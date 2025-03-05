package translator

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/cloudspannerecosystem/dynamodb-adapter/third_party/amazon_apache/translator/PartiQLParser/parser"
)

// Methods for UPDATE Listener
func (l *UpdateQueryListener) EnterUpdateClause(ctx *parser.UpdateClauseContext) {
	l.Table = ctx.TableBaseReference().GetText()
}

func (l *UpdateQueryListener) EnterSetCommand(ctx *parser.SetCommandContext) {
	for _, setAssign := range ctx.AllSetAssignment() {
		column := setAssign.PathSimple().GetText()
		value := setAssign.Expr().GetText()

		l.SetClauses = append(l.SetClauses, SetClause{
			Column:   column,
			Operator: "=",
			Value:    value,
		})
	}
}

func (l *UpdateQueryListener) EnterPredicateComparison(ctx *parser.PredicateComparisonContext) {
	column := ctx.GetLhs().GetText()
	operator := ctx.GetOp().GetText()
	value := ctx.GetRhs().GetText()

	condition := Condition{
		Column:   column,
		Operator: operator,
		Value:    value,
	}

	// Check logical operator from parent context node
	if parent, ok := ctx.GetParent().(antlr.ParserRuleContext); ok {
		switch parent.(type) {
		case *parser.ExprAndContext:
			condition.LogicOp = "AND"
		case *parser.ExprOrContext:
			condition.LogicOp = "OR"
		}
	}

	l.Where = append(l.Where, condition)
}

func (l *UpdateQueryListener) EnterExprAnd(ctx *parser.ExprAndContext) {
}

func (l *UpdateQueryListener) EnterExprOr(ctx *parser.ExprOrContext) {
}

func (t *Translator) ToSpannerUpdate(query string) (*DeleteUpdateQueryMap, error) {
	updateQueryMap := &DeleteUpdateQueryMap{}

	// Lexer and parser setup
	lexer := parser.NewPartiQLLexer(antlr.NewInputStream(query))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewPartiQLParser(stream)

	updateListener := &UpdateQueryListener{}
	antlr.ParseTreeWalkerDefault.Walk(updateListener, p.Root())

	updateQueryMap.Table = updateListener.Table
	updateQueryMap.PartiQLQuery = query
	updateQueryMap.QueryType = "UPDATE"
	for _, clause := range updateListener.SetClauses {
		updateQueryMap.UpdateSetValues = append(updateQueryMap.UpdateSetValues, UpdateSetValue{
			Column:   clause.Column,
			Operator: clause.Operator,
			Value:    clause.Value,
		})
	}
	if len(updateListener.Where) > 0 {
		for i := range updateListener.Where {
			updateQueryMap.Clauses = append(updateQueryMap.Clauses, Clause{
				Column:   updateListener.Where[i].Column,
				Operator: updateListener.Where[i].Operator,
				Value:    updateListener.Where[i].Value,
			})
		}
	}
	updateQueryMap.SpannerQuery = formSpannerUpdateQuery(updateQueryMap)
	return updateQueryMap, nil
}

func formSpannerUpdateQuery(updateQueryMap *DeleteUpdateQueryMap) string {
	return "UPDATE " + updateQueryMap.Table + buildSetValues(updateQueryMap.UpdateSetValues) + buildWhereClause(updateQueryMap.Clauses) + ";"
}

func buildSetValues(updateSetValues []UpdateSetValue) string {
	setValues := ""
	for _, val := range updateSetValues {
		column := "`" + val.Column + "`"
		if val.Value == questionMarkLiteral {
			val.Value = "@" + val.Column
		}
		value := val.Value

		if setValues != "" {
			setValues += " , "
		}
		setValues += fmt.Sprintf("%s = %s", column, value)
	}
	if setValues != "" {
		setValues = " SET " + setValues
	}
	return setValues
}

func buildWhereClause(clauses []Clause) string {
	whereClause := ""
	for _, val := range clauses {
		column := "`" + val.Column + "`"
		if val.Value == questionMarkLiteral {
			val.Value = "@" + val.Column
		}
		value := val.Value

		if val.Operator == "IN" {
			value = fmt.Sprintf("UNNEST(%s)", val.Value)
		}
		if whereClause != "" {
			whereClause += " AND "
		}
		whereClause += fmt.Sprintf("%s %s %s", column, val.Operator, value)
	}

	if whereClause != "" {
		whereClause = " WHERE " + whereClause
	}
	return whereClause
}
