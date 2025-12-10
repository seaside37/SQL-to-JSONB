package converter

import (
	"fmt"
	"regexp"
	"strings"

	"sqlalchemy/db"
)

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

	cfg := db.DBConfig{
		Host:     host,
		Port:     port,
		DBName:   dbName,
		User:     user,
		Password: password,
	}

	topicFields, err := db.LoadNumericFields(cfg, table, payloadCol)
	if err != nil {
		return "", fmt.Errorf("load numeric fields failed: %w", err)
	}

	numericFields := make(map[string]struct{})
	for _, fields := range topicFields {
		for f := range fields {
			numericFields[f] = struct{}{}
		}
	}

	// 处理 SELECT * 的情况
	allFields, err := db.LoadAllFields(cfg, table, payloadCol)
	if err != nil {
		return "", fmt.Errorf("load all fields failed: %w", err)
	}
	re := regexp.MustCompile(`(?i)SELECT\s+\*\s+FROM\s+(\S+)`)
	matches := re.FindStringSubmatch(originalSQL)
	if len(matches) == 2 {
		fromTable := matches[1]

		fields, ok := allFields[fromTable]
		if ok && len(fields) > 0 {
			fieldExprs := make([]string, 0, len(fields))
			for _, f := range fields {
				expr := fmt.Sprintf("(%s ->> '%s') AS %s", payloadCol, f, f)
				fieldExprs = append(fieldExprs, expr)
			}
			fieldsStr := strings.Join(fieldExprs, ", ")
			// 替换 * 为具体字段
			originalSQL = re.ReplaceAllString(originalSQL, fmt.Sprintf("SELECT %s FROM %s", fieldsStr, fromTable))
		}
	}

	mapper, err := NewSQLMapper(originalSQL, numericFields, table, payloadCol, topicField)
	if err != nil {
		return "", fmt.Errorf("SQL parse/map failed: %w", err)
	}

	return mapper.MappedSQL, nil
}
