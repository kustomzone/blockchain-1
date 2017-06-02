package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	tmpBlk     *Block
	blockchain []*Block

	iPeer    = *flag.String("iperr", "", "init peer address")
	wsPort   = *flag.String("hport", "", "set http port")
	httpPort = *flag.String("wsport", "", "set ws port")
	nodes    []websocket.Addr

	successMineNotify = make(chan *Block)
)

type Block struct {
	index     int
	prevHash  string
	hash      string
	timestamp time.Time
	facts     []*interface{}
	task      *Task
}

type Task struct {
	start      int
	end        int
	complexity int
}

func init() {
	flag.Parse()

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
	return b.prevHash + b.timestamp.String() +
		fmt.Sprint(b.facts, b.index, b.task.end, b.task.start, b.task.complexity)
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
	if strings.Contains(
		decision[tmpBlk.task.start:tmpBlk.task.end],
		strconv.Itoa(tmpBlk.task.complexity),
	) {
		if isValidBlock(tmpBlk, latestBlock()) {
			tmpBlk.hash = calcHash(tmpBlk)
			blockchain = append(blockchain, tmpBlk)

			tmpBlk = createNextBlock()

			return true
		}
	}
	return false
}

func addFact(data *interface{}) {
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
		// 0 - 31
		start = rand.Intn(32)
		// 1 - 32
		end = rand.Intn(32) + 1

		if start < end {
			break
		}
	}

	return &Task{
		start:      start,
		end:        end,
		complexity: rand.Intn(end) + start,
	}
}

func main() {
	done := make(chan os.Signal)
	defer close(done)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	// http server
	go func() {
		http.HandleFunc("/blocks", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(blockchain))
		})

		http.HandleFunc("/facts", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				var facts string
				for _, block := range blockchain {
					facts += fmt.Sprint(block.facts)
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(facts))
			case http.MethodPost:
				var (
					buf  = make([]byte, 10240)
					fact interface{}
				)
				n, err := r.Body.Read(buf)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					break
				}
				err = json.Unmarshal(buf[:n], fact)
				tmpBlk.facts = append(tmpBlk.facts, &fact)
				w.WriteHeader(http.StatusOK)
			}
		})

		http.HandleFunc("/mine", func(w http.ResponseWriter, r *http.Request) {
			mine(r.URL.Query().Get("decision"))
		})

		http.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(nodes))
		})

		log.Fatal(http.ListenAndServe(":"+httpPort, nil))
	}()

	// websocket server
	go func() {
		http.Handle("/peer", websocket.Handler(func(ws *websocket.Conn) {
			go func() {}()
			go func() {}()
		}))

		log.Fatal(http.ListenAndServe(":"+wsPort, nil))
	}()

	// client
	r, err := http.Get(iPeer + "/nodes")
	if err != nil {
		log.Fatal(err)
	}
	defer r.Body.Close()

	err = json.NewDecoder(r.Body).Decode(nodes)
	if err != nil {
		log.Fatal(err)
	}

	r, err = http.Get(iPeer + "/blocks")
	if err != nil {
		log.Fatal(err)
	}
	defer r.Body.Close()

	err = json.NewDecoder(r.Body).Decode(blockchain)
	if err != nil {
		log.Fatal(err)
	}

	for _, node := range nodes {
		go func() {
			ws, err := websocket.Dial(node.String()+"peer", "", "http://localhost")
			if err != nil {
				log.Fatal(err)
			}

			for {
				blk, ok := <-successMineNotify
				if ok {
					blkjson, err := json.Marshal(blk)
					if err != nil {
						log.Fatal(err)
					}

					_, err = ws.Write(blkjson)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}()
	}

	<-done
}
