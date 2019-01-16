# Web3 CLI Tool

Simple command line tool for interacting with web3 enabled blockchains - GoChain, Ethereum, etc.

## Local installation

Clone the repo:

```sh
git clone https://github.com/gochain-io/web3
```

You also should have Docker installed [Docker](https://docs.docker.com/install/linux/docker-ce/ubuntu/)

Build:

```sh
make build
```

Run:

 ```sh
 ./web3 help
 ```

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
