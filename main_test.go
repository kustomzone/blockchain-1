package main

import (
	"net/http"
	"testing"
)

func Test(t *testing.T) {
	var (
		nodes      = 2
		iterations = 10000
	)

	for i := 0; i < nodes; i++ {
		go func() {
			for k := 0; k < iterations; k++ {
				http.Get("http://localhost:100" + string(i) + "/mine?nonce=" + string(k))
			}
		}()
	}
}
