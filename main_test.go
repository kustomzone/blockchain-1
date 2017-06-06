package main

import (
	"net/http"
	"strconv"
	"testing"
)

func Test(t *testing.T) {
	for i := 0; i < 10000; i++ {
		http.Get("http://localhost:1000/mine?nonce=" + strconv.Itoa(i))
	}
}
