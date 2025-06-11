// Copyright 2021
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

package storage

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	translator "github.com/cloudspannerecosystem/dynamodb-adapter/translator/utils"
)

func Test_parseRow(t *testing.T) {
	tests := []struct {
		name             string
		spannerTableName string
		row              *spanner.Row
		tableDDL         map[string]string
		tableSpannerDDL  map[string]string
		want             map[string]interface{}
		wantError        bool
	}{
		{
			name:             "ParseStringValue",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"strCol"}, []interface{}{
					spanner.NullString{StringVal: "my-text", Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"strCol": "S"},
			tableSpannerDDL: map[string]string{"strCol": "STRING(MAX)"},
			want:            map[string]interface{}{"strCol": "my-text"},
		},
		{
			name:             "ParseIntValue",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"intCol"}, []interface{}{
					spanner.NullInt64{Int64: 314, Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"intCol": "N"},
			tableSpannerDDL: map[string]string{"intCol": "INT64"},
			want:            map[string]interface{}{"intCol": int64(314)},
		},
		{
			name:             "ParseFloatValue",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"floatCol"}, []interface{}{
					spanner.NullFloat64{Float64: 3.14, Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"floatCol": "N"},
			tableSpannerDDL: map[string]string{"floatCol": "FLOAT64"},
			want:            map[string]interface{}{"floatCol": 3.14},
		},
		{
			name:             "ParseNumericValue",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"numericCol"}, []interface{}{
					spanner.NullNumeric{
						Numeric: func() big.Rat { r, _ := new(big.Rat).SetString("3.14"); return *r }(),
						Valid:   true,
					},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"numericCol": "N"},
			tableSpannerDDL: map[string]string{"numericCol": "NUMERIC"},
			want:            map[string]interface{}{"numericCol": func() big.Rat { r, _ := new(big.Rat).SetString("3.14"); return *r }()},
		},
		{
			name:             "ParseTimestampValue",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"timestampCol"}, []interface{}{
					spanner.NullTime{
						Time:  time.Date(2024, 6, 18, 12, 0, 0, 0, time.UTC),
						Valid: true,
					},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"timestampCol": "N"},
			tableSpannerDDL: map[string]string{"timestampCol": "TIMESTAMP"},
			want:            map[string]interface{}{"timestampCol": time.Date(2024, 6, 18, 12, 0, 0, 0, time.UTC)}},
		{
			name:             "ParseBoolValue",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"boolCol"}, []interface{}{
					spanner.NullBool{Bool: true, Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"boolCol": "BOOL"},
			tableSpannerDDL: map[string]string{"boolCol": "BOOL"},
			want:            map[string]interface{}{"boolCol": true},
		},
		{
			name:             "RemoveNulls",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"strCol"}, []interface{}{
					spanner.NullString{StringVal: "", Valid: false},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"strCol": "S"},
			tableSpannerDDL: map[string]string{"strCol": "STRING(MAX)"},
			want:            map[string]interface{}{"strCol": nil},
		},
		{
			name:             "SkipCommitTimestamp",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"commit_timestamp"}, []interface{}{
					nil,
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"commit_timestamp": "S"},
			tableSpannerDDL: map[string]string{"commit_timestamp": "STRING(MAX)"},
			want:            map[string]interface{}{},
		},
		{
			name:             "MultiValueRow",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"boolCol", "intCol", "strCol"}, []interface{}{
					spanner.NullBool{Bool: true, Valid: true},
					spanner.NullFloat64{Float64: 32, Valid: true},
					spanner.NullString{StringVal: "my-text", Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"boolCol": "BOOL", "intCol": "N", "strCol": "S"},
			tableSpannerDDL: map[string]string{"boolCol": "BOOL", "intCol": "FLOAT64", "strCol": "STRING(MAX)"},
			want:            map[string]interface{}{"boolCol": true, "intCol": 32.0, "strCol": "my-text"},
		},
		{
			name:             "ParseStringArray",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"arrayCol"}, []interface{}{
					[]spanner.NullString{
						{StringVal: "element1", Valid: true},
						{StringVal: "element2", Valid: true},
						{StringVal: "element3", Valid: true},
					},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"arrayCol": "SS"},
			tableSpannerDDL: map[string]string{"arrayCol": "ARRAY<STRING(MAX)>"},
			want:            map[string]interface{}{"arrayCol": []string{"element1", "element2", "element3"}},
			wantError:       false,
		},
		{
			name:             "MissingColumnTypeInDDL",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"strCol"}, []interface{}{
					spanner.NullString{StringVal: "test", Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"strCol": ""},
			tableSpannerDDL: map[string]string{"strCol": "STRING(MAX)"},
			want:            nil,
			wantError:       true,
		},
		{
			name:             "InvalidTypeConversion",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"strCol"}, []interface{}{
					spanner.NullFloat64{Float64: 123.45, Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"strCol": "S"},
			tableSpannerDDL: map[string]string{"strCol": "STRING(MAX)"},
			want:            nil,
			wantError:       true,
		},
		{
			name:             "ColumnNotInDDL",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"someOtherCol"}, []interface{}{
					spanner.NullString{StringVal: "missing-column", Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"strCol": "S"},
			tableSpannerDDL: map[string]string{"strCol": "STRING(MAX)"},
			want:            nil,
			wantError:       true,
		},
		{
			name:             "ParseNumberArray",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"numberArrayCol"}, []interface{}{
					[]spanner.NullFloat64{
						{Float64: 1.1, Valid: true},
						{Float64: 2.2, Valid: true},
						{Float64: 3.3, Valid: true},
					},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"numberArrayCol": "NS"},
			tableSpannerDDL: map[string]string{"numberArrayCol": "ARRAY<FLOAT64>"},
			want:            map[string]interface{}{"numberArrayCol": []float64{1.1, 2.2, 3.3}},
			wantError:       false,
		},
		{
			name:             "ParseBinaryArray",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"binaryArrayCol"}, []interface{}{
					[][]byte{
						[]byte("binaryData1"),
						[]byte("binaryData2"),
					},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"binaryArrayCol": "BS"},
			tableSpannerDDL: map[string]string{"binaryArrayCol": "ARRAY<BYTES(MAX)>"},
			want:            map[string]interface{}{"binaryArrayCol": [][]byte{[]byte("binaryData1"), []byte("binaryData2")}},
			wantError:       false,
		},
		{
			name:             "EmptyNumberArray",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"numberArrayCol"}, []interface{}{
					[]spanner.NullFloat64{},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"numberArrayCol": "NS"},
			tableSpannerDDL: map[string]string{"numberArrayCol": "ARRAY<FLOAT64>"},
			want:            map[string]interface{}{},
			wantError:       false,
		},
		{
			name:             "EmptyBinaryArray",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"binaryArrayCol"}, []interface{}{
					[][]byte{},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"binaryArrayCol": "BS"},
			tableSpannerDDL: map[string]string{"binaryArrayCol": "ARRAY<BYTES(MAX)>"},
			want:            map[string]interface{}{},
			wantError:       false,
		},
		{
			name:             "InvalidNumberArrayConversion",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"numberArrayCol"}, []interface{}{
					[]spanner.NullString{
						{StringVal: "not-a-number", Valid: true},
					},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"numberArrayCol": "NS"},
			tableSpannerDDL: map[string]string{"numberArrayCol": "ARRAY<FLOAT64>"},
			want:            nil,
			wantError:       true,
		},
		{
			name:             "InvalidBinaryArrayConversion",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"binaryArrayCol"}, []interface{}{
					[]spanner.NullString{
						{StringVal: "not-binary-data", Valid: true},
					},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"binaryArrayCol": "BS"},
			tableSpannerDDL: map[string]string{"binaryArrayCol": "ARRAY<BYTES(MAX)>"},
			want:            nil,
			wantError:       true,
		},
		{
			name:             "ParseNullValue",
			spannerTableName: "TestTable",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"nullCol"}, []interface{}{
					spanner.NullString{Valid: false},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			tableDDL:        map[string]string{"nullCol": "NULL"},
			tableSpannerDDL: map[string]string{"nullCol": "STRING(MAX)"},
			want:            map[string]interface{}{"nullCol": nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the DDLs for this test case
			models.TableDDL[tt.spannerTableName] = tt.tableDDL
			models.TableSpannerDDL[tt.spannerTableName] = tt.tableSpannerDDL

			got, _, err := parseRow(tt.row, tt.spannerTableName)
			if (err != nil) != tt.wantError {
				t.Errorf("parseRow() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildStmt(t *testing.T) {
	// Set up the test data
	query := &translator.DeleteUpdateQueryMap{
		SpannerQuery: "UPDATE Users SET age = @age WHERE name = @name",
		Params: map[string]interface{}{
			"age":  30,
			"name": "John Doe",
		},
	}

	// Expected result
	expectedStmt := &spanner.Statement{
		SQL:    query.SpannerQuery,
		Params: query.Params,
	}

	// Call the function
	result := buildStmt(query)

	// Assert that the result matches the expected statement
	if result.SQL != expectedStmt.SQL {
		t.Errorf("Expected SQL: %s, but got: %s", expectedStmt.SQL, result.SQL)
	}

	for key, expectedValue := range expectedStmt.Params {
		if result.Params[key] != expectedValue {
			t.Errorf("Expected param[%s]: %v, but got: %v", key, expectedValue, result.Params[key])
		}
	}
}

func TestBuildCommitOptions(t *testing.T) {
	storage := Storage{}

	// Call the function to test
	options := storage.BuildCommitOptions()

	// Verify that MaxCommitDelay is not nil
	if options.MaxCommitDelay == nil {
		t.Fatal("Expected MaxCommitDelay to be non-nil")
	}

	// Verify that it matches the expected default commit delay
	expectedDelay := defaultCommitDelay
	if *options.MaxCommitDelay != expectedDelay {
		t.Errorf("Expected MaxCommitDelay: %v, but got: %v", expectedDelay, *options.MaxCommitDelay)
	}
}
