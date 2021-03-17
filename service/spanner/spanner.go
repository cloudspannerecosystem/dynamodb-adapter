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

package spanner

import (
	"context"
	"regexp"
	"strings"

	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/storage"

	"cloud.google.com/go/spanner"
)

var colNameRg = regexp.MustCompile("^[a-zA-Z0-9_]*$")
var chars = []string{"]", "^", "\\\\", "/", "[", ".", "(", ")", "-"}
var ss = strings.Join(chars, "")
var specialCharRg = regexp.MustCompile("[" + ss + "]+")

// ParseDDL - this will parse DDL of spannerDB and set all the table configs in models
// This fetches the spanner schema config from dynamodb_adapter_table_ddl table and stored it in
// global map object which is used to read and write data into spanner tables
func ParseDDL(updateDB bool) error {

	stmt := spanner.Statement{}
	stmt.SQL = "SELECT * FROM dynamodb_adapter_table_ddl"
	ms, err := storage.GetStorageInstance().ExecuteSpannerQuery(context.Background(), "dynamodb_adapter_table_ddl", []string{"tableName", "column", "dataType", "originalColumn"}, false, stmt)
	if err != nil {
		return err
	}

	if len(ms) > 0 {
		for i := 0; i < len(ms); i++ {
			tableName := ms[i]["tableName"].(string)
			column := ms[i]["column"].(string)
			column = strings.Trim(column, "`")
			dataType := ms[i]["dataType"].(string)
			originalColumn, ok := ms[i]["originalColumn"].(string)
			if ok {
				originalColumn = strings.Trim(originalColumn, "`")
				if column != originalColumn && originalColumn != "" {
					models.TableColChangeMap[tableName] = struct{}{}
					models.ColumnToOriginalCol[originalColumn] = column
					models.OriginalColResponse[column] = originalColumn
				}
			}
			_, found := models.TableColumnMap[tableName]
			if !found {
				models.TableDDL[tableName] = make(map[string]string)
				models.TableColumnMap[tableName] = []string{}
			}
			models.TableColumnMap[tableName] = append(models.TableColumnMap[tableName], column)
			models.TableDDL[tableName][column] = dataType
		}
	}
	return nil
}
