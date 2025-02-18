package main

import (
	"encoding/json"
	"fmt"

	translator "github.com/cloudspannerecosystem/dynamodb-adapter/translator/utils"
)

func main() {
	transaltorObj := translator.Translator{}
	query := "INSERT INTO employee VALUE {'emp_id': 10, 'first_name': 'Marc', 'last_name': 'Richards1', 'age': 10, 'address': 'Shamli'};"
	// query := "SELECT * FROM users WHERE name = 'Alice' OR city = 'New York';"
	// query := "SELECT employee_id, employee_name FROM employees WHERE data -> 'address' -> 'city' = 'New York' AND data -> 'age' > 30;"
	res, err := transaltorObj.ToSpannerInsert(query)
	if err != nil {
		fmt.Println(err)
	}
	if err != nil {
		fmt.Println(err)
	}
	a, _ := json.Marshal(res)
	fmt.Println("response-> ", string(a))
}
