package config

import (
	"testing"

	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"gopkg.in/go-playground/assert.v1"
)

func TestGetTableConf(t *testing.T) {
	DbConfigMap = map[string]models.TableConfig{
		"employee_data": {
			PartitionKey:     "emp_id",
			SortKey:          "emp_name",
			Indices:          nil,
			SpannerIndexName: "emp_id",
			ActualTable:      "employee%data",
		},
		"employee%data": {
			PartitionKey:     "e_id",
			SortKey:          "e_name",
			Indices:          nil,
			SpannerIndexName: "emp_id",
			ActualTable:      "employee%data",
		},
		"department": {
			PartitionKey:     "d_id",
			SortKey:          "d_name",
			Indices:          nil,
			SpannerIndexName: "d_id",
			ActualTable:      "",
		},
	}

	tests := []struct {
		testName  string
		tableName string
		want      models.TableConfig
	}{
		{
			"empty table Name",
			"",
			models.TableConfig{},
		},
		{
			"table which is not present",
			"xyz",
			models.TableConfig{},
		},
		{
			"table which does not have actual table name",
			"department",
			models.TableConfig{
				PartitionKey:     "d_id",
				SortKey:          "d_name",
				Indices:          nil,
				SpannerIndexName: "d_id",
				ActualTable:      "department",
			},
		},
		{
			"table which is present",
			"employee_data",
			models.TableConfig{
				PartitionKey:     "e_id",
				SortKey:          "e_name",
				Indices:          nil,
				SpannerIndexName: "emp_id",
				ActualTable:      "employee%data",
			},
		},
	}

	for _, tc := range tests {
		got, _ := GetTableConf(tc.tableName)
		assert.Equal(t, got, tc.want)
	}
}

func TestChangeTableNameForSP(t *testing.T) {
	tests := []struct {
		testName  string
		tableName string
		want      string
	}{
		{
			"empty table Name",
			"",
			"",
		},
		{
			"table name without underscore",
			"department",
			"department",
		},
		{
			"table name with one underscore",
			"department-data",
			"department_data",
		},
		{
			"table name with more than one underscore",
			"department-data-1-7",
			"department_data_1_7",
		},
	}

	for _, tc := range tests {
		got := changeTableNameForSP(tc.tableName)
		assert.Equal(t, got, tc.want)
	}
}
