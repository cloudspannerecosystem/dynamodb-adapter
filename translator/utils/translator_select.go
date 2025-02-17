package translator

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/cloudspannerecosystem/dynamodb-adapter/translator/PartiQLParser/parser"
)

func (l *SelectQueryListener) EnterExprAnd(ctx *parser.ExprAndContext) {
	l.CurrentLogic = "AND"
	l.LogicStack = append(l.LogicStack, LogicalGroup{Operator: "AND"})
}

func (l *SelectQueryListener) EnterExprOr(ctx *parser.ExprOrContext) {
	l.CurrentLogic = "OR"
	l.LogicStack = append(l.LogicStack, LogicalGroup{Operator: "OR"})
}

func (l *SelectQueryListener) EnterProjectionItems(ctx *parser.ProjectionItemsContext) {
	for _, proj := range ctx.AllProjectionItem() {
		l.Columns = append(l.Columns, proj.GetText())
	}
}

func (l *SelectQueryListener) EnterFromClause(ctx *parser.FromClauseContext) {
	l.Tables = append(l.Tables, ctx.TableReference().GetText())
}

func (l *SelectQueryListener) EnterOrderByClause(ctx *parser.OrderByClauseContext) {
	for _, orderSpec := range ctx.AllOrderSortSpec() {
		l.OrderBy = append(l.OrderBy, orderSpec.GetText())
	}
}

func (l *SelectQueryListener) EnterLimitClause(ctx *parser.LimitClauseContext) {
	l.Limit = ctx.GetText()
}

func (l *SelectQueryListener) EnterOffsetByClause(ctx *parser.OffsetByClauseContext) {
	l.Offset = ctx.GetText()
}

// Extracts WHERE conditions for SELECT
func (l *SelectQueryListener) EnterPredicateComparison(ctx *parser.PredicateComparisonContext) {
	column := ctx.GetLhs().GetText()
	operator := ctx.GetOp().GetText()
	value := ctx.GetRhs().GetText()

	condition := Condition{
		Column:   strings.ReplaceAll(column, `'`, ""),
		Operator: operator,
		Value:    value,
	}

	if len(l.LogicStack) > 0 {
		lastGroup := &l.LogicStack[len(l.LogicStack)-1]
		lastGroup.Conditions = append(lastGroup.Conditions, condition)
	} else {
		// Avoid adding logic operators among the base conditions themselves
		l.Where = append(l.Where, condition)
	}
}

func (l *SelectQueryListener) ExitExprAnd(ctx *parser.ExprAndContext) {
	if len(l.LogicStack) > 0 {
		lastGroup := l.LogicStack[len(l.LogicStack)-1]
		l.LogicStack = l.LogicStack[:len(l.LogicStack)-1] // Pop from stack
		for i, cond := range lastGroup.Conditions {
			// Ensure all conditions except the first get AND
			if i > 0 {
				cond.ANDOpr = "AND"
			}
			l.Where = append(l.Where, cond)
		}
	}
}

func (l *SelectQueryListener) ExitExprOr(ctx *parser.ExprOrContext) {
	if len(l.LogicStack) > 0 {
		lastGroup := l.LogicStack[len(l.LogicStack)-1]
		l.LogicStack = l.LogicStack[:len(l.LogicStack)-1] // Pop from stack
		for i, cond := range lastGroup.Conditions {
			// Ensure all conditions except the first get OR
			if i > 0 {
				cond.OROpr = "OR"
			}
			l.Where = append(l.Where, cond)
		}
	}
}

func (t *Translator) ToSpannerSelect(query string) (*SelectQueryMap, error) {
	var err error
	var whereConditions []Condition // Local variable to store WHERE conditions temporarily

	// Lexer and parser setup
	lexer := parser.NewPartiQLLexer(antlr.NewInputStream(query))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewPartiQLParser(stream)

	selectListener := &SelectQueryListener{}
	antlr.ParseTreeWalkerDefault.Walk(selectListener, p.Root())

	// Capture WHERE conditions
	whereConditions = append(whereConditions, selectListener.Where...)

	// Build the SelectQueryMap
	selectQueryMap := &SelectQueryMap{
		PartiQLQuery:      query,
		SpannerQuery:      "",       // TODO: Assign translated Spanner SQL
		QueryType:         "SELECT", // Assuming SELECT by context
		Table:             selectListener.Tables[0],
		ParamKeys:         []string{}, // Populate if params are used
		ProjectionColumns: selectListener.Columns,
		Limit:             selectListener.Limit,
		OrderBy:           selectListener.OrderBy,
		Offset:            selectListener.Offset,
		Where:             whereConditions,
	}
	// Generate Spanner query string
	selectQueryMap.SpannerQuery, err = formSpannerSelectQuery(selectQueryMap, whereConditions)
	if err != nil {
		return nil, err
	}

	return selectQueryMap, nil
}
