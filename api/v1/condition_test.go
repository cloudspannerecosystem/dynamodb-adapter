package v1

import (
	"testing"
)

func TestUpdateExpression(t *testing.T) {
}

func Test_parseUpdateExpresstion(t *testing.T) {
	expr := parseUpdateExpresstion(" a = if_exists(value, :y),b = :t")
	t.Logf("%+v", expr)
}

func Test_extractOperations(t *testing.T) {
	rs := extractOperations("SET #col_lastUpdatedOn = :val_lastUpdatedOn")
	t.Logf("%+v", rs)
}
