package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// const for choose information type
	// when send valid / new block or fact to nodes
	BLOCK = iota
	FACT
)

// Nodes type for store current connections
type Nodes struct {
	Conns []*websocket.Conn `json:"conns"`
	Addrs []string          `json:"addrs"`
}

// Fact type for fact
type Fact struct {
	// has unique id for identify
	Id   string       `json:"id"`
	Fact *interface{} `json:"fact,omitempty"`
}

// Block type for store block
type Block struct {
	Index int `json:"index"`
	// calculated from block info
	Hash string `json:"hash"`
	// point to previous block hash
	PrevHash  string    `json:"prev_hash"`
	Timestamp time.Time `json:"timestamp"`
	Facts     []*Fact   `json:"facts,omitempty"`
	// mining complexity
	Complexity int `json:"complexity"`
}

// BlockAPI type for send valid block / new mining block to other nodes
type BlockAPI struct {
	ValidBlock  *Block `json:"valid_block"`
	MiningBlock *Block `json:"mining_block,omitempty"`
}

// API type for communicate with other nodes or clients
type API struct {
	// information type
	// used only when send valid / new block or new fact
	// to other nodes
	Type       int       `json:"type,omitempty"`
	Complexity int       `json:"complexity,omitempty"`
	Error      string    `json:"error,omitempty"`
	Fact       *Fact     `json:"fact,omitempty"`
	Blocks     *BlockAPI `json:"blocks,omitempty"`
	Nodes      []string  `json:"nodes,omitempty"`
	Facts      []*Fact   `json:"facts,omitempty"`
	Blockchain []*Block  `json:"blockchain,omitempty"`
}

var (
	blockchain []*Block
	block      *Block
	facts      []*Fact
	nodes      = &Nodes{}

	iPeer    = flag.String("ipeer", "", "init peer address")
	httpPort = flag.String("hport", "", "set http port")
	wsPort   = flag.String("wsport", "", "set ws port")
	verbose  = flag.Bool("v", false, "enable verbose output")

	mineNotify = make(chan *BlockAPI)
	factNotify = make(chan *Fact)
)

func init() {
	flag.Parse()

	if *iPeer != "" {
		initNode()
	} else {
		initRootNode()
	}
}

func initRootNode() {
	blockchain = []*Block{{
		Timestamp: time.Now(),
	}}
	blockchain[0].Hash = calcHash(blockchain[0].String())
	block = createNextBlock()
}

func initNode() {
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

	err = json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		panic(err)
	}
	block = t.Blocks.BlkN
	blockchain = t.Blockchain

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
		case t, ok := <-mineNotify:
			if ok {
				block = t.BlkN
				for _, node := range nodes.Conns {
					err := websocket.JSON.Send(node, &API{
						Blocks: &BlockAPI{
							BlkN: t.BlkN,
							BlkS: t.BlkS,
						},
					})
					if err != nil {
						removePeer(node)
					}
				}
			}
		case fact, ok := <-factNotify:
			if ok {
				for _, node := range nodes.Conns {
					err := websocket.JSON.Send(node, API{
						Type: FACT, Fact: fact,
					})
					if err != nil {
						removePeer(node)
					}
				}
			}
		}
	}
}

func removePeer(ws *websocket.Conn) {
	info("client disconnect", ws.RemoteAddr())

	for i, addr := range nodes.Addrs {
		if ws.RemoteAddr().String() == addr {
			nodes.Addrs = append(nodes.Addrs[:i], nodes.Addrs[i+1:]...)
			nodes.Conns = append(nodes.Conns[:i], nodes.Conns[i+1:]...)
		}
	}
}

func read(ws *websocket.Conn) {
	for {
		t := &API{}

		err := websocket.JSON.Receive(ws, t)
		if err != nil {
			removePeer(ws)
			return
		}

		switch t.Type {
		case BLOCK:
			if isValidBlock(t.Blocks.BlkS, latestBlock()) {
				blockchain = append(blockchain, t.Blocks.BlkS)
			}

			block = t.Blocks.BlkN

			for _, tFact := range t.Blocks.BlkS.Facts {
				for i, lFact := range facts {
					if tFact.Id == lFact.Id {
						facts = append(facts[:i], facts[i+1:]...)
					}
				}
			}

			break
		case FACT:
			info("new fact from", ws.RemoteAddr(), *t.Fact.Fact)
			facts = append(facts, t.Fact)
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

func handleBlock(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(API{
		Blockchain: blockchain,
		Blocks: &BlockAPI{
			BlkN: block,
		},
	})
	if err != nil {
		panic(err)
	}
}

func handleFact(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")

		id, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil || id < 0 || id > len(blockchain)-1 {
			w.WriteHeader(http.StatusInternalServerError)
			err = json.NewEncoder(w).Encode(API{
				Error: "invalid id",
			})
			if err != nil {
				panic(err)
			}
			return
		}

		err = json.NewEncoder(w).Encode(API{
			Facts: blockchain[id].Facts,
		})
		if err != nil {
			panic(err)
		}

		break
	case http.MethodPost:
		var fact interface{}
		err := json.NewDecoder(r.Body).Decode(&fact)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			err = json.NewEncoder(w).Encode(API{
				Error: "invalid incoming data",
			})
			if err != nil {
				panic(err)
			}
			return
		}

		t := &Fact{Id: time.Now().String(), Fact: &fact}
		factNotify <- t
		facts = append(facts, t)
	}
}

func handleMine(w http.ResponseWriter, r *http.Request) {
	go mine(r.URL.Query().Get("nonce"))
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
	if strings.Count(calcHash(block.Hash + nonce)[:block.Complexity], "0") == block.Complexity {
		if isValidBlock(block, latestBlock()) {
			blockchain = append(blockchain, block)
			mineNotify <- &BlockAPI{BlkS: block, BlkN: createNextBlock()}
			info(block.String())
		}
	}
}

func (b *Block) String() string {
	var facts string
	for _, fact := range b.Facts {
		facts += fact.Id
		facts += fmt.Sprint(*fact.Fact)
	}

	return b.PrevHash + b.Timestamp.String() +
		fmt.Sprint(b.Index, facts, b.Complexity)
}

func calcHash(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
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
			Facts:     facts,
		}
	)

	facts = nil

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

func info(info ...interface{}) {
	if *verbose {
		log.Println(info)
	}
}

func main() {
	go func() {
		http.HandleFunc("/blocks", handleBlock)
		http.HandleFunc("/fact", handleFact)
		http.HandleFunc("/mine", handleMine)
		http.HandleFunc("/nodes", handleNodes)

		info("http server starting at port:", *httpPort)
		panic(http.ListenAndServe(":"+*httpPort, nil))
	}()

	go func() {
		http.Handle("/peer", websocket.Handler(handlePeer))

		info("ws server starting at port:", *wsPort)
		panic(http.ListenAndServe(":"+*wsPort, nil))
	}()

	notify()
}
