package translator

import (
	"testing"

	"github.com/tj/assert"
)

func TestToSpannerDelete(t *testing.T) {
	translator := &Translator{}

	tests := []struct {
		name            string
		query           string
		expectedTable   string
		expectedClauses []Clause
	}{
		{
			name:          "Simple delete with conditions",
			query:         "DELETE FROM employee WHERE age > 30;",
			expectedTable: "employee",
			expectedClauses: []Clause{
				{Column: "age", Operator: ">", Value: "30"},
			},
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteQueryMap, err := translator.ToSpannerDelete(tt.query)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			assert.Equal(t, tt.expectedTable, deleteQueryMap.Table)
			assert.Equal(t, tt.expectedClauses, deleteQueryMap.Clauses)
		})
	}
}
