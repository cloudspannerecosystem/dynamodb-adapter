package constant

import "errors"

var ErrTypeInvalidPartitionKey = errors.New("HashExp should be a partition key.")
var ErrTypeInvalidIndexPartitionKey = errors.New("HashExp should be a partition key of the index.")
var ErrIndicesNotFound = errors.New("Indices not found.")
var ErrTypeCastingForProjectionType = errors.New("Could not type cast projectionType.")
var ErrTypeCasting = errors.New("Could not type cast.")
var ErrInvalidTableConfigMap = errors.New("Config Map error.")
var ErrConditionalCheckFailedException = errors.New("ConditionalCheckFailedException.")
var ErrFetchRowException = errors.New("Error while fetching rowkey.")
var ErrInvalidTableName = errors.New("Invalid Table name")
var ErrInvalidPrimaryKey = errors.New("Invalid Primary Key")
var ErrWriteRows = errors.New("Error while writing rows.")
var ErrDeleteRows = errors.New("Error while deleting rows.")

type key int

const (
	KeyPrincipalID key = iota
)
