package translator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimSingleQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "With single quotes",
			input:    "'Hello, World!'",
			expected: "Hello, World!",
		},
		{
			name:     "Without single quotes",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "Only quotes",
			input:    "''",
			expected: "",
		},
		{
			name:     "Spaces with quotes",
			input:    "'   '",
			expected: "   ", // maintaining spaces
		},
		{
			name:     "Single quote at start only",
			input:    "'Hello, World!",
			expected: "'Hello, World!", // single quote not at the end should remain
		},
		{
			name:     "Single quote at end only",
			input:    "Hello, World!'",
			expected: "Hello, World!'", // single quote not at the start should remain
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimSingleQuotes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormSpannerSelectQuery(t *testing.T) {
	tests := []struct {
		name            string
		selectQueryMap  SelectQueryMap
		whereConditions []Condition
		expectedQuery   string
	}{
		{
			name: "Basic Select with Where Clause",
			selectQueryMap: SelectQueryMap{
				PartiQLQuery:      "SELECT age, address FROM employee WHERE age > 30 AND address = 'abc'",
				SpannerQuery:      "",
				QueryType:         "SELECT",
				Table:             "employee",
				ProjectionColumns: []string{"age", "address"},
				OrderBy:           []string{"age"},
				Limit:             "10",
				Offset:            "5",
			},
			whereConditions: []Condition{
				{Column: "age", Operator: ">", Value: "30", ANDOpr: "AND"},
				{Column: "address", Operator: "=", Value: "'abc'", OROpr: ""},
			},
			expectedQuery: "SELECT age, address FROM employee WHERE age > 30 AND address = 'abc' ORDER BY age LIMIT 10 OFFSET 5",
		},
		{
			name: "Select All with No Where Clause",
			selectQueryMap: SelectQueryMap{
				PartiQLQuery:      "SELECT * FROM employee",
				SpannerQuery:      "",
				QueryType:         "SELECT",
				Table:             "employee",
				ProjectionColumns: []string{},
				OrderBy:           []string{},
				Limit:             "",
				Offset:            "",
			},
			whereConditions: []Condition{},
			expectedQuery:   "SELECT * FROM employee",
		},
		{
			name: "Select with Multiple Where Conditions",
			selectQueryMap: SelectQueryMap{
				PartiQLQuery:      "SELECT age FROM employee",
				SpannerQuery:      "",
				QueryType:         "SELECT",
				Table:             "employee",
				ProjectionColumns: []string{"age"},
				OrderBy:           []string{"age"},
				Limit:             "",
				Offset:            "",
			},
			whereConditions: []Condition{
				{Column: "age", Operator: ">", Value: "30", ANDOpr: "AND"},
				{Column: "status", Operator: "=", Value: "'active'", OROpr: ""},
			},
			expectedQuery: "SELECT age FROM employee WHERE age > 30 AND status = 'active' ORDER BY age",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := formSpannerSelectQuery(&tt.selectQueryMap, tt.whereConditions)
			assert.NoError(t, err)                   // Ensure no error occurred
			assert.Equal(t, tt.expectedQuery, query) // Compare the generated query
		})
	}
}
