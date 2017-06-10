# Simple blockchain
Implementation of a simple blockchain, which can store facts with the possibility of mining
### [How it work?](BLOCKCHAIN.md)
## Usage
### Docker Compose
1. Build docker image
```
$ docker build -t blkchn .
```
2. Run containers
```
$ docker-compose up -d
```
### CLI
```
  -h string
    	set node http server port
  -i string
    	set initial node address
  -v	enable verbose output
  -ws string
    	set node websocket server port
```
   	
1. First need to run root node
```
$ go run main.go -v -h 1000 -ws 2000
```
2. Than run first node
```
$ go run main.go -v -i 1000 -h 1001 -ws 2001
```
3. Repeat second point to start each node
## API
### Get nodes
REQUEST
```
GET /nodes HTTP/1.1
```
RESPONSE
```
HTTP/1.1 200 OK
Content-Type: application/json
{
  "nodes": [
    "ws://localhost:1000/"
  ]
}
```
### Get blockchain
REQUEST
```
GET /blockchain HTTP/1.1
```
RESPONSE
```
HTTP/1.1 200 OK
Content-Type: application/json
{
  "vm_blocks": {
    "mining_block": {
      "index": 1,
      "hash": "7fb53dcaaaa23b3a46a750bad25b04b226a97f235be0c4fdfb0842e5c577a022",
      "prev_hash": "3368823cb6d6fab32c4535265579f83ed79830664dc346ea4f9acddc21ebf02a",
      "timestamp": "2017-06-09T23:19:33.3462461+03:00",
      "complexity": 1,
      "nonce": ""
    }
  },
  "blockchain": [
    {
      "index": 0,
      "hash": "3368823cb6d6fab32c4535265579f83ed79830664dc346ea4f9acddc21ebf02a",
      "prev_hash": "",
      "timestamp": "2017-06-09T23:19:33.2947309+03:00",
      "complexity": 0,
      "nonce": ""
    }
  ]
}
```
### Mine
REQUEST
```
GET /mine?nonce=0 HTTP/1.1
```
RESPONSE
```
HTTP/1.1 200 OK
```
### Post fact
REQUEST
```
POST /fact HTTP/1.1
{
  "data": ".",
   ...
}
```
RESPONSE
```
HTTP/1.1 200 OK
```
### Get block facts
REQUEST
```
GET /fact?id=0 HTTP/1.1
```
RESPONSE
```
{
  "facts": [
    {
      "id": "7e2daaed828fb122fc827c7ef75ce3f6242d159c64db3ebd75360df125ca78c7",
      "fact": {
        "data": "."
      }
    }
  ]
}
```
