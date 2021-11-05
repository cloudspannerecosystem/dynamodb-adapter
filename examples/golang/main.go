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
	"strconv"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/aws/aws-sdk-go/aws/awserr"
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
	getCustomerContactDetails(svc,"CUST#0000000000","EMAIL#homer@email.com")

	fmt.Println("\nDynamo PK/SK Query:")
	getCustomerOrderDetails(svc)

	fmt.Println("\nDynamo Index Query:")
	getCustomerMostRecentOrder(svc)

	fmt.Println("\nDynamo PK Query:")
	getOrderLineItemDetails(svc)

	fmt.Println("\nDynamo Index Query")
	getProductsByCategory(svc)

 	fmt.Println("\nDynamo UpdateItem")
	updateCustomerDetails(svc)

 	fmt.Println("\nDynamo PutItem")
	addNewCustomer(svc)

	fmt.Println("\nDynamo DeleteItem")
	deleteCustomer(svc)

	fmt.Println("\nDynamo ScanItem")
	getCustomerWithSameId(svc)

	fmt.Println("\nDynamo BatchGetItem")
	getProductManufacturer(svc)

	fmt.Println("\nDynamo BatchWriteItem")
	addNewCustomerBatch(svc)
}

func getCustomerContactDetails(svc *dynamodb.DynamoDB,pk string,sk string) {
	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("Customer_Order"),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(pk),
			},
			"SK": {
				S: aws.String(sk),
			},
		},
	})

	if err != nil {
		fmt.Println(err.Error())
	}
	if len(result.Item) == 0 {
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
		TableName:              aws.String("Customer_Order"),
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
		TableName:              aws.String("Customer_Order"),
		IndexName:              aws.String("By_customer"),
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
		TableName:              aws.String("Customer_Order"),
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
		TableName:              aws.String("Product"),
		IndexName:              aws.String("By_Product_Category"),
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

func updateCustomerDetails(svc *dynamodb.DynamoDB) {
	_, err := svc.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: aws.String("Customer_Order"),
		Key: map[string]*dynamodb.AttributeValue{
    			"PK": {
    				S: aws.String("CUST#0000000000"),
    			},
    			"SK": {
    				S: aws.String("EMAIL#homer@email.com"),
    			},
    		},
    ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":number_of_items": {
                S: aws.String("100"),
            },
        },
    ReturnValues: aws.String("UPDATED_NEW"),
    UpdateExpression: aws.String("seT number_of_items = :number_of_items"),
	})

	if err != nil {
		fmt.Println(err.Error())
	}
	getCustomerContactDetails(svc,"CUST#0000000000","EMAIL#homer@email.com")


}


func addNewCustomer(svc *dynamodb.DynamoDB) {

  itemtoput := ItemToPut{
        PK:  "CUST#9922885566",
        SK:  "EMAIL#mosby@gmail.com",
        Addresses:  "{Shipping:  Maclarens pub, 240 W 55th St, New York, NY}",
        Email:  "mosby@gmail.com",
        Fname:  "Ted",
        Id:  "9922885566",
        Lname:  "Mosby",
    }
  av, err := dynamodbattribute.MarshalMap(itemtoput)
  if err != nil {
      fmt.Println(err.Error())
  }
  fmt.Println("Adding new customer with PK=CUST#9922885566 and SK=EMAIL#mosby@gmail.com")
	_, error := svc.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("Customer_Order"),
		Item: av,
	})

	if error != nil {
		fmt.Println(error.Error())
	}
  fmt.Println("Verifying the customer has been added")
	getCustomerContactDetails(svc,"CUST#9922885566","EMAIL#mosby@gmail.com")
}

func deleteCustomer(svc *dynamodb.DynamoDB) {

  fmt.Println("Deleting the customer with PK=CUST#9922885566 and SK=EMAIL#mosby@gmail.com")
	_, err := svc.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String("Customer_Order"),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String("CUST#9922885566"),
			},
			"SK": {
				S: aws.String("EMAIL#mosby@gmail.com"),
			},
		},
	})

	if err != nil {
      fmt.Println(err.Error())
  }  else {
      fmt.Println("Deleted the customer with PK=CUST#9922885566 and SK=EMAIL#mosby@gmail.com")
  }

  fmt.Println("Verifying the customer has been deleted")
  getCustomerContactDetails(svc,"CUST#9922885566","EMAIL#mosby@gmail.com")

}

func getCustomerWithSameId(svc *dynamodb.DynamoDB) {

  fmt.Println("Running a scan operation to find customers with Id 0000000000 who has number_of_items greater than 3")
  filt := expression.Name("customer_id").Equal(expression.Value("0000000000"))
  proj := expression.NamesList(expression.Name("customer_fname"), expression.Name("customer_lname"), expression.Name("number_of_items"))
  expr, err := expression.NewBuilder().WithFilter(filt).WithProjection(proj).Build()

  if err != nil {
      fmt.Println(err.Error())
  }
	result, err := svc.Scan(&dynamodb.ScanInput{
		  TableName: aws.String("Customer_Order"),
		  ExpressionAttributeNames:  expr.Names(),
      ExpressionAttributeValues: expr.Values(),
      FilterExpression:          expr.Filter(),
      ProjectionExpression:      expr.Projection(),
	})

	if err != nil {
		fmt.Println(err.Error())
	}

  for _, i := range result.Items {
      customer := Customer{}

      err = dynamodbattribute.UnmarshalMap(i, &customer)

      if err != nil {
          fmt.Println(err.Error())
      }
      intItem,err := strconv.Atoi(customer.Items)
      if err != nil {
                fmt.Println(err.Error())
      }
      if intItem > 3 && customer.Fname!= "" && customer.Lname!="" {
          fmt.Println("Customer Name: ", customer.Fname, customer.Lname)
      }
  }

}

func getProductManufacturer(svc *dynamodb.DynamoDB) {

  fmt.Println("Running a BatchGetItem operation to find manufacturer based on the model of the product and customer first name based on their PK and SK")
	result, err := svc.BatchGetItem(&dynamodb.BatchGetItemInput{
		  RequestItems: map[string]*dynamodb.KeysAndAttributes{
		    "Product": {
          Keys: []map[string]*dynamodb.AttributeValue{
             {
               "PK": &dynamodb.AttributeValue{
                   S: aws.String("1003003"),
                },
               "SK": &dynamodb.AttributeValue{
                   S: aws.String("Software#Musical Instruments#Recording Equipment#Hard Rock TrackPak - Mac"),
                },
             },
             {
               "PK": &dynamodb.AttributeValue{
                   S: aws.String("1003012"),
                },
               "SK": &dynamodb.AttributeValue{
                   S: aws.String("HardGood#Toys, Games & Drones#TV, Movie & Character Toys#Aquarius - Fender Playing Cards Gift Tin - Red/Black"),
                },
             },
             {
               "PK": &dynamodb.AttributeValue{
                   S: aws.String("1003021"),
                },
               "SK": &dynamodb.AttributeValue{
                   S: aws.String("HardGood#Musical Instruments#Musical Instrument Accessories#LoDuca Bros Inc - Deluxe Keyboard Bench - Black"),
                },
             },
             {
               "PK": &dynamodb.AttributeValue{
                   S: aws.String("1003049"),
                },
               "SK": &dynamodb.AttributeValue{
                   S: aws.String("HardGood#Toys, Games & Drones#TV, Movie & Character Toys#Trumpet Multimedia - Trumpets That Work 2015 Calendar - Black"),
                },
             },
            },
          ProjectionExpression: aws.String("manufacturer"),
          },
        "Customer_Order" : {
            Keys: []map[string]*dynamodb.AttributeValue{
              {
                "PK": &dynamodb.AttributeValue{
                    S: aws.String("CUST#6666666666"),
                },
                "SK": &dynamodb.AttributeValue{
                    S: aws.String("EMAIL#august@email.com"),
                },
              },
            },
            ProjectionExpression: aws.String("customer_fname"),
          },
        },
	})

  if err != nil {
      if aerr, ok := err.(awserr.Error); ok {
          switch aerr.Code() {
          case dynamodb.ErrCodeProvisionedThroughputExceededException:
              fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
          case dynamodb.ErrCodeResourceNotFoundException:
              fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
          case dynamodb.ErrCodeRequestLimitExceeded:
              fmt.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
          case dynamodb.ErrCodeInternalServerError:
              fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
          default:
              fmt.Println(aerr.Error())
          }
      } else {
          fmt.Println(err.Error())
      }
      return
  }

  for _, i := range result.Responses   {
           manufacturers := Manufacturers{}
           for _,j := range i {
              err = dynamodbattribute.UnmarshalMap(j, &manufacturers)
              if len(manufacturers.Manufacturer)>0 {
                fmt.Println("Product Manufacturers:", manufacturers.Manufacturer)
               }
           }
    }
  for _, i := range result.Responses   {
           firstnames := Firstnames{}
           for _,j := range i {
              err = dynamodbattribute.UnmarshalMap(j, &firstnames)
              if len(firstnames.Firstname) > 0 {
                fmt.Println("Customer First Names:", firstnames.Firstname)
               }
           }
    }
}

func addNewCustomerBatch(svc *dynamodb.DynamoDB) {

  fmt.Println("Running a BatchWriteItem operation to add customers")
	result, err := svc.BatchWriteItem(&dynamodb.BatchWriteItemInput{
		  RequestItems: map[string][]*dynamodb.WriteRequest{
              "Customer_Order": {
                  {
                      PutRequest: &dynamodb.PutRequest{
                          Item: map[string]*dynamodb.AttributeValue{
                              "PK": {
                                  S: aws.String("CUST#0070070070"),
                              },
                              "SK": {
                                  S: aws.String("EMAL#bond@gmail.com"),
                              },
                              "customer_fname": {
                                  S: aws.String("James"),
                              },
                              "customer_lname": {
                                  S: aws.String("Bond"),
                              },
                              "customer_email": {
                                  S: aws.String("bond@gmail.com"),
                              },
                              "customer_id": {
                                  S: aws.String("0070070070"),
                              },
                              "customer_addresses": {
                                  S: aws.String("{Shipping:  Casino Royal, Las Vegas, NY}"),
                              },
                              "item_quantity": {
                                  S: aws.String("100"),
                              },
                          },
                      },
                  },
                  {
                      PutRequest: &dynamodb.PutRequest{
                          Item: map[string]*dynamodb.AttributeValue{
                              "PK": {
                                  S: aws.String("CUST#1111111111"),
                              },
                              "SK": {
                                  S: aws.String("EMAIL#johns@email.com"),
                              },
                              "item_quantity": {
                                  S: aws.String("10"),
                              },
                          },
                      },
                  },
                  {
                      PutRequest: &dynamodb.PutRequest{
                          Item: map[string]*dynamodb.AttributeValue{
                              "PK": {
                                  S: aws.String("CUST#987654321"),
                              },
                              "SK": {
                                  S: aws.String("EMAIL#reallynoone@email.com"),
                              },
                          },
                      },
                  },
                  {
                      DeleteRequest: &dynamodb.DeleteRequest{
                          Key: map[string]*dynamodb.AttributeValue{
                              "PK": {
                                        S: aws.String("CUST#2222222222"),
                               },
                               "SK": {
                                  S: aws.String("EMAIL#alices@email.com"),
                               },
                          },
                      },
                  },

              },
              "Product": {
                  {
                      PutRequest: &dynamodb.PutRequest{
                          Item: map[string]*dynamodb.AttributeValue{
                              "PK": {
                                  S: aws.String("1002651"),
                              },
                              "SK": {
                                  S: aws.String("HardGood#Car Electronics & GPS#Car Audio#Polk Audio - 12\" Single-Voice-Coil 4-Ohm Subwoofer - Black"),
                              },
                              "manufacturer": {
                                  S: aws.String("Polk Audio"),
                              },
                              "price": {
                                  S: aws.String("89.99"),
                              },
                              "shipping_amount": {
                                  S: aws.String("10"),
                              },
                              "product_category": {
                                  S: aws.String("HardGood#Car Electronics & GPS#Car Audio"),
                              },
                               "product_id": {
                                   S: aws.String("1002651"),
                               },
                          },
                      },
                  },
                  {
                      PutRequest: &dynamodb.PutRequest{
                          Item: map[string]*dynamodb.AttributeValue{
                              "PK": {
                                  S: aws.String("1003003"),
                              },
                              "SK": {
                                  S: aws.String("Software#Musical Instruments#Recording Equipment#Hard Rock TrackPak - Mac"),
                              },
                              "manufacturer": {
                                  S: aws.String("Hal Leonard"),
                              },
                              "price": {
                                  S: aws.String("19.99"),
                              },
                              "shipping_amount": {
                                  S: aws.String("8.50"),
                              },
                              "product_category": {
                                  S: aws.String("HardGood#Car Electronics & GPS#Car Audio"),
                              },
                               "product_id": {
                                   S: aws.String("1003003"),
                               },
                          },
                      },
                  },
              },
      },
	})
  if err != nil {
    if aerr, ok := err.(awserr.Error); ok {
        switch aerr.Code() {
        case dynamodb.ErrCodeProvisionedThroughputExceededException:
            fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
        case dynamodb.ErrCodeResourceNotFoundException:
            fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
        case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
            fmt.Println(dynamodb.ErrCodeItemCollectionSizeLimitExceededException, aerr.Error())
        case dynamodb.ErrCodeRequestLimitExceeded:
            fmt.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
        case dynamodb.ErrCodeInternalServerError:
            fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
        default:
            fmt.Println(aerr.Error())
        }
    } else {
        fmt.Println(err.Error())
    }
    return
  } else {
    fmt.Println("An empty map or an empty unprocessed key in a map indicates a successful operation below")
    fmt.Println(result)
  }


}


func printCustomer(c Customer) {
	fmt.Println("Customer Info:")
	fmt.Println("\tId:        ", c.Id)
	fmt.Println("\tEmail:     ", c.Email)
	fmt.Println("\tFname:     ", c.Fname)
	fmt.Println("\tLname:     ", c.Lname)
	fmt.Println("\tAddresses: ", c.Addresses)
	fmt.Println("\tItems: ",c.Items)
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

var region = os.Getenv("AWS_REGION")

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
	return createSession("http://localhost:9050/v1")
}