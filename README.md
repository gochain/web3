# GoChain cli tool

Simple command line tool for interacting with web3 enabled blockchains - GoChain/Ethereum etc.

## Local installation

Clone the repo

`git clone https://github.com/gochain-io/web3-cli`


you also should have Docker installed [Docker] (https://docs.docker.com/install/linux/docker-ce/ubuntu/)


Build:

`go build`

Run:

 `./web3-cli help`


## List of available commands

### Global parameters
`$NETWORK as env variable or -network as command parameter` - the name of the network (testnet/mainnet/ethereum/ropsten/localhost)

`$RPC_URL as env variable or -rpc-url as command parameter` - The network RPC URL (ie http://localhost:8545)

`-verbose as command parameter` - Verbose logging

### Show the clique snapshot
```
./web3-cli snapshot
```

**Parameters:**
none


### Show information about the block
```
./web3-cli block BLOCK_ID
```

**Parameters:**

- BLOCK_ID - id of the block


### Show information about the transaction
```
./web3-cli transaction TX_HASH
```

**Parameters:**

- TX_HASH - hash of the transaction


### Show information about the address
```
./web3-cli transaction ADDRESS_HASH
```

**Parameters:**

- ADDRESS_HASH - hash of the address

### Build a smart contract
```
./web3-cli contract build FILENAME
```

**Parameters:**

- FILENAME - the name of the .sol file

### Deploy a smart contract to a network
```
./web3-cli contract deploy FILENAME
```

**Parameters:**

- FILENAME - the name of the .bin file
- $PRIVATE_KEY as env variable or -private-key as command parameter - the private key of the wallet

