# Blockchain
A blockchain – is a distributed database that is used to maintain a 
continuously growing list of records, called blocks.
Each block contains a timestamp and a link to a previous block.
A blockchain is typically managed by a peer-to-peer network collectively 
adhering to a protocol for validating new blocks. By design, blockchains 
are inherently resistant to modification of the data. Once recorded, 
the data in any given block cannot be altered retroactively without 
the alteration of all subsequent blocks and the collusion of the network.

## How it work
### Root node
It initializes the blockchain and mining block. 
The first block is genesis block.
###### Genesis block
```
{
    "index": 0,
    "hash": "3368823cb6d6fab32c4535265579f83ed79830664dc346ea4f9acddc21ebf02a",
    "prev_hash": "",
    "timestamp": "2017-06-09T23:19:33.2947309+03:00",
    "complexity": 0,
    "nonce": ""
}
```

### Other nodes
First, node requests the initialization node for following information:
1. Current blockchain
2. Current mining block
3. List of current nodes

Then node connects to each node by WebSockets.

### HTTP and WebSocket
Nodes raises the HTTP and WebSocket server 
to work with other nodes (WebSocket) and (HTTP) to view information about blockchain:
1. Blockchain
2. Current mining block
3. Block facts
4. Nodes
5. Handler for block mining

### Block
#### Block contains following data:
- Index - block index
- Hash - calculated from block data (sha256)
- Previous block hash - latest block hash
- Timestamp - created time
- Facts - confirmed facts
- Complexity - solution complexity
- Nonce - number to solve block

Each block contains hash of previous block to preserve chain integrity.

#### Block has been validated if:
1. its index is equal to index latest block + 1
2. latest block hash is equal to previous hash of current block 
3. calculation of hash of current block is equal to its hash

#### Creation of the next block is:
1. Index = latest block index + 1
2. Previous hash = latest block hash
3. Timestamp = current time
4. Facts = take unconfirmed facts
5. Complexity = increase if more than 10 seconds have passed since 
creation of previous block, otherwise decrease
6. Nonce = ""
7. Hash = calculated from block data

#### Mining process
To solve block, it is necessary to find such a number (nonce)
that this number + hash of block contained number of leading zeros 
greater than or equal to complexity of block.

### Work process
When node is initialized, it will be connected to others 
via a WebSockets, and node is ready to receive a new block or fact.

When you add a new fact to node, it sends it to other nodes 
and enters to list of unconfirmed facts.

If block is successfully solved, 
node creates a new block for solution on the basis of newly solved
than sends solved block to other nodes for verification, 
if it passes check, it is added to chain, 
together with solved block, node sends next mining block, 
so that all nodes solve block with same complexity.

Nodes, when checking block, look through list of confirmed facts, 
if a fact is found that coincides with fact from unconfirmed ones, 
then it is removed therefrom, compared by a unique id consisting of
sha256 hash from creation time.
