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

type Nodes struct {
	Conns []*websocket.Conn `json:"conns"`
	Addrs []string          `json:"addrs"`
}

type Fact struct {
	Id   string
	Fact *interface{} `json:"fact,omitempty"`
}

type Block struct {
	Index      int       `json:"index"`
	Hash       string    `json:"hash"`
	PrevHash   string    `json:"prev_hash"`
	Timestamp  time.Time `json:"timestamp"`
	Complexity int       `json:"complexity"`
	Facts      []*Fact   `json:"facts,omitempty"`
}

type BlockAPI struct {
	BlkS *Block `json:"blk_s,omitempty"`
	BlkN *Block `json:"blk_n,omitempty"`
}

type API struct {
	Type       int       `json:"type,omitempty"`
	Nodes      []string  `json:"nodes,omitempty"`
	Blocks     *BlockAPI `json:"blocks,omitempty"`
	Complexity int       `json:"complexity,omitempty"`
	Facts      []*Fact   `json:"facts,omitempty"`
	Fact       *Fact     `json:"fact,omitempty"`
	Blockchain []*Block  `json:"blockchain,omitempty"`
}

var (
	blockchain []*Block
	block      *Block

	iPeer    = flag.String("ipeer", "", "init peer address")
	httpPort = flag.String("hport", "", "set http port")
	wsPort   = flag.String("wsport", "", "set ws port")
	verbose  = flag.Bool("v", false, "enable verbose output")

	facts []*Fact

	mineNotify = make(chan *BlockAPI)
	factNotify = make(chan *Fact)

	nodes = &Nodes{}
)

func init() {
	flag.Parse()

	if *iPeer != "" {
		nodeInit()
	} else {
		blockchain = []*Block{{
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

	err = json.NewDecoder(r.Body).Decode(t)
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
						panic(err)
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
			if isValidBlock(t.Blocks.BlkS, latestBlock()) {
				blockchain = append(blockchain, t.Blocks.BlkS)
			}

			for _, tFact := range t.Facts {
				log("LEL")
				for i, lFact := range facts {
					log("LAL")
					if tFact.Id == lFact.Id {
						log(true)
						facts = append(facts[:i], facts[i+1:]...)
					}
				}
			}

			block = t.Blocks.BlkN
			facts = nil

			break
		case FACT:
			log("new fact from", ws.RemoteAddr(), *t.Fact.Fact)
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

func handleBlock(w http.ResponseWriter, r *http.Request) {
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
		id, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			panic(err)
		}

		if id < 0 || id > len(blockchain)-1 {
			// send that id is incorrect
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
			panic(err)
		}

		t := &Fact{Id: time.Now().String(), Fact: &fact}
		factNotify <- t
		facts = append(facts, t)
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
	if strings.Count(calcHash(block.Hash + nonce)[:block.Complexity], "0") == block.Complexity {
		if isValidBlock(block, latestBlock()) {
			blockchain = append(blockchain, block)
			mineNotify <- &BlockAPI{BlkS: block, BlkN: createNextBlock()}
			log(block.String())
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
