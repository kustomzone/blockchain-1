package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/net/websocket"
	l "log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	BLOCK = iota
	FACT
)

var (
	blockchain []*Block
	block      *Block

	iPeer    = flag.String("ipeer", "", "init peer address")
	httpPort = flag.String("hport", "", "set http port")
	wsPort   = flag.String("wsport", "", "set ws port")
	verbose  = flag.Bool("v", false, "enable verbose output")

	records []*interface{}

	mineNotify = make(chan *Block)
	factNotify = make(chan *interface{})

	nodes = &Nodes{}
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
	Type   int            `json:"type,omitempty"`
	Nodes  []string       `json:"nodes,omitempty"`
	Block  *Block         `json:"block,omitempty"`
	Fact   []*interface{} `json:"fact,omitempty"`
	Record *interface{}   `json:"record"`
}

func init() {
	flag.Parse()

	if *iPeer != "" {
		nodeInit()
	} else {
		blockchain = []*Block{{
			Index:     0,
			PrevHash:  "0",
			Timestamp: time.Now(),
		}}
		blockchain[0].Hash = calcHash(blockchain[0].String())
		block = createNextBlock()
	}
}

func nodeInit() {
	r, err := http.Get("http://" + *iPeer + "/nodes")
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()

	var t *API
	err = json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		panic(err)
	}
	nodes.Addrs = t.Nodes

	r, err = http.Get("http://" + *iPeer + "/blocks")
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()

	err = json.NewDecoder(r.Body).Decode(&blockchain)
	if err != nil {
		panic(err)
	}
	block = createNextBlock()

	origin := "ws://localhost:" + *wsPort + "/peer"
	for i, addr := range nodes.Addrs {
		if addr == origin {
			nodes.Addrs = append(nodes.Addrs[:i], nodes.Addrs[i+1:]...)
			continue
		}

		ws, err := websocket.Dial(addr, "", origin)
		if err != nil {
			panic(err)
		}

		go read(ws)
		nodes.Conns = append(nodes.Conns, ws)
	}

	ws, err := websocket.Dial("ws://"+*iPeer+"/peer", "", origin)
	if err != nil {
		panic(err)
	}

	go read(ws)
	nodes.Addrs = append(nodes.Addrs, ws.RemoteAddr().String())
	nodes.Conns = append(nodes.Conns, ws)
}

func notify() {
	for {
		select {
		case blk, ok := <-mineNotify:
			if ok {
				for _, node := range nodes.Conns {
					err := websocket.JSON.Send(node, API{
						Type: BLOCK, Block: blk,
					})
					if err != nil {
						panic(err)
					}
				}
			}
		case fact, ok := <-factNotify:
			if ok {
				for _, node := range nodes.Conns {
					err := websocket.JSON.Send(node, API{
						Type: FACT, Record: fact,
					})
					if err != nil {
						panic(err)
					}
				}
			}
		}
	}
}

func read(ws *websocket.Conn) {
	for {
		t := &API{}

		err := websocket.JSON.Receive(ws, t)
		if err != nil {
			panic(err)
		}

		switch t.Type {
		case BLOCK:
			log(t.Block)
			if isValidBlock(t.Block, latestBlock()) {
				blockchain = append(blockchain, t.Block)
			}
			break
		case FACT:
			log("new fact from", ws.RemoteAddr(), *t.Record)
			records = append(records, t.Record)
		}
	}
}

func handlePeer(ws *websocket.Conn) {
	search := false
	for _, node := range nodes.Addrs {
		if node == ws.RemoteAddr().String() {
			search = true
		}
	}
	if !search {
		nodes.Addrs = append(nodes.Addrs, ws.RemoteAddr().String())
		nodes.Conns = append(nodes.Conns, ws)
	}

	read(ws)
}

func handleBlock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(blockchain)
	if err != nil {
		panic(err)
	}
}

func handleFact(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			panic(err)
		}

		if id < 0 || id > len(blockchain)-1 {
			// send that id is incorrect
			return
		}

		err = json.NewEncoder(w).Encode(API{
			Fact: blockchain[id].Facts,
		})
		if err != nil {
			panic(err)
		}
		break
	case http.MethodPost:
		var fact interface{}
		err := json.NewDecoder(r.Body).Decode(&fact)
		if err != nil {
			panic(err)
		}

		factNotify <- &fact
		records = append(records, &fact)
	}
}

func handleMine(w http.ResponseWriter, r *http.Request) {
	mine(r.URL.Query().Get("nonce"))
	w.WriteHeader(http.StatusOK)
}

func handleNodes(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(API{Nodes: nodes.Addrs})
	if err != nil {
		panic(err)
	}
}

func mine(nonce string) {
	if strings.Count(calcHash(nonce)[:block.Complexity], "0") == block.Complexity {
		if isValidBlock(block, latestBlock()) {
			mineNotify <- block
			blockchain = append(blockchain, block)
			block = createNextBlock()
			records = nil
		}
	}
}

func (b *Block) String() string {
	var facts string

	for _, fact := range b.Facts {
		facts += fmt.Sprint(*fact)
	}

	return b.PrevHash + b.Timestamp.String() +
		fmt.Sprint(b.Index, facts, b.Complexity)
}

func calcHash(data string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
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

func log(info ...interface{}) {
	if *verbose {
		l.Println(info)
	}
}

func main() {
	// http server
	go func() {
		http.HandleFunc("/blocks", handleBlock)
		http.HandleFunc("/fact", handleFact)
		http.HandleFunc("/mine", handleMine)
		http.HandleFunc("/nodes", handleNodes)

		log("http server starting at port:", *httpPort)
		panic(http.ListenAndServe(":"+*httpPort, nil))
	}()

	// websocket server
	go func() {
		http.Handle("/peer", websocket.Handler(handlePeer))

		log("ws server starting at port:", *wsPort)
		panic(http.ListenAndServe(":"+*wsPort, nil))
	}()

	notify()
}
