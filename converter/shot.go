package converter

import (
	"fmt"

	"sqlalchemy/db"
)

// MapSQLShot
// Wrap entire SQL mapping pipeline:
// - Load numeric fields from PostgreSQL
// - Build SQL mapper
// - Return mapped SQL
//
// Parameters:
//
//	host, port, dbName, user, password : DB connection info
//	table        : TSDB table name
//	payloadCol   : JSONB column
//	topicField   : JSONB topic field name
//	originalSQL  : original SQL string
//
// Returns:
//
//	mappedSQL string, error
//
// Usage:
//
//	mapped, err := MapSQLOnce("127.0.0.1", 5432, "tsdb", "postgres", "123456",
//	                          "tsdb_table", "payload", "topic", sql)
func MapSQLShot(
	host string,
	port int,
	dbName string,
	user string,
	password string,
	table string,
	payloadCol string,
	topicField string,
	originalSQL string,
) (string, error) {

	// 1️⃣ Build DB config
	cfg := db.DBConfig{
		Host:     host,
		Port:     port,
		DBName:   dbName,
		User:     user,
		Password: password,
	}

	// 2️⃣ Load numeric fields from PostgreSQL
	topicFields, err := db.LoadNumericFields(cfg, table, payloadCol)
	if err != nil {
		return "", fmt.Errorf("load numeric fields failed: %w", err)
	}

	// Flatten the topic→fields map to a simple field set
	numericFields := make(map[string]struct{})
	for _, fields := range topicFields {
		for f := range fields {
			numericFields[f] = struct{}{}
		}
	}

	// 3️⃣ Create SQL mapper
	mapper, err := NewSQLMapper(originalSQL, numericFields, table, payloadCol, topicField)
	if err != nil {
		return "", fmt.Errorf("SQL parse/map failed: %w", err)
	}

	// 4️⃣ Return result
	return mapper.MappedSQL, nil
}
