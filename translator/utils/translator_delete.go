package translator

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/cloudspannerecosystem/dynamodb-adapter/third_party/amazon_apache/translator/PartiQLParser/parser"
)

// Methods for DELETE Listener
func (l *DeleteQueryListener) EnterDeleteCommand(ctx *parser.DeleteCommandContext) {
	if fromCtx, ok := ctx.FromClauseSimple().(*parser.FromClauseSimpleExplicitContext); ok {
		l.Table = fromCtx.PathSimple().GetText()
	}
}

// Extracts WHERE conditions for DELETE
func (l *DeleteQueryListener) EnterPredicateComparison(ctx *parser.PredicateComparisonContext) {
	column := ctx.GetLhs().GetText()
	operator := ctx.GetOp().GetText()
	value := ctx.GetRhs().GetText()

	l.Where = append(l.Where, Condition{
		Column:   column,
		Operator: operator,
		Value:    value,
	})
}
func (t *Translator) ToSpannerDelete(query string) (*DeleteQueryMap, error) {
	deleteQueryMap := &DeleteQueryMap{}
	deleteListener := &DeleteQueryListener{}

	// Lexer and parser setup
	lexer := parser.NewPartiQLLexer(antlr.NewInputStream(query))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewPartiQLParser(stream)

	// Parse the input query
	antlr.ParseTreeWalkerDefault.Walk(deleteListener, p.Root())
	deleteQueryMap.Table = deleteListener.Table

	// Populate deleteQueryMap.Clauses from deleteListener.Where
	if len(deleteListener.Where) > 0 {
		for _, cond := range deleteListener.Where {
			deleteQueryMap.Clauses = append(deleteQueryMap.Clauses, Clause{
				Column:   cond.Column,
				Operator: cond.Operator,
				Value:    cond.Value,
			})
		}
	}
	deleteQueryMap.PartiQL = query
	deleteQueryMap.SpannerQuery = createSpannerDeleteQuery(deleteListener.Table, deleteQueryMap.Clauses)
	return deleteQueryMap, nil
}

// createSpannerDeleteQuery generates the Spanner delete query using Parsed information.
//
// It takes the table name and an array of clauses as input and returns the generated query string.
func createSpannerDeleteQuery(table string, clauses []Clause) string {
	return "DELETE FROM " + table + buildWhereClause(clauses) + ";"
}
