package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func main() {
	// Set up the DynamoDB client to point to the proxy
	sess := session.Must(session.NewSession(&aws.Config{
		Region:   aws.String("us-west-2"),                // Replace with your region if needed
		Endpoint: aws.String("http://localhost:9050/v1"), // Proxy URL
	}))

	// Create DynamoDB service client
	svc := dynamodb.New(sess)
	fmt.Println("svc", svc)

	createItem(svc)     //PutItem
	readItems(svc)      //Scan
	updateItem(svc)     //UpdateItem
	deleteItem(svc)     //DeleteItem
	getItem(svc)        //GetItem
	queryItems(svc)     //QueryItem
	batchGetItem(svc)   //BatchGetItem
	batchWriteItem(svc) //BatchWriteItem
}

func createItem(svc *dynamodb.DynamoDB) {

	item := map[string]*dynamodb.AttributeValue{
		"emp_id": {
			N: aws.String("123"),
		},
		"emp_name": {
			S: aws.String("test"),
		},
		"isHired": {
			BOOL: aws.Bool(false),
		},
		"emp_image": {
			B: []byte("binary_data_here"),
		},
		"emp_status": {
			NULL: aws.Bool(true), // Explicitly setting NULL
		},
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String("employee_table"),
		Item:      item,
	}

	// Perform the PutItem operation
	_, err := svc.PutItem(input)
	if err != nil {
		fmt.Println("Error putting item:", err)
		return
	}

	fmt.Println("Successfully added item:", item)
}

func readItems(svc *dynamodb.DynamoDB) {
	// Define the input for the Scan operation
	input := &dynamodb.ScanInput{
		TableName: aws.String("employee_table"), // Table name
	}

	// Perform the Scan operation
	result, err := svc.Scan(input)
	if err != nil {
		fmt.Println("Error performing scan:", err)
		return
	}

	// Print the scan results
	fmt.Println("Scan result:", result)
}

func updateItem(svc *dynamodb.DynamoDB) {
	//currentTime := time.Now().UTC().Format(time.RFC3339)
	key := map[string]*dynamodb.AttributeValue{
		"emp_id": {
			N: aws.String("123"),
		},
	}

	// Use the correct timestamp format (RFC3339)
	update := &dynamodb.UpdateItemInput{
		TableName:        aws.String("employee_table"),
		Key:              key,
		UpdateExpression: aws.String("SET emp_name = :crl"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":crl": {
				S: aws.String("value"), // New value for contact_ranking_list
			},
		},
		ReturnValues: aws.String("UPDATED_NEW"),
	}

	// Perform the UpdateItem operation
	result, err := svc.UpdateItem(update)
	if err != nil {
		fmt.Println("Error updating item:", err)
		return
	}

	// Print the result of the update
	fmt.Println("Successfully updated item:", result)
}

func deleteItem(svc *dynamodb.DynamoDB) {
	// Define the input for the DeleteItem operation
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String("employee_table"),
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {
				N: aws.String("123"),
			},
		},
	}
	fmt.Println(input)
	// Perform the DeleteItem operation
	_, err := svc.DeleteItem(input)
	if err != nil {
		fmt.Println("Error deleting item:", err)
		return
	}

	// Print a success message
	fmt.Println("Successfully deleted item with guid:", "00XK5C0X6112TMBON4B2A5F88CYT548")
}

func getItem(svc *dynamodb.DynamoDB) {
	// Define the key for the GetItem operation
	input := &dynamodb.GetItemInput{
		TableName: aws.String("employee_table"), // Table name
		Key: map[string]*dynamodb.AttributeValue{
			"emp_id": {
				S: aws.String("123"), // Primary Key
			},
		},
	}

	// Perform the GetItem operation
	result, err := svc.GetItem(input)
	if err != nil {
		fmt.Println("Error getting item:", err)
		return
	}

	// Print the GetItem result
	fmt.Println("GetItem result:", result.Item)
}

func queryItems(svc *dynamodb.DynamoDB) {
	// Define the input for the Query operation
	input := &dynamodb.QueryInput{
		TableName:              aws.String("employee_table"),
		KeyConditionExpression: aws.String("emp_id = :emp_id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":emp_id": {
				N: aws.String("123"), // Primary Key
			},
		},
	}

	// Perform the Query operation
	result, err := svc.Query(input)
	if err != nil {
		fmt.Println("Error performing query:", err)
		return
	}

	// Print the Query results
	fmt.Println("Query result:", result.Items)
}

func batchGetItem(svc *dynamodb.DynamoDB) {
	// Define the input for the BatchGetItem operation
	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			"employee_table": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{
						"emp_id": {
							N: aws.String("124"),
						},
					},
					{
						"emp_id": {
							N: aws.String("156"),
						},
					},
				},
			},
		},
	}

	// Perform the BatchGetItem operation
	result, err := svc.BatchGetItem(input)
	if err != nil {
		fmt.Println("Error performing batch get:", err)
		return
	}

	// Print the BatchGetItem result
	for tableName, items := range result.Responses {
		fmt.Printf("Table: %s\n", tableName)
		for _, item := range items {
			fmt.Println(item)
		}
	}
}

func batchWriteItem(svc *dynamodb.DynamoDB) {
	// Define the input for the BatchWriteItem operation
	input := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			"employee_table": {
				{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id": {
								N: aws.String("124"),
							},
							"emp_name": {
								S: aws.String("jacob"),
							},
							"isHired": {
								BOOL: aws.Bool(true),
							},
						},
					},
				},
				{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id": {
								N: aws.String("156"),
							},
							"emp_name": {
								S: aws.String("sara"),
							},
							"emp_image": {
								B: []byte("binary_data_here"),
							},
							"isHired": {
								BOOL: aws.Bool(true),
							},
						},
					},
				},
				{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id": {
								N: aws.String("15666"),
							},
							"emp_name": {
								S: aws.String("jennifer"),
							},
							"emp_image": {
								B: []byte("binary_data_here"),
							},
							"isHired": {
								BOOL: aws.Bool(true),
							},
						},
					},
				},
				{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							"emp_id": {
								N: aws.String("1564"),
							},
							"emp_name": {
								S: aws.String("john"),
							},
							"emp_image": {
								B: []byte("binary_data_here"),
							},
							"isHired": {
								BOOL: aws.Bool(true),
							},
						},
					},
				},
			},
		},
	}

	// Perform the BatchWriteItem operation
	_, err := svc.BatchWriteItem(input)
	if err != nil {
		fmt.Println("Error performing batch write:", err)
		return
	}

	// Print success message
	fmt.Println("Successfully performed batch write operation")
}
