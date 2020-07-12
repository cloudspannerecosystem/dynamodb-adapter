package utils

import (
	"testing"
)

func TestCreateConditionExpression(t *testing.T) {
	r, err := CreateConditionExpression("b < :c AND attribute_exists(c)", map[string]interface{}{":c": 12345})
	t.Log(r)
	t.Log(err)
}

func Test_writeIncrement(t *testing.T) {

}
