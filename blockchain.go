package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	blockchain []*Block
	block      *Block

	iPeer    = flag.String("iperr", "", "init peer address")
	httpPort = flag.String("hport", "", "set http port")
	wsPort   = flag.String("wsport", "", "set ws port")

	records    []*interface{}
	mineNotify = make(chan *Block)

	complexity = 1
)

type Block struct {
	Index     int            `json:"index"`
	Hash      string         `json:"hash"`
	PrevHash  string         `json:"prev_hash"`
	Timestamp time.Time      `json:"timestamp"`
	Facts     []*interface{} `json:"facts,omitempty"`
}

func init() {
	flag.Parse()

	blockchain = []*Block{{
		Index:     0,
		PrevHash:  "0",
		Timestamp: time.Now(),
	}}
	blockchain[0].Hash = calcHash(blockchain[0].String())

	block = createNextBlock()
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

	blk.Hash = calcHash(blk.String())
	return blk
}

func mine(nonce string) {
	if strings.Count(calcHash(nonce)[:complexity], "0") == complexity {
		if isValidBlock(block, latestBlock()) {
			if time.Since(block.Timestamp) < time.Second*10 {
				complexity++
			} else {
				complexity--
			}
			block.Facts = records
			blockchain = append(blockchain, block)

			mineNotify <- block

			block = createNextBlock()
			records = nil
		}
	}
}

func isValidBlock(nBlock, pBlock *Block) bool {
	if pBlock.Index+1 != nBlock.Index ||
		pBlock.Hash != nBlock.PrevHash ||
		calcHash(nBlock.String()) != nBlock.Hash {

		return false
	}
	return true
}

func main() {
	var nodes []string

	// http server
	go func() {
		http.HandleFunc("/blocks", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(blockchain)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal(err)
			}
		})

		http.HandleFunc("/fact", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				var fact interface{}
				err := json.NewDecoder(r.Body).Decode(&fact)
				if err != nil {
					log.Fatal(err)
				}

				records = append(records, &fact)
			}
		})

		http.HandleFunc("/mine", func(w http.ResponseWriter, r *http.Request) {
			go mine(r.URL.Query().Get("nonce"))
			w.WriteHeader(http.StatusOK)
		})

		http.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
			err := json.NewEncoder(w).Encode(&map[string]interface{}{
				"nodes":    nodes,
				"initaddr": "localhost:" + *wsPort,
			})
			if err != nil {
				log.Panic(err)
			}
		})

		log.Println("http server start at port:", *httpPort)
		log.Fatal(http.ListenAndServe(":"+*httpPort, nil))
	}()

	// websocket server
	go func() {
		http.Handle("/peer", websocket.Handler(func(ws *websocket.Conn) {
			nodes = append(nodes, ws.RemoteAddr().String())

			for {
				buf := make([]byte, 10240)
				n, err := ws.Read(buf)
				if err != nil {
					log.Fatal(err)
				}

				var blk *Block
				err = json.Unmarshal(buf[:n], blk)
				if err != nil {
					log.Fatal(err)
				}

				if isValidBlock(blk, latestBlock()) {
					blockchain = append(blockchain, blk)
				}
			}
		}))

		log.Fatal(http.ListenAndServe(":"+*wsPort, nil))
	}()

	if *iPeer != "" {
		// client
		r, err := http.Get(*iPeer + "/nodes")
		if err != nil {
			log.Fatal(err)
		}
		defer r.Body.Close()

		var lal map[string]interface{}

		err = json.NewDecoder(r.Body).Decode(&lal)
		if err != nil {
			log.Panic(err)
		}

		log.Println(lal)

		r, err = http.Get(*iPeer + "/blocks")
		if err != nil {
			log.Fatal(err)
		}
		defer r.Body.Close()

		err = json.NewDecoder(r.Body).Decode(&blockchain)
		if err != nil {
			log.Fatal(err)
		}

		//nodes = append(nodes, ipper)

		for _, node := range nodes {
			go func() {
				ws, err := websocket.Dial(node+"/peer", "", "http://localhost")
				if err != nil {
					log.Fatal(err)
				}

				for {
					blk, ok := <-mineNotify
					if ok {
						blkjson, err := json.Marshal(&blk)
						if err != nil {
							log.Fatal(err)
						}

						_, err = ws.Write(blkjson)
						if err != nil {
							log.Fatal(err)
						}
					} else {
						log.Fatal("error blk notify")
					}
				}
			}()
		}
	}

	done := make(chan os.Signal)
	defer close(done)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
}
