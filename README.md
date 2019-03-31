# Web3

Simple command line tool for interacting with web3 enabled blockchains - GoChain, Ethereum, etc.
This repository also exports the backing golang `package web3`.

[![API Reference](
https://camo.githubusercontent.com/915b7be44ada53c290eb157634330494ebe3e30a/68747470733a2f2f676f646f632e6f72672f6769746875622e636f6d2f676f6c616e672f6764646f3f7374617475732e737667
)](https://godoc.org/github.com/gochain-io/web3)

## Install web3

Quick one line install:

```sh
curl -LSs https://raw.githubusercontent.com/gochain-io/web3/master/install.sh | sh
```

[Install Docker](https://docs.docker.com/install/) (optional) - While not required for all commands, many of the developer commands require Docker so we recommend installing it.

[More options](#More-installation-options)

## Quickstart

If you just plan to read from the blockchain, you do not need any GO tokens and you do not need to set your `PRIVATE_KEY`. If you plan to deploy contracts or write anything to the blockchain, you'll need tokens and you'll need to set your `PRIVATE_KEY` for the account that has those tokens.

### Pick a network to use

#### a) Run a local node

Run this command to start a local node. It will print 10 addresses with keys upon starting that you can use to deploy and interact.

```sh
web3 start
export WEB3_NETWORK=localhost
```

#### b) Use the GoChain testnet

```sh
export WEB3_NETWORK=testnet
```

To do any write operations, [get yourself some GO testnet tokens](https://help.gochain.io/en/article/getting-started-4tlo7a/) so you can deploy and interact with your contract.

#### c) Use the GoChain mainnet or another web3 network

```sh
export WEB3_NETWORK=gochain
```

You'll need mainnet GO for this which you can [buy on various exchanges](https://help.gochain.io/en/article/how-to-buy-go-tokens-z12xkb/).

You can also point this to other web3 based networks such as Ethereum. Ethereum is supported by default and you
can use one of the following: `ethereum` or `ropsten`.

### Set Private Key (optional)

Required if you plan to deploy or write transactions.

```sh
export WEB3_PRIVATE_KEY=0x...
```

### Deploy a contract

Copy [contracts/hello.sol](contracts/hello.sol) into your current directory.

Then:

```sh
web3 contract build hello.sol
web3 contract deploy Hello.bin
```

This will return a contract address, copy it and use below.

### Read from a contract

Let's call a read function (which is free):

```sh
web3 contract call --address 0xCONTRACT_ADDRESS --abi Hello.abi --function hello
```

That should return: `[Hello World]`.

### Write to a contract

Now let's change the name:

```sh
web3 contract call --address 0xCONTRACT_ADDRESS --abi Hello.abi --function setName "Johnny"
```

And call the hello function again to see if the name changed:

```sh
web3 contract call --address 0xCONTRACT_ADDRESS --abi Hello.abi --function hello
```

Now it should return `[Hello Johnny]`

:boom:

### Troubleshooting

If it doesn't return Hello Johnny, you can check the logs and receipt with:

```sh
web3 rc TX_HASH
```

## Testing

To automate testing using web3 CLI, enable the JSON format flag with `--format json`. This will
return easily parseable results for your tests. Eg:

```sh
web3 --format json contract call --address 0xCONTRACT_ADDRESS --abi Hello.abi --function hello
```

And you'll get a JSON response like this:

```json
{
  "response": [
    "Hello",
    "World"
  ]
}
```

## List of available commands

### Global parameters

`$NETWORK as env variable or -network as command parameter` - the name of the network. Available networks are:

* gochain (default)
* testnet
* ethereum
* ropsten
* localhost

`$RPC_URL as env variable or -rpc-url as command parameter` - The network RPC URL (ie http://localhost:8545)

`-verbose as command parameter` - Verbose logging

### Show information about a block

```sh
web3 block BLOCK_ID
```

**Parameters:**

- BLOCK_ID - id of a block (omit for `latest`)

### Show information about a transaction

```sh
web3 transaction TX_HASH
```

**Parameters:**

- TX_HASH - hash of a transaction

### Show information about an address

```sj
web3 transaction ADDRESS_HASH
```

**Parameters:**

- ADDRESS_HASH - hash of the address

### Build a smart contract

```sh
web3 contract build FILENAME.sol --solc-version SOLC_VERSION
```

**Parameters:**

- FILENAME - the name of the .sol file, eg: `hello.sol`
- SOLC_VERSION - the version of the solc compiler

### Deploy a smart contract to a network

```sh
web3 contract deploy FILENAME.bin
```

**Parameters:**

- FILENAME - the name of the .bin file
- $WEB3_PRIVATE_KEY as env variable or -private-key as command parameter - the private key of the wallet

### Call a function of a deployed contract

```sh
web3 contract call --amount AMOUNT --address CONTRACT_ADDRESS --abi CONTRACT_ABI_FILE --function FUNCTION_NAME FUNCTION_PARAMETERS
```

or using bundled abi files

```sh
web3 contract call --amount AMOUNT --address CONTRACT_ADDRESS --abi erc20|erc721 --function FUNCTION_NAME FUNCTION_PARAMETERS
```

**Parameters:**

- CONTRACT_ADDRESS - the address of the deployed contract
- CONTRACT_ABI_FILE - the abi file of the deployed contract (take into account that there are some bundled abi files like erc20 and erc721 so you could use them without downloading or compiling them)
- FUNCTION_NAME - the name of the function you want to call
- FUNCTION_PARAMETERS - the list of the function parameters
- AMOUNT - amount of wei to be send with transaction (require only for paid transact functions)
- $WEB3_PRIVATE_KEY as env variable or -private-key as command parameter - the private key of the wallet

### List functions in an ABI

```sh
web3 contract list --abi CONTRACT_ABI_FILE
```

**Parameters:**

- CONTRACT_ABI_FILE - the abi file of the compiled contract

### Transfer amount to the address

```sh
web3 send --to RECIPIENT_ADDRESS AMOUNT
```

**Parameters:**

- RECIPIENT_ADDRESS - the address of the recepient
- AMOUNT - the amount that should be send in the transaction ie - 1go (allowed units: go,eth,nanogo,gwei,attogo,wei)

### Generate common contracts - ERC20, ERC721, etc

```sh
web3 generate contract [erc20/erc721] --name "TEST Tokens" --symbol "TEST"
```

See `web3 generate contract --help` for more information.

### Generate ABI bindings

```sh
web3 generate code --abi CONTRACT_ABI_FILE --out OUT_FILENAME --lang [go|objc|java] --pkg PGK_NAME
```

See `web3 generate code --help` for more information.

**Parameters:**
- CONTRACT_ABI_FILE - the abi file of the compiled contract
- OUT_FILENAME - the output file
- PGK_NAME - package name

## More installation options

### Install a specific version

You can use the script to install a specific version:

```sh
curl -LSs https://raw.githubusercontent.com/gochain-io/web3/master/install.sh | sh -s v0.0.9
```

### Install using the Go language

```sh
go install github.com/gochain-io/web3/cmd/web3
```

### Build from source

Clone this repo:

```sh
git clone https://github.com/gochain-io/web3
cd web3
make build
./web3 help
```
