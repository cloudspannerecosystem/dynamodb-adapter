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

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"gopkg.in/yaml.v2"
)

var readFile = os.ReadFile

func main() {
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Build the Spanner database name
	databaseName := fmt.Sprintf(
		"projects/%s/instances/%s/databases/%s",
		config.Spanner.ProjectID, config.Spanner.InstanceID, config.Spanner.DatabaseName,
	)
	switch cmd := os.Args[1]; cmd {
	case "setup":
		w := log.Writer()
		if err := createDatabase(w, databaseName); err != nil {
			log.Fatal(err)
		}
		if err := initData(w, databaseName); err != nil {
			log.Fatal(err)
		}
	case "teardown":
		w := log.Writer()
		if err := deleteDatabase(w, databaseName); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("Unknown command: use 'setup', 'teardown'")
	}
}

func loadConfig(filename string) (*models.Config, error) {
	data, err := readFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func createDatabase(w io.Writer, db string) error {
	matches := regexp.MustCompile("^(.*)/databases/(.*)$").FindStringSubmatch(db)
	if matches == nil || len(matches) != 3 {
		return fmt.Errorf("invalid database id %s", db)
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
		ExtraStatements: []string{
			`CREATE TABLE dynamodb_adapter_table_ddl (
				tableName STRING(MAX) NOT NULL,
				column STRING(MAX) NOT NULL,
				dynamoDataType STRING(MAX) NOT NULL,
				originalColumn STRING(MAX) NOT NULL,
				partitionKey STRING(MAX),
				sortKey STRING(MAX),
				spannerIndexName STRING(MAX),
				actualTable STRING(MAX),
				spannerDataType STRING(MAX)
			) PRIMARY KEY (tableName, column)`,
			`CREATE TABLE employee (
				emp_id          FLOAT64,
				address         STRING(MAX),
				age            FLOAT64,
				first_name      STRING(MAX),
				last_name       STRING(MAX),
				phone_numbers   ARRAY<STRING(MAX)>,
				profile_pics    ARRAY<BYTES(MAX)>,
				salaries        ARRAY<FLOAT64>
			) PRIMARY KEY (emp_id)`,
			`CREATE TABLE department (
				d_id 		 FLOAT64,
				d_name 		 STRING(MAX),
				d_specialization STRING(MAX),
			) PRIMARY KEY (d_id)`,
			`CREATE TABLE test_table (
				updated_at STRING(MAX),
				rank_list STRING(MAX),
				category STRING(MAX),
				id STRING(MAX),
				list_type JSON,
			  ) PRIMARY KEY(rank_list)`,
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
	if err := adminClient.DropDatabase(ctx, &databasepb.DropDatabaseRequest{
		Database: db,
	}); err != nil {
		return err
	}
	fmt.Fprintf(w, "Deleted database [%s]\n", db)
	return nil
}

func initData(w io.Writer, db string) error {
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		return err
	}
	defer client.Close()

	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `INSERT INTO dynamodb_adapter_table_ddl 
			(tableName, column, dynamoDataType, originalColumn, partitionKey, sortKey, spannerIndexName, actualTable, spannerDataType) 
			VALUES
			('employee', 'emp_id', 'N', 'emp_id', 'emp_id', '', 'emp_id', 'employee', 'FLOAT64'),
			('employee', 'address', 'S', 'address', 'emp_id', '', 'address', 'employee', 'STRING(MAX)'),
			('employee', 'age', 'N', 'age', 'emp_id', '', 'age', 'employee', 'FLOAT64'),
			('employee', 'first_name', 'S', 'first_name', 'emp_id', '', 'first_name', 'employee', 'STRING(MAX)'),
			('employee', 'last_name', 'S', 'last_name', 'emp_id', '', 'last_name', 'employee', 'STRING(MAX)'),
			('employee', 'phone_numbers', 'SS', 'phone_numbers', 'emp_id', '', 'phone_numbers', 'employee', 'ARRAY<STRING(MAX)>'),
			('employee', 'profile_pics', 'BS', 'profile_pics', 'emp_id', '', 'profile_pics', 'employee', 'ARRAY<BYTES(MAX)>'),
			('employee', 'salaries', 'NS', 'salaries', 'emp_id', '', 'salaries', 'employee', 'ARRAY<FLOAT64>'),
			('department', 'd_id', 'N', 'd_id', 'd_id', '', 'd_id', 'department', 'FLOAT64'),
			('department', 'd_name', 'S', 'd_name', 'd_id', '', 'd_name', 'department', 'STRING(MAX)'),
			('department', 'd_specialization', 'S', 'd_specialization', 'd_id', '', 'd_specialization', 'department', 'STRING(MAX)'),
			('test_table', 'updated_at', 'S', 'updated_at', 'rank_list', '', 'updated_at', 'test_table', 'STRING(MAX)'),
			('test_table', 'rank_list', 'S', 'rank_list', 'rank_list', '', 'rank_list', 'test_table', 'STRING(MAX)'),
			('test_table', 'category', 'S', 'category', 'rank_list', '', 'category', 'test_table', 'STRING(MAX)'),
			('test_table', 'id', 'S', 'id', 'rank_list', '', 'id', 'test_table', 'STRING(MAX)'),
			('test_table', 'list_type', 'L', 'list_type', 'rank_list', '', 'list_type', 'test_table', 'JSON');`,
		}
		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%d record(s) inserted.\n", rowCount)
		return err
	})
	if err != nil {
		return err
	}

	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `INSERT INTO employee (emp_id, address, age, first_name, last_name, phone_numbers, profile_pics, salaries) VALUES
				(1, 'Shamli', 10, 'Marc', 'Richards', 
				['+1111111111', '+1222222222'], 
				[FROM_BASE64('U29tZUJ5dGVzRGF0YTE='), FROM_BASE64('U29tZUJ5dGVzRGF0YTI=')], 
				[1000.50, 2000.75]),				
				(2, 'New York', 20, 'Catalina', 'Smith', 
				['+1333333333'], 
				[FROM_BASE64('U29tZUJ5dGVzRGF0YTM=')], 
				[3000.00]),
				(3, 'Pune', 30, 'Alice', 'Trentor', 
				['+1444444444', '+1555555555'], 
				[FROM_BASE64('U29tZUJ5dGVzRGF0YTQ='), FROM_BASE64('U29tZUJ5dGVzRGF0YTU=')], 
				[4000.25, 5000.50, 6000.75]),
				(4, 'Silicon Valley', 40, 'Lea', 'Martin', 
				['+1666666666'], 
				[FROM_BASE64('U29tZUJ5dGVzRGF0YTY=')], 
				[7000.00, 8000.25]),
				(5, 'London', 50, 'David', 'Lomond', 
				['+1777777777', '+1888888888', '+1999999999'], 
				[FROM_BASE64('U29tZUJ5dGVzRGF0YTc='), FROM_BASE64('U29tZUJ5dGVzRGF0YTg=')], 
				[9000.50]);`,
		}
		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%d record(s) inserted.\n", rowCount)
		return err
	})
	if err != nil {
		return err
	}

	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `INSERT department (d_id, d_name, d_specialization) VALUES
						(100, 'Engineering', 'CSE, ECE, Civil'),
						(200, NULL, 'BA'),
						(300, 'Culture', 'History')`,
		}
		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%d record(s) inserted.\n", rowCount)
		return err
	})
	if err != nil {
		return err
	}

	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `INSERT INTO test_table (updated_at, rank_list, category, id, list_type) VALUES
    ('2024-12-04T11:02:02Z', 'rank_list', 'category', 'testing', JSON '[ "John Doe", "62536", true ]'),
    ('2024-12-04T11:02:02Z', 'rank_list1', 'category1', 'id', JSON '["string_value","12345",true,[1, 2, 3],"testing"]'),
    ('2024-12-04T11:02:02Z', 'rank_list2', 'category2', 'id2', JSON '["test","dummy_value","62536"]')`,
		}
		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%d record(s) inserted.\n", rowCount)
		return err
	})
	if err != nil {
		return err
	}

	return err
}
