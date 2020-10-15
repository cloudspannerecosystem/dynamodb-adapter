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

package logger

import (
	"testing"
)

func TestLogError(t *testing.T) {
	err := struct {
		Error string
	}{
		"Test",
	}
	LogError(err)
}
func TestLogInfo(t *testing.T) {
	info := struct {
		Info string
	}{
		"Test info",
	}
	LogInfo(info)
}

func TestLogDebug(t *testing.T) {
	info := struct {
		Info string
	}{
		"Test info",
	}
	LogDebug(info)
}

// 20000	     70479 ns/op	    1379 B/op	      12 allocs/op
// above shows the Benchmark for using LogError in Project
func BenchmarkLogError(b *testing.B) {
	err := struct {
		Error string
	}{
		"Test",
	}
	for index := 0; index < b.N; index++ {
		LogError(err)
	}
}
