// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"testing"

	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"gopkg.in/go-playground/assert.v1"
)

// Actual table name comes from the DynamoDB table name
// Key in table config is the Spanner table name
// We fetch from GetTableConf using the DynamoDB table name which get's converted
func TestGetTableConf(t *testing.T) {
	models.DbConfigMap = map[string]models.TableConfig{
		"employee_data": {
			PartitionKey:     "emp_id",
			SortKey:          "emp_name",
			Indices:          nil,
			SpannerIndexName: "emp_id",
			ActualTable:      "employee_data",
		},
		"employee_data_2": {
			PartitionKey:     "e_id",
			SortKey:          "e_name",
			Indices:          nil,
			SpannerIndexName: "emp_id",
			ActualTable:      "employee-data-2",
		},
		"department": {
			PartitionKey:     "d_id",
			SortKey:          "d_name",
			Indices:          nil,
			SpannerIndexName: "d_id",
			ActualTable:      "",
		},
	}

	tests := []struct {
		testName  string
		tableName string
		want      models.TableConfig
	}{
		{
			"Empty table name",
			"",
			models.TableConfig{},
		},
		{
			"Missing table",
			"xyz",
			models.TableConfig{},
		},
		{
			"Table missing ActualTable name, defaults to Spanner table name",
			"department",
			models.TableConfig{
				PartitionKey:     "d_id",
				SortKey:          "d_name",
				Indices:          nil,
				SpannerIndexName: "d_id",
				ActualTable:      "department",
			},
		},
		{
			"Table where DynamoDB table name is same as Spanner table name",
			"employee_data",
			models.TableConfig{
				PartitionKey:     "emp_id",
				SortKey:          "emp_name",
				Indices:          nil,
				SpannerIndexName: "emp_id",
				ActualTable:      "employee_data",
			},
		},
		{
			"Table where DynamoDB table name is different from Spanner table name",
			"employee-data-2",
			models.TableConfig{
				PartitionKey:     "e_id",
				SortKey:          "e_name",
				Indices:          nil,
				SpannerIndexName: "emp_id",
				ActualTable:      "employee-data-2",
			},
		},
	}

	for _, tc := range tests {
		got, _ := GetTableConf(tc.tableName)
		assert.Equal(t, got, tc.want)
	}
}
