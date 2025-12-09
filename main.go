package main

import (
	"fmt"
	"log"

	"sqlalchemy/converter"
	"sqlalchemy/db"
)

func main() {
	// 1️⃣ 配置数据库
	cfg := db.DBConfig{
		Host:     "127.0.0.1",
		Port:     5432,
		DBName:   "tsdb",
		User:     "postgres",
		Password: "123456",
	}

	// 2️⃣ 动态参数
	table := "tsdb_table"   // TSDB 表名
	payloadCol := "payload" // JSONB 列名
	topic := "topic"        // topic 字段名

	fmt.Println("🔍 Loading numeric fields from PostgreSQL...")

	// 3️⃣ 加载 numeric fields
	topicFields, err := db.LoadNumericFields(cfg, table, payloadCol)
	if err != nil {
		log.Fatalf("❌ load numeric fields failed: %v", err)
	}

	numericFields := make(map[string]struct{})
	for _, fields := range topicFields {
		for field := range fields {
			numericFields[field] = struct{}{}
		}
	}

	fmt.Println("📦 Numeric fields loaded:", numericFields)

	// 4️⃣ 原始 SQL
	originalSQL := `
	SELECT code,
	       COUNT(*) as alarm_count,
	       MAX(value) as max_value,
	       MIN(value) as min_value,
	       AVG(value) as avg_value
	FROM factory_alarm_pump_alarm
	WHERE threshold > 20 OR threshold < 16
	GROUP BY code
	HAVING COUNT(*) > 0
	ORDER BY alarm_count DESC, max_value DESC
	LIMIT 10;
	`
	mapper, err := converter.NewSQLMapper(originalSQL, numericFields, table, payloadCol, topic)
	if err != nil {
		log.Fatalf("❌ SQL parse/map failed: %v", err)
	}

	// 6️⃣ 输出结果
	fmt.Println("\n====================================")
	fmt.Println("Original SQL:")
	fmt.Println(originalSQL)

	fmt.Println("\nMapped SQL:")
	fmt.Println(mapper.MappedSQL)

	fmt.Println("\n📌 Done.")
}
