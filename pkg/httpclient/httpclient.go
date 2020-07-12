package httpclient

import (
	"net/http"
	"sync"
	"time"
)

var client *http.Client
var once sync.Once

// Get - this will return singletone object of http client
func Get() *http.Client {
	once.Do(func() {
		client = &http.Client{
			Timeout:   time.Duration(time.Second * 10),
			Transport: &http.Transport{},
		}
	})
	return client
}
