# DynamoDB Adapter Demo Setup

This README outlines the steps to set up a test environment for
using the DynamoDB adapter with Cloud Spanner.
Follow these instructions to create the necessary tables,
insert sample data, and configure the adapter.

## 1. Create a Sample Table for Demo Operations

The following SQL statement creates a table `employee_table` in Cloud Spanner,
which will be used to demonstrate DynamoDB operations through the adapter:

```sql
CREATE TABLE employee_table (
  emp_name STRING(MAX),
  emp_id FLOAT64,
  emp_image BYTES(MAX),
  isHired BOOL,
  emp_status STRING(MAX),
  emp_details JSON
) PRIMARY KEY(emp_id);
```

## 2. Create `dynamodb_adapter_table_ddl` Table

The DynamoDB adapter requires a table to store metadata about
the schema for other tables.
Use the following SQL statement to create this table:

```sql
CREATE TABLE dynamodb_adapter_table_ddl (
tableName STRING(MAX) NOT NULL,
column STRING(MAX) NOT NULL,
dynamoDataType STRING(MAX) NOT NULL,
originalColumn STRING(MAX) NOT NULL,
partitionKey STRING(MAX),
sortKey STRING(MAX),
spannerIndexName STRING(MAX),
actualTable STRING(MAX),
spannerDataType STRING(MAX)
) PRIMARY KEY (tableName, column)
```

## 3. Insert Data into `dynamodb_adapter_table_ddl`

Once the `dynamodb_adapter_table_ddl` table is created,
insert the metadata for `employee_table` as follows:

```sql
INSERT INTO dynamodb_adapter_table_ddl (tableName, column, dynamoDataType, originalColumn, partitionKey,sortKey, spannerIndexName, actualTable, spannerDataType) VALUES ('employee_table','emp_name','S','emp_name','emp_id', '', 'emp_name', 'employee_table','STRING(MAX)');

INSERT INTO dynamodb_adapter_table_ddl (tableName, column, dynamoDataType, originalColumn, partitionKey,sortKey, spannerIndexName, actualTable, spannerDataType) VALUES ('employee_table','emp_id','N','emp_id','emp_id', '', 'emp_id', 'employee_table','FLOAT64');

INSERT INTO dynamodb_adapter_table_ddl (tableName, column, dynamoDataType, originalColumn, partitionKey,sortKey, spannerIndexName, actualTable, spannerDataType)
VALUES ('employee_table','emp_image','B','emp_image','emp_id', '', 'emp_image', 'employee_table','BYTES(MAX)');

INSERT INTO dynamodb_adapter_table_ddl (tableName, column, dynamoDataType, originalColumn, partitionKey,sortKey, spannerIndexName, actualTable, spannerDataType)
VALUES ('employee_table','isHired','BOOL','isHired','emp_id', '', 'isHired', 'employee_table','BOOL');

INSERT INTO dynamodb_adapter_table_ddl (tableName, column, dynamoDataType, originalColumn, partitionKey,sortKey, spannerIndexName, actualTable, spannerDataType)
VALUES ('employee_table','emp_status','S','emp_status','emp_id', '', 'emp_status', 'employee_table','STRING(MAX)');

INSERT INTO dynamodb_adapter_table_ddl (tableName, column, dynamoDataType, originalColumn, partitionKey,sortKey, spannerIndexName, actualTable, spannerDataType)
VALUES ('employee_table','emp_details','L','emp_details','emp_id', '', 'emp_details', 'employee_table','JSON');
```

## 4. Configuration Files

The DynamoDB adapter requires several configuration files to function properly.
Below are examples of these configuration files:

## config.yaml

This file defines the necessary settings for the adapter.
A sample configuration might look like this:

spanner:
        project_id: "my-project-id"
        instance_id: "my-instance-id"
        database_name: "my-database-name"
        query_limit: "query_limit"
        dynamo_query_limit: "dynamo_query_limit"

The fields are:
project_id: The Google Cloud project ID.
instance_id: The Spanner instance ID.
database_name: The database name in Spanner.
query_limit: Database query limit.
dynamo_query_limit: DynamoDb query limit.

## 5. Build the Project

Once the configuration files are set up,
build the DynamoDB Adapter project by running:

```bash
go build
```

## 6. Start the Adapter

After building the project, start the DynamoDB Adapter by running the following command:

```bash
./dynamodb-adapter
```

---

Follow these steps to set up your DynamoDB Adapter environment
and perform necessary operations using the `employee_table`.
Let me know if you need further clarification or assistance!
