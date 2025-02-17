package translator

import (
	"testing"

	"github.com/antlr4-go/antlr/v4"
	"github.com/cloudspannerecosystem/dynamodb-adapter/translator/PartiQLParser/parser"
	"github.com/stretchr/testify/assert"
)

// Test the ToSpannerInsert function
func TestToSpannerInsert(t *testing.T) {
	query := "INSERT INTO employee VALUE {'emp_id': 10, 'first_name': 'Marc', 'last_name': 'Richards1', 'age': 10, 'address': 'Shamli'};"

	translator := &Translator{}

	// Set up lexer and parser
	lexer := parser.NewPartiQLLexer(antlr.NewInputStream(query))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewPartiQLParser(stream)

	// Prepare the listener for capturing parsed data
	insertListener := &InsertQueryListener{}
	antlr.ParseTreeWalkerDefault.Walk(insertListener, p.Root())

	// Call ToSpannerInsert after parsing
	insertStatement, err := translator.ToSpannerInsert(query)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Expected results
	expectedTable := "employee"
	expectedColumns := []string{"emp_id", "first_name", "last_name", "age", "address"}
	expectedValues := []string{"10", "'Marc'", "'Richards1'", "10", "'Shamli'"}

	// Assertions for Table
	assert.Equal(t, expectedTable, insertStatement.Table)

	// Assertions for Columns
	assert.Equal(t, expectedColumns, insertStatement.Columns)

	// Assertions for Values
	assert.Equal(t, expectedValues, insertStatement.Values)

	// Assertions for AdditionalMap
	assert.Empty(t, insertStatement.OnConflict) // Check OnConflict if necessary
}
