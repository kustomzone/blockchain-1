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

		json0 = []byte(`{"data":"0."}`)
		json1 = []byte(`{"data":"1."}`)
		json2 = []byte(`{"data":"2."}`)
		json3 = []byte(`{"data":"3."}`)
	)
	defer close(done)

	http.Post("http://localhost:1000/fact", "", bytes.NewBuffer(json0))
	http.Post("http://localhost:1001/fact", "", bytes.NewBuffer(json1))
	http.Post("http://localhost:1002/fact", "", bytes.NewBuffer(json2))
	http.Post("http://localhost:1003/fact", "", bytes.NewBuffer(json3))

	send := func(clientPort int) {
		for i := 0; i < mining; i++ {
			http.Get("http://localhost:100" + strconv.Itoa(clientPort) + "/mine?nonce=" + string(i))
		}
		done <- true
	}

	for i := 0; i < clients; i++ {
		go send(i)
	}

	for i := 0; i < clients; i++ {
		<-done
	}
	fmt.Println("lal")
}
