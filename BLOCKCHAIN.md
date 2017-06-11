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
The first block is the genesis block.
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
The node raises the http and websocket server 
to work with other nodes and to view information about the blockchain.

### Other nodes
First, the node requests the initialization node the following information:
1. Current blockchain
2. Current mining block
3. List of current nodes

The node raises the http and websocket server 
to work with other nodes and to view information about the blockchain.

Then the node connects to each node by websockets.

### Block
##### The block contains following data:
- Index - block index
- Hash - calculated from block data
- Previous block hash - latest block hash
- Timestamp - created time
- Facts - confirmed facts
- Complexity - solution complexity
- Nonce - number to solve the block

Each block contains the hash of the previous block 
to preserve the chain integrity.

##### The block has been validated if:
1. its index is equal to the index latest block + 1
2. latest block hash is equal to the previous hash of the current block 
3. calculation of the hash of the current block is equal to its hash

##### The creation of the next block is:
1. Index = latest block index + 1
2. Previous hash = latest block hash
3. Timestamp = current time
4. Facts = take unconfirmed facts
5. Complexity = increase if more than 10 seconds have passed since the 
creation of the previous block, otherwise decrease
6. Nonce = ""
7. Hash = calculated from block data

##### The decision process of the block
To solve the block, it is necessary to find such a number (nonce)
that this number + hash of the block contained the number of leading zeros 
greater than or equal to the complexity of the block.

### Work process
