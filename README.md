# Web3

Simple command line tool for interacting with web3 enabled blockchains - GoChain, Ethereum, etc. 
This repository also exports the backing golang `package web3`.

## Local installation

### I have the Go language installed

```sh
go install github.com/gochain-io/web3/cmd/web3
```

### I don't have the go language installed

#### a) Download a prebuilt release binary

Coming soon.

#### a) Build from source

Clone the repo:

```sh
git clone https://github.com/gochain-io/web3
```

Build:

```sh
make build
```

Run:

 ```sh
 ./web3 help
 ```
 
 Note: Some commands require [Docker](https://docs.docker.com/install/linux/docker-ce/ubuntu/).

## List of available commands

### Global parameters

`$NETWORK as env variable or -network as command parameter` - the name of the network (testnet/mainnet/ethereum/ropsten/localhost)

`$RPC_URL as env variable or -rpc-url as command parameter` - The network RPC URL (ie http://localhost:8545)

`-verbose as command parameter` - Verbose logging

### Show the clique snapshot

```sh
./web3 snapshot
```

**Parameters:**
none

### Show information about the block

```sh
./web3 block BLOCK_ID
```

**Parameters:**

- BLOCK_ID - id of the block (omit for `latest`)

### Show information about the transaction

```sh
./web3 transaction TX_HASH
```

**Parameters:**

- TX_HASH - hash of the transaction


### Show information about the address

```sj
./web3 transaction ADDRESS_HASH
```

**Parameters:**

- ADDRESS_HASH - hash of the address

### Build a smart contract

```sh
./web3 contract build FILENAME
```

**Parameters:**

- FILENAME - the name of the .sol file

### Deploy a smart contract to a network

```sh
./web3 contract deploy FILENAME
```

**Parameters:**

- FILENAME - the name of the .bin file
- $PRIVATE_KEY as env variable or -private-key as command parameter - the private key of the wallet
