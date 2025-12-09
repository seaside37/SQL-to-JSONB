# SQL-to-JSONB转换器

## 📋 项目概述

SQL-to-JSONB转换器是一个用于将标准SQL查询转换为针对PostgreSQL JSONB格式数据的查询。它适用于需要从JSONB字段中提取和查询结构化数据的场景。

### 名词解释：
- **表名**：进行查询的数据所在的实际 PostgreSQL 表。
- **JSONB 列名**：存放 JSONB 格式结构化数据的字段。
- **JSONB 主题字段名（topic）**：用于标识不同 JSON 结构所属的"逻辑表"。所有拥有相同 topic 的 JSONB 记录，结构必须完全一致。

### 适用场景：
当系统将原本分散在多个数据表中的数据统一存储到一张 PostgreSQL 表中时，每条原始表记录会转换为 JSONB 格式并写入这张表。此时：

- **表名**：由用户自定义（例如：tsdb_table）
- **JSONB 列名**：由用户自定义（例如：payload）
- **JSONB 主题字段名**：使用原始表名作为 topic（例如：factory_alarm_pump_alarm）

该工具可根据这些信息，将标准 SQL 自动转换成面向 JSONB 的查询语句。

## 🏗️ 项目结构
```
project/  
├── converter/ # SQL转换核心模块  
│ ├── converter.go  
│ └── types.go  
├── db/ # 数据库相关模块  
│ ├── dbconfig.go # 数据库配置  
│ └── numeric.go # 数值字段检测  
└── README.md  
```

## ✨ 核心特性

### 1. 智能SQL转换
- 自动添加基于topic字段的过滤条件，将表名映射为真实的表名
- 支持JOIN、子查询、UNION等复杂查询

### 2. JSONB字段处理
- 自动将列引用转换为 `(payload ->> 'column_name')`
- 智能识别数值字段并添加 `::FLOAT` 类型转换

### 3. 语法支持
- ✅ SELECT语句
- ✅ WHERE条件（AND/OR）
- ✅ JOIN操作（INNER/LEFT/RIGHT JOIN）
- ✅ 聚合函数（COUNT、SUM、AVG等）
- ✅ GROUP BY / HAVING
- ✅ ORDER BY / LIMIT / OFFSET
- ✅ UNION

## 📖 API 参考

### converter 包

#### 1. `ParseAndMapSQL(sql string, numericFields map[string]struct{}) (*SQLMapper, error)`

**作用**：解析并转换SQL语句。

**参数**：
- `sql`: 要转换的标准SQL语句
- `numericFields`: 数值字段集合（通过`db.LoadNumericFields`获取）

**返回值**：
- `*SQLMapper`: 包含原始SQL和转换后SQL的结构体
- `error`: 转换过程中的错误

#### 2. SQLMapper 结构体

```go
type SQLMapper struct {
    OriginalSQL   string                 // 原始SQL语句
    MappedSQL     string                 // 转换后的SQL语句
    NumericFields map[string]struct{}    // 数值字段集合
    
    TableName     string                 // 原始数据库数据表名
    PayloadCol    string                 // 原始数据库JSONB列名
    Topic         string                 // 原始数据库JSONB主题字段名
}
```
### db 包

#### 1. `LoadNumericFields(cfg DBConfig, table string, jsonbCol string) (map[string]map[string]struct{}, error)`

**作用**：从数据库加载所有topic的数值字段。

**参数**：
- `cfg`: 数据库配置
- `table`: 表名
- `jsonbCol`: JSONB列名（默认为"payload"）

**返回值**：
- `map[string]map[string]struct{}`: 按topic分组的数值字段集合
- `error`: 查询错误

#### 2. DBConfig 结构体

```go
type DBConfig struct {
    Host     string
    Port     int
    User     string
    Database string
    Password string
    SSLMode  string
}
```

## 🔍 SQL转换示例

### 示例1 简单查询
**输入SQL：**

```sql
SELECT id, name, price FROM products WHERE category = 'electronics'
-- TableName=order_table PayloadCol=payload Topic=topic
```
**输出SQL：**
```sql
SELECT 
    (payload ->> 'id') AS id,
    (payload ->> 'name') AS name,
    (payload ->> 'price')::FLOAT AS price
FROM order_table
WHERE (payload ->> 'topic') = 'products'
  AND (payload ->> 'category') = 'electronics'
```

### 示例2 带聚合函数的查询
**输入SQL：**
```sql
SELECT c.code, value, message
FROM factory_alarm_pump_alarm
WHERE threshold > 20 OR threshold < 16
ORDER BY value DESC
LIMIT 10
-- TableName=tsdb_table PayloadCol=payload Topic=topic
```
**输出SQL：**
```sql
SELECT 
    (payload ->> 'code') AS c,
    (payload ->> 'value') AS value,
    (payload ->> 'message') AS message
FROM tsdb_table
WHERE 
    (payload ->> 'topic') = 'factory_alarm_pump_alarm'
    AND (
        (payload ->> 'threshold')::FLOAT > 20
        OR (payload ->> 'threshold')::FLOAT < 16
    )
ORDER BY 
    value DESC
LIMIT 10;
```

### 示例3 带JOIN的查询
**输入SQL：**

```sql
SELECT o.order_id, c.customer_name, SUM(oi.quantity * oi.price) as total
FROM orders o
JOIN customers c ON o.customer_id = c.customer_id
JOIN order_items oi ON o.order_id = oi.order_id
WHERE o.order_date >= '2024-01-01'
GROUP BY o.order_id, c.customer_name
-- TableName=order_table PayloadCol=payload Topic=topic
```
**输出SQL：**
```sql
SELECT 
    (o.payload ->> 'order_id') AS order_id,
    (c.payload ->> 'customer_name') AS customer_name,
    SUM((oi.payload ->> 'quantity')::FLOAT * (oi.payload ->> 'price')::FLOAT) as total
FROM tsdb_table AS o
JOIN tsdb_table AS c ON (o.payload ->> 'customer_id') = (c.payload ->> 'customer_id')
JOIN tsdb_table AS oi ON (o.payload ->> 'order_id') = (oi.payload ->> 'order_id')
WHERE o.payload ->> 'topic' = 'orders'
  AND c.payload ->> 'topic' = 'customers'
  AND oi.payload ->> 'topic' = 'order_items'
  AND (o.payload ->> 'order_date') >= '2024-01-01'
GROUP BY (o.payload ->> 'order_id'), (c.payload ->> 'customer_name')
```

## 🚀 快速开始
### JSONB数据结构
```json
{
  "code": "PUMP_VIBRATION",
  "level": "LOW",
  "topic": "factory_alarm_pump_alarm",
  "value": 28.61,
  "message": "Abnormal vibration detected",
  "threshold": 12.93
}
```
### 基本用法
```go
package main

import (
    "fmt"
    "log"
    
    "sqlalchemy/converter"
    "sqlalchemy/db"
)

func main() {
    // 1. 配置数据库连接
    cfg := db.DBConfig{
        Host:     "127.0.0.1",      // PostgreSQL 主机地址
        Port:     5432,             // 端口号
        DBName:   "tsdb",           // 数据库名
        User:     "postgres",       // 用户名
        Password: "your_password",  // 密码
    }
    
    // 2. 指定数据表信息
    table := "tsdb_table"      // 你的 TimescaleDB 表名
    payloadCol := "payload"    // JSONB 字段列名
    topic := "topic"           // JSONB 中标识数据类型的字段名
    
    // 3. 自动检测数值类型字段
    topicFields, err := db.LoadNumericFields(cfg, table, payloadCol)
    if err != nil {
        log.Fatalf("❌ 加载数值字段失败: %v", err)
    }
    
    // 4. 合并所有 topic 的数值字段
    numericFields := make(map[string]struct{})
    for _, fields := range topicFields {
        for field := range fields {
            numericFields[field] = struct{}{}
        }
    }
    
    // 5. 定义要转换的 SQL 查询
    originalSQL := `
        SELECT device_id, 
               AVG(temperature) as avg_temp,
               MAX(pressure) as max_pressure
        FROM sensor_data
        WHERE temperature > 30
        GROUP BY device_id
        HAVING COUNT(*) > 100
        ORDER BY avg_temp DESC
        LIMIT 20
    `
    
    // 6. 执行 SQL 转换
    mapper, err := converter.NewSQLMapper(
        originalSQL,      // 原始 SQL
        numericFields,    // 数值字段映射
        table,            // 物理表名
        payloadCol,       // JSONB 列名
        topic,            // topic 字段名
    )
    if err != nil {
        log.Fatalf("❌ SQL 转换失败: %v", err)
    }
    
    // 7. 输出结果
    fmt.Println("📝 原始 SQL:")
    fmt.Println(mapper.OriginalSQL)
    
    fmt.Println("\n🔧 转换后的 SQL:")
    fmt.Println(mapper.MappedSQL)
}
```