    ██╗    ██╗███████╗██████╗ ██████╗      ██████╗██╗     ██╗
    ██║    ██║██╔════╝██╔══██╗╚════██╗    ██╔════╝██║     ██║
    ██║ █╗ ██║█████╗  ██████╔╝ █████╔╝    ██║     ██║     ██║
    ██║███╗██║██╔══╝  ██╔══██╗ ╚═══██╗    ██║     ██║     ██║
    ╚███╔███╔╝███████╗██████╔╝██████╔╝    ╚██████╗███████╗██║
     ╚══╝╚══╝ ╚══════╝╚═════╝ ╚═════╝      ╚═════╝╚══════╝╚═╝

Zero-CGO, pure-Go command-line tool and SDK (`package web3`) for interacting with EVM blockchains — GoChain, Ethereum, Base, Arbitrum, Optimism, Polygon, BNB Chain, Avalanche, Sepolia, and more.

[![API Reference](https://pkg.go.dev/badge/github.com/gochain/web3)](https://pkg.go.dev/github.com/gochain/web3)
[![Test](https://github.com/gochain/web3/actions/workflows/test.yml/badge.svg)](https://github.com/gochain/web3/actions/workflows/test.yml)
[![Release](https://github.com/gochain/web3/actions/workflows/release.yml/badge.svg)](https://github.com/gochain/web3/actions/workflows/release.yml)

```sh
web3 --help
NAME:
   web3 - Modern EVM & Web3 command-line tool

USAGE:
   web3 [global options] command [command options] [arguments...]

COMMANDS:
   block, bl        Block details for a block number (decimal) or hash (0x...). Omit for latest.
   transaction, tx  Transaction details for a tx hash
   receipt, rc      Transaction receipt for a tx hash
   address, addr    Account details for an address or WEB3_PRIVATE_KEY
   balance          Get native or ERC20 token balance for an address or WEB3_PRIVATE_KEY
   storage          Read a 32-byte contract storage slot (eth_getStorageAt)
   logs             Query event logs (eth_getLogs) with optional ABI event decoding
   transfer, send   Transfer native coins (GO/ETH) or ERC20 tokens (e.g. `web3 transfer 1.5 to 0xADDRESS`)
   increasegas      Speed up a pending transaction by bumping its gas price / priority fee
   cancel           Cancel a pending transaction by sending a 0-value self-transfer at the same nonce with higher gas
   replace          Replace a pending transaction at a specific nonce
   contract, c      Smart contract build, deploy, call, verify, events, and EIP-1967 proxy operations
   generate, g      Generate OpenZeppelin v5 smart contracts or Go ABI bindings
   account, a       Create accounts and manage Web3 Secret Storage V3 keystores
   myaddress        Print the EIP-55 address derived from WEB3_PRIVATE_KEY
   sign             Sign a message using EIP-191 personal_sign
   verify-sig       Recover signer (and optionally verify address) for an EIP-191 personal_sign signature
   keccak, hash     Compute the Keccak-256 hash of UTF-8 text or --hex bytes
   encode, calldata ABI-encode function calldata offline
   decode           ABI-decode function calldata, return values (--output), or Solidity reverts (--revert)
   units            Convert an amount across wei, gwei, and eth/go
   snapshot, sn     Clique consensus snapshot
   id               Network/Chain ID and genesis hash information
   start            Start a local GoChain development node via Docker
   env              List WEB3_* environment variables
   help, h          Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --network value, -n value  Target network (arbitrum/avalanche/base/base-sepolia/bsc/ethereum/gochain/holesky/localhost/optimism/polygon/sepolia/testnet). Default: "gochain" [$WEB3_NETWORK]
   --testnet                  Shorthand for '--network testnet'
   --rpc-url value            Custom JSON-RPC HTTP(S) or WS(S) URL [$WEB3_RPC_URL]
   --verbose                  Enable verbose logging
   --format value, -f value   Output format: json (default: human-readable)
   --help, -h                 show help
   --version, -v              print the version
```

## Features

- **Pure Go (`CGO_ENABLED=0`)**: Zero full-node dependencies, fast static builds on Linux, macOS, and Windows.
- **EIP-1559 & Legacy Transactions**: Auto-detects EIP-1559 (`Type 2`) chains via `baseFeePerGas` and falls back to EIP-155 (`Type 0`) on legacy networks like GoChain.
- **Automatic Gas Estimation**: Estimates gas via `eth_estimateGas` automatically whenever `--gas-limit` is omitted.
- **Full Solidity ABI Engine**: Supports dynamic tuples/structs, slices, fixed arrays, function overloads, human-readable inline signatures, and `Error(string)` / `Panic(uint256)` / custom Solidity error revert decoding.
- **Standard EIP-1967 Upgradeable Proxies**: Deploy, inspect (`target`, `admin`), upgrade (`upgradeTo` / `upgradeToAndCall`), `pause`, and `resume` standard EIP-1967 proxies recognized by Etherscan and Blockscout.
- **OpenZeppelin v5 Contract Generation**: Scaffold ERC-20, ERC-721, and ERC-1155 contracts in one command.

## Install `web3`

Quick one-line install:

```sh
curl -fsSL https://raw.githubusercontent.com/gochain/web3/master/install.sh | sh
```

Or install with Go (`1.26+`):

```sh
go install github.com/gochain/web3/cmd/web3@latest
```

## Quickstart

### 1. Pick a network

Built-in `--network` (`WEB3_NETWORK`) presets:
- `gochain` (default), `testnet`, `localhost`
- `ethereum`, `sepolia`, `holesky`
- `base`, `base-sepolia`, `arbitrum`, `optimism`, `polygon`, `bsc`, `avalanche`

```sh
# Local development node
web3 start
export WEB3_NETWORK=localhost

# Built-in network preset
export WEB3_NETWORK=base

# Or any custom HTTP(S) / WS(S) RPC endpoint
export WEB3_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
```

### 2. Set Private Key (optional — only needed for signing/writing)

```sh
export WEB3_PRIVATE_KEY=0x...
```

### 3. Build, Deploy & Call a Contract

Copy [hello.sol](contracts/hello.sol) into your working directory, then:

```sh
web3 contract build hello.sol
web3 contract deploy Hello.bin
```

Read from the contract (free `eth_call`):

```sh
web3 contract call --address 0xCONTRACT_ADDRESS --abi Hello.abi --function hello
```

Write to the contract (auto-estimates gas and uses EIP-1559 or Legacy fees automatically):

```sh
web3 contract call --address 0xCONTRACT_ADDRESS --abi Hello.abi --function setName "Johnny" --wait
```

You can also call contracts without an `.abi` file using an inline Solidity signature or bundled ABI (`erc20`, `erc721`, `erc1155`, `proxy`):

```sh
web3 contract call --address 0xUSDC --abi erc20 --function balanceOf 0xUSER_ADDRESS
web3 contract call --address 0xCONTRACT --function "hello():(string,string)"
```

## Generating OpenZeppelin v5 Contracts

Generate an ERC-20 token:

```sh
web3 generate contract erc20 --name "Test Tokens" --symbol TEST --initial-supply 1000000
```

Generate an ERC-721 NFT collection:

```sh
web3 generate contract erc721 --name "Kitties" --symbol CAT
```

Generate an ERC-1155 multi-token contract:

```sh
web3 generate contract erc1155 --symbol ITEMS --uri "https://example.com/metadata/{id}.json"
```

## Deploying an EIP-1967 Upgradeable Contract

Deploy any contract behind a standard EIP-1967 upgradeable proxy with `--upgradeable`:

```sh
web3 contract deploy --upgradeable Hello.bin
export WEB3_ADDRESS=0xPROXY_ADDRESS
```

Inspect, initialize, upgrade, pause, and resume:

```sh
# Read EIP-1967 implementation address (works on any EIP-1967 proxy!)
web3 contract target

# Initialize state through the proxy
web3 contract call --abi Hello.abi --function setName "World" --wait

# Upgrade to a new implementation contract
web3 contract build goodbye.sol
web3 contract deploy Goodbye.bin
web3 contract upgrade --to 0xGOODBYE_IMPLEMENTATION_ADDRESS

# Emergency pause & resume
web3 contract pause
web3 contract resume
```

## Offline Crypto, ABI & Chain Inspection Utilities

```sh
# Sign & verify EIP-191 messages
web3 sign "Hello Web3!"
web3 verify-sig --message "Hello Web3!" --signature 0xSIG --address 0xEXPECTED

# Offline ABI calldata encoding, output decoding & revert decoding
web3 encode "transfer(address,uint256)" 0x70997970C51812dc3A010C7d01b50e0d17dc79C8 1000000
web3 decode "transfer(address,uint256)" 0xa9059cbb...
web3 decode --revert --abi MyContract.abi 0x08c379a0...

# Keccak-256 & unit conversions
web3 keccak "Transfer(address,address,uint256)"
web3 units 1.5eth

# Storage slots, event logs, and pending transaction management
web3 storage 0xCONTRACT_ADDRESS 0
web3 logs --address 0xTOKEN --abi erc20 --event Transfer --from-block 20000000
web3 increasegas --tx 0xPENDING_HASH --amount 2.5
web3 cancel --tx 0xPENDING_HASH
```

## JSON Output Mode

Pass `--format json` (`-f json`) on any command for machine-readable output in scripts and CI:

```sh
web3 --format json balance 0xADDRESS
web3 --format json contract call --address 0xCONTRACT --abi Hello.abi --function hello
```

## Build from Source

```sh
git clone https://github.com/gochain/web3
cd web3
make build
make test
```
