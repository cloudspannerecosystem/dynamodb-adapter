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

	"cloud.google.com/go/spanner"
)

func Test_parseRow(t *testing.T) {
	simpleStringRow, _ := spanner.NewRow([]string{"strCol"}, []interface{}{"my-text"})
	simpleIntRow, _ := spanner.NewRow([]string{"intCol"}, []interface{}{int64(314)})
	simpleFloatRow, _ := spanner.NewRow([]string{"floatCol"}, []interface{}{3.14})
	simpleNumericIntRow, _ := spanner.NewRow([]string{"numericCol"}, []interface{}{big.NewRat(314, 1)})
	simpleNumericFloatRow, _ := spanner.NewRow([]string{"numericCol"}, []interface{}{big.NewRat(13, 4)})
	simpleBoolRow, _ := spanner.NewRow([]string{"boolCol"}, []interface{}{true})
	removeNullRow, _ := spanner.NewRow([]string{"strCol", "nullCol"}, []interface{}{"my-text", spanner.NullString{}})
	skipCommitTimestampRow, _ := spanner.NewRow([]string{"strCol", "commit_timestamp"}, []interface{}{"my-text", "2021-01-01"})
	multipleValuesRow, _ := spanner.NewRow([]string{"strCol", "intCol", "nullCol", "boolCol"}, []interface{}{"my-text", int64(32), spanner.NullString{}, true})


	type args struct {
		r      *spanner.Row
		colDDL map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{
			"ParseStringValue",
			args{simpleStringRow, map[string]string{"strCol": "STRING(MAX)"}}, 
			map[string]interface{}{"strCol": "my-text"},
			false,
		},
		{
			"ParseIntValue",
			args{simpleIntRow, map[string]string{"intCol": "INT64"}}, 
			map[string]interface{}{"intCol": int64(314)},
			false,
		},
		{
			"ParseFloatValue",
			args{simpleFloatRow, map[string]string{"floatCol": "FLOAT64"}}, 
			map[string]interface{}{"floatCol": 3.14},
			false,
		},
		{
			"ParseNumericIntValue",
			args{simpleNumericIntRow, map[string]string{"numericCol": "NUMERIC"}}, 
			map[string]interface{}{"numericCol": int64(314)},
			false,
		},
		{
			"ParseNumericFloatValue",
			args{simpleNumericFloatRow, map[string]string{"numericCol": "NUMERIC"}}, 
			map[string]interface{}{"numericCol": 3.25},
			false,
		},
		{
			"ParseBoolValue",
			args{simpleBoolRow, map[string]string{"boolCol": "BOOL"}}, 
			map[string]interface{}{"boolCol": true},
			false,
		},
		{
			"RemoveNulls",
			args{removeNullRow, map[string]string{"strCol": "STRING(MAX)", "nullCol": "STRING(MAX)"}}, 
			map[string]interface{}{"strCol": "my-text"},
			false,
		},
		{
			"SkipCommitTimestamp",
			args{skipCommitTimestampRow, map[string]string{"strCol": "STRING(MAX)", "commit_timestamp": "TIMESTAMP"}}, 
			map[string]interface{}{"strCol": "my-text"},
			false,
		},
		{
			"MultiValueRow",
			args{multipleValuesRow, map[string]string{"strCol": "STRING(MAX)", "intCol": "INT64", "nullCol": "STRING(MAX)", "boolCol": "BOOL"}}, 
			map[string]interface{}{"strCol": "my-text", "intCol": int64(32), "boolCol": true},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRow(tt.args.r, tt.args.colDDL)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRowForNull() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRowForNull() = %v, want %v", got, tt.want)
			}
		})
	}
}
