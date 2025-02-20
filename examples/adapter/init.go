// Copyright 2021 Google LLC
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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	rice "github.com/GeertJohan/go.rice"
	"github.com/cloudspannerecosystem/dynamodb-adapter/config"
	"google.golang.org/api/iterator"
)

const (
	expectedRowCount = 45
)

var (
	colNameRg     = regexp.MustCompile("^[a-zA-Z0-9_]*$")
	chars         = []string{"]", "^", "\\\\", "/", "[", ".", "(", ")", "-"}
	ss            = strings.Join(chars, "")
	specialCharRg = regexp.MustCompile("[" + ss + "]+")
	adapterTables = map[string]string{
		"dynamodb_adapter_table_ddl": `CREATE TABLE dynamodb_adapter_table_ddl (
			column	       STRING(MAX),
			tableName      STRING(MAX),
			dataType       STRING(MAX),
			originalColumn STRING(MAX),
		) PRIMARY KEY (tableName, column)`,
		"dynamodb_adapter_config_manager": `CREATE TABLE dynamodb_adapter_config_manager (
			tableName     STRING(MAX),
			config 	      STRING(MAX),
			cronTime      STRING(MAX),
			enabledStream STRING(MAX),
			pubsubTopic   STRING(MAX),
			uniqueValue   STRING(MAX),
		) PRIMARY KEY (tableName)`,
	}
)

func main() {
	box := rice.MustFindBox("./config-files")

	// read the config variables
	ba, err := box.Bytes("staging/config.json")
	if err != nil {
		log.Fatal("error reading staging config json: ", err.Error())
	}
	var conf = &config.Configuration{}
	if err = json.Unmarshal(ba, &conf); err != nil {
		log.Fatal(err)
	}

	// read the spanner table configurations
	var m = make(map[string]string)
	ba, err = box.Bytes("staging/spanner.json")
	if err != nil {
		log.Fatal("error reading spanner config json: ", err.Error())
	}
	if err = json.Unmarshal(ba, &m); err != nil {
		log.Fatal(err)
	}

	var databaseName = fmt.Sprintf(
		"projects/%s/instances/%s/databases/%s", conf.GoogleProjectID, m["dynamodb_adapter_table_ddl"], conf.SpannerDb,
	)

	switch cmd := os.Args[1]; cmd {
	case "setup":
		w := log.Writer()
		if err := createDatabase(w, databaseName); err != nil {
			log.Fatal(err)
		}

		if err := createDynamodbAdatperTableDDL(w, databaseName); err != nil {
			log.Fatal(err)
		}

		if err := updateDynamodbAdapterTableDDL(w, databaseName); err != nil {
			log.Fatal(err)
		}

		count, err := verifySpannerSetup(databaseName)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, "Found %d rows\n", count)
		if count != expectedRowCount {
			log.Fatalf("Rows found: %d, exepected %d\n", count, expectedRowCount)
		}
	default:
		log.Fatal("Unknown command: use 'setup'")
	}
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

	op, err := adminClient.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          matches[1],
		CreateStatement: "CREATE DATABASE `" + matches[2] + "`",
	})
	if err != nil {
		if strings.Contains(err.Error(), "AlreadyExists") {
			fmt.Fprintf(w, "[%s] already exists.\n", db)
			return nil
		}
		return err
	}
	if _, err := op.Wait(ctx); err != nil {
		return err
	}

	fmt.Fprintf(w, "Created database [%s]\n", db)
	return nil
}

func createDynamodbAdatperTableDDL(w io.Writer, db string) error {
	ctx := context.Background()
	adminClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return err
	}
	defer adminClient.Close()

	for name, table := range adapterTables {
		op, err := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database: db,
			Statements: []string{
				table,
			},
		})
		if err != nil {
			return err
		}
		if err := op.Wait(ctx); err != nil {
			if strings.Contains(err.Error(), "Duplicate name") {
				fmt.Fprintf(w, "[%s] already exists\n", name)
			} else {
				return err
			}
		}
	}

	fmt.Fprintf(w, "dynamodb_adapter tables are ready\n")
	return nil
}

func updateDynamodbAdapterTableDDL(w io.Writer, db string) error {
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
			fmt.Fprintf(w, "[%s, %s, %s, %s]\n", currentTable, colName, colType, originalColumn)
			mutations = append(mutations, mut)
		}
	}

	return spannerBatchPut(context.Background(), db, mutations)
}

func readDatabaseSchema(db string) ([]string, error) {
	ctx := context.Background()
	adminClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return nil, err
	}
	defer adminClient.Close()

	ddlResp, err := adminClient.GetDatabaseDdl(ctx,
		&databasepb.GetDatabaseDdlRequest{Database: db},
	)
	if err != nil {
		return nil, err
	}

	return ddlResp.GetStatements(), nil
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
