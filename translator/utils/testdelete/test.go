package main

import (
	"encoding/json"
	"fmt"

	translator "github.com/cloudspannerecosystem/dynamodb-adapter/translator/utils"
)

func main() {
	transaltorObj := translator.Translator{}
	query := "DELETE FROM employee WHERE emp_id = 'hello' AND age = 20;"
	res, err := transaltorObj.ToSpannerDelete(query)
	if err != nil {
		fmt.Println(err)
	}
	a, _ := json.Marshal(res)
	fmt.Println("response-> ", string(a))
}
