// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package config implements the functions for reading
// configuration files and saving them into Golang objects

package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func main() {
	var sess *session.Session

	switch cmd := os.Args[1]; cmd {
	case "spanner":
		sess = createAdapterSession()
	case "dynamo":
		sess = createSession("")
	}

	svc := dynamodb.New(sess)

	fmt.Println("Dynamo GetItem:")
	getCustomerContactDetails(svc)
	
	fmt.Println("\nDynamo PK/SK Query:")
	getCustomerOrderDetails(svc)
	
	fmt.Println("\nDynamo Index Query:")
	getCustomerMostRecentOrder(svc)

	fmt.Println("\nDynamo PK Query:")
	getOrderLineItemDetails(svc)

	fmt.Println("\nDynamo Index Query")
	getProductsByCategory(svc)
}

func getCustomerContactDetails(svc *dynamodb.DynamoDB) {
	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("Customer_Order"),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String("CUST#0000000000"),
			},
			"SK": {
				S: aws.String("EMAIL#homer@email.com"),
			},
		},
	})

	if err != nil {
		fmt.Println(err.Error())
	}
	if result.Item == nil {
		fmt.Printf("No record found")
	}
	
	customer := Customer{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &customer)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
	}

	printCustomer(customer)
}

func getCustomerOrderDetails(svc *dynamodb.DynamoDB) {
	result, err := svc.Query(&dynamodb.QueryInput{
		TableName: aws.String("Customer_Order"),
		KeyConditionExpression: aws.String("PK = :customer_id and SK = :order_id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":customer_id": {
				S: aws.String("CUST#0000000000"),
			},
			":order_id": {
				S: aws.String("ORDER#ej68vuldzgps"),
			},
		},
	})

	if err != nil {
		fmt.Println(err.Error())
	}
	if result.Items == nil {
		fmt.Printf("No record found")
	}
	
	customer := Customer{}
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &customer)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
	}

	order := Order{}
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &order)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
	}

	printCustomer(customer)
	printOrder(order)
}

func getCustomerMostRecentOrder(svc *dynamodb.DynamoDB) {
	result, err := svc.Query(&dynamodb.QueryInput{
		TableName: aws.String("Customer_Order"),
		IndexName: aws.String("By_Customer_And_Order_TS"),
		KeyConditionExpression: aws.String("customer_id = :customer_id AND order_ts BETWEEN :order_ts_start AND :order_ts_end"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":customer_id": {
				S: aws.String("0000000000"),
			},
			":order_ts_start": {
				S: aws.String("2021-05-01T00:00:00.000000"),
			},
			":order_ts_end": {
				S: aws.String("2021-05-31T00:00:00.000000"),
			},
		},
	})

	if err != nil {
		fmt.Println(err.Error())
	}
	if result.Items == nil {
		fmt.Printf("No record found")
	}
	
	customer := Customer{}
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &customer)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
	}
	printCustomer(customer)

	for _, item := range result.Items {
		order := Order{}
		err = dynamodbattribute.UnmarshalMap(item, &order)
		if err != nil {
			panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
		}

		printOrder(order)
	}
}

func getOrderLineItemDetails(svc *dynamodb.DynamoDB) {
	result, err := svc.Query(&dynamodb.QueryInput{
		TableName: aws.String("Customer_Order"),
		KeyConditionExpression: aws.String("PK = :order_id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":order_id": {
				S: aws.String("ORDER#ej68vuldzgps"),
			},
		},
	})

	if err != nil {
		fmt.Println(err.Error())
	}
	if result.Items == nil {
		fmt.Printf("No record found")
	}

	order := Order{}
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &order)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
	}
	printOrder(order)

	for _, item := range result.Items {
		lineItem := LineItem{}
		err = dynamodbattribute.UnmarshalMap(item, &lineItem)
		if err != nil {
			panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
		}
		printLineItem(lineItem)
	}
}

func getProductsByCategory(svc *dynamodb.DynamoDB) {
	result, err := svc.Query(&dynamodb.QueryInput{
		TableName: aws.String("Product"),
		IndexName: aws.String("By_Product_Category"),
		KeyConditionExpression: aws.String("product_category = :product_category"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":product_category": {
				S: aws.String("HardGood#Musical Instruments#Drums & Percussion"),
			},
		},
	})

	if err != nil {
		fmt.Println(err.Error())
	}
	if result.Items == nil {
		fmt.Printf("No record found")
	}

	for _, item := range result.Items {
		electronic := Electronic{}
		err = dynamodbattribute.UnmarshalMap(item, &electronic)
		if err != nil {
			panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
		}
		printElectronic(electronic)
	}
}

func printCustomer(c Customer) {
	fmt.Println("Customer Info:")
	fmt.Println("\tId:        ", c.Id)
	fmt.Println("\tEmail:     ", c.Email)
	fmt.Println("\tFname:     ", c.Fname)
	fmt.Println("\tLname:     ", c.Lname)
	fmt.Println("\tAddresses: ", c.Addresses)
}

func printOrder(o Order) {
	fmt.Println("Order details:")
	fmt.Println("\tId:         ", o.Id)
	fmt.Println("\tStatus:     ", o.Status)
	fmt.Println("\tAmount:     ", o.Amount)
	fmt.Println("\tItem Count: ", o.NumberOfItems)
	fmt.Println("\tOrder Time: ", o.Ts)
}

func printLineItem(i LineItem) {
	fmt.Println("LineItem details:")
	fmt.Println("\tId:         ", i.Id)
	fmt.Println("\tProduct Id: ", i.ProductId)
	fmt.Println("\tPrice:      ", i.Price)
	fmt.Println("\tDiscount:   ", i.Discount)
	fmt.Println("\tStatus:     ", i.Status)
	fmt.Println("\tComment:    ", i.Comment)
}

func printElectronic(e Electronic) {
	fmt.Println("Electronic details:")
	fmt.Println("\tId:       ", e.Id)
	fmt.Println("\tName:     ", e.Name)
	fmt.Println("\tCategory: ", e.Category)
	fmt.Println("\tModel:    ", e.Model)
	fmt.Println("\tPrice:    ", e.Price)
	fmt.Println("\tShipping: ", e.ShippingAmount)
}

var region = "us-east-2"

func createSession(url string) *session.Session {
	if url == "" {
		url = "https://dynamodb." + region + ".amazonaws.com"
	}

	return session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Endpoint: aws.String(url),
		},
	}))
}

func createAdapterSession() *session.Session {
	return createSession("http://localhost:9050/v1/GetItem")
}
