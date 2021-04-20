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

package integrationtest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"google.golang.org/api/iterator"
	adminpb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
)

// tableColumnMap - this contains the list of columns for the tables

var (
	colNameRg     = regexp.MustCompile("^[a-zA-Z0-9_]*$")
	chars         = []string{"]", "^", "\\\\", "/", "[", ".", "(", ")", "-"}
	ss            = strings.Join(chars, "")
	specialCharRg = regexp.MustCompile("[" + ss + "]+")
)

func createInstance(instance string) error {
	return nil
}

func createDatabase(w io.Writer, db string) error {
	matches := regexp.MustCompile("^(.*)/databases/(.*)$").FindStringSubmatch(db)
	if matches == nil || len(matches) != 3 {
		return fmt.Errorf("Invalid database id %s", db)
	}

	ctx := context.Background()
	adminClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return err
	}
	defer adminClient.Close()

	op, err := adminClient.CreateDatabase(ctx, &adminpb.CreateDatabaseRequest{
		Parent:          matches[1],
		CreateStatement: "CREATE DATABASE `" + matches[2] + "`",
		ExtraStatements: []string{
			`CREATE TABLE dynamodb_adapter_table_ddl (
				column	       STRING(MAX),
				tableName      STRING(MAX),
				dataType       STRING(MAX),
				originalColumn STRING(MAX),
			) PRIMARY KEY (tableName, column)`,
			`CREATE TABLE dynamodb_adapter_config_manager (
				tableName     STRING(MAX),
				config 	      STRING(MAX),
				cronTime      STRING(MAX),
				enabledStream STRING(MAX),
				pubsubTopic   STRING(MAX),
				uniqueValue   STRING(MAX),
			) PRIMARY KEY (tableName)`,
			`CREATE TABLE employee (
				emp_id 	   FLOAT64,
				address    STRING(MAX),
				age 	   FLOAT64,
				first_name STRING(MAX),
				last_name  STRING(MAX),
			) PRIMARY KEY (emp_id)`,
			`CREATE TABLE department (
				d_id 		 FLOAT64,
				d_name 		 STRING(MAX),
				d_specialization STRING(MAX),
			) PRIMARY KEY (d_id)`,
		},
	})
	if err != nil {
		return err
	}
	if _, err := op.Wait(ctx); err != nil {
		return err
	}
	fmt.Fprintf(w, "Created database [%s]\n", db)
	return nil
}

func deleteDatabase(w io.Writer, db string) error {
	ctx := context.Background()
	adminClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return err
	}
	if err := adminClient.DropDatabase(ctx, &adminpb.DropDatabaseRequest{
		Database: db,
	}); err != nil {
		return err
	}
	fmt.Fprintf(w, "Deleted database [%s]\n", db)
	return nil
}

func updateDynamodbAdapterTableDDL(db string) error {
	stmt, err := readDatabaseSchema(db)
	if err != nil {
		return err
	}

	var mutations []*spanner.Mutation
	for i := 0; i < len(stmt); i++ {
		tokens := strings.Split(stmt[i], "\n")
		if len(tokens) == 1 {
			continue
		}
		var currentTable, colName, colType, originalColumn string

		for j := 0; j < len(tokens); j++ {
			if strings.Contains(tokens[j], "PRIMARY KEY") {
				continue
			}
			if strings.Contains(tokens[j], "CREATE TABLE") {
				currentTable = getTableName(tokens[j])
				continue
			}
			colName, colType = getColNameAndType(tokens[j])
			originalColumn = colName

			if !colNameRg.MatchString(colName) {
				colName = specialCharRg.ReplaceAllString(colName, "_")
			}
			colType = strings.Replace(colType, ",", "", 1)
			var mut = spanner.InsertOrUpdateMap(
				"dynamodb_adapter_table_ddl",
				map[string]interface{}{
					"tableName":      currentTable,
					"column":         colName,
					"dataType":       colType,
					"originalColumn": originalColumn,
				},
			)
			mutations = append(mutations, mut)
		}
	}
	return spannerBatchPut(context.Background(), db, mutations)
}

func readDatabaseSchema(db string) ([]string, error) {
	ctx := context.Background()
	cli, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	ddlResp, err := cli.GetDatabaseDdl(ctx, &adminpb.GetDatabaseDdlRequest{Database: db})
	if err != nil {
		return nil, err
	}
	return ddlResp.GetStatements(), nil
}

func getTableName(stmt string) string {
	tokens := strings.Split(stmt, " ")
	return tokens[2]
}

func getColNameAndType(stmt string) (string, string) {
	stmt = strings.TrimSpace(stmt)
	tokens := strings.Split(stmt, " ")
	tokens[0] = strings.Trim(tokens[0], "`")
	return tokens[0], tokens[1]
}

func changeTableNameForSP(tableName string) string {
	tableName = strings.ReplaceAll(tableName, "-", "_")
	return tableName
}

// spannerBatchPut - this insert or update data in batch
func spannerBatchPut(ctx context.Context, db string, m []*spanner.Mutation) error {
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		log.Fatalf("Failed to create client %v", err)
		return err
	}
	defer client.Close()

	if _, err = client.Apply(ctx, m); err != nil {
		return errors.New("ResourceNotFoundException: " + err.Error())
	}
	return nil
}

func verifySpannerSetup(db string) (int, error) {
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	var iter = client.Single().Read(ctx, "dynamodb_adapter_table_ddl", spanner.AllKeys(),
		[]string{"column", "tableName", "dataType", "originalColumn"})

	var count int
	for {
		if _, err := iter.Next(); err != nil {
			if err == iterator.Done {
				break
			}
			return 0, err
		}
		count++
	}
	return count, nil
}
