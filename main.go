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
	// constants are used to understand
	// what data came from the node

	// VMBLOCKS means that received valid / mining block
	VMBLOCKS = iota
	// FACT means that received new fact
	FACT
)

// Nodes type for store current connections
type Nodes struct {
	// store to send data to the nodes
	Conns []*websocket.Conn `json:"conns"`
	// store to send the current node list to a new node
	Addrs []string `json:"addrs"`
}

// Fact type for store fact
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
	// random number to form a hash for successful mining
	Nonce string `json:"nonce"`
}

// VMBlocks type for send valid / mining block to other nodes
type VMBlocks struct {
	ValidBlock  *Block `json:"valid_block"`
	MiningBlock *Block `json:"mining_block,omitempty"`
}

// API type for communicate with other nodes or clients
type API struct {
	// information type
	// used only when send valid / mining block or new fact
	// to other nodes
	Type int `json:"type,omitempty"`
	// mining complexity
	Complexity int `json:"complexity,omitempty"`
	// error message
	Error    string    `json:"error,omitempty"`
	Fact     *Fact     `json:"fact,omitempty"`
	VMBlocks *VMBlocks `json:"vm_blocks,omitempty"`
	// nodes addresses
	Nodes      []string `json:"nodes,omitempty"`
	Facts      []*Fact  `json:"facts,omitempty"`
	Blockchain []*Block `json:"blockchain,omitempty"`
}

var (
	// blockchain
	blockchain []*Block
	// mining block
	miningBlock *Block
	// unconfirmed facts
	unconfirmedFacts []*Fact
	// nodes
	nodes = &Nodes{}

	// initial peer addr
	iPeer = flag.String("i", "", "set initial peer addr")
	// node http server port
	hPort = flag.String("h", "", "set node http port")
	// node websocket server port
	wsPort = flag.String("ws", "", "set node websocket port")
	// verbose output flag
	v = flag.Bool("v", false, "enable verbose output")

	// channel announcing nodes about successful mining
	miningSuccessNotice = make(chan *VMBlocks)
	// channel announcing nodes about new fact
	newFactNotice = make(chan *Fact)
)

func init() {
	// parse flags
	flag.Parse()

	// if have init peer flag
	if *iPeer != "" {
		// init new node
		initNode()
	} else {
		// init root node
		initRootNode()
	}
}

// init root node
func initRootNode() {
	// init blockchain with genesis block
	blockchain = []*Block{{
		Timestamp: time.Now(),
	}}
	// calc hash for genesis block
	blockchain[0].Hash = blockchain[0].calcHash()

	// init mining block
	miningBlock = createMiningBlock()
}

// init node
func initNode() {
	var (
		t *API
		// origin node address
		// needed for send other nodes
		// that they know with which node to interact
		origin = "ws://localhost:" + *wsPort
	)

	// get current nodes
	r, err := http.Get("http://" + *iPeer + "/nodes")
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()

	err = json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		panic(err)
	}
	// set current nodes addr
	nodes.Addrs = t.Nodes

	// get current blockchain and mining block
	r, err = http.Get("http://" + *iPeer + "/blockchain")
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()

	err = json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		panic(err)
	}
	// set current mining block
	miningBlock = t.VMBlocks.MiningBlock
	// set current blockchain
	blockchain = t.Blockchain

	// connect to each nodes
	for _, addr := range nodes.Addrs {
		// dial to node
		ws, err := websocket.Dial(addr, "", origin)
		if err != nil {
			panic(err)
		}

		// start receiving node
		go receive(ws)
		// added to connections
		nodes.Conns = append(nodes.Conns, ws)
	}

	// dial to init node
	ws, err := websocket.Dial("ws://"+*iPeer+"/p2p", "", origin)
	if err != nil {
		panic(err)
	}

	// start receiving init node
	go receive(ws)
	// added to connections and addrs
	nodes.Addrs = append(nodes.Addrs, ws.RemoteAddr().String())
	nodes.Conns = append(nodes.Conns, ws)
}

// returns latest blockchain block
func latestBlock() *Block {
	return blockchain[len(blockchain)-1]
}

// create next mining block
func createMiningBlock() *Block {
	var (
		// get latest block
		latestBlk = latestBlock()

		// create new block
		blk = &Block{
			Index:     latestBlk.Index + 1,
			PrevHash:  latestBlk.Hash,
			Timestamp: time.Now(),
			Facts:     unconfirmedFacts,
		}
	)
	// flush unconfirmed facts
	// now he in new mining block
	unconfirmedFacts = nil

	// if time since create latest block < 10s
	// increase complexity
	if time.Since(latestBlk.Timestamp) < time.Second*10 {
		blk.Complexity = latestBlk.Complexity + 1
	} else {
		// if < 10s -> decrease
		blk.Complexity = latestBlk.Complexity - 1
	}

	blk.Hash = blk.calcHash()
	return blk
}

// calc hash for block
func (b *Block) calcHash() string {
	// need to convert all the facts into a string to pass in sha256
	facts := ""
	for _, fact := range b.Facts {
		facts += fact.Id
		facts += fmt.Sprint(*fact.Fact)
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(
		b.PrevHash+b.Timestamp.String()+b.Nonce+
			fmt.Sprint(b.Index, facts, b.Complexity)),
	))
}

// receive data from node
func receive(ws *websocket.Conn) {
	for {
		t := &API{}

		err := websocket.JSON.Receive(ws, t)
		if err != nil {
			// if error -> node disconnect
			nodeRemove(ws)
			return
		}

		// switch data type
		switch t.Type {
		case VMBLOCKS:
			// if block
			// valid this block
			if isValidBlock(t.VMBlocks.ValidBlock) {
				// if valid -> append to blockchain
				blockchain = append(blockchain, t.VMBlocks.ValidBlock)
			}

			// update mining block
			miningBlock = t.VMBlocks.MiningBlock

			// check on the repetition of facts
			for _, tFact := range t.VMBlocks.ValidBlock.Facts {
				for i, lFact := range unconfirmedFacts {
					if tFact.Id == lFact.Id {
						// if found -> remove fact
						unconfirmedFacts = append(unconfirmedFacts[:i], unconfirmedFacts[i+1:]...)
					}
				}
			}

			break
		case FACT:
			// if fact
			// append to unconfirmed facts
			unconfirmedFacts = append(unconfirmedFacts, t.Fact)
		}
	}
}

// remove node from nodes storage
func nodeRemove(ws *websocket.Conn) {
	// search node id
	for i, addr := range nodes.Addrs {
		// if found
		if ws.RemoteAddr().String() == addr {
			// remove from store
			nodes.Addrs = append(nodes.Addrs[:i], nodes.Addrs[i+1:]...)
			nodes.Conns = append(nodes.Conns[:i], nodes.Conns[i+1:]...)
		}
	}
}

// block validation
func isValidBlock(unconfirmedBlk *Block) bool {
	latestBlk := latestBlock()

	if latestBlk.Index+1 != unconfirmedBlk.Index ||
		latestBlk.Hash != unconfirmedBlk.PrevHash ||
		unconfirmedBlk.calcHash() != unconfirmedBlk.Hash {

		return false
	}
	return true
}

// print info log in verbose mode
func info(info ...interface{}) {
	if *v {
		log.Println(info)
	}
}

// notify the nodes of a successful mining or new fact
func notify() {
	for {
		select {
		case t, ok := <-miningSuccessNotice:
			// if successful mining
			if ok {
				// update mining block
				miningBlock = t.MiningBlock
				// notify nodes
				for _, node := range nodes.Conns {
					err := websocket.JSON.Send(node, &API{
						Type: VMBLOCKS,
						VMBlocks: &VMBlocks{
							ValidBlock:  t.ValidBlock,
							MiningBlock: t.MiningBlock,
						},
					})
					if err != nil {
						// if err -> node disconnect
						nodeRemove(node)
					}
				}
			}
		case fact, ok := <-newFactNotice:
			// if new fact
			if ok {
				// notify nodes
				for _, node := range nodes.Conns {
					err := websocket.JSON.Send(node, API{
						Type: FACT, Fact: fact,
					})
					if err != nil {
						// if err -> node disconnect
						nodeRemove(node)
					}
				}
			}
		}
	}
}

// handle new peer
func handlePeer(ws *websocket.Conn) {
	// add peer to connections
	nodes.Addrs = append(nodes.Addrs, ws.RemoteAddr().String())
	nodes.Conns = append(nodes.Conns, ws)

	// start receiving
	receive(ws)
}

// handle block request
// sending blockchain and mining block
func handleBlockchain(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(API{
		Blockchain: blockchain,
		VMBlocks: &VMBlocks{
			MiningBlock: miningBlock,
		},
	})
	if err != nil {
		panic(err)
	}
}

// handler, that when requested by method get,
// sends the facts of the specified block,
// and, if requested by method post, takes a new unconfirmed fact
func handleFact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:

		id, err := strconv.Atoi(r.URL.Query().Get("id"))
		// send that received id is invalid
		if err != nil || id < 0 || id > len(blockchain)-1 {
			w.WriteHeader(http.StatusInternalServerError)
			err = json.NewEncoder(w).Encode(API{
				Error: "Invalid block id",
			})
			if err != nil {
				panic(err)
			}
			return
		}

		// send block facts
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
			w.WriteHeader(http.StatusInternalServerError)
			err = json.NewEncoder(w).Encode(API{
				Error: "Invalid incoming data",
			})
			if err != nil {
				panic(err)
			}
			return
		}

		t := &Fact{Id: time.Now().String(), Fact: &fact}
		// notify nodes of a new fact
		newFactNotice <- t
		// append to other unconfirmed facts
		unconfirmedFacts = append(unconfirmedFacts, t)
	}
}

// handle that try mining
func handleMine(w http.ResponseWriter, r *http.Request) {
	// try mining
	go tryMining(r.URL.Query().Get("nonce"))
}

// handler that send nodes addresses
func handleNodes(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(API{Nodes: nodes.Addrs})
	if err != nil {
		panic(err)
	}
}

// try mining
func tryMining(nonce string) {
	// update nonce
	miningBlock.Nonce = nonce

	// solve a problem
	if strings.Count(
		miningBlock.calcHash()[:miningBlock.Complexity],
		"0") == miningBlock.Complexity {

		// if solved -> validate block
		if isValidBlock(miningBlock) {
			// if block valid -> append to blockchain
			// and notify nodes
			blockchain = append(blockchain, miningBlock)
			miningSuccessNotice <- &VMBlocks{
				ValidBlock:  miningBlock,
				MiningBlock: createMiningBlock(),
			}
		}
	}
}

func main() {
	// start http server
	go func() {
		http.HandleFunc("/blockchain", handleBlockchain)
		http.HandleFunc("/fact", handleFact)
		http.HandleFunc("/mine", handleMine)
		http.HandleFunc("/nodes", handleNodes)

		panic(http.ListenAndServe(":"+*hPort, nil))
	}()

	// start websocket server
	go func() {
		http.Handle("/p2p", websocket.Handler(handlePeer))

		panic(http.ListenAndServe(":"+*wsPort, nil))
	}()

	// notify nodes
	notify()
}
