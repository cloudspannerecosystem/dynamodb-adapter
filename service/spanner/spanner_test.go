package spanner

import (
	"testing"

	"gopkg.in/go-playground/assert.v1"
)

func Test_getColNameAndType(t *testing.T) {
	tests := []struct {
		inputStr string
		want1    string
		want2    string
	}{
		{
			"",
			"",
			"",
		},
		{
			"oneValue",
			"",
			"",
		},
		{
			"name STRING",
			"name",
			"STRING",
		},
		{
			"subjects ARRAY",
			"subjects",
			"ARRAY",
		},
		{
			"date_of_birth DATE",
			"date_of_birth",
			"DATE",
		},
		{
			"Available BOOL",
			"Available",
			"BOOL",
		},
	}

	for _, tc := range tests {
		got1, got2 := getColNameAndType(tc.inputStr)
		assert.Equal(t, got1, tc.want1)
		assert.Equal(t, got2, tc.want2)
	}
}

func Test_getTableName(t *testing.T) {
	tests := []struct {
		inputStr string
		want     string
	}{
		{
			"",
			"",
		},
		{
			"oneValue",
			"",
		},
		{
			"two value",
			"",
		},
		{
			"three value here",
			"here",
		},
		{
			"create table Employee",
			"Employee",
		},
	}

	for _, tc := range tests {
		got := getTableName(tc.inputStr)
		assert.Equal(t, got, tc.want)
	}
}
