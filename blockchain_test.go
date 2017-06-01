package main

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}

func TestMine(t *testing.T) {
	var (
		localBlockChainLen = 0
		timeMining         time.Time
	)

	for range time.Tick(time.Millisecond * 25) {
		mine(fmt.Sprintf("%x", sha256.Sum256(RandStringBytes(10))))

		if localBlockChainLen != len(blockchain) {
			localBlockChainLen = len(blockchain)
			fmt.Println(localBlockChainLen, " --- ", time.Since(timeMining))

			timeMining = time.Now()
		}
	}
}
