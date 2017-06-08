package main

import (
	"net/http"
	"testing"
)

func Test(t *testing.T) {
	var iterations = 10000

	for k := 0; k < iterations; k++ {
		http.Get("http://localhost:1003/mine?nonce=" + string(k))
	}
}
