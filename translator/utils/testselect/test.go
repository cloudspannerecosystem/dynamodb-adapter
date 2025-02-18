package main

import (
	"encoding/json"
	"fmt"

	translator "github.com/cloudspannerecosystem/dynamodb-adapter/translator/utils"
)

func main() {
	transaltorObj := translator.Translator{}
	query := "SELECT age, address FROM employee WHERE age > 30 AND address = 'abc' ORDER BY age LIMIT 10 OFFSET 5;"
	// query := "SELECT * FROM users WHERE name = 'Alice' OR city = 'New York';"
	// query := "SELECT employee_id, employee_name FROM employees WHERE data -> 'address' -> 'city' = 'New York' AND data -> 'age' > 30;"
	res, err := transaltorObj.ToSpannerSelect(query)
	if err != nil {
		fmt.Println(err)
	}
	a, _ := json.Marshal(res)
	fmt.Println("response-> ", string(a))
}
