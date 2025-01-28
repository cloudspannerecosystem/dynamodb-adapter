// Copyright (c) DataStax, Inc.
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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"cloud.google.com/go/spanner"
	Admindatabase "cloud.google.com/go/spanner/admin/database/apiv1"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"github.com/cloudspannerecosystem/dynamodb-adapter/utils"
	"google.golang.org/genproto/googleapis/spanner/admin/database/v1"
	"gopkg.in/yaml.v3"
)

// Define a global variable for reading files (mockable for tests)
var readFile = os.ReadFile

// DDL statement to create the DynamoDB adapter table in Spanner
var (
	adapterTableDDL = `
	CREATE TABLE dynamodb_adapter_table_ddl (
		column STRING(MAX) NOT NULL,
		tableName STRING(MAX) NOT NULL,
		dynamoDataType STRING(MAX) NOT NULL,
		originalColumn STRING(MAX) NOT NULL,
		partitionKey STRING(MAX),
		sortKey STRING(MAX),
		spannerIndexName STRING(MAX),
		actualTable STRING(MAX),
		spannerDataType STRING(MAX)
	) PRIMARY KEY (tableName, column)`
)

// Entry point for the application
func main() {
	// Parse command-line arguments for dry-run mode
	dryRun := flag.Bool("dry_run", false, "Run the program in dry-run mode to output DDL and queries without making changes")
	flag.Parse()

	// Load configuration from a YAML file
	config, err := loadConfig("../config.yaml")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Build the Spanner database name
	databaseName := fmt.Sprintf(
		"projects/%s/instances/%s/databases/%s",
		config.Spanner.ProjectID, config.Spanner.InstanceID, config.Spanner.DatabaseName,
	)
	ctx := context.Background()
	adminClient, err := Admindatabase.NewDatabaseAdminClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create Spanner Admin client: %v", err)
	}
	defer adminClient.Close()
	// Decide execution mode based on the dry-run flag
	if *dryRun {
		fmt.Println("-- Dry Run Mode: Generating Spanner DDL and Insert Queries Only --")
		runDryRun(config.Spanner.DynamoQueryLimit)
	} else {
		fmt.Println("-- Executing Setup on Spanner --")
		executeSetup(ctx, adminClient, databaseName)
	}
}

// Load configuration from a YAML file
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

// Run in dry-run mode to output DDL and insert queries without making changes
func runDryRun(limit int32) {
	fmt.Println("-- Spanner DDL to create the adapter table --")
	fmt.Println(adapterTableDDL)

	client := createDynamoClient()
	tables, err := listDynamoTables(client)
	if err != nil {
		log.Fatalf("Failed to list DynamoDB tables: %v", err)
	}

	// Process each DynamoDB table
	for _, tableName := range tables {
		fmt.Printf("Processing table: %s\n", tableName)

		// Generate and print table-specific DDL
		ddl := generateTableDDL(tableName, client)
		fmt.Printf("-- DDL for table: %s --\n%s\n", tableName, ddl)

		// Generate and print insert queries
		generateInsertQueries(tableName, client)
	}
}

// Generate DDL statement for a specific DynamoDB table
func generateTableDDL(tableName string, client *dynamodb.Client) string {
	attributes, partitionKey, sortKey, err := fetchTableAttributes(client, tableName, models.GlobalConfig.Spanner.DynamoQueryLimit)
	if err != nil {
		log.Printf("Failed to fetch attributes for table %s: %v", tableName, err)
		return ""
	}

	var columns []string
	for column, dataType := range attributes {
		columns = append(columns, fmt.Sprintf("%s %s", column, utils.ConvertDynamoTypeToSpannerType(dataType)))
	}
	primaryKey := fmt.Sprintf("PRIMARY KEY (%s%s)", partitionKey, func() string {
		if sortKey != "" {
			return ", " + sortKey
		}
		return ""
	}())

	return fmt.Sprintf(
		"CREATE TABLE %s (\n\t%s\n) %s",
		tableName, strings.Join(columns, ",\n\t"), primaryKey,
	)
}

// Generate insert queries for a given DynamoDB table
func generateInsertQueries(tableName string, client *dynamodb.Client) {
	attributes, partitionKey, sortKey, err := fetchTableAttributes(client, tableName, models.GlobalConfig.Spanner.DynamoQueryLimit)
	if err != nil {
		log.Printf("Failed to fetch attributes for table %s: %v", tableName, err)
		return
	}

	for column, dataType := range attributes {
		spannerDataType := utils.ConvertDynamoTypeToSpannerType(dataType)
		query := fmt.Sprintf(
			`INSERT INTO dynamodb_adapter_table_ddl 
			(column, tableName, dataType, originalColumn, partitionKey, sortKey, spannerIndexName, actualTable, spannerDataType) 
			VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s');`,
			column, tableName, dataType, column, partitionKey, sortKey, column, tableName, spannerDataType,
		)
		fmt.Println(query)
	}
}

// Execute the setup process: create database, tables, and migrate data
func executeSetup(ctx context.Context, adminClient *Admindatabase.DatabaseAdminClient, databaseName string) {

	// Create the Spanner database if it doesn't exist
	if err := createDatabase(ctx, adminClient, databaseName); err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// Create the adapter table
	if err := createTable(ctx, adminClient, databaseName, adapterTableDDL); err != nil {
		log.Fatalf("Failed to create adapter table: %v", err)
	}

	// Process each DynamoDB table
	client := createDynamoClient()
	tables, err := listDynamoTables(client)
	if err != nil {
		log.Fatalf("Failed to list DynamoDB tables: %v", err)
	}

	for _, tableName := range tables {
		// Generate and apply table-specific DDL
		ddl := generateTableDDL(tableName, client)
		if err := createTable(ctx, adminClient, databaseName, ddl); err != nil {
			log.Printf("Failed to create table %s: %v", tableName, err)
			continue
		}

		// Migrate table metadata to Spanner
		err := migrateDynamoTableToSpanner(ctx, databaseName, tableName, client)
		if err != nil {
			log.Printf("Error migrating table %s: %v", tableName, err)
		}
	}

	fmt.Println("Initial setup complete.")
}

// migrateDynamoTableToSpanner migrates a DynamoDB table schema and metadata to Spanner.
func migrateDynamoTableToSpanner(ctx context.Context, db, tableName string, client *dynamodb.Client) error {
	// Load configuration
	config, err := loadConfig("../config.yaml")
	if err != nil {
		return fmt.Errorf("error loading configuration: %v", err)
	}
	models.SpannerTableMap[tableName] = config.Spanner.InstanceID

	// Fetch table attributes and keys from DynamoDB
	attributes, partitionKey, sortKey, err := fetchTableAttributes(client, tableName, int32(config.Spanner.DynamoQueryLimit))
	if err != nil {
		return fmt.Errorf("failed to fetch attributes for table %s: %v", tableName, err)
	}

	// Fetch the current Spanner schema for the table
	spannerSchema, err := fetchSpannerSchema(ctx, db, tableName)
	if err != nil {
		return fmt.Errorf("failed to fetch Spanner schema for table %s: %v", tableName, err)
	}

	// Generate and apply DDL statements for missing columns
	var ddlStatements []string
	for column, dynamoType := range attributes {
		if _, exists := spannerSchema[column]; !exists {
			spannerType := utils.ConvertDynamoTypeToSpannerType(dynamoType)
			ddlStatements = append(ddlStatements, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, column, spannerType))
		}
	}
	if len(ddlStatements) > 0 {
		if err := applySpannerDDL(ctx, db, ddlStatements); err != nil {
			return fmt.Errorf("failed to apply DDL to table %s: %v", tableName, err)
		}
		log.Printf("Schema updated for table %s in Spanner.", tableName)
	}

	// Check for columns that are in Spanner but not in DynamoDB (columns that should be dropped)
	var dropColumnStatements []string
	for column := range spannerSchema {
		if _, exists := attributes[column]; !exists {
			dropColumnStatements = append(dropColumnStatements, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, column))
		}
	}

	// Apply DDL to drop removed columns
	if len(dropColumnStatements) > 0 {
		if err := applySpannerDDL(ctx, db, dropColumnStatements); err != nil {
			return fmt.Errorf("failed to apply DROP COLUMN DDL to table %s: %v", tableName, err)
		}
		log.Printf("Removed columns from table %s in Spanner.", tableName)
	}

	// Prepare mutations to insert metadata into the adapter table
	var mutations []*spanner.Mutation
	for column, dataType := range attributes {
		mutations = append(mutations, spanner.InsertOrUpdate(
			"dynamodb_adapter_table_ddl",
			[]string{"column", "tableName", "dataType", "originalColumn", "partitionKey", "sortKey", "spannerIndexName", "actualTable", "spannerDataType"},
			[]interface{}{column, tableName, dataType, column, partitionKey, sortKey, column, tableName},
		))
	}

	// Perform batch insert into Spanner
	if err := spannerBatchInsert(ctx, db, mutations); err != nil {
		return fmt.Errorf("failed to insert metadata for table %s into Spanner: %v", tableName, err)
	}

	log.Printf("Successfully migrated metadata for table %s to Spanner.", tableName)
	return nil
}

// createDatabase creates a new Spanner database if it does not exist.
func createDatabase(ctx context.Context, adminClient *Admindatabase.DatabaseAdminClient, db string) error {
	// Parse database ID
	matches := regexp.MustCompile("^(.*)/databases/(.*)$").FindStringSubmatch(db)
	if matches == nil || len(matches) != 3 {
		return fmt.Errorf("invalid database ID: %s", db)
	}
	parent, dbName := matches[1], matches[2]

	// Initiate database creation
	op, err := adminClient.CreateDatabase(ctx, &database.CreateDatabaseRequest{
		Parent:          parent,
		CreateStatement: "CREATE DATABASE `" + dbName + "`",
	})
	if err != nil {
		if strings.Contains(err.Error(), "AlreadyExists") {
			log.Printf("Database `%s` already exists. Skipping creation.", dbName)
			return nil
		}
		return fmt.Errorf("failed to initiate database creation: %v", err)
	}

	// Wait for database creation to complete
	if _, err = op.Wait(ctx); err != nil {
		return fmt.Errorf("error while waiting for database creation: %v", err)
	}
	log.Printf("Database `%s` created successfully.", dbName)
	return nil
}

// createTable creates a table in Spanner if it does not already exist.
func createTable(ctx context.Context, adminClient *Admindatabase.DatabaseAdminClient, db, ddl string) error {
	// Extract table name from DDL
	re := regexp.MustCompile(`CREATE TABLE (\w+)`)
	matches := re.FindStringSubmatch(ddl)
	if len(matches) < 2 {
		return fmt.Errorf("unable to extract table name from DDL: %s", ddl)
	}
	tableName := matches[1]

	// Create Spanner client
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to create Spanner client: %v", err)
	}
	defer client.Close()

	// Check if the table already exists
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = @tableName`,
		Params: map[string]interface{}{
			"tableName": tableName,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var tableCount int64
	err = iter.Do(func(row *spanner.Row) error {
		return row.Columns(&tableCount)
	})
	if err != nil {
		return fmt.Errorf("failed to query table existence: %w", err)
	}
	if tableCount > 0 {
		log.Printf("Table `%s` already exists. Skipping creation.", tableName)
		return nil
	}

	// Create the table
	op, err := adminClient.UpdateDatabaseDdl(ctx, &database.UpdateDatabaseDdlRequest{
		Database:   db,
		Statements: []string{ddl},
	})
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return op.Wait(ctx)
}

func listDynamoTables(client *dynamodb.Client) ([]string, error) {
	output, err := client.ListTables(context.TODO(), &dynamodb.ListTablesInput{})
	if err != nil {
		return nil, err
	}
	return output.TableNames, nil
}

// fetchTableAttributes retrieves attributes and key schema (partition and sort keys) of a DynamoDB table.
// It describes the table to get its key schema and scans the table to infer attribute types.
func fetchTableAttributes(client *dynamodb.Client, tableName string, limit int32) (map[string]string, string, string, error) {
	// Describe the DynamoDB table to get its key schema and attributes.
	output, err := client.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to describe table %s: %w", tableName, err)
	}

	var partitionKey, sortKey string
	// Extract the partition key and sort key from the table's key schema.
	for _, keyElement := range output.Table.KeySchema {
		switch keyElement.KeyType {
		case dynamodbtypes.KeyTypeHash:
			partitionKey = aws.ToString(keyElement.AttributeName) // Partition key
		case dynamodbtypes.KeyTypeRange:
			sortKey = aws.ToString(keyElement.AttributeName) // Sort key
		}
	}

	// Map to store inferred attribute types.
	attributes := make(map[string]string)

	// Scan the table to retrieve data and infer attribute types.
	scanOutput, err := client.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(tableName),
		Limit:     aws.Int32(limit),
	})
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to scan table %s: %w", tableName, err)
	}

	// Iterate through the items and infer the attribute types.
	for _, item := range scanOutput.Items {
		for attr, value := range item {
			attributes[attr] = inferDynamoDBType(value)
		}
	}

	return attributes, partitionKey, sortKey, nil
}

// inferDynamoDBType determines the type of a DynamoDB attribute based on its value.
func inferDynamoDBType(attr dynamodbtypes.AttributeValue) string {
	// Check the attribute type and return the corresponding DynamoDB type.
	switch attr.(type) {
	case *dynamodbtypes.AttributeValueMemberS:
		return "S" // String type
	case *dynamodbtypes.AttributeValueMemberN:
		return "N" // Number type
	case *dynamodbtypes.AttributeValueMemberB:
		return "B" // Binary type
	case *dynamodbtypes.AttributeValueMemberBOOL:
		return "BOOL" // Boolean type
	case *dynamodbtypes.AttributeValueMemberSS:
		return "SS" // String Set type
	case *dynamodbtypes.AttributeValueMemberNS:
		return "NS" // Number Set type
	case *dynamodbtypes.AttributeValueMemberBS:
		return "BS" // Binary Set type
	case *dynamodbtypes.AttributeValueMemberNULL:
		return "NULL" // Null type
	case *dynamodbtypes.AttributeValueMemberM:
		return "M" // Map type
	case *dynamodbtypes.AttributeValueMemberL:
		return "L" // List type
	default:
		log.Printf("Unknown DynamoDB attribute type: %T\n", attr)
		return "Unknown" // Unknown type
	}
}

// spannerBatchInsert applies a batch of mutations to a Spanner database.
func spannerBatchInsert(ctx context.Context, databaseName string, mutations []*spanner.Mutation) error {
	// Create a Spanner client.
	client, err := spanner.NewClient(ctx, databaseName)
	if err != nil {
		return fmt.Errorf("failed to create Spanner client: %w", err)
	}
	defer client.Close() // Ensure the client is closed after the operation.

	// Apply the batch of mutations to the database.
	_, err = client.Apply(ctx, mutations)
	return err
}

// createDynamoClient initializes a DynamoDB client using default AWS configuration.
func createDynamoClient() *dynamodb.Client {
	// Load the default AWS configuration.
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}
	return dynamodb.NewFromConfig(cfg) // Return the configured client.
}

// fetchSpannerSchema retrieves the schema of a Spanner table by querying the INFORMATION_SCHEMA.
func fetchSpannerSchema(ctx context.Context, db, tableName string) (map[string]string, error) {
	// Create a Spanner client.
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create Spanner client: %v", err)
	}
	defer client.Close() // Ensure the client is closed after the operation.

	// Query the schema information for the specified table.
	stmt := spanner.Statement{
		SQL: `SELECT COLUMN_NAME, SPANNER_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = @tableName`,
		Params: map[string]interface{}{
			"tableName": tableName,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop() // Ensure the iterator is stopped after use.

	// Map to store the schema information.
	schema := make(map[string]string)
	err = iter.Do(func(row *spanner.Row) error {
		var columnName, spannerType string
		// Extract column name and type from the row.
		if err := row.Columns(&columnName, &spannerType); err != nil {
			return err
		}
		schema[columnName] = spannerType
		return nil
	})
	if err != nil {
		return nil, err
	}
	return schema, nil
}

// applySpannerDDL executes DDL statements on a Spanner database.
func applySpannerDDL(ctx context.Context, db string, ddlStatements []string) error {
	// Create a Spanner Admin client.
	adminClient, err := Admindatabase.NewDatabaseAdminClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Spanner Admin client: %v", err)
	}
	defer adminClient.Close() // Ensure the client is closed after the operation.

	// Initiate the DDL update operation.
	op, err := adminClient.UpdateDatabaseDdl(ctx, &database.UpdateDatabaseDdlRequest{
		Database:   db,
		Statements: ddlStatements,
	})
	if err != nil {
		return fmt.Errorf("failed to initiate DDL update: %v", err)
	}

	// Wait for the DDL update operation to complete.
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("error while waiting for DDL update to complete: %v", err)
	}
	return nil
}
