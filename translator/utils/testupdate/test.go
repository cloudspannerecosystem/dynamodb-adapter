package main

import (
	"encoding/json"
	"fmt"

	translator "github.com/cloudspannerecosystem/dynamodb-adapter/translator/utils"
)

func main() {
	transaltorObj := translator.Translator{}
	query := "UPDATE employee SET status = 'active', address = 'new address', age = 31 WHERE emp_id = 'eqi' OR age > 30;"
	res, err := transaltorObj.ToSpannerUpdate(query)
	if err != nil {
		fmt.Println(err)
	}
	a, _ := json.Marshal(res)
	fmt.Println("response-> ", string(a))
}
