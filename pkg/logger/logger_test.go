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
