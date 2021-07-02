# Overview

These scripts create a sample set of DynamoDB tables and load them with data.
The data model represents a simple ecommerce application, where you have a
table of customers, orders and order line items, in the Customer_Order table.
The model also contains a Product table which contains a set of books and
electronics that could be purchased via the ecommerce application.

## Setup

Export your AWS credentials:

```shell
export AWS_REGION=[your region]
export AWS_ACCESS_KEY_ID=[your access key id]
export AWS_SECRET_ACCESS_KEY=[your secret key]
export AWS_SESSION_TOKEN=[if using multi-factor authentication]
```

Install the Python requirements:

```shell
pip install
```

Run the load scripts:

```shell
python create_customer_order.py
python create_product.py
python put_customer_order_items.py
puyton put_product_items.py
```
