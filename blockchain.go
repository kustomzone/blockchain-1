package blockchain

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
	tmpBlk     *Block
	blockchain []*Block
)

type Block struct {
	index        int
	prevHash     string
	hash         string
	timestamp    time.Time
	transactions []*Transaction
	task         *Task
}

type Task struct {
	start      int
	end        int
	complexity int
}

type Transaction struct {
	amount float64
	from   string
	to     string
}

func init() {
	rand.Seed(time.Now().UnixNano())

	blk := &Block{
		timestamp: time.Now(),
		prevHash:  "0",
		task:      generateTask(),
		index:     0,
	}
	blk.hash = calcHash(blk)

	blockchain = []*Block{blk}

	tmpBlk = &Block{
		index:     1,
		prevHash:  latestBlock().hash,
		timestamp: time.Now(),
		task:      generateTask(),
	}
}

func (b *Block) String() string {
	var transactions string
	for _, t := range b.transactions {
		transactions += strconv.FormatFloat(t.amount, 'f', 6, 64) + t.from + t.to
	}

	return b.prevHash + b.timestamp.String() + transactions +
		string(b.index+b.task.end+b.task.start+b.task.complexity)
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
		task:      generateTask(),
	}
}

func mine(decision string) bool {
	if strings.Contains(decision[tmpBlk.task.start:tmpBlk.task.end], strconv.Itoa(tmpBlk.task.complexity)) {
		if isValidBlock(tmpBlk, latestBlock()) {
			tmpBlk.hash = calcHash(tmpBlk)
			blockchain = append(blockchain, tmpBlk)

			tmpBlk = createNextBlock()

			return true
		}
	}
	return false
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

func generateTask() *Task {
	var start, end int

	for {
		start = rand.Intn(32) + 0
		end = rand.Intn(33) + 1

		if start < end {
			break
		}
	}

	return &Task{
		start:      start,
		end:        end,
		complexity: rand.Intn(end-1) + start,
	}
}
