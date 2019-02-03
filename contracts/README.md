# Quickstart

Get some Testnet tokens [TODO: LINK TO HOW TO GET TOKENS PAGE, should have telegram, create wallet on explorer wallet, /send tokens from tg to new address].

Set `PRIVATE_KEY` and `NETWORK` env vars:

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

Let's call a read function (which is free):

```sh
web3 contract call --contract ADDRESS --contract-abi HelloWorld.abi --function hello
```

That should return: `[Hello World]`.

Now let's change the name:

```sh
web3 contract call --contract 0x633a073E3C8c809b484585C97df10Cf879F2c66b --contract-abi HelloWorld.abi --function setName "Johnny"
```

And call the hello function again to see if the name changed:

```sh
web3 contract call --contract 0x633a073E3C8c809b484585C97df10Cf879F2c66b --contract-abi HelloWorld.abi --function hello
```

Now it should return `[Hello Johnny]`

:boom:

## Troubleshooting

If it doesn't return Hello Johnny, you can check the logs and receipt with:

```sh
web3 rc TX_HASH
```
