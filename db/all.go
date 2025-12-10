package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// LoadAllFields returns a map where topic is the key, and the value is a slice of all jsonb field names except "topic"
func LoadAllFields(
	cfg DBConfig,
	table string,
	jsonbCol string,
) (map[string][]string, error) {

	// PostgreSQL connection
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
        WHERE field.key != 'topic'
        ORDER BY ts.topic, field.key;
    `, table, jsonbCol)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query all fields failed: %v", err)
	}
	defer rows.Close()

	result := make(map[string][]string)

	for rows.Next() {
		var topic, field string
		if err := rows.Scan(&topic, &field); err != nil {
			return nil, err
		}
		result[topic] = append(result[topic], field)
	}

	return result, rows.Err()
}
