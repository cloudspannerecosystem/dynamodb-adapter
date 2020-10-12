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

	stmt1 := spanner.Statement{}
	stmt1.SQL = "SELECT * FROM dynamodb_adapter_table_ddl"
	ms, err := storage.GetStorageInstance().ExecuteSpannerQuery(context.Background(), "dynamodb_adapter_table_ddl", []string{"tableName", "column", "dataType", "originalColumn"}, false, stmt1)
	if err != nil {
		return err
	}
	dataAvailable := len(ms) > 0

	if dataAvailable {
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

func getColNameAndType(stmt string) (string, string) {
	if stmt == "" {
		return "", ""
	}
	stmt = strings.TrimSpace(stmt)
	tokens := strings.Split(stmt, " ")
	if len(tokens) < 2 {
		return "", ""
	}
	tokens[0] = strings.Trim(tokens[0], "`")
	return tokens[0], tokens[1]
}

func getTableName(stmt string) string {
	if stmt == "" {
		return ""
	}
	tokens := strings.Split(stmt, " ")
	if len(tokens) < 3 {
		return ""
	}
	return tokens[2]
}
