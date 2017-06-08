package main

import (
	"bytes"
	"net/http"
	"strconv"
	"testing"
)

func Test(t *testing.T) {
	var (
		mining  = 1000
		clients = 2
		done    = make(chan bool, clients)

		jd = []byte(`{"data":"."}`)
	)
	defer close(done)

	send := func(clientPort int) {
		for i := 0; i < mining; i++ {
			r, _ := http.Get("http://localhost:100" + strconv.Itoa(clientPort) + "/mine?nonce=" + string(i))
			defer r.Body.Close()
		}
		done <- true
	}

	for i := 0; i < clients; i++ {
		r, _ := http.Post("http://localhost:100"+strconv.Itoa(i)+"/fact", "", bytes.NewBuffer(jd))
		defer r.Body.Close()

		go send(i)
	}

	for i := 0; i < clients; i++ {
		<-done
	}
}
