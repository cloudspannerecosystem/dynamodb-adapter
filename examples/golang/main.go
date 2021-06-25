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
		return
	}
	if result.Item == nil {
		fmt.Printf("No record found")
	}
	fmt.Printf(result.String())

	customer := Customer{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &customer)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
	}

	fmt.Println("Found item:")
	fmt.Println("Id:        ", customer.Id)
	fmt.Println("Email:     ", customer.Email)
	fmt.Println("Fname:     ", customer.Fname)
	fmt.Println("Lname:     ", customer.Lname)
	fmt.Println("Addresses: ", customer.Addresses)
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
