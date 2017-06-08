package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

func Test(t *testing.T) {
	var (
		mining  = 10000
		clients = 4
		done    = make(chan bool, clients)

		jd = []byte(`{"data":"."}`)
	)
	defer close(done)

	send := func(clientPort int) {
		for i := 0; i < mining; i++ {
			http.Get("http://localhost:100" + strconv.Itoa(clientPort) + "/mine?nonce=" + string(i))
		}
		done <- true
	}

	for i := 0; i < clients; i++ {
		http.Post("http://localhost:100"+strconv.Itoa(i)+"/fact", "", bytes.NewBuffer(jd))

		go send(i)
	}

	for i := 0; i < clients; i++ {
		<-done
	}
	fmt.Println("lal")
}
