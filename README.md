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

**OUTDATED** - The adapter project includes an example application and sample eCommerce
data model. The [instructions](./examples/README.md) for the sample
application include migration using [Harbourbridge](https://github.com/cloudspannerecosystem/harbourbridge)
and [setup](./examples/README.md#initialize_the_adapter_configuration) for
the adapter.

## Compatibility Matrix

### Supported Actions

DynamoDB Adapter currently supports the following operations:

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
| TransactGetItems |
| TransactWriteItems |

### Supported Data Types

DynamoDB Adapter currently supports the following DynamoDB data types

| DynamoDB Data Type            | Spanner Data Types |
| ------------------------------| ------------------ |
| `N` (number type)             | `INT64`, `FLOAT64`, `NUMERIC`, `TIMESTAMP` (EPOCH seconds) |
| `BOOL` (boolean)              | `BOOL` |
| `B` (binary type)             | `BYTES(MAX)` |
| `S` (string and data values)  | `STRING(MAX)` |
| `SS` (string set)             | `ARRAY<STRING(MAX)>` |
| `NS` (number set)             | `ARRAY<FLOAT64>` |
| `BS` (binary set)             | `ARRAY<BYTES(MAX)>` |
| `L` (List Type)               | `JSON` |
| `M` (Map Type)                | `JSON` |

Note: Map and List datatypes does not support the Set datatypes.

## Configuration

This DynamoDB Adapter requires some initial setup in order to work. There is an initialization section to help bootstrap and create required Spanner tables. Running the init code isn't required but keep in mind that you will have to manually create resources (noted below).

### Auth

#### Spanner Credentials
The adapter and initialization both expect GOOGLE_APPLICATION_CREDENTIALS in order to run. On platforms like GCE, GKE, etc., this will be auto picked up at runtime. Locally, make sure to run the following:
```sh
gcloud auth application-default login
```

#### DynamoDB Credentials
These can be set either in the `.env` file or by running the following:
```sh
export AWS_ACCESS_KEY_ID=YOUR_ACCESS_KEY
export AWS_SECRET_ACCESS_KEY=YOUR_SECRET_KEY
export AWS_REGION=YOUR_REGION
```

### config.yaml

This file defines the necessary settings for the adapter. You should avoid changing this file directly as all fields can be overwritten with env vars. Both the adapters `main.go` and the initilaization's `init.go` will pull in the `config.yaml` file and allow overriding of any value via env vars or a `.env` file.

You can override `config.yaml` like the following:

`config.yaml`
```
spanner:
  project_id: SPANNER_PROJECT_ID
```

`.env`
```
SPANNER_PROJECT_ID=<yourprojectid>
```

### .env

The `.env` file is used to override `config.yaml`. It is not required and you can simply set env vars directly. For deployments on platforms like Docker or GKE, you likely will set env vars specifically for that platform.

Copy the `.env.example` file to `.env` and set needed variables.
```sh
cp .env.example .env
```

### Required Spanner Tables

* `dynamodb_adapter_table_ddl`
  * Stores the metadata for all DynamoDB tables now
stored in Cloud Spanner. It is used when the adapter starts up to create a map
for all the columns names present in Spanner tables with the columns of tables
present in DynamoDB. This mapping is required because DynamoDB supports the
special characters in column names while Cloud Spanner only supports
underscores(_). For more: [Spanner Naming Conventions](https://cloud.google.com/spanner/docs/data-definition-language#naming_conventions)
  * This table also maps DynamoDB types to the underlying Spanner types. This is particularly important for mapping DynamoDB Number to Spanner since there isn't a 1to1 mapping. The adapter will read the Spanner column type and auto-convert reads/writes for that attribute to the closest type to match.
* `dynamodb_adapter_config_manager`

If you opt to not use the init code, you can create these tables manually by running:
```SQL
CREATE TABLE dynamodb_adapter_table_ddl (
  column	     STRING(MAX),
  tableName      STRING(MAX),
  dataType       STRING(MAX),
  originalColumn STRING(MAX),
) PRIMARY KEY (tableName, column)

CREATE TABLE dynamodb_adapter_config_manager (
  tableName     STRING(MAX),
  config 	    STRING(MAX),
  cronTime      STRING(MAX),
  enabledStream STRING(MAX),
  uniqueValue   STRING(MAX),
) PRIMARY KEY (tableName)
```

## Initialization

This repo provides init code to bootstrap the needed resources for the adapter to run. Running this is not required but note that you will have to manually setup required tables. Commands will only run if the resources don't already exist. The init will perform the following:

* Creates new database in Spanner
* Creates adapter required tables
  * `dynamodb_adapter_table_ddl`
  * `dynamodb_adapter_config_manager`
* Reads from source DynamoDB tables
* Creates tables in Spanner converting names to match Spanner restrictions
* Creates table columns converting DynamoDB types to Spanner types on a best effort basis.
  * Note that by default, all number types are mapped to FLOAT64. You will have to manually adjust the schema for other types.
* Creates Spanner indexes converting from DynamoDB GSIs and LSIs
* Inserts rows into the `dynamodb_adapter_table_ddl` table to map DynamoDB -> Spanner attributes

Before starting, make sure you followed the Configuration setup above. You can ignore any steps calling for manually creating resources.

You can run the init in dry run mode which will only generate statements instead of creating resources. You can then remove the `--dry_run` flag to actually create needed resources:
```sh
go run config-files/init.go --dry_run
```

## Starting DynamoDB Adapter

To start from scratch, complete the steps described in https://cloud.google.com/spanner/docs/getting-started/set-up, which
covers creating and setting a default Google Cloud project, enabling billing,
enabling the Cloud Spanner API, and setting up OAuth 2.0 to get authentication
credentials to use the Cloud Spanner API.

Ensure you have already followed the Configuration section notes above.

### Locally or with Binary
Run directly
```sh
go run main.go
```

Or

Build
```sh
go build \
  -ldflags "-X github.com/cloudspannerecosystem/dynamodb-adapter/config.proxyReleaseVersion=$(cat VERSION)" \
  -o dynamodb-adapter
```

Run Binary
```sh
./dynamodb-adapter
```

### Docker

Set needed env vars (for publishing)
```
export ARTIFACT_REGISTRY_NAME="<registry>"
export ARTIFACT_REGISTRY_PROJECT_ID="<project>"
export ARTIFACT_REGISTRY_REGION="<region>"
```

Build
```sh
docker build \
  --platform linux/amd64 \
  --build-arg "PROXY_RELEASE_VERSION=$(cat VERSION)" \
  --tag $ARTIFACT_REGISTRY_REGION-docker.pkg.dev/$ARTIFACT_REGISTRY_PROJECT_ID/$ARTIFACT_REGISTRY_NAME/dynamodb-adapter:$(cat VERSION) .
```

Running locally (passes in local GCP creds)
```sh
docker run \
  --publish 9050:9050 \
  --name dynamodb-adapter \
  --detach \
  --volume $HOME/.config/gcloud/application_default_credentials.json:/app/application_default_credentials.json:ro \
  --env GOOGLE_APPLICATION_CREDENTIALS=/app/application_default_credentials.json \
  --env-file .env \
  $ARTIFACT_REGISTRY_REGION-docker.pkg.dev/$ARTIFACT_REGISTRY_PROJECT_ID/$ARTIFACT_REGISTRY_NAME/dynamodb-adapter:$(cat VERSION)
```

Publish
```sh
gcloud auth configure-docker $ARTIFACT_REGISTRY_REGION-docker.pkg.dev
docker push $ARTIFACT_REGISTRY_REGION-docker.pkg.dev/$ARTIFACT_REGISTRY_PROJECT_ID/$ARTIFACT_REGISTRY_NAME/dynamodb-adapter:$(cat VERSION)
```

## API Documentation

This is can be imported in Postman or can be used for Swagger UI.
You can get open-api-spec file here [here](https://github.com/cldcvr/dynamodb-adapter/wiki/Open-API-Spec)

## Development

### Unit Tests

Run with 
```sh
go test ./... -short 
```

### Integration Tests

The integrations tests run against a live Spanner instance to perform validations. Integration tests rely on the same `config.yaml` but a local `.env` file if being used.

If using `.env` files, copy the example to the integrationtest path:
```sh
cp .env.example integrationtest/.env
```

```sh
cd integrationtest
go test . -v
```

You can also manually setup/teardown the integration test DB
```sh
go run setup.go setup
go run setup.go teardown
```
