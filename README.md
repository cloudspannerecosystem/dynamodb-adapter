# dynamodb-adapter

[![Join the chat at
https://gitter.im/cloudspannerecosystem/dynamodb-adapter](https://badges.gitter.im/cloudspannerecosystem/dynamodb-adapter.svg)](https://gitter.im/cloudspannerecosystem/dynamodb-adapter?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)


## Introduction
Dynamodb-adapter is an API tool which translates the AWS Dynamodb queries to Cloud Spanner equivalent queries and runs those queries on Cloud Spanner. By running this project locally or on cloud, this would work seemlessly.

Additionally, It also support the primary index and secondary index in similar way as dynamodb supports.

It will be helpful for moving to the Spanner from dynamodb environment without changing the code of dynamodb. APIs created by this project can be directly consumed at place of dynamodb queries.

This project require two tables to store metadata and configuation for the project.
* dynamodb_adapter_table_ddl (for meta data of all tables)
* dynamodb_adapter_config_manager (for pubsub configuation)

It supports two mode  - 
* Production
* Staging

## Usage
Follow the given steps to setup the project and generate the apis.

### 1. Creation for the Base Tables in Spanner
#### Table: dynamodb_adapter_table_ddl
This table will be used to store the metadata for other tables. It will be used at the time of initiation of project to create a map for all the columns names present in Spanner tables with the columns of tables present in DynamoDB. This mapping is required by because dynamoDB supports the special characters in column names while spanner does not support special characters other than underscores(_). 
For more: [Spanner Naming Conventions](https://cloud.google.com/spanner/docs/data-definition-language#naming_conventions)

```
CREATE TABLE 
dynamodb_adapter_table_ddl 
(
 column		    STRING(MAX),
 tableName	    STRING(MAX),
 dataType 	    STRING(MAX),
 originalColumn     STRING(MAX),
) PRIMARY KEY (tableName, column)
```

Add the meta data of all the tables in the similar way as shown below.

![dynamodb_adapter_table_ddl sample data](images/config_spanner.png)

#### Table: dynamodb_adapter_config_manager
This table will be used to store the configuration info for publishing the data in Pub/Sub topic for other processes on change of data. It will be used to do some additional operation required on the change of data in tables. It can trigger New and Old data on given Pub/Sub topic. 

```
CREATE TABLE 
dynamodb_adapter_config_manager 
(
 tableName 	STRING(MAX),
 config 	STRING(MAX),
 cronTime 	STRING(MAX),
 enabledStream 	STRING(MAX),
 pubsubTopic    STRING(MAX),
 uniqueValue    STRING(MAX),
) PRIMARY KEY (tableName)
```


### 2. Creation for configuration files
There are two folders in [config-files](./config-files). 
* **production** : It will be used to store the config files related to Production Environment.
* **staging** : It will be used to store the config files related to Production Environment. 

Add the configuration in the given files:
#### config.{env}.json 
| Key | Used For |
| ------ | ------ |
| GOOGLE_PROJECT_ID | Your Google Project ID |
| SPANNER_DB | Your Spanner Database Name |

For example:
```
{
    "GOOGLE_PROJECT_ID" : "first-project",
    "SPANNER_DB"        : "test-db"
}
```

#### spanner.{env}.json
It is a mapping file for table name with instance id. It will be helpful to query data on particular instance.
All table's instance-id will be stored in this file in this format: 
"TableName" : "instance-id"

For example:

```
{
    "dynamodb_adapter_table_ddl": "spanner-2 ",
    "dynamodb_adapter_config_manager": "spanner-2",
    "tableName1": "spanner-1",
    "tableName2": "spanner-1"
}
```

#### tabels.{env}.json
All table's primary key, columns, index information will be stored here.

| Key | Used For |
| ------ | ------ |
| tableName | table name present in dbynamo db |
| paritionKey | Primary key |
| sortKey| Sorting key |
| attributeTypes | Coulmn names and type present |
| indices | indexes present in the table |


For example:

```
{
    "tableName":{
        "partitionKey":"primary key or Partition key",
        "sortKey": "sorting key of dynamoDB adaptar",
        "attributeTypes": {
			"ColmnnName1": "Type of like N & S for ColmnnName1",
			"ColmnnName2": "Type of like N & S for ColmnnName2"
        },
        "indices": { 
			"indexName1": {
				"sortKey": "sort key indexName1",
				"partitionKey": "partition key for indexName1"
			}
		}
    }
}
```


### 3. Creation of rice-box.go file

##### install rice package
This package is required to load the config files. This is required in the first step of the running dynamoDB-adapter.

```
go get github.com/GeertJohan/go.rice
go get github.com/GeertJohan/go.rice/rice
```
##### run command for creating the file.
This is required to increase the performance when any config file is changed so that configuration files can be loaded directly from go file.
```
rice embed-go
```

### 4. Run 
* Setup GCP project on **gcloud cli** 

* Run for **staging**
    ```
    go run main.go
    ```
* Run for **Production**
    ```
    export ACTIVE_ENV=PRODUCTION
    go run main.go
    ```

## Starting Process
* Step 1: DynamoDB-adapter will load the configuration according the Environment Variable *ACTIVE_ENV*
* Step 2: DynamoDB-adapter will initialize all the connections for all the instances so that it doesn't need to start the connection again and again for every request.
* Step 3: DynamoDB-adapter will parse the data inside dynamodb_adapter_table_ddl table and will store in ram for faster access of data.
* Step 4: DynamoDB-adapter will parse the dynamodb_adapter_config_manager table then will load it in ram. It will check for every 1 min if data has been changed in this table or not. If data is changed then It will update the data for this in ram. 
* Step 5: After all these steps, DynamoDB-adapter will start the APIs which are similar to dynamodb APIs.


## API Documentation
This is can be imported in Postman or can be used for Swagger UI.
You can get open-api-spec file here [here](https://github.com/cldcvr/dynamodb-adapter/wiki/Open-API-Spec)

