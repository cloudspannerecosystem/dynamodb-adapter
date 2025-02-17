package translator

import (
	"github.com/cloudspannerecosystem/dynamodb-adapter/translator/PartiQLParser/parser"
	"go.uber.org/zap"
)

type Translator struct {
	Logger *zap.Logger
	Debug  bool
}

type Condition struct {
	Column   string
	Operator string
	Value    string
	ANDOpr   string
	OROpr    string
	LogicOp  string
}

type LogicalGroup struct {
	Conditions []Condition
	Operator   string // AND or OR
}

type SelectQueryListener struct {
	*parser.BasePartiQLParserListener
	Columns      []string
	Tables       []string
	Where        []Condition
	OrderBy      []string
	Limit        string
	Offset       string
	LogicStack   []LogicalGroup // Stack to track logical groups
	CurrentLogic string         // Tracks current logical operator
}

// SelectQueryMap represents the mapping of a select query along with its translation details.
type SelectQueryMap struct {
	PartiQLQuery      string
	SpannerQuery      string
	QueryType         string
	Table             string
	ParamKeys         []string
	ProjectionColumns []string
	OrderBy           []string // Ensure OrderBy is part of this struct
	Limit             string   // Ensure Limit is part of this struct
	Offset            string   // Ensure Offset is part of this struct
	Where             []Condition
}

type Clause struct {
	Column       string
	Operator     string
	Value        string
	IsPrimaryKey bool
}

type UpdateQueryMap struct {
	PartiQLQuery    string // Original query string
	SpannerQuery    string
	QueryType       string           // Type of the query (e.g., UPDATE)
	Table           string           // Table involved in the query
	UpdateSetValues []UpdateSetValue // Values to be updated
	Clauses         []Clause         // List of clauses in the update query
	PrimaryKeys     []string         // Primary keys of the table                   // Flag to indicate if local IDs pattern is used
}
type UpdateSetValue struct {
	Column   string
	Value    string
	Operator string
	RawValue interface{}
}

// Listener for DELETE queries.
type DeleteQueryListener struct {
	*parser.BasePartiQLParserListener
	Table string
	Where []Condition
}

type DeleteQueryMap struct {
	PartiQL           string // Original query string
	SpannerQuery      string
	QueryType         string   // Type of the query (e.g., DELETE)
	Table             string   // Table involved in the query
	Clauses           []Clause // List of clauses in the delete query
	PrimaryKeys       []string // Primary keys of the table
	ExecuteByMutation bool     // Flag to indicate if the delete should be executed by mutation
}

type InsertStatement struct {
	PartiQL       string // Original query string
	SpannerQuery  string
	Table         string
	Columns       []string
	Values        []string
	OnConflict    string
	AdditionalMap map[string]interface{} //
}

// Listener for INSERT queries.
type InsertQueryListener struct {
	*parser.BasePartiQLParserListener
	InsertData InsertStatement
}

type SetClause struct {
	Column   string
	Operator string
	Value    string
}

// Listener for UPDATE queries.
type UpdateQueryListener struct {
	*parser.BasePartiQLParserListener
	Table      string
	SetColumns []string // Storage for column-specific updates.
	Where      []Condition
	SetClauses []SetClause
}
