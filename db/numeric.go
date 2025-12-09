package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func LoadNumericFields(
	cfg DBConfig,
	table string,
	jsonbCol string,
) (map[string]map[string]struct{}, error) {

	// PostgreSQL connection with DBConfig
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect failed: %v", err)
	}
	defer db.Close()

	query := fmt.Sprintf(`
        WITH topic_samples AS (
            SELECT DISTINCT ON (%[2]s->>'topic') 
                   %[2]s->>'topic' as topic,
                   %[2]s as data
            FROM %[1]s
            WHERE %[2]s IS NOT NULL
              AND jsonb_typeof(%[2]s) = 'object'
              AND %[2]s != '{}'
              AND %[2]s ? 'topic'
            ORDER BY %[2]s->>'topic'
        )
        SELECT ts.topic, field.key
        FROM topic_samples ts,
        LATERAL jsonb_each(ts.data) AS field(key, value)
        WHERE jsonb_typeof(field.value) = 'number'
          AND field.key != 'topic'
        ORDER BY ts.topic, field.key;
    `, table, jsonbCol)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query numeric fields failed: %v", err)
	}
	defer rows.Close()

	result := make(map[string]map[string]struct{})

	for rows.Next() {
		var topic, field string
		if err := rows.Scan(&topic, &field); err != nil {
			return nil, err
		}
		if _, ok := result[topic]; !ok {
			result[topic] = make(map[string]struct{})
		}
		result[topic][field] = struct{}{}
	}

	return result, rows.Err()
}
