package converter

import (
	"fmt"
	"strings"

	"github.com/xwb1989/sqlparser"
)

func NewSQLMapper(sql string, numericFields map[string]struct{}, table, payloadCol, topic string) (*SQLMapper, error) {
	mapper := &SQLMapper{
		OriginalSQL:   sql,
		NumericFields: numericFields,
		TableName:     table,
		PayloadCol:    payloadCol,
		Topic:         topic,
	}

	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return nil, err
	}

	switch stmt := stmt.(type) {
	case *sqlparser.Select:
		mapper.MappedSQL = mapper.mapSelectStatement(stmt)
	case *sqlparser.Union:
		mapper.MappedSQL = mapper.mapUnionStatement(stmt)
	default:
		return nil, fmt.Errorf("SQL Type not supported: %T", stmt)
	}

	return mapper, nil
}

func ParseAndMapSQL(sql string, numericFields map[string]struct{}) (*SQLMapper, error) {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return nil, err
	}

	mapper := &SQLMapper{
		OriginalSQL:   sql,
		NumericFields: numericFields,
	}

	switch stmt := stmt.(type) {
	case *sqlparser.Select:
		mapper.MappedSQL = mapper.mapSelectStatement(stmt)
	case *sqlparser.Union:
		mapper.MappedSQL = mapper.mapUnionStatement(stmt)
	default:
		return nil, fmt.Errorf("SQL Type not supported: %T", stmt)
	}

	return mapper, nil
}

func (mapper *SQLMapper) mapSelectStatement(selectStmt *sqlparser.Select) string {
	aliasMap := make(map[string]string)
	selectExprs := make([]string, 0, len(selectStmt.SelectExprs))
	for _, selectExpr := range selectStmt.SelectExprs {
		mapped := mapper.mapSelectExpr(selectExpr)
		selectExprs = append(selectExprs, mapped)

		if ae, ok := selectExpr.(*sqlparser.AliasedExpr); ok {
			alias := ae.As.String()
			if alias == "" {
				switch col := ae.Expr.(type) {
				case *sqlparser.ColName:
					alias = col.Name.String()
				case *sqlparser.FuncExpr:
					alias = col.Name.String()
				default:
					alias = sqlparser.String(ae.Expr)
				}
			}
			aliasMap[alias] = alias
		}
	}

	fromClause := ""
	hasTableCondition := false
	if selectStmt.From != nil {
		fromParts := make([]string, 0, len(selectStmt.From))
		for _, tableExpr := range selectStmt.From {
			mappedTable, hasCondition := mapper.mapTableExprWithCondition(tableExpr)
			fromParts = append(fromParts, mappedTable)
			if hasCondition {
				hasTableCondition = true
			}
		}
		fromClause = "FROM " + strings.Join(fromParts, ", ")
	}

	parts := []string{
		"SELECT",
		strings.Join(selectExprs, ", "),
	}
	if fromClause != "" {
		parts = append(parts, fromClause)
	}

	whereConditions := make([]string, 0)
	if hasTableCondition {
		tableConditions := mapper.extractTableConditions(selectStmt.From)
		if len(tableConditions) > 0 {
			whereConditions = append(whereConditions, tableConditions...)
		}
	}
	if selectStmt.Where != nil {
		whereConditions = append(whereConditions, mapper.mapExpr(selectStmt.Where.Expr))
	}
	if len(whereConditions) > 0 {
		whereClause := ""
		if len(whereConditions) >= 2 {
			if _, ok := selectStmt.Where.Expr.(*sqlparser.OrExpr); ok {
				whereClause = fmt.Sprintf("%s AND (%s)", whereConditions[0], whereConditions[1])
			} else {
				whereClause = strings.Join(whereConditions, " AND ")
			}
		} else {
			whereClause = whereConditions[0]
		}
		parts = append(parts, "WHERE", whereClause)
	}

	if selectStmt.GroupBy != nil {
		groupByParts := make([]string, 0, len(selectStmt.GroupBy))
		for _, expr := range selectStmt.GroupBy {
			colName := ""
			if c, ok := expr.(*sqlparser.ColName); ok {
				colName = c.Name.String()
			}
			if alias, ok := aliasMap[colName]; ok {
				groupByParts = append(groupByParts, alias)
			} else {
				groupByParts = append(groupByParts, mapper.mapExpr(expr))
			}
		}
		parts = append(parts, "GROUP BY", strings.Join(groupByParts, ", "))
	}

	if selectStmt.Having != nil {
		havingStr := ""
		switch h := selectStmt.Having.Expr.(type) {
		case *sqlparser.FuncExpr:
			if alias, ok := aliasMap[h.Name.String()]; ok {
				havingStr = alias
			} else {
				havingStr = mapper.mapExpr(selectStmt.Having.Expr)
			}
		case *sqlparser.ColName:
			if alias, ok := aliasMap[h.Name.String()]; ok {
				havingStr = alias
			} else {
				havingStr = mapper.mapExpr(h)
			}
		default:
			havingStr = mapper.mapExpr(selectStmt.Having.Expr)
		}
		parts = append(parts, "HAVING", havingStr)
	}

	if selectStmt.OrderBy != nil {
		orderByParts := make([]string, 0, len(selectStmt.OrderBy))
		for _, order := range selectStmt.OrderBy {
			orderStr := ""
			switch c := order.Expr.(type) {
			case *sqlparser.ColName:
				if alias, ok := aliasMap[c.Name.String()]; ok {
					orderStr = alias
				} else {
					orderStr = mapper.mapExpr(order.Expr)
				}
			case *sqlparser.FuncExpr:
				if alias, ok := aliasMap[c.Name.String()]; ok {
					orderStr = alias
				} else {
					orderStr = mapper.mapExpr(order.Expr)
				}
			default:
				orderStr = mapper.mapExpr(order.Expr)
			}

			if order.Direction != "" {
				orderStr += " " + order.Direction
			}
			orderByParts = append(orderByParts, orderStr)
		}
		parts = append(parts, "ORDER BY", strings.Join(orderByParts, ", "))
	}

	if selectStmt.Limit != nil {
		limitParts := []string{"LIMIT", mapper.mapExpr(selectStmt.Limit.Rowcount)}
		if selectStmt.Limit.Offset != nil {
			limitParts = append([]string{"OFFSET", mapper.mapExpr(selectStmt.Limit.Offset)}, limitParts...)
		}
		parts = append(parts, strings.Join(limitParts, " "))
	}

	return strings.Join(parts, " ")
}

func (mapper *SQLMapper) mapTableExprWithCondition(tableExpr sqlparser.TableExpr) (string, bool) {
	switch expr := tableExpr.(type) {
	case *sqlparser.AliasedTableExpr:
		return mapper.mapAliasedTableExprWithCondition(expr)
	case *sqlparser.JoinTableExpr:
		return mapper.mapJoinTableExpr(expr), true
	case *sqlparser.ParenTableExpr:
		innerParts := make([]string, 0, len(expr.Exprs))
		hasCondition := false
		for _, innerExpr := range expr.Exprs {
			mapped, innerHasCondition := mapper.mapTableExprWithCondition(innerExpr)
			innerParts = append(innerParts, mapped)
			if innerHasCondition {
				hasCondition = true
			}
		}
		return "(" + strings.Join(innerParts, ", ") + ")", hasCondition
	default:
		return sqlparser.String(tableExpr), false
	}
}

func (mapper *SQLMapper) mapAliasedTableExprWithCondition(table *sqlparser.AliasedTableExpr) (string, bool) {
	switch expr := table.Expr.(type) {
	case sqlparser.TableName:
		tableName := expr.Name.String()
		if tableName != "" && tableName != mapper.TableName {
			mappedTable := mapper.TableName
			if !table.As.IsEmpty() {
				mappedTable += " AS " + table.As.String()
			}
			return mappedTable, true
		}
		return sqlparser.String(table), false
	case *sqlparser.Subquery:
		if selectStmt, ok := expr.Select.(*sqlparser.Select); ok {
			mappedSubquery := mapper.mapSelectStatement(selectStmt)
			result := "(" + mappedSubquery + ")"
			if !table.As.IsEmpty() {
				result += " AS " + table.As.String()
			}
			return result, false
		}
	}
	return sqlparser.String(table), false
}

func (mapper *SQLMapper) extractTableConditions(tableExprs sqlparser.TableExprs) []string {
	conditions := make([]string, 0)
	for _, tableExpr := range tableExprs {
		mapper.extractConditionsFromTableExpr(tableExpr, &conditions)
	}
	return conditions
}

func (mapper *SQLMapper) extractConditionsFromTableExpr(tableExpr sqlparser.TableExpr, conditions *[]string) {
	switch expr := tableExpr.(type) {
	case *sqlparser.AliasedTableExpr:
		switch table := expr.Expr.(type) {
		case sqlparser.TableName:
			tableName := table.Name.String()
			if tableName != "" && tableName != mapper.TableName {
				condition := fmt.Sprintf("(%s ->> '%s') = '%s'", mapper.PayloadCol, mapper.Topic, tableName)
				if !expr.As.IsEmpty() {
					condition = fmt.Sprintf("(%s.%s ->> '%s') = '%s'", expr.As.String(), mapper.PayloadCol, mapper.Topic, tableName)
				}
				*conditions = append(*conditions, condition)
			}
		}
	case *sqlparser.JoinTableExpr:
		mapper.extractConditionsFromTableExpr(expr.LeftExpr, conditions)
		mapper.extractConditionsFromTableExpr(expr.RightExpr, conditions)
	case *sqlparser.ParenTableExpr:
		for _, innerExpr := range expr.Exprs {
			mapper.extractConditionsFromTableExpr(innerExpr, conditions)
		}
	}
}

func (mapper *SQLMapper) mapJoinTableExpr(join *sqlparser.JoinTableExpr) string {
	leftTable, _ := mapper.mapTableExprWithCondition(join.LeftExpr)
	rightTable, _ := mapper.mapTableExprWithCondition(join.RightExpr)
	condition := mapper.mapExpr(join.Condition.On)

	return fmt.Sprintf("%s %s %s ON %s", leftTable, join.Join, rightTable, condition)
}

func (mapper *SQLMapper) mapSelectExpr(expr sqlparser.SelectExpr) string {
	switch se := expr.(type) {
	case *sqlparser.AliasedExpr:
		if fn, ok := se.Expr.(*sqlparser.FuncExpr); ok {
			mapped := mapper.mapExpr(fn)
			alias := ""
			if !se.As.IsEmpty() {
				alias = se.As.String()
			} else {
				alias = fn.Name.String()
			}
			return fmt.Sprintf("%s AS %s", mapped, alias)
		}

		if col, ok := se.Expr.(*sqlparser.ColName); ok {
			colName := col.Name.String()
			var alias string
			if !se.As.IsEmpty() {
				alias = se.As.String()
			} else if col.Qualifier.Name.String() != "" {
				alias = col.Qualifier.Name.String()
			} else {
				alias = colName
			}
			return fmt.Sprintf("(%s ->> '%s') AS %s", mapper.PayloadCol, colName, alias)
		}

		mapped := mapper.mapExpr(se.Expr)
		alias := se.As.String()
		if alias == "" {
			alias = mapped
		}
		return fmt.Sprintf("%s AS %s", mapped, alias)

	case *sqlparser.StarExpr:
		return "*"
	default:
		return sqlparser.String(expr)
	}
}

func (mapper *SQLMapper) mapExpr(expr sqlparser.Expr) string {
	if expr == nil {
		return ""
	}

	switch e := expr.(type) {
	case *sqlparser.ColName:
		columnName := e.Name.String()
		mapped := fmt.Sprintf("(%s ->> '%s')", mapper.PayloadCol, columnName)
		if _, ok := mapper.NumericFields[columnName]; ok {
			mapped += "::FLOAT"
		}
		if e.Qualifier.Name.String() != "" {
			mapped = fmt.Sprintf("(%s.%s ->> '%s')", e.Qualifier.Name.String(), mapper.PayloadCol, columnName)
			if _, ok := mapper.NumericFields[columnName]; ok {
				mapped += "::FLOAT"
			}
		}
		return mapped

	case *sqlparser.SQLVal:
		switch e.Type {
		case sqlparser.StrVal:
			return "'" + string(e.Val) + "'"
		default:
			return string(e.Val)
		}
	case *sqlparser.BinaryExpr:
		return fmt.Sprintf("%s %s %s", mapper.mapExpr(e.Left), e.Operator, mapper.mapExpr(e.Right))
	case *sqlparser.ParenExpr:
		return "(" + mapper.mapExpr(e.Expr) + ")"
	case *sqlparser.ComparisonExpr:
		return fmt.Sprintf("%s %s %s", mapper.mapExpr(e.Left), e.Operator, mapper.mapExpr(e.Right))
	case *sqlparser.IsExpr:
		left := mapper.mapExpr(e.Expr)
		if e.Operator == "is not null" {
			return fmt.Sprintf("%s IS NOT NULL", left)
		}
		return fmt.Sprintf("%s IS %s", left, e.Operator)
	case *sqlparser.FuncExpr:
		args := make([]string, 0, len(e.Exprs))
		for _, expr := range e.Exprs {
			switch ae := expr.(type) {
			case *sqlparser.AliasedExpr:
				args = append(args, mapper.mapExpr(ae.Expr))
			case *sqlparser.StarExpr:
				args = append(args, "*")
			default:
				args = append(args, mapper.mapExpr(ae.(sqlparser.Expr)))
			}
		}
		return fmt.Sprintf("%s(%s)", e.Name.String(), strings.Join(args, ", "))
	case *sqlparser.OrExpr:
		return fmt.Sprintf("%s OR %s", mapper.mapExpr(e.Left), mapper.mapExpr(e.Right))
	case *sqlparser.AndExpr:
		return fmt.Sprintf("%s AND %s", mapper.mapExpr(e.Left), mapper.mapExpr(e.Right))
	case *sqlparser.RangeCond:
		return fmt.Sprintf("%s %s %s AND %s", mapper.mapExpr(e.Left), e.Operator, mapper.mapExpr(e.From), mapper.mapExpr(e.To))
	case *sqlparser.UnaryExpr:
		return fmt.Sprintf("%s%s", e.Operator, mapper.mapExpr(e.Expr))
	default:
		return sqlparser.String(expr)
	}
}

func (mapper *SQLMapper) mapUnionStatement(union *sqlparser.Union) string {
	left := mapper.mapSelectStatement(union.Left.(*sqlparser.Select))
	right := mapper.mapSelectStatement(union.Right.(*sqlparser.Select))
	parts := []string{left, "UNION"}
	if strings.Contains(strings.ToUpper(sqlparser.String(union)), "UNION ALL") {
		parts = append(parts, "ALL")
	}
	parts = append(parts, right)
	if union.OrderBy != nil {
		orderByParts := make([]string, 0, len(union.OrderBy))
		for _, order := range union.OrderBy {
			orderStr := mapper.mapExpr(order.Expr)
			if order.Direction != "" {
				orderStr += " " + order.Direction
			}
			orderByParts = append(orderByParts, orderStr)
		}
		parts = append(parts, "ORDER BY", strings.Join(orderByParts, ", "))
	}
	if union.Limit != nil {
		limitParts := []string{"LIMIT", mapper.mapExpr(union.Limit.Rowcount)}
		if union.Limit.Offset != nil {
			limitParts = append([]string{"OFFSET", mapper.mapExpr(union.Limit.Offset)}, limitParts...)
		}
		parts = append(parts, strings.Join(limitParts, " "))
	}
	return strings.Join(parts, " ")
}
