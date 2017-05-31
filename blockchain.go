package main

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// blockchain
var blockchain []*block

// block
type block struct {
	index     int
	pHash     string
	hash      string
	timestamp time.Time
	data      string
}

// calc sha256x2 hash
func calcHash(b *block) string {
	h := sha256.New()
	h.Write([]byte(string(b.index) + b.pHash + b.timestamp.String() + b.data))
	h.Write([]byte(string(b.index) + b.pHash + b.timestamp.String() + b.data))
	return string(h.Sum(nil))
}

// returns next block
func createNextBlock(data string) *block {
	var latestBlock = latestBlock()

	blk := &block{
		index:     latestBlock.index + 1,
		pHash:     latestBlock.hash,
		timestamp: time.Now(),
		data:      data,
	}
	blk.hash = calcHash(blk)
	return blk
}

// returns latest block
func latestBlock() *block {
	return blockchain[len(blockchain)-1]
}

// returns genesis block
func genesisBlock() *block {
	blk := &block{
		index:     0,
		pHash:     "0",
		timestamp: time.Now(),
		data:      "genesis block",
	}
	blk.hash = calcHash(blk)
	return blk
}

// block validation
func isValidBlock(nBlock, pBlock *block) bool {
	if pBlock.index+1 != nBlock.index &&
		pBlock.hash != nBlock.pHash &&
		calcHash(nBlock) != nBlock.hash {

		return false
	}
	return true
}

func main() {
	blockchain = append(blockchain, genesisBlock())
	blockchain = append(blockchain, createNextBlock("some data"))
	fmt.Println(isValidBlock(blockchain[1], blockchain[0]))
}
