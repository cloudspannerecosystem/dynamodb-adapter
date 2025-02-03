# Integration Tests

### config.yaml
This file defines the necessary settings for the adapter. A sample configuration might look like this:


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
```

## Execute tests

```sh
go run integrationtest/setup.go setup
go test integrationtest/api_test.go
go run integrationtest/setup.go teardown
```
