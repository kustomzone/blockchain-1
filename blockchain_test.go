package main

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestMine(t *testing.T) {
	var timeMining = time.Now()

	for i := 0; i < 1000000; i++ {
		if mine(fmt.Sprintf("%x", sha256.Sum256([]byte(strconv.Itoa(i))))) {
			t.Log(time.Since(timeMining))
			timeMining = time.Now()
		}
	}
}
