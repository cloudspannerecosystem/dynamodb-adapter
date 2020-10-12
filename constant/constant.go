// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package constant implements constant variables for application
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
