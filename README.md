# DynamoDB Adapter

[![Join the chat at
https://gitter.im/cloudspannerecosystem/dynamodb-adapter](https://badges.gitter.im/cloudspannerecosystem/dynamodb-adapter.svg)](https://gitter.im/cloudspannerecosystem/dynamodb-adapter?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

[![cloudspannerecosystem](https://circleci.com/gh/cloudspannerecosystem/dynamodb-adapter.svg?style=svg)](https://circleci.com/gh/cloudspannerecosystem/dynamodb-adapter)

## Introduction

DynamoDB Adapter is a tool that translates AWS DynamoDB queries to Cloud
Spanner equivalent queries and runs those queries on Cloud Spanner. The
adapter serves as a proxy whereby applications that use DynamoDB can send their
queries to the adapter where they are then translated and executed against
Cloud Spanner. DynamoDB Adapter is helpful when moving to Cloud Spanner from
a DynamoDB environment without changing the code for DynamoDB queries. The APIs
created by this project can be directly consumed where DynamoDB queries are
used in your application.

The adapter supports the basic data types and operations required for most
applications.  Additionally, it also supports primary and secondary indexes in
a similar way as DynamoDB. For detailed comparison of supported operations and
data types, refer to the [Compatibility Matrix](#compatibility_matrix)

## Examples and Quickstart

The adapter project includes an example application and sample eCommerce
data model. The [instructions](./examples/README.md) for the sample
application include migration using [Harbourbridge](https://github.com/cloudspannerecosystem/harbourbridge)
and [setup](./examples/README.md#initialize_the_adapter_configuration) for
the adapter.

## Compatibility Matrix

### Supported Actions

DynamoDB Adapter currently supports the folowing operations:

| DynamoDB Action |
|----------------|
| BatchGetItem |
| BatchWriteItem |
| DeleteItem |
| GetItem |
| PutItem |
| Query |
| Scan |
| UpdateItem |
| TransactWriteItems |

### Supported Data Types

DynamoDB Adapter currently supports the following DynamoDB data types

| DynamoDB Data Type            | Spanner Data Types |
| ------------------------------| ------------------ |
| `N` (number type)             | `INT64`, `FLOAT64`, `NUMERIC` |
| `BOOL` (boolean)              | `BOOL` |
| `B` (binary type)             | `BYTES(MAX)` |
| `S` (string and data values)  | `STRING(MAX)` |
| `SS` (string set)             | `ARRAY<STRING(MAX)>` |
| `NS` (number set)             | `ARRAY<FLOAT64>` |
| `BS` (binary set)             | `ARRAY<BYTES(MAX)>` |
| `L` (List Type)               | `JSON` |

## Configuration

### config.yaml

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

### dynamodb_adapter_table_ddl

`dynamodb_adapter_table_ddl` stores the metadata for all DynamoDB tables now
stored in Cloud Spanner. It is used when the adapter starts up to create a map
for all the columns names present in Spanner tables with the columns of tables
present in DynamoDB. This mapping is required because DynamoDB supports the
special characters in column names while Cloud Spanner only supports
underscores(_). For more: [Spanner Naming Conventions](https://cloud.google.com/spanner/docs/data-definition-language#naming_conventions)

### Initialization Modes

DynamoDB Adapter supports two modes of initialization:

#### Dry Run Mode

This mode generates the Spanner queries required to:

Create the dynamodb_adapter_table_ddl table in Spanner.
Insert metadata for all DynamoDB tables into dynamodb_adapter_table_ddl.
These queries are printed to the console without executing them on Spanner,
allowing you to review them before making changes.

```sh
go run config-files/init.go --dry_run
```

#### Execution Mode

This mode executes the Spanner queries generated
during the dry run on the Spanner instance. It will:

Create the dynamodb_adapter_table_ddl table in Spanner if it does not exist.
Insert metadata for all DynamoDB tables into the dynamodb_adapter_table_ddl table.

```sh

go run config-files/init.go

```

### Prerequisites for Initialization

AWS CLI:
Configure AWS credentials:

```sh

aws configure set aws_access_key_id YOUR_ACCESS_KEY
aws configure set aws_secret_access_key YOUR_SECRET_KEY
aws configure set default.region YOUR_REGION
aws configure set aws_session_token YOUR_SESSION_TOKEN

```

Google Cloud CLI:
Authenticate and set up your environment:

```sh

gcloud auth application-default login
gcloud config set project [MY_PROJECT_NAME]

```

## Starting DynamoDB Adapter

Complete the steps described in
[Set up](https://cloud.google.com/spanner/docs/getting-started/set-up), which
covers creating and setting a default Google Cloud project, enabling billing,
enabling the Cloud Spanner API, and setting up OAuth 2.0 to get authentication
credentials to use the Cloud Spanner API.

In particular, ensure that you run

```sh

gcloud auth application-default login

```

to set up your local development environment with authentication credentials.

Set the GCLOUD_PROJECT environment variable to your Google Cloud project ID:

```sh

gcloud config set project [MY_PROJECT NAME]

```

```sh
go run main.go

```

```

## API Documentation

This is can be imported in Postman or can be used for Swagger UI.
You can get open-api-spec file here [here](https://github.com/cldcvr/dynamodb-adapter/wiki/Open-API-Spec)
