# Examples

As an example we have created a golang application that you can use to
experiment with the DynamoDB adapter. The sample application uses the ecommerce
data in the `sample-data` directory. Using those scripts you will create tables
in DynamoDB, once that is done follow these instructions to migrate the data,
setup the adapter and then run the sample application.

## Create sample data

To create the sample data follow the instructions in the
[sample-data README](sample-data/README.md).

## Create target Cloud Spanner instance

Now you will create a Cloud Spanner instance to migrate the data into

```shell
gcloud config set project [your project id]
gcloud spanner instances create dynamodb-migration --config=regional-us-central1 --nodes=1
```

## Mirgrate the data

For the data migration we'll use [Harbourbridge](https://github.com/cloudspannerecosystem/harbourbridge).
Follow the [installation instructions](https://github.com/cloudspannerecosystem/harbourbridge#installing-harbourbridge)
to get Harbourbridge installed. Harbourbridge will analyze the data to
determine the schema and then also copy the data across to Cloud Spanner.

Export your AWS credentials:

```shell
export AWS_REGION=[your region]
export AWS_ACCESS_KEY_ID=[your access key id]
export AWS_SECRET_ACCESS_KEY=[your secret key]
export AWS_SESSION_TOKEN=[if using multi-factor authentication]
```

Migrate the data:

```shell
harbourbridge -driver=dynamodb -instance=dynamodb-migration -dbname=ecommerce
```

Create the indexes:
```
CREATE INDEX By_customer
  ON Customer_Order (customer_id, order_ts ASC);

CREATE INDEX By_Product_Category
  ON Product (product_category, product_id ASC);
```

## Initialize the adapter configuration

The DynamoDB adapter uses tables in Cloud Spanner to store some configuration
data. `adapter/init.go` is a helper tool that will create those tables and
populate them using the schema that was just created by Harbourbridge.
`adapter/init.go` also uses the sample configuration files stored in
[adapter/config-files/staging](adapter/config-files/staging) contains examples
of the config-files needed for the ecommerce sample.

Set your project id in the sample configuration files:

```shell
cd examples
sed -i "s/YOUR_PROJECT_HERE/[your project id]/g" adapter/config-files/staging/config.json
```

Initialize the adapter's configuration tables:

```shell
cd adapter
go run init.go setup
cd ..
```

## Build the adapter

```shell
cd ..
Replace config-files with config-files under example dir

sed -i 's#rice\.MustFindBox("config-files")#rice\.MustFindBox("examples/adapter/config-files")'#g main.go

go build
cp dyanmodb-adapter examples/adpater/
```

## Start the adapter

```shell
cd examples/adapter
./dynamodb-adapter
```

Should see output similar to:

```shell
2021-09-14T23:16:45.994-0600	DEBUG	logger/logger.go:66	[Fetching starts]
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:	export GIN_MODE=release
 - using code:	gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /debug/pprof/             --> github.com/gin-contrib/pprof.pprofHandler.func1 (3 handlers)
[GIN-debug] GET    /debug/pprof/cmdline      --> github.com/gin-contrib/pprof.pprofHandler.func1 (3 handlers)
[GIN-debug] GET    /debug/pprof/profile      --> github.com/gin-contrib/pprof.pprofHandler.func1 (3 handlers)
[GIN-debug] POST   /debug/pprof/symbol       --> github.com/gin-contrib/pprof.pprofHandler.func1 (3 handlers)
[GIN-debug] GET    /debug/pprof/symbol       --> github.com/gin-contrib/pprof.pprofHandler.func1 (3 handlers)
[GIN-debug] GET    /debug/pprof/trace        --> github.com/gin-contrib/pprof.pprofHandler.func1 (3 handlers)
[GIN-debug] GET    /debug/pprof/allocs       --> github.com/gin-contrib/pprof.pprofHandler.func1 (3 handlers)
[GIN-debug] GET    /debug/pprof/block        --> github.com/gin-contrib/pprof.pprofHandler.func1 (3 handlers)
[GIN-debug] GET    /debug/pprof/goroutine    --> github.com/gin-contrib/pprof.pprofHandler.func1 (3 handlers)
[GIN-debug] GET    /debug/pprof/heap         --> github.com/gin-contrib/pprof.pprofHandler.func1 (3 handlers)
[GIN-debug] GET    /debug/pprof/mutex        --> github.com/gin-contrib/pprof.pprofHandler.func1 (3 handlers)
[GIN-debug] GET    /debug/pprof/threadcreate --> github.com/gin-contrib/pprof.pprofHandler.func1 (3 handlers)
[GIN-debug] GET    /doc/*any                 --> github.com/swaggo/gin-swagger.CustomWrapHandler.func1 (3 handlers)
[GIN-debug] GET    /                         --> main.main.func1 (3 handlers)
[GIN-debug] POST   /v1                       --> github.com/cloudspannerecosystem/dynamodb-adapter/api/v1.RouteRequest (3 handlers)
[GIN-debug] Listening and serving HTTP on :9050

```

## Run the sample application

The sample application can run queries directly against DynamoDB using `dynamo`
argument or run queries against Cloud Spanner using the `spanner` argument.

Query the ecommerce data using the DynamoDB adapter:

```shell
cd golang
go build
./golang spanner
```
