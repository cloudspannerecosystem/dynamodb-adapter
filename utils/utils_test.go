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

package utils

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/tj/assert"

	"github.com/antonmedv/expr"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
)

func TestGetStringInBetween(t *testing.T) {
	tests := []struct {
		testName, strValue, firstStr, secondStr, want string
	}{
		{"Empty Value for String", "", "s", "l", ""},
		{"Correct Values here", "school", "sc", "ol", "ho"},
		{"Correct Values with 2 similar letters", "stool", "o", "l", "o"},
		{"Empty 1st string", "school", "", "l", "schoo"},
		{"Empty 2nd string", "school", "o", "", ""},
	}

	for _, tc := range tests {
		got := GetStringInBetween(tc.strValue, tc.firstStr, tc.secondStr)
		assert.Equal(t, got, tc.want)
	}
}

func TestGetFieldNameFromConditionalExpression(t *testing.T) {
	tests := []struct {
		testName, condExpr, want string
	}{
		{"Empty Value", "", ""},
		{"Any String passed", "Any stirng", "Any stirng"},
		{"String with attribute_exists ", "attribute_exists(name)", "name"},
		{"String with attribute_not_exists", "attribute_not_exists(some_field)", "some_field"},
	}

	for _, tc := range tests {
		got := GetFieldNameFromConditionalExpression(tc.condExpr)
		assert.Equal(t, got, tc.want)
	}
}

func TestCreateConditionExpression(t *testing.T) {
	cond1, _ := expr.Compile(`TOKEN0 > "20" && TOKEN4 `)

	tests := []struct {
		testName            string
		conditionExpression string
		attributeMap        map[string]interface{}
		want                *models.Eval
	}{
		{
			"empty Conditonal Expression",
			"",
			nil,
			new(models.Eval),
		},
		{
			"Attribute map not present",
			"age > :val AND attribute_exists(c)",
			nil,
			nil,
		},
		{
			"Conditonal Expression with attributeMap",
			"age > :val AND attribute_exists(c)",
			map[string]interface{}{":val": "20"},
			&models.Eval{
				Cond:       cond1,
				Attributes: []string{"age", "attribute_exists(c)"},
				Cols:       []string{"age", "c"},
				Tokens:     []string{"TOKEN0", "TOKEN4"},
				ValueMap:   make(map[string]interface{}),
			},
		},
	}

	for _, tc := range tests {
		got, _ := CreateConditionExpression(tc.conditionExpression, tc.attributeMap)
		assert.Equal(t, got, tc.want)
	}
}

func TestEvaluateExpression(t *testing.T) {
	cond1, _ := expr.Compile(`TOKEN0 > "20" && TOKEN4 `)
	tests := []struct {
		testName string
		input    *models.Eval
		want     bool
	}{
		{
			"No Input",
			nil,
			true,
		},
		{
			"Cond is nil in input",
			&models.Eval{
				Cond:       nil,
				Attributes: []string{"age", "attribute_exists(c)"},
				Cols:       []string{"age", "c"},
				Tokens:     []string{"TOKEN0", "TOKEN4"},
				ValueMap:   make(map[string]interface{}),
			},
			true,
		},
		{
			"ValueMap is nil",
			&models.Eval{
				Cond:       cond1,
				Attributes: []string{"age", "attribute_exists(c)"},
				Cols:       []string{"age", "c"},
				Tokens:     []string{"TOKEN0", "TOKEN4"},
				ValueMap:   nil,
			},
			false,
		},
		{
			"Correct Params",
			&models.Eval{
				Cond:       cond1,
				Attributes: []string{"age", "attribute_exists(c)"},
				Cols:       []string{"age", "c"},
				Tokens:     []string{"TOKEN0", "TOKEN4"},
				ValueMap: map[string]interface{}{
					"TOKEN0": "age",
					"TOKEN4": true,
				},
			},
			true,
		},
	}

	// EvaluateExpression()
	for _, tc := range tests {
		got, _ := EvaluateExpression(tc.input)
		assert.Equal(t, got, tc.want)
	}
}

func TestParseBeginsWith(t *testing.T) {
	tests := []struct {
		testName, rangeExpression string
		want                      map[string]string
	}{
		{
			"Empty rangeExpression",
			"",
			map[string]string{
				"first":  "",
				"second": "",
				"third":  "",
			},
		},
		{
			"rangeExpression with begins_with()",
			"begins_with(name, :val)",
			map[string]string{
				"first":  "name",
				"second": ":val",
				"third":  "begins_with(name, :val)",
			},
		},
		{
			"ragneEpression without begins_with()",
			"age > 20",
			map[string]string{
				"first":  "",
				"second": "",
				"third":  "age > 20",
			},
		},
		{
			"ragneEpression with special symbols GT",
			"age GT 20",
			map[string]string{
				"first":  "",
				"second": "",
				"third":  "age > 20",
			},
		},
		{
			"ragneEpression with special symbols LT",
			"age GT 20",
			map[string]string{
				"first":  "",
				"second": "",
				"third":  "age > 20",
			},
		},
	}

	for _, tc := range tests {
		first, second, third := ParseBeginsWith(tc.rangeExpression)
		assert.Equal(t, first, tc.want["first"])
		assert.Equal(t, second, tc.want["second"])
		assert.Equal(t, third, tc.want["third"])
	}
}

func TestChangeTableNameForSpanner(t *testing.T) {
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
		got := ChangeTableNameForSpanner(tc.tableName)
		assert.Equal(t, got, tc.want)
	}
}

func TestRemoveDuplicatesString(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{[]string{"apple", "banana", "apple", "orange"}, []string{"apple", "banana", "orange"}},
		{[]string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{[]string{"one", "two", "three"}, []string{"one", "two", "three"}}, // No duplicates
		{[]string{}, []string{}}, // Empty slice
	}

	for _, test := range tests {
		result := RemoveDuplicatesString(test.input)
		if len(result) == 0 && len(test.expected) == 0 {
			continue
		}
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("RemoveDuplicatesString(%v) = %v; want %v", test.input, result, test.expected)
		}

	}
}

func TestRemoveDuplicatesFloat(t *testing.T) {
	tests := []struct {
		input    []float64
		expected []float64
	}{
		{[]float64{1.1, 2.2, 3.3, 1.1, 2.2}, []float64{1.1, 2.2, 3.3}},
		{[]float64{0.5, 0.5, 0.5}, []float64{0.5}},
		{[]float64{10.0, 20.0, 30.0}, []float64{10.0, 20.0, 30.0}}, // No duplicates
		{[]float64{}, []float64{}},                                 // Empty slice
	}

	for _, test := range tests {
		result := RemoveDuplicatesFloat(test.input)
		if len(result) == 0 && len(test.expected) == 0 {
			continue
		}
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("RemoveDuplicatesString(%v) = %v; want %v", test.input, result, test.expected)
		}

	}
}

func TestRemoveDuplicatesByteSlice(t *testing.T) {
	tests := []struct {
		input    [][]byte
		expected [][]byte
	}{
		{
			[][]byte{[]byte("foo"), []byte("bar"), []byte("foo"), []byte("baz")},
			[][]byte{[]byte("foo"), []byte("bar"), []byte("baz")},
		},
		{
			[][]byte{[]byte("apple"), []byte("banana"), []byte("apple")},
			[][]byte{[]byte("apple"), []byte("banana")},
		},
		{
			[][]byte{[]byte("one"), []byte("two"), []byte("three")},
			[][]byte{[]byte("one"), []byte("two"), []byte("three")},
		},
		{
			[][]byte{},
			[][]byte{},
		},
	}

	for _, test := range tests {
		result := RemoveDuplicatesByteSlice(test.input)
		if !equalByteSlices(result, test.expected) {
			t.Errorf("RemoveDuplicatesByteSlice(%v) = %v; want %v", test.input, result, test.expected)
		}
	}
}

// Helper function to compare [][]byte slices
func equalByteSlices(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestParseListRemoveTarget(t *testing.T) {
	tests := []struct {
		name          string
		target        string
		expected      string
		expectedIndex int
	}{
		{"Valid target", "listAttr[2]", "listAttr", 2},
		{"Invalid target", "listAttr", "listAttr", -1},
		{"Invalid target with no brackets", "listAttr2", "listAttr2", -1},
		{"Invalid target with no index", "listAttr[]", "listAttr[]", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualName, actualIndex := ParseListRemoveTarget(tt.target)
			if actualName != tt.expected {
				t.Errorf("expected name %q, got %q", tt.expected, actualName)
			}
			if actualIndex != tt.expectedIndex {
				t.Errorf("expected index %d, got %d", tt.expectedIndex, actualIndex)
			}
		})
	}
}

func TestRemoveListElement(t *testing.T) {
	tests := []struct {
		name     string
		list     []interface{}
		idx      int
		expected []interface{}
	}{
		{"Remove from middle", []interface{}{1, 2, 3, 4, 5}, 2, []interface{}{1, 2, 4, 5}},
		{"Remove from start", []interface{}{1, 2, 3, 4, 5}, 0, []interface{}{2, 3, 4, 5}},
		{"Remove from end", []interface{}{1, 2, 3, 4, 5}, 4, []interface{}{1, 2, 3, 4}},
		{"Invalid index", []interface{}{1, 2, 3, 4, 5}, 5, []interface{}{1, 2, 3, 4, 5}},
		{"Invalid index negative", []interface{}{1, 2, 3, 4, 5}, -1, []interface{}{1, 2, 3, 4, 5}},
		{"Empty list", []interface{}{}, 0, []interface{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := RemoveListElement(tt.list, tt.idx)
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, actual)
			}
		})
	}
}

func TestUpdateFieldByPath(t *testing.T) {
	data := map[string]interface{}{
		"first": map[string]interface{}{
			"second": map[string]interface{}{
				"third": "value",
			},
		},
	}

	// Successful update
	success := UpdateFieldByPath(data, ".first.second.third", "newValue")
	assert.True(t, success)
	assert.Equal(t, "newValue", data["first"].(map[string]interface{})["second"].(map[string]interface{})["third"])

	// Invalid path
	success = UpdateFieldByPath(data, ".first.invalid_key.third", "newValue")
	assert.False(t, success)
}
func TestTrimSingleQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "With single quotes",
			input:    "'Hello, World!'",
			expected: "Hello, World!",
		},
		{
			name:     "Without single quotes",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "Only quotes",
			input:    "''",
			expected: "",
		},
		{
			name:     "Spaces with quotes",
			input:    "'   '",
			expected: "   ", // maintaining spaces
		},
		{
			name:     "Single quote at start only",
			input:    "'Hello, World!",
			expected: "'Hello, World!", // single quote not at the end should remain
		},
		{
			name:     "Single quote at end only",
			input:    "Hello, World!'",
			expected: "Hello, World!'", // single quote not at the start should remain
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TrimSingleQuotes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
func TestCleanExpressionSpacing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Removes space after function name",
			input:    "attribute_exists (foo)",
			expected: "attribute_exists(foo)",
		},
		{
			name:     "Multiple functions and logical operators",
			input:    "attribute_exists (foo) AND begins_with (bar, :val)",
			expected: "attribute_exists(foo) AND begins_with(bar, :val)",
		},
		{
			name:     "Logical operators not affected",
			input:    "foo AND bar OR baz",
			expected: "foo AND bar OR baz",
		},
		{
			name:     "Lowercase logical operators not affected",
			input:    "foo and bar or baz",
			expected: "foo and bar or baz",
		},
		{
			name:     "Mixed case logical operators not affected",
			input:    "foo Or bar aNd baz",
			expected: "foo Or bar aNd baz",
		},
		{
			name:     "No space after function name",
			input:    "attribute_exists(foo)",
			expected: "attribute_exists(foo)",
		},
		{
			name:     "Space after logical operator and function",
			input:    "AND begins_with (bar, :val)",
			expected: "AND begins_with(bar, :val)",
		},
		{
			name:     "Function with multiple spaces before paren",
			input:    "begins_with    (bar, :val)",
			expected: "begins_with(bar, :val)",
		},
		{
			name:     "Function with tab before paren",
			input:    "begins_with\t(bar, :val)",
			expected: "begins_with(bar, :val)",
		},
		{
			name:     "Function with space and tab before paren",
			input:    "begins_with \t(bar, :val)",
			expected: "begins_with(bar, :val)",
		},
		{
			name:     "No function, just text",
			input:    "foo bar baz",
			expected: "foo bar baz",
		},
		{
			name:     "Function name is logical operator (should not remove space)",
			input:    "AND (foo)",
			expected: "AND (foo)",
		},
		{
			name:     "Lowercase logical operator as function name (should not remove space)",
			input:    "or (foo)",
			expected: "or (foo)",
		},
		{
			name:     "Function with underscore",
			input:    "my_func (x)",
			expected: "my_func(x)",
		},
		{
			name:     "Function with digits",
			input:    "func123 (x)",
			expected: "func123(x)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanExpressionSpacing(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
func TestStripWrappingParens(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"(foo)", "foo"},
		{"((foo))", "foo"},
		{"(foo", "foo"},
		{"foo)", "foo"},
		{"((foo)", "foo"},
		{"(size(bar))", "size(bar)"},
		{"(size(bar)", "size(bar)"},
		{"size(bar))", "size(bar)"},
		{"size(bar)", "size(bar)"},
		{"foo(bar(baz))", "foo(bar(baz))"},
		{"((foo(bar)))", "foo(bar)"},
		{"(((foo)))", "foo"},
		{"(foo(bar))", "foo(bar)"},
		{"", ""},
		{"()", ""},
		{"((()))", ""},
		{"(foo)   ", "foo"},
		{"   (foo)", "foo"},
		{"   (foo)   ", "foo"},
		{"(((foo(bar(baz)))))", "foo(bar(baz))"},
		{"(foo(bar(baz))", "foo(bar(baz))"},
		{"foo(bar(baz)))", "foo(bar(baz))"},
		{"((foo(bar(baz))))", "foo(bar(baz))"},
		{"(foo(bar(baz)))))", "foo(bar(baz))"},
		{"(((foo(bar(baz)))", "foo(bar(baz))"},
		{"(foo(bar(baz)) extra)", "foo(bar(baz)) extra"},
		{"(foo)bar", "(foo)bar"},
		{"foo(bar)", "foo(bar)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stripWrappingParens(tt.input)
			if result != tt.expected {
				t.Errorf("stripWrappingParens(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}
func TestCountParens(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"()", 0},
		{"((", 2},
		{"))", -2},
		{"(()", 1},
		{"())", -1},
		{"(foo)", 0},
		{"((foo)", 1},
		{"(foo))", -1},
		{"foo(bar(baz))", 0},
		{"foo(bar(baz)", 1},
		{"foo)bar(baz(", 1},
		{"((foo(bar)))", 0},
		{"(((foo)))", 0},
		{"(foo(bar(baz)) extra)", 0},
		{"(foo)bar", 0},
		{"foo(bar)", 0},
		{"(((((", 5},
		{")))))", -5},
		{"(()())", 0},
		{"(()", 1},
		{"())", -1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := countParens(tt.input)
			if result != tt.expected {
				t.Errorf("countParens(%q) = %d; want %d", tt.input, result, tt.expected)
			}
		})
	}
}
