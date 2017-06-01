package main

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestMine(t *testing.T) {
	var (
		localBlockChainLen = 0
		timeMining         time.Time
	)

	go checkBlockLifetime()

	for i := 0; i < 1000000; i++ {
		mine(fmt.Sprintf("%x", sha256.Sum256([]byte(strconv.Itoa(i)))))

		if localBlockChainLen != len(blockchain) {
			localBlockChainLen = len(blockchain)
			fmt.Println(localBlockChainLen, " --- ", time.Since(timeMining))

			timeMining = time.Now()
		}
	}
}
