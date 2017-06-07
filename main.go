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

const (
	BLOCK = iota
	RECORD
)

var (
	blockchain []*Block
	block      *Block

	iPeer    = flag.String("ipeer", "", "init peer address")
	httpPort = flag.String("hport", "", "set http port")
	wsPort   = flag.String("wsport", "", "set ws port")
	verbose  = flag.Bool("v", false, "enable verbose output")

	records      []*interface{}
	mineNotify   = make(chan *Block)
	recordNotify = make(chan *interface{})
	nodes        = &Nodes{}
)

type Nodes struct {
	Conns []*websocket.Conn `json:"conns"`
	Addrs []string          `json:"addrs"`
}

type Block struct {
	Index      int            `json:"index"`
	Hash       string         `json:"hash"`
	PrevHash   string         `json:"prev_hash"`
	Timestamp  time.Time      `json:"timestamp"`
	Facts      []*interface{} `json:"facts,omitempty"`
	Complexity int            `json:"complexity"`
}

type API struct {
	Type       int          `json:"type,omitempty"`
	NodesAddrs []string     `json:"nodes_addrs,omitempty"`
	Block      *Block       `json:"block,omitempty"`
	Record     *interface{} `json:"record,omitempty"`
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

		var t *API
		err = json.NewDecoder(r.Body).Decode(&t)
		if err != nil {
			log.Panic(err)
		}
		nodes.Addrs = t.NodesAddrs
        fmt.Println(nodes.Addrs)

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

		for _, node := range nodes.Addrs {
			ws, err := websocket.Dial(node, "", "ws://localhost:"+*wsPort+"/peer")
			if err != nil {
				log.Panic(err)
			}

			nodes.Conns = append(nodes.Conns, ws)
			go read(ws)
		}

		ws, err := websocket.Dial("ws://"+*iPeer+"/peer", "", "ws://localhost:"+*wsPort+"/peer")
		if err != nil {
			log.Panic(err)
		}
		go read(ws)

		nodes.Conns = append(nodes.Conns, ws)
		nodes.Addrs = append(nodes.Addrs, ws.RemoteAddr().String())
	} else {
		blockchain = []*Block{{
			Timestamp: time.Now(),
		}}
		blockchain[0].Hash = calcHash(blockchain[0].String())
		block = createNextBlock()
	}

	for {
		select {
		case blk, ok := <-mineNotify:
			if ok {
				for _, node := range nodes.Conns {
					err := websocket.JSON.Send(node, API{
						Type:  BLOCK,
						Block: blk,
					})
					if err != nil {
						log.Panic(err)
					}
				}
			}
		case record, ok := <-recordNotify:
			if ok {
				for _, node := range nodes.Conns {
					err := websocket.JSON.Send(node, API{
						Type:   RECORD,
						Record: record,
					})
					if err != nil {
						log.Panic(err)
					}
				}
			}
		}
	}
}

func read(ws *websocket.Conn) {
	var t *API

	for {
		err := websocket.JSON.Receive(ws, &t)
		if err != nil {
			log.Panic(err)
		}
		switch t.Type {
		case BLOCK:
			if isValidBlock(t.Block, latestBlock()) {
				blockchain = append(blockchain, t.Block)
			}
			break
		case RECORD:
			records = append(records, t.Record)
		}
	}
}

func handlePeer(ws *websocket.Conn) {
	var search bool

	if len(nodes.Conns) == 0 {
		nodes.Conns = append(nodes.Conns, ws)
		nodes.Addrs = append(nodes.Addrs, ws.RemoteAddr().String())
	}
	for _, node := range nodes.Addrs {
		if node == ws.RemoteAddr().String() {
			search = true
		}
	}
	if !search {
		nodes.Conns = append(nodes.Conns, ws)
		nodes.Addrs = append(nodes.Addrs, ws.RemoteAddr().String())
	}

	read(ws)
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

		recordNotify <- &fact
		records = append(records, &fact)
	}
}

func handleMine(w http.ResponseWriter, r *http.Request) {
	mine(r.URL.Query().Get("nonce"))
	w.WriteHeader(http.StatusOK)
}

func handleNodes(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(API{
		NodesAddrs: nodes.Addrs,
	})
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
		fmt.Sprint(b.Index, b.Facts)
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
