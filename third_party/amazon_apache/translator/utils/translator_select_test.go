package translator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToSpannerSelect(t *testing.T) {
	query := "SELECT age, address FROM employee WHERE age > 30 AND address = 'abc' ORDER BY age LIMIT 10 OFFSET 5;"

	translator := Translator{}
	response, err := translator.ToSpannerSelect(query)

	assert.NoErrorf(t, err, "should not throw an error", err)
	// Prepare expected results
	expectedTable := "employee"
	expectedColumns := []string{"age", "address"}
	expectedWhereConditions := []Condition{
		{Column: "age", Operator: ">", Value: "30"},
		{Column: "address", Operator: "=", Value: `'abc'`},
	}
	expectedOrderBy := []string{"age"}
	expectedLimit := "LIMIT10"
	expectedOffset := "OFFSET5"

	// Assertions for Table
	assert.Equal(t, expectedTable, response.Table)

	// Assertions for Projection Columns
	assert.Equal(t, expectedColumns, response.ProjectionColumns)

	// Assertions for WHERE conditions
	assert.Equal(t, len(expectedWhereConditions), len(response.ProjectionColumns))
	for i, cond := range response.Where {
		assert.Equal(t, expectedWhereConditions[i].Column, cond.Column)
		assert.Equal(t, expectedWhereConditions[i].Operator, cond.Operator)
		assert.Equal(t, expectedWhereConditions[i].Value, cond.Value)
	}

	// Assertions for ORDER BY clause
	assert.Equal(t, expectedOrderBy, response.OrderBy)

	// Assertions for LIMIT clause
	assert.Equal(t, expectedLimit, response.Limit)

	// Assertions for OFFSET clause
	assert.Equal(t, expectedOffset, response.Offset)
}
