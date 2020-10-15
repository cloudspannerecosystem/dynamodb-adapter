// Copyright 2020 Google LLC
//
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

package services

import (
	"testing"

	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	"gopkg.in/go-playground/assert.v1"
)

func init() {
	models.ConfigController.StreamEnable = map[string]struct{}{
		"TestTable": {},
		"Sample":    {},
	}

	models.ConfigController.PubSubTopic = map[string]string{
		"TestTable": "topic1",
		"Sample":    "topic2",
	}
}

func TestIsMyStreamEnabled(t *testing.T) {
	tests := []struct {
		testName  string
		tableName string
		want      bool
	}{
		{
			"empty TableName",
			"",
			false,
		},
		{
			"wrong TableName",
			"sometable",
			false,
		},
		{
			"correct Table Name",
			"TestTable",
			true,
		},
		{
			"another Table Name",
			"Sample",
			true,
		},
	}

	for _, tc := range tests {
		got := IsMyStreamEnabled(tc.tableName)
		assert.Equal(t, got, tc.want)
	}
}

func TestIsPubSubAllowed(t *testing.T) {
	tests := []struct {
		testName  string
		tableName string
		want1     string
		want2     bool
	}{
		{
			"empty TableName",
			"",
			"",
			false,
		},
		{
			"wrong TableName",
			"sometable",
			"",
			false,
		},
		{
			"correct Table Name",
			"TestTable",
			"topic1",
			true,
		},
		{
			"another Table Name",
			"Sample",
			"topic2",
			true,
		},
	}

	for _, tc := range tests {
		got1, got2 := IsPubSubAllowed(tc.tableName)
		assert.Equal(t, got1, tc.want1)
		assert.Equal(t, got2, tc.want2)
	}
}
