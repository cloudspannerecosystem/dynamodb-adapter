package translator

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/cloudspannerecosystem/dynamodb-adapter/translator/PartiQLParser/parser"
)

// Funtion to create lexer and parser object for the partiql query
func NewPartiQLParser(partiQL string, isDebug bool) (*parser.PartiQLParser, error) {
	if partiQL == "" {
		return nil, fmt.Errorf("invalid input string")
	}

	lexer := parser.NewPartiQLLexer(antlr.NewInputStream(partiQL))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewPartiQLParser(stream)
	if p == nil {
		return nil, fmt.Errorf("error while creating parser object")
	}
	if !isDebug {
		p.RemoveErrorListeners()
	}
	return p, nil
}

func trimSingleQuotes(s string) string {
	// Check if the string starts and ends with single quotes
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		// Remove the quotes from the beginning and end
		s = s[1 : len(s)-1]
	}
	return s
}

func formSpannerSelectQuery(selectQueryMap *SelectQueryMap, whereConditions []Condition) (string, error) {
	spannerQuery := "SELECT "

	// Construct projection columns or use * if empty
	if len(selectQueryMap.ProjectionColumns) == 0 {
		spannerQuery += "* "
	} else {
		spannerQuery += strings.Join(selectQueryMap.ProjectionColumns, ", ") + " "
	}

	spannerQuery += "FROM " + selectQueryMap.Table

	// Construct WHERE clause
	if len(whereConditions) > 0 {
		var whereClauses []string
		for i, cond := range whereConditions {
			clause := fmt.Sprintf("%s %s %s", cond.Column, cond.Operator, cond.Value)

			// Add logical operators if it's not the first condition
			if i > 0 {
				if cond.ANDOpr != "" {
					clause = cond.ANDOpr + " " + clause
				} else if cond.OROpr != "" {
					clause = cond.OROpr + " " + clause
				}
			}
			whereClauses = append(whereClauses, clause)
		}
		// Join the WHERE clauses using the appropriate spacing
		spannerQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Append ORDER BY clause if present
	if len(selectQueryMap.OrderBy) > 0 {
		spannerQuery += " ORDER BY " + strings.Join(selectQueryMap.OrderBy, ", ")
	}

	// Append LIMIT clause if present
	if selectQueryMap.Limit != "" {
		spannerQuery += " LIMIT " + strings.Trim(selectQueryMap.Limit, "LIMIT")
	}

	// Append OFFSET clause if present
	if selectQueryMap.Offset != "" {
		spannerQuery += " OFFSET " + strings.Trim(selectQueryMap.Offset, "OFFSET")
	}

	return spannerQuery, nil
}
