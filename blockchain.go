package main

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	tempBlock = &block{
		height:      latestBlock().height + 1,
		pHash:       latestBlock().hash,
		timestamp:   time.Now(),
		transaction: nil,
	}

	task       = "0"
	complexity = 1

	blockchain = []*block{genesisBlock()}
)

type block struct {
	sync.Mutex
	height      int
	pHash       string
	hash        string
	timestamp   time.Time
	transaction []*transaction
}

type transaction struct {
	cash float64
	from string
	to   string
}

func (b *block) String() string {
	var data string

	for _, d := range b.transaction {
		data += strconv.FormatFloat(d.cash, 'f', 6, 64) + d.from + d.to
	}

	return string(b.height) + b.pHash + b.timestamp.String() + data
}

func calcHash(b *block) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(b.String())))
}

func latestBlock() *block {
	return blockchain[len(blockchain)-1]
}

func mine(decision string) {
	if strings.Contains(decision, task) {
		if isValidBlock(tempBlock, latestBlock()) {
			tempBlock.hash = calcHash(tempBlock)

			blockchain = append(blockchain, tempBlock)

			tempBlock = &block{
				height:      tempBlock.height + 1,
				pHash:       tempBlock.hash,
				timestamp:   time.Now(),
				transaction: nil,
			}

			task = ""
			complexity = rand.Intn(2) + 2
			for i := 0; i < complexity; i++ {
				task += "0"
			}
		}
	}
}

func addTransaction(cash float64, from, to string) {
	tempBlock.transaction = append(tempBlock.transaction, &transaction{
		cash: cash,
		from: from,
		to:   to,
	})
}

func getTransactions(height int) []*transaction {
	return blockchain[len(blockchain)-1].transaction
}

func getAllTransactions() (t []*transaction) {
	for _, block := range blockchain {
		for _, bt := range block.transaction {
			t = append(t, bt)
		}
	}
	return
}

func genesisBlock() *block {
	blk := &block{
		height:      0,
		pHash:       "0",
		timestamp:   time.Now(),
		transaction: []*transaction{},
	}
	blk.hash = calcHash(blk)
	return blk
}

func isValidBlock(nBlock, pBlock *block) bool {
	if pBlock.height+1 != nBlock.height &&
		pBlock.hash != nBlock.pHash &&
		calcHash(nBlock) != nBlock.hash {

		return false
	}
	return true
}
