package main

import (
	"fmt"
	"log"

	"sqlalchemy/converter"
)

func main() {

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

	mappedSQL, err := converter.MapSQLShot(
		"127.0.0.1",
		5432,
		"tsdb",
		"postgres",
		"123456",
		"tsdb_table",
		"payload",
		"topic",
		originalSQL,
	)
	if err != nil {
		log.Fatalf("❌ Error: %v", err)
	}

	fmt.Println("✨ Mapped SQL:")
	fmt.Println(mappedSQL)
}
