# Quickstart

Get some Testnet tokens [TODO: LINK TO HOW TO GET TOKENS PAGE, should have telegram, create wallet on explorer wallet, /send tokens from tg to new address].

Set `PRIVATE_KEY` and network:

```sh
export NETWORK=testnet
export PRIVATE_KEY=0x...
```

Copy [hello.sol](hello.sol) into your current directory.

Then:

```sh
web3 contract build hello.sol
web3 contract deploy HelloWorld.bin
```

This will return a contract address, copy it and use below.

```sh
web3 contract call --contract 0x633a073E3C8c809b484585C97df10Cf879F2c66b --contract-abi HelloWorld.abi --function hello
```
