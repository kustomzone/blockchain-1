package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var (
	tmpBlk = &Block{
		index:     1,
		prevHash:  latestBlock().hash,
		timestamp: time.Now(),
	}

	task string

	blockchain = []*Block{{
		index:        0,
		prevHash:     "0",
		timestamp:    time.Now(),
		transactions: []*Transaction{},
		hash:         "0",
	}}
)

func init() {
	rand.Seed(time.Now().UnixNano())
	task = generateTask()
}

type Block struct {
	index        int
	prevHash     string
	hash         string
	timestamp    time.Time
	transactions []*Transaction
}

type Transaction struct {
	amount float64
	from   string
	to     string
}

func (b *Block) String() string {
	var transactions string
	for _, t := range b.transactions {
		transactions += strconv.FormatFloat(t.amount, 'f', 6, 64) + t.from + t.to
	}

	return string(b.index) + b.prevHash + b.timestamp.String() + transactions
}

func calcHash(b *Block) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(b.String())))
}

func latestBlock() *Block {
	return blockchain[len(blockchain)-1]
}

func createNextBlock() *Block {
	latestBlock := latestBlock()
	return &Block{
		index:     latestBlock.index + 1,
		timestamp: time.Now(),
		prevHash:  latestBlock.hash,
	}
}

func mine(decision string) {
	if strings.Contains(decision, task) {
		if isValidBlock(tmpBlk, latestBlock()) {
			tmpBlk.hash = calcHash(tmpBlk)
			blockchain = append(blockchain, tmpBlk)

			tmpBlk = createNextBlock()
			task = generateTask()
		}
	}
}

func addTransaction(amount float64, from, to string) {
	tmpBlk.transactions = append(tmpBlk.transactions, &Transaction{
		amount: amount,
		from:   from,
		to:     to,
	})
}

func transaction(index int) ([]*Transaction, error) {
	if index >= len(blockchain) || index < 0 {
		return nil, errors.New("invalid block index")
	}
	return blockchain[index].transactions, nil
}

func transactions() (t []*Transaction) {
	for _, block := range blockchain {
		for _, bt := range block.transactions {
			t = append(t, bt)
		}
	}
	return
}

func isValidBlock(nBlock, pBlock *Block) bool {
	if pBlock.index+1 != nBlock.index &&
		pBlock.hash != nBlock.prevHash &&
		calcHash(nBlock) != nBlock.hash {

		return false
	}
	return true
}

func checkBlockLifetime() {
	for range time.Tick(time.Second) {
		if time.Since(latestBlock().timestamp) > time.Second*10 {
			removeBlock(latestBlock().index)
		}
	}
}

func removeBlock(index int) {
	blockchain = append(blockchain[:index], blockchain[index+1:]...)
}

func generateTask() string {
	var (
		task       = ""
		complexity = rand.Intn(9) + 2
	)

	for i := 0; i < complexity; i++ {
		task += "0"
	}

	return task
}
