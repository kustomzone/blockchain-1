package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	blockchain []*Block
	block      *Block

	iPeer    = flag.String("ipeer", "", "init peer address")
	httpPort = flag.String("hport", "", "set http port")
	wsPort   = flag.String("wsport", "", "set ws port")

	records []*interface{}

	mineNotify = make(chan *Block)

	nodes []*websocket.Conn
)

type Block struct {
	Index      int            `json:"index"`
	Hash       string         `json:"hash"`
	PrevHash   string         `json:"prev_hash"`
	Timestamp  time.Time      `json:"timestamp"`
	Facts      []*interface{} `json:"facts,omitempty"`
	Complexity int            `json:"complexity"`
}

type LAL struct {
	nodes []*websocket.Conn
}

func main() {
	flag.Parse()

	// http server
	go func() {
		http.HandleFunc("/blocks", handleBlock)
		http.HandleFunc("/fact", handleFact)
		http.HandleFunc("/mine", handleMine)
		http.HandleFunc("/nodes", handleNodes)

		log.Println("http server starting at port:", *httpPort)
		log.Panic(http.ListenAndServe(":"+*httpPort, nil))
	}()

	// websocket server
	go func() {
		http.Handle("/peer", websocket.Handler(handlePeer))

		log.Println("ws server starting at port:", *wsPort)
		log.Panic(http.ListenAndServe(":"+*wsPort, nil))
	}()

	if *iPeer != "" {
		r, err := http.Get("http://" + *iPeer + "/nodes")
		if err != nil {
			log.Panic(err)
		}
		defer r.Body.Close()

		var lal LAL
		err = json.NewDecoder(r.Body).Decode(&lal)
		if err != nil {
			log.Panic(err)
		}
		nodes = lal.nodes

		r, err = http.Get("http://" + *iPeer + "/blocks")
		if err != nil {
			log.Panic(err)
		}
		defer r.Body.Close()

		err = json.NewDecoder(r.Body).Decode(&blockchain)
		if err != nil {
			log.Panic(err)
		}
		block = createNextBlock()

		for _, node := range nodes {
			log.Println("lalasdasdsa")
			ws, err := websocket.Dial(node.RemoteAddr().String(), "", "http://localhost")
			if err != nil {
				log.Panic(err)
			}

			nodes = append(nodes, ws)
		}

		ws, err := websocket.Dial("ws://"+*iPeer+"/peer", "", "http://localhost")
		if err != nil {
			log.Panic(err)
		}

		go func() {
			for {
				var blk *Block
				err := websocket.JSON.Receive(ws, &blk)
				if err != nil {
					log.Panic(err)
				}
				log.Println(blk)
				if isValidBlock(blk, latestBlock()) {
					blockchain = append(blockchain, blk)
				}
			}
		}()
		log.Println("here", ws.RemoteAddr())
		nodes = append(nodes, ws)
	} else {
		blockchain = []*Block{{
			Index:     0,
			PrevHash:  "0",
			Timestamp: time.Now(),
		}}
		blockchain[0].Hash = calcHash(blockchain[0].String())
		block = createNextBlock()
	}

	for {
		if blk, ok := <-mineNotify; ok {
			for _, node := range nodes {
				log.Println("sd")
				err := websocket.JSON.Send(node, blk)
				if err != nil {
					log.Panic(err)
				}
			}
		}
	}
}

func handlePeer(ws *websocket.Conn) {
	var search bool

	if len(nodes) == 0 {
		nodes = append(nodes, ws)
	}
	for _, node := range nodes {
		if node == ws {
			search = true
		}
	}
	if !search {
		nodes = append(nodes, ws)
	}

	for {
		var blk *Block
		err := websocket.JSON.Receive(ws, &blk)
		if err != nil {
			log.Panic(err)
		}
		log.Println(blk)
		if isValidBlock(blk, latestBlock()) {
			blockchain = append(blockchain, blk)
		}
	}
}

func handleBlock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(blockchain)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}
}

func handleFact(_ http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var fact interface{}
		err := json.NewDecoder(r.Body).Decode(&fact)
		if err != nil {
			log.Fatal(err)
		}

		records = append(records, &fact)
	}
}

func handleMine(w http.ResponseWriter, r *http.Request) {
	mine(r.URL.Query().Get("nonce"))
	w.WriteHeader(http.StatusOK)
}

func handleNodes(w http.ResponseWriter, _ *http.Request) {
	err := json.NewEncoder(w).Encode(LAL{nodes})
	if err != nil {
		log.Panic(err)
	}
}

func mine(nonce string) {
	if strings.Count(calcHash(nonce)[:block.Complexity], "0") == block.Complexity {
		if isValidBlock(block, latestBlock()) {
			blockchain = append(blockchain, block)

			mineNotify <- block

			block = createNextBlock()
			records = nil
		}
	}
}

func (b *Block) String() string {
	return b.PrevHash + b.Timestamp.String() +
		fmt.Sprint(b.Index, b.Facts, b.Complexity)
}

func calcHash(str string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(str)))
}

func latestBlock() *Block {
	return blockchain[len(blockchain)-1]
}

func createNextBlock() *Block {
	var (
		latestBlk = latestBlock()

		blk = &Block{
			Index:     latestBlk.Index + 1,
			PrevHash:  latestBlk.Hash,
			Timestamp: time.Now(),
			Facts:     records,
		}
	)

	if time.Since(latestBlk.Timestamp) < time.Second*10 {
		blk.Complexity = latestBlk.Complexity + 1
	} else {
		blk.Complexity = latestBlk.Complexity - 1
	}

	blk.Hash = calcHash(blk.String())
	return blk
}

func isValidBlock(nBlock, pBlock *Block) bool {
	if pBlock.Index+1 != nBlock.Index ||
		pBlock.Hash != nBlock.PrevHash ||
		calcHash(nBlock.String()) != nBlock.Hash {

		return false
	}
	return true
}
