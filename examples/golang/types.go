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

type Customer struct {
	Id        string      `dynamodbav:"customer_id"`
	Fname     string      `dynamodbav:"customer_fname"`
	Lname     string      `dynamodbav:"customer_lname"`
	Email     string      `dynamodbav:"customer_email"`
	Addresses interface{} `dynamodbav:"customer_addresses"`
}

type Order struct {
	Id            string  `dynamodbav:"order_id"`
	Status        string  `dynamodbav:"order_status"`
	Ts            string  `dynamodbav:"order_ts"`
	Amount        float32 `dynamodbav:"order_amount"`
	NumberOfItems int     `dynamodbav:"number_of_items"`
}

type LineItem struct {
	Id        string  `dynamodbav:"line_item_id"`
	ProductId string  `dynamodbav:"product_id"`
	Price     float32 `dynamodbav:"item_price"`
	Discount  float32 `dynamodbav:"item_discount"`
	Status    string  `dynamodbav:"item_status"`
	Comment   string  `dynamodbav:"comment"`
}

type Book struct {
	Id              string   `dynamodbav:"product_id"`
	Category        string   `dynamodbav:"product_category"`
	ISBN            string   `dynamodbav:"isbn"`
	Authors         []string `dynamodbav:"book_authors"`
	Title           string   `dynamodbav:"book_title"`
	Publisher       string   `dynamodbav:"book_publisher"`
	PublicationDate string   `dynamodbav:"publication_date"`
	Format          string   `dynamodbav:"book_format"`
	Price           float32  `dynamodbav:"price"`
	ShippingAmount  float32  `dynamodbav:"shipping_amount"`
}

type Electronic struct {
	Id             string  `dynamodbav:"product_id"`
	Name           string  `dynamodbav:"product_name"`
	Description    string  `dynamodbav:"product_description"`
	Category       string  `dynamodbav:"product_category"`
	UPC            string  `dynamodbav:"upc"`
	Manufacturer   string  `dynamodbav:"manufacturer"`
	Model          string  `dynamodbav:"model"`
	Price          float32 `dynamodbav:"price"`
	ShippingAmount float32 `dynamodbav:"shipping_amount"`
}
