package models

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	// "github.com/zhouzhuojie/conditions"

	"github.com/antonmedv/expr/vm"

	"sync"
)

// Meta struct
type Meta struct {
	TableName                 string                              `json:"tableName"`
	AttrMap                   map[string]interface{}              `json:"attrMap"`
	ConditionalExp            string                              `json:"conditionalExp"`
	ExpressionAttributeValues map[string]interface{}              `json:"expressionAttributeValues"`
	DynamoObjectAttr          map[string]*dynamodb.AttributeValue `json:"dynamoObjectAttrVal"`
	DynamoObject              map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
}

// GetKeyMeta struct
type GetKeyMeta struct {
	Key          string                              `json:"key"`
	Type         string                              `json:"type"`
	DynamoObject map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
}

// SetKeyMeta struct
type SetKeyMeta struct {
	Key          string                              `json:"key"`
	Type         string                              `json:"type"`
	Value        string                              `json:"value"`
	DynamoObject map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
}

// BatchMetaUpdate struct
type BatchMetaUpdate struct {
	TableName    string                                `json:"tableName"`
	ArrAttrMap   []map[string]interface{}              `json:"arrAttrMap"`
	DynamoObject []map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
}

// BatchMeta struct
type BatchMeta struct {
	TableName    string                                `json:"tableName"`
	KeyArray     []map[string]interface{}              `json:"keyArray"`
	DynamoObject []map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
}

// GetItemMeta struct
type GetItemMeta struct {
	TableName                string                              `json:"TableName"`
	PrimaryKeyMap            map[string]interface{}              `json:"PrimaryKeyMap"`
	ProjectionExpression     string                              `json:"ProjectionExpression"`
	ExpressionAttributeNames map[string]string                   `json:"ExpressionAttributeNames"`
	Key                      map[string]*dynamodb.AttributeValue `json:"Key"`
}

//BatchGetMeta struct
type BatchGetMeta struct {
	RequestItems map[string]BatchGetWithProjectionMeta `json:"RequestItems"`
}

// BatchGetWithProjectionMeta struct
type BatchGetWithProjectionMeta struct {
	TableName                string                                `json:"tableName"`
	KeyArray                 []map[string]interface{}              `json:"keyArray"`
	ProjectionExpression     string                                `json:"projectionExpression"`
	ExpressionAttributeNames map[string]string                     `json:"expressionAttributeNames"`
	Keys                     []map[string]*dynamodb.AttributeValue `json:"Keys"`
}

// Delete struct
type Delete struct {
	TableName                 string                              `json:"tableName"`
	PrimaryKeyMap             map[string]interface{}              `json:"primaryKeyMap"`
	ConditionalExpression     string                              `json:"conditionalExpression"`
	ExpressionAttributeValues map[string]interface{}              `json:"expressionAttributeValues"`
	DynamoObject              map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
	DynamoObjectAttrVal       map[string]*dynamodb.AttributeValue `json:"dynamoObjectAttrVal"`
}

// BulkDelete struct
type BulkDelete struct {
	TableName          string                                `json:"tableName"`
	PrimaryKeyMapArray []map[string]interface{}              `json:"keyArray"`
	DynamoObject       []map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
}

// Query struct
type Query struct {
	TableName                string                              `json:"tableName"`
	IndexName                string                              `json:"indexName"`
	OnlyCount                bool                                `json:"onlyCount"`
	Limit                    int64                               `json:"limit"`
	SortAscending            bool                                `json:"sortAscending"`
	StartFrom                map[string]interface{}              `json:"startFrom"`
	HashExp                  string                              `json:"hashExp"`
	HashVal                  interface{}                         `json:"hasVal"`
	HashValDDB               *dynamodb.AttributeValue            `json:"hasValDDB"`
	ProjectionExpression     string                              `json:"projectionExpression"`
	ExpressionAttributeNames map[string]string                   `json:"expressionAttributeNames"`
	FilterExp                string                              `json:"filterExp"`
	FilterVal                interface{}                         `json:"filterVal"`
	FilterValDDB             *dynamodb.AttributeValue            `json:"filterValDDB"`
	RangeExp                 string                              `json:"rangeExp"`
	RangeVal                 interface{}                         `json:"rangeVal"`
	RangeValDDB              *dynamodb.AttributeValue            `json:"rangeValDDB"`
	RangeValMap              map[string]interface{}              `json:"rangeValMap"`
	RangeValMapDDB           map[string]*dynamodb.AttributeValue `json:"rangeValMapDDB"`
	DynamoObject             map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
}

// UpdateAttr struct
type UpdateAttr struct {
	TableName                 string                              `json:"tableName"`
	PrimaryKeyMap             map[string]interface{}              `json:"primaryKeyMap"`
	ReturnValues              string                              `json:"returnValues"`
	UpdateExpression          string                              `json:"updateExpression"`
	ConditionalExpression     string                              `json:"conditionalExp"`
	ExpressionAttributeValues map[string]interface{}              `json:"attrVals"`
	ExpressionAttributeNames  map[string]string                   `json:"attrNames"`
	DynamoObject              map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
	DynamoObjectAttr          map[string]*dynamodb.AttributeValue `json:"dynamoObjectAttrVal"`
}

//ScanMeta for Scan request
type ScanMeta struct {
	TableName    string                              `json:"tableName"`
	Limit        int64                               `json:"limit"`
	StartFrom    map[string]interface{}              `json:"startFrom"`
	DynamoObject map[string]*dynamodb.AttributeValue `json:"dynamoObject"`
}

type TableConfig struct {
	PartitionKey     string                 `json:"partitionKey,omitempty"`
	SortKey          string                 `json:"sortKey,omitempty"`
	Indices          map[string]TableConfig `json:"indices,omitempty"`
	GCSSourcePath    string                 `json:"gcsSourcePath,omitempty"`
	DDBIndexName     string                 `json:"ddbIndexName,omitempty"`
	SpannerIndexName string                 `json:"table,omitempty"`
	IsPadded         bool                   `json:"isPadded,omitempty"`
	IsComplement     bool                   `json:"isComplement,omitempty"`
	TableSource      string                 `json:"tableSource,omitempty"`
	ActualTable      string                 `json:"actualTable,omitempty"`
}

// TableDDL - This contains the DDL
var TableDDL map[string]map[string]string

// TableColumnMap - this contains the list of columns for the tables
var TableColumnMap map[string][]string

var TableColChangeMap map[string]struct{}

var ColumnToOriginalCol map[string]string
var OriginalColResponse map[string]string

func init() {
	TableDDL = make(map[string]map[string]string)
	TableDDL["dynamodb_adapter_table_ddl"] = map[string]string{"tableName": "STRING(MAX)", "column": "STRING(MAX)", "dataType": "STRING(MAX)", "originalColumn": "STRING(MAX)"}
	TableDDL["dynamodb_adapter_config_manager"] = map[string]string{"tableName": "STRING(MAX)", "config": "STRING(MAX)", "cronTime": "STRING(MAX)", "uniqueValue": "STRING(MAX)", "enabledStream": "STRING(MAX)", "pubsubTopic": "STRING(MAX)"}
	TableColumnMap = make(map[string][]string)
	TableColumnMap["dynamodb_adapter_table_ddl"] = []string{"tableName", "column", "dataType", "originalColumn"}
	TableColumnMap["dynamodb_adapter_config_manager"] = []string{"tableName", "config", "cronTime", "uniqueValue", "enabledStream", "pubsubTopic"}
	TableColChangeMap = make(map[string]struct{})
	ColumnToOriginalCol = make(map[string]string)
	OriginalColResponse = make(map[string]string)
}

type Eval struct {
	// Cond       conditions.Expr
	Cond       *vm.Program
	Attributes []string
	Cols       []string
	Tokens     []string
	ValueMap   map[string]interface{}
}

type UpdateExpressionCondition struct {
	Field     []string
	Value     []string
	Condition []string
	ActionVal string
	AddValues map[string]float64
}

type dynamodb_adapter_table_ddl struct {
	Table    string
	Column   string
	DataType string
}

type DBAudit struct {
	ID           string                 `json:"id,omitempty"`
	Timestamp    int64                  `json:"timestamp,omitempty"`
	AWSResponse  map[string]interface{} `json:"awsResponse,omitempty"`
	GCPResponse  map[string]interface{} `json:"gcpResponse,omitempty"`
	TableName    string                 `json:"tableName,omitempty"`
	PartitionKey string                 `json:"partitionKey,omitempty"`
	SortKey      string                 `json:"sortKey,omitempty"`
	Operation    string                 `json:"operation"`
	GCPDbErr     string                 `json:"gcpDbErr"`
	AWSDbErr     string                 `json:"awsDbErr"`
	Payload      string                 `json:"payload"`
}

type ConfigControllerModel struct {
	Mux               sync.RWMutex
	UniqueVal         string
	CornTime          string
	StopConfigManager bool
	ReadMap           map[string]struct{}
	WriteMap          map[string]struct{}
	StreamEnable      map[string]struct{}
	PubSubTopic       map[string]string
}

var ConfigController *ConfigControllerModel

var SpannerTableMap = make(map[string]string)

func init() {
	ConfigController = new(ConfigControllerModel)
	ConfigController.CornTime = "1"
	ConfigController.Mux = sync.RWMutex{}
	ConfigController.ReadMap = make(map[string]struct{})
	ConfigController.WriteMap = make(map[string]struct{})
	ConfigController.StreamEnable = make(map[string]struct{})
	ConfigController.PubSubTopic = make(map[string]string)
}

type StreamDataModel struct {
	OldImage       map[string]interface{} `json:"oldImage"`
	NewImage       map[string]interface{} `json:"newImage"`
	Keys           map[string]interface{} `json:"keys"`
	Timestamp      int64                  `json:"timestamp"`
	Table          string                 `json:"tableName"`
	EventName      string                 `json:"eventName"`
	SequenceNumber int64                  `json:"sequenceNumber"`
	EventID        string                 `json:"eventId"`
	EventSourceArn string                 `json:"eventSourceArn"`
}
