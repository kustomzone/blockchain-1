package main

import (
	"net/http"
	"strconv"
	"testing"
)

func TestMine(t *testing.T) {
	for i := 0; i < 1000; i++ {
		resp, _ := http.Get("http://localhost:1000/mine?nonce=" + strconv.Itoa(i))
		defer resp.Body.Close()
	}
}
