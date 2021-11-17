# Integration Tests

Running the integration tests will require the files present in the
[staging](./config-files/staging) folder to be configured as below:

## Config Files

`config-files/staging/config.json`

```json
{
    "GoogleProjectID": "<your-project-id>",
    "SpannerDb": "<any-db-name>",
    "QueryLimit": 5000
}
```

`config-files/staging/spanner.json`

```json
{
    "dynamodb_adapter_table_ddl": "<spanner-instance-name>",
    "dynamodb_adapter_config_manager": "<spanner-instance-name>",
    "department": "<spanner-instance-name>",
    "employee": "<spanner-instance-name>"
}
```

`config-files/staging/tables.json`

```json
{
    "employee": {
        "partitionKey": "emp_id",
        "sortKey": "",
        "attributeTypes": {
            "emp_id": "N",
            "first_name": "S",
            "last_name": "S",
            "address": "S",
            "age": "N"
        },
        "indices": {}
    },
    "department": {
        "partitionKey": "d_id",
        "sortKey": "",
        "attributeTypes": {
            "d_id": "N",
            "d_name": "S",
            "d_specialization": "S"
        },
        "indices": {}
    }
}
```

## Execute tests

```sh
go run integrationtest/setup.go setup
go test integrationtest/api_test.go
go run integrationtest/setup.go teardown
```
