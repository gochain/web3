    ██╗    ██╗███████╗██████╗ ██████╗      ██████╗██╗     ██╗
    ██║    ██║██╔════╝██╔══██╗╚════██╗    ██╔════╝██║     ██║
    ██║ █╗ ██║█████╗  ██████╔╝ █████╔╝    ██║     ██║     ██║
    ██║███╗██║██╔══╝  ██╔══██╗ ╚═══██╗    ██║     ██║     ██║
    ╚███╔███╔╝███████╗██████╔╝██████╔╝    ╚██████╗███████╗██║
    ╚══╝╚══╝ ╚══════╝╚═════╝ ╚═════╝      ╚═════╝╚══════╝╚═╝

Simple command line tool for interacting with web3 enabled blockchains - GoChain, Ethereum, etc.
This repository also exports the backing golang `package web3`.

[![API Reference](
https://camo.githubusercontent.com/915b7be44ada53c290eb157634330494ebe3e30a/68747470733a2f2f676f646f632e6f72672f6769746875622e636f6d2f676f6c616e672f6764646f3f7374617475732e737667
)](https://godoc.org/github.com/gochain/web3)
![](https://github.com/gochain/web3/workflows/release/badge.svg)

```sh
web3 --help
NAME:
   web3 - web3 cli tool

USAGE:
   web3 [global options] command [command options] [arguments...]

VERSION:
   0.2.34

COMMANDS:
   block, bl        Block details for a block number (decimal integer) or hash (hexadecimal with 0x prefix). Omit for latest.
   transaction, tx  Transaction details for a tx hash
   receipt, rc      Transaction receipt for a tx hash
   address, addr    Account details for a specific address, or the one corresponding to the private key.
   balance          Get balance for your private key or an address passed in(you could also use "block" as an optional parameter). eg: `balance 0xABC123` 
   increasegas      Increase gas for a transaction. Useful if a tx is taking too long and you want it to go faster.
   replace          Replace transaction. If a transaction is still pending, you can attempt to replace it.
   contract, c      Contract operations
   snapshot, sn     Clique snapshot
   id, id           Network/Chain information
   start            Start a local GoChain development node
   myaddress        Returns the address associated with WEB3_PRIVATE_KEY
   account, a       Account operations
   transfer, send   Transfer GO/ETH to an account. eg: `web3 transfer 10.1 to 0xADDRESS`
   env              List environment variables
   generate, g      Generate code
   did              Distributed identity operations
   claim            Verifiable claims operations
   help, h          Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --network value, -n value  The name of the network. Options: gochain/testnet/ethereum/ropsten/localhost. (default: "gochain") [$WEB3_NETWORK]
   --testnet                  Shorthand for '-network testnet'.
   --rpc-url value            The network RPC URL [$WEB3_RPC_URL]
   --verbose                  Enable verbose logging
   --format value, -f value   Output format. Options: json. Default: human readable output.
   --help, -h                 show help
   --version, -v              print the version
```


## Install web3

Quick one line install:

```sh
curl -LSs https://raw.githubusercontent.com/gochain/web3/master/install.sh | sh
```

[Install Docker](https://docs.docker.com/install/) (optional) - not required for all commands, but if you plan on building and deploying smart contracts, 
you'll need Docker installed.

[More installation options](#More-installation-options)

## Quickstart

If you just plan to read from the blockchain, you do not need any GO tokens and you do not need to set your `WEB3_PRIVATE_KEY`. If you plan to deploy contracts or write anything to the blockchain, you'll need tokens and you'll need to set your `WEB3_PRIVATE_KEY` for the account that has those tokens.

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

To do any write operations, you'll need some testnet GO. You can get some at https://faucet.gochain.io/ or ask in [GoChain Developers Telegram Group](https://t.me/gochain_testnet). 

#### c) Use the GoChain mainnet or another web3 network

```sh
export WEB3_NETWORK=gochain
```

You'll need mainnet GO for this which you can [buy on various exchanges](https://gochain.io/go).

#### d) Ethereum or any other web3 compatible network

Most people use Infura for Ethereum which requires an API key to use. Sign in to [Infura](https://infura.io), create a project, click the settings tab and find your unique mainnet RPC URL under "ENDPOINTS". Then set web3 to use it with:

```sh
export WEB3_RPC_URL=https://mainnet.infura.io/v3/YOURUNIQUEKEY
```

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

you could also verify it in the block explorer after deployment
```sh
web3 contract deploy --verify hello_flatten.sol Hello.bin
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

## Generating Common Contracts

web3 includes some of the most common contracts so you can generate and deploy things like a token contract (ERC20)
or a collectible contract (ERC721) in seconds. The generated contract uses [OpenZeppelin](https://openzeppelin.org/) contracts so you can be sure these are secure and industry standard.

Generate an ERC20 contract:

```sh
web3 generate contract erc20 --name "Test Tokens" --symbol TEST
```

That's it! Now you can literally just deploy it and be done. Or open the generated code to see what was generated and modify it to your liking. To see all the available options for generating an ERC20 contract, use `web3 generate contract erc20 --help`

Generate an ERC721 contract:

```sh
web3 generate contract erc721 --name "Kitties" --symbol CAT
```

To see all the available options for generating an ERC721 contract, use `web3 generate contract erc721 --help`

## Deploying an Upgradeable Contract

The `web3` tool comes with built-in support for deploying contracts that can be
upgraded later. To deploy an upgradeable contract, simply specify the
`--upgradeable` flag while deploying. From our `Hello` example above:

```sh
web3 contract deploy --upgradeable Hello.bin
```

This will return the contract address. Let's set the contract address environment variable so you can use it throughout the rest of this
tutorial (alternatively you can pass in the `--address CONTRACT_ADDRESS` flag on all the commands).

```sh
export WEB3_ADDRESS=0xCONTRACT_ADDRESS
```

Internally, deploying an upgradeable contract will actually deploy two separate contracts:

1. Your original `Hello` contract.
2. A proxy contract for redirecting calls and storage.

The returned contract address is the address of your proxy. To see the contract
address that your proxy is pointing to, you can use the `target` command in
the CLI:

```sh
web3 contract target
```

One caveat to using upgradeable contracts is that their constructors will not
execute. To get around this, we will have to initialize our contract with an 
initial call to `setName`:

```sh
web3 contract call --abi Hello.abi --function setName "World"
```

Now we can interact with our upgradeable contract just like a normal contract:

```sh
web3 contract call --abi Hello.abi --function hello
# returns: [Hello World]
```

Alright, so we have a working contract. Let's upgrade it!

### Upgrading the contract

We can now deploy a different contract (without the `upgradeable` flag) and 
redirect our upgradeable contract to point to that new contract.

Copy [contracts/goodbye.sol](contracts/goodbye.sol) into your current directory
and build and deploy it:

```sh
web3 contract build goodbye.sol
web3 contract deploy Goodbye.bin
```

Using the new `Goodbye` contract address, we can upgrade our previous contract
using the `contract upgrade` command:

```sh
web3 contract upgrade --to 0xGOODBYE_CONTRACT_ADDRESS
```

We can see that our proxy contract now points to this new contract by
calling the `hello` function again:

```sh
web3 contract call --abi Hello.abi --function hello
# returns: [Goodbye World]
```

Note that contracts can only be upgraded by the account that created them.

### Pausing and resuming a contract

Upgradeable contracts also include the ability to pause & resume execution.
This can be useful if you discover a bug in your contract and you wish to cease
operation until you can upgrade to a fixed version.

Pausing a contract is simple:

```sh
web3 contract pause
```

Wait a minute for the transaction to go through, then try to use the contract again and it will fail:

```sh
web3 contract call --abi Hello.abi --function hello
# returns: ERROR: Cannot call the contract: abi: unmarshalling empty output
```

Contracts can be upgraded while they are paused. To execute any other contract functions, you
will need to first resume operation:

```sh
web3 contract resume
```

## The Most Common Available commands

### Global parameters

#### Choosing a network

To choose a network, you can either set `WEB3_NETWORK` or `WEB3_RPC_URL` environment variables or pass it in explicitly
on each command with the `--network` or `--rpc-url` flag.

Available name networks are:

* gochain (default)
* testnet
* ethereum
* ropsten
* localhost

The RPC URL is a full URL to a host, for eg: `https://rpc.gochain.io` or `http://localhost:8545`

#### Setting your private key

Set your private key in the environment so it can be used in all the commands below:

```sh
export WEB3_PRIVATE_KEY=0xKEY
```

### Check balance

```sh
web3 balance
```

### Transfer tokens

```sh
web3 transfer 0.1 to 0x67683dd2a499E765BCBE0035439345f48996892f
```

### Get transaction details

```sh
web3 tx TX_HASH
```

### Build a smart contract

```sh
web3 contract build FILENAME.sol --solc-version SOLC_VERSION
```

**Parameters:**

* FILENAME - the name of the .sol file, eg: `hello.sol`
* SOLC_VERSION - the version of the solc compiler

### Flatten a smart contract

Sometimes to verify a contract you have to flatten it before.

```sh
web3 contract flatten FILENAME.sol -o OUTPUT_FILE
```

**Parameters:**

* FILENAME - the name of the .sol file, eg: `hello.sol`

* OUTPUT_FILE (optional) - the output file

### Deploy a smart contract to a network

```sh
web3 contract deploy FILENAME.bin
```

**Parameters:**

* FILENAME - the name of the .bin

### Call a function of a deployed contract

Note: you can set `WEB3_ADDRESS=0xCONTRACT_ADDRESS` environment variable to skip the `--address` flag in the commands below.

```sh
web3 contract call --amount AMOUNT --address CONTRACT_ADDRESS --abi CONTRACT_ABI_FILE --function FUNCTION_NAME FUNCTION_PARAMETERS
```

or using bundled abi files

```sh
web3 contract call --amount AMOUNT --address CONTRACT_ADDRESS --abi erc20|erc721 --function FUNCTION_NAME FUNCTION_PARAMETERS
```

**Parameters:**

* CONTRACT_ADDRESS - the address of the deployed contract
* CONTRACT_ABI_FILE - the abi file of the deployed contract (take into account that there are some bundled abi files like erc20 and erc721 so you could use them without downloading or compiling them)
* FUNCTION_NAME - the name of the function you want to call
* FUNCTION_PARAMETERS - the list of the function parameters
* AMOUNT - amount of wei to be send with transaction (require only for paid transact functions)

### List functions in an ABI

```sh
web3 contract list --abi CONTRACT_ABI_FILE
```

**Parameters:**

* CONTRACT_ABI_FILE - the abi file of the compiled contract

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

### Show information about a block

```sh
web3 block BLOCK_ID
```

**Parameters:**

- BLOCK_ID - id of a block (omit for `latest`)

### Show information about an address

```sj
web3 transaction ADDRESS_HASH
```

**Parameters:**

* ADDRESS_HASH - hash of the address

### Verify a smart contract to a block explorer

```sh
web3 contract verify --explorer-api EXPLORER_API_URL --address CONTRACT_ADDRESS  --contract-name CONTRACT_NAME FILENAME.sol
```

**Parameters:**

* EXPLORER_API_URL - URL for block explorer API (eg https://testnet-explorer.gochain.io/api) - Optional for GoChain networks, which use `{testnet-}explorer.gochain.io` by default.
* CONTRACT_ADDRESS - address of a deployed contract
* CONTRACT_NAME - name of a deployed contract
* FILENAME - the name of the .sol file with a contract source

## More installation options

### Install a specific version

You can use the script to install a specific version:

```sh
curl -LSs https://raw.githubusercontent.com/gochain/web3/master/install.sh | sh -s v0.0.9
```

### Install using the Go language

```sh
go install github.com/gochain/web3/cmd/web3
```

### Build from source

Clone this repo:

```sh
git clone https://github.com/gochain/web3
cd web3
make install
# or just `make build` to build it into current directory
web3 help
```
