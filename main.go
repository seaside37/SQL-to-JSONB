// package main

// import (
// 	"fmt"
// 	"log"

// 	"sqlalchemy/converter"
// )

// func main() {
// 	// 原始 SQL
// 	// originalSQL := `
// 	// 	SELECT c.code, value, message
// 	// 	FROM factory_alarm_pump_alarm
// 	// 	WHERE threshold > 20 OR threshold < 16
// 	// 	ORDER BY value DESC
// 	// 	LIMIT 10
// 	// `
// 	originalSQL :=
// 		`SELECT code,
// 		COUNT(*) as alarm_count,
// 		MAX(value) as max_value,
// 		MIN(value) as min_value,
// 		AVG(value) as avg_value
// 		FROM factory_alarm_pump_alarm
// 		WHERE threshold > 20 OR threshold < 16
// 		GROUP BY code
// 		HAVING COUNT(*) > 0
// 		ORDER BY alarm_count DESC, max_value DESC
// 		LIMIT 10;`

// 	// 调用 converter 解析并映射
// 	numericFields := map[string]struct{}{
// 		"threshold": {},
// 		"value":     {},
// 	}

// 	mapper, err := converter.ParseAndMapSQL(originalSQL, numericFields)
// 	if err != nil {
// 		log.Fatalf("SQL parse/map failed: %v", err)
// 	}

//		// 输出结果
//		fmt.Println("Original SQL:")
//		fmt.Println(mapper.OriginalSQL)
//		fmt.Println("\nMapped SQL:")
//		fmt.Println(mapper.MappedSQL)
//	}

// package main

// import (
// 	"fmt"
// 	"log"

// 	"sqlalchemy/db"
// )

// func main() {

// 	// 1) 初始化数据库配置（使用 DBConfig）
// 	cfg := db.DBConfig{
// 		Host:     "127.0.0.1",
// 		Port:     5432,
// 		DBName:   "tsdb",
// 		User:     "postgres",
// 		Password: "123456",
// 	}

// 	// 2) 目标 JSONB 表及列
// 	table := "tsdb_table"
// 	jsonbCol := "payload"

// 	fmt.Println("🔍 Loading numeric fields from PostgreSQL...")

// 	// 3) 调用优化后的函数（只需传 DBConfig + 表和列名）
// 	topicFields, err := db.LoadNumericFields(cfg, table, jsonbCol)
// 	if err != nil {
// 		log.Fatalf("❌ load numeric fields failed: %v", err)
// 	}

// 	fmt.Println("\n🚀 Numeric Fields by Topic")
// 	fmt.Println("====================================")

// 	// 4) 输出结果
// 	for topic, fields := range topicFields {
// 		fmt.Printf("📌 Topic: %s\n", topic)
// 		for field := range fields {
// 			fmt.Printf("  - %s\n", field)
// 		}
// 		fmt.Println("------------------------------------")
// 	}

//		fmt.Println("✅ Done.")
//	}
package main

import (
	"fmt"
	"log"

	"sqlalchemy/converter"
	"sqlalchemy/db"
)

func main() {
	cfg := db.DBConfig{
		Host:     "127.0.0.1",
		Port:     5432,
		DBName:   "tsdb",
		User:     "postgres",
		Password: "123456",
	}

	table := "tsdb_table"
	jsonbCol := "payload"

	fmt.Println("🔍 Loading numeric fields from PostgreSQL...")

	topicFields, err := db.LoadNumericFields(cfg, table, jsonbCol)
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

	originalSQL :=
		`SELECT code,
		COUNT(*) as alarm_count,
		MAX(value) as max_value,
		MIN(value) as min_value,
		AVG(value) as avg_value
		FROM factory_alarm_pump_alarm
		WHERE threshold > 20 OR threshold < 16
		GROUP BY code
		HAVING COUNT(*) > 0
		ORDER BY alarm_count DESC, max_value DESC
		LIMIT 10;`

	mapper, err := converter.ParseAndMapSQL(originalSQL, numericFields)
	if err != nil {
		log.Fatalf("❌ SQL parse/map failed: %v", err)
	}

	fmt.Println("\n====================================")
	fmt.Println("Original SQL:")
	fmt.Println(originalSQL)

	fmt.Println("\nMapped SQL:")
	fmt.Println(mapper.MappedSQL)

	fmt.Println("\n📌 Done.")
}
