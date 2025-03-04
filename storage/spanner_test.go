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
	"reflect"
	"testing"

	"cloud.google.com/go/spanner"
)

func Test_parseRow(t *testing.T) {
	tests := []struct {
		name      string
		row       *spanner.Row
		colDDL    map[string]string
		want      map[string]interface{}
		wantError bool
	}{
		{
			name: "ParseStringValue",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"strCol"}, []interface{}{
					spanner.NullString{StringVal: "my-text", Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL: map[string]string{"strCol": "S"},
			want:   map[string]interface{}{"strCol": "my-text"},
		},
		{
			name: "ParseIntValue",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"intCol"}, []interface{}{
					spanner.NullFloat64{Float64: 314, Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL: map[string]string{"intCol": "N"},
			want:   map[string]interface{}{"intCol": 314.0},
		},
		{
			name: "ParseFloatValue",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"floatCol"}, []interface{}{
					spanner.NullFloat64{Float64: 3.14, Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL: map[string]string{"floatCol": "N"},
			want:   map[string]interface{}{"floatCol": 3.14},
		},
		{
			name: "ParseBoolValue",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"boolCol"}, []interface{}{
					spanner.NullBool{Bool: true, Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL: map[string]string{"boolCol": "BOOL"},
			want:   map[string]interface{}{"boolCol": true},
		},
		{
			name: "RemoveNulls",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"strCol"}, []interface{}{
					spanner.NullString{StringVal: "", Valid: false},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL: map[string]string{"strCol": "S"},
			want:   map[string]interface{}{"strCol": nil}, // Null value should be removed
		},
		{
			name: "SkipCommitTimestamp",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"commit_timestamp"}, []interface{}{
					nil, // Commit timestamp should be skipped
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL: map[string]string{"commit_timestamp": "S"},
			want:   map[string]interface{}{}, // Commit timestamp should not appear in the result
		},
		{
			name: "MultiValueRow",
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
			colDDL: map[string]string{"boolCol": "BOOL", "intCol": "N", "strCol": "S"},
			want:   map[string]interface{}{"boolCol": true, "intCol": 32.0, "strCol": "my-text"},
		},
		{
			name: "ParseStringArray",
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
			colDDL:    map[string]string{"arrayCol": "SS"},
			want:      map[string]interface{}{"arrayCol": []string{"element1", "element2", "element3"}},
			wantError: false,
		},
		{
			name: "MissingColumnTypeInDDL",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"strCol"}, []interface{}{
					spanner.NullString{StringVal: "test", Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL:    map[string]string{"strCol": ""}, // Missing type in DDL
			want:      nil,
			wantError: true,
		},
		{
			name: "InvalidTypeConversion",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"strCol"}, []interface{}{
					spanner.NullFloat64{Float64: 123.45, Valid: true}, // Trying to parse float as a string
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL:    map[string]string{"strCol": "S"},
			want:      nil,
			wantError: true,
		},
		{
			name: "ColumnNotInDDL",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"someOtherCol"}, []interface{}{
					spanner.NullString{StringVal: "missing-column", Valid: true},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL:    map[string]string{"strCol": "S"}, // Column "someOtherCol" not in DDL
			want:      nil,
			wantError: true,
		},
		{
			name: "ParseNumberArray",
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
			colDDL:    map[string]string{"numberArrayCol": "NS"},
			want:      map[string]interface{}{"numberArrayCol": []float64{1.1, 2.2, 3.3}},
			wantError: false,
		},
		{
			name: "ParseBinaryArray",
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
			colDDL:    map[string]string{"binaryArrayCol": "BS"},
			want:      map[string]interface{}{"binaryArrayCol": [][]byte{[]byte("binaryData1"), []byte("binaryData2")}},
			wantError: false,
		},
		{
			name: "EmptyNumberArray",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"numberArrayCol"}, []interface{}{
					[]spanner.NullFloat64{},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL:    map[string]string{"numberArrayCol": "NS"},
			want:      map[string]interface{}{},
			wantError: false,
		},
		{
			name: "EmptyBinaryArray",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"binaryArrayCol"}, []interface{}{
					[][]byte{},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL:    map[string]string{"binaryArrayCol": "BS"},
			want:      map[string]interface{}{},
			wantError: false,
		},
		{
			name: "InvalidNumberArrayConversion",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"numberArrayCol"}, []interface{}{
					[]spanner.NullString{
						{StringVal: "not-a-number", Valid: true}, // Invalid conversion
					},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL:    map[string]string{"numberArrayCol": "NS"},
			want:      nil,
			wantError: true,
		},
		{
			name: "InvalidBinaryArrayConversion",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"binaryArrayCol"}, []interface{}{
					[]spanner.NullString{
						{StringVal: "not-binary-data", Valid: true}, // Invalid conversion
					},
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL:    map[string]string{"binaryArrayCol": "BS"},
			want:      nil,
			wantError: true,
		},
		{
			name: "ParseNullValue",
			row: func() *spanner.Row {
				row, err := spanner.NewRow([]string{"nullCol"}, []interface{}{
					spanner.NullString{Valid: false}, // Represents a NULL value
				})
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
				return row
			}(),
			colDDL: map[string]string{"nullCol": "NULL"},   // Define the column type as NULL
			want:   map[string]interface{}{"nullCol": nil}, // Expect a nil value in the output
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := parseRow(tt.row, tt.colDDL)
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
