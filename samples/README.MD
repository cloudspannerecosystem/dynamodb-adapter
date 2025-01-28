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
  isHired BOOL
) PRIMARY KEY(emp_id);
```

## 2. Create `dynamodb_adapter_table_ddl` Table

The DynamoDB adapter requires a table to store metadata about
the schema for other tables.
Use the following SQL statement to create this table:

```sql
CREATE TABLE dynamodb_adapter_table_ddl (
  column         STRING(MAX),
  tableName      STRING(MAX),
  dataType       STRING(MAX),
  originalColumn STRING(MAX),
) PRIMARY KEY (tableName, column);
```

## 3. Insert Data into `dynamodb_adapter_table_ddl`

Once the `dynamodb_adapter_table_ddl` table is created,
insert the metadata for `employee_table` as follows:

```sql
INSERT INTO dynamodb_adapter_table_ddl (column, tableName, dataType, originalColumn)
VALUES ('emp_name', 'employee_table', 'STRING(MAX)', 'emp_name');

INSERT INTO dynamodb_adapter_table_ddl (column, tableName, dataType, originalColumn)
VALUES ('emp_id', 'employee_table', 'FLOAT64', 'emp_id');

INSERT INTO dynamodb_adapter_table_ddl (column, tableName, dataType, originalColumn)
VALUES ('emp_image', 'employee_table', 'BYTES(MAX)', 'emp_image');

INSERT INTO dynamodb_adapter_table_ddl (column, tableName, dataType, originalColumn)
VALUES ('isHired', 'employee_table', 'BOOL', 'isHired');
```

## 4. Configuration Files

The DynamoDB adapter requires several configuration files to function properly.
Below are examples of these configuration files:

### 4.1. `config.json`

The `config.json` file contains the basic settings for the DynamoDB Adapter.
Replace the `GoogleProjectID` and `SpannerDb` values with your own project
and database names:

```json
{
    "GoogleProjectID": "cassandra-to-spanner",
    "SpannerDb": "cluster10",
    "QueryLimit": 5000
}
```

### 4.2. `spanner.json`

The `spanner.json` file maps table names to Cloud Spanner instance IDs.
Define the instance where each table resides:

```json
{
    "dynamodb_adapter_table_ddl": "spanner-instance-dev",
    "dynamodb_adapter_config_manager": "spanner-instance-dev",
    "employee_table": "spanner-instance-dev"
}
```

### 4.3. `tables.json`

The `tables.json` file describes the DynamoDB table schema,
including the partition and sort keys. Here's an example for `employee_table`:

```json
{
    "employee_table": {
        "partitionKey": "guid",
        "sortKey": "context",
        "spannerIndexName": "guid",
        "actualTable": "employee_table"
    }
}
```

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






