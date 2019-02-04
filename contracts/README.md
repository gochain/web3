# Quickstart

First, [get yourself some GO testnet tokens](https://help.gochain.io/en/article/getting-started-4tlo7a/) so you can deploy and interact with your contract.

Set `WEB3_PRIVATE_KEY` and `WEB3_NETWORK` env vars:

```sh
export WEB3_NETWORK=testnet
export WEB3_PRIVATE_KEY=0x...
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
web3 contract call --address 0xCONTRACT_ADDRESS --abi HelloWorld.abi --function hello
```

That should return: `[Hello World]`.

Now let's change the name:

```sh
web3 contract call --address 0xCONTRACT_ADDRESS --abi HelloWorld.abi --function setName "Johnny"
```

And call the hello function again to see if the name changed:

```sh
web3 contract call --address 0xCONTRACT_ADDRESS --abi HelloWorld.abi --function hello
```

Now it should return `[Hello Johnny]`

:boom:

## Troubleshooting

If it doesn't return Hello Johnny, you can check the logs and receipt with:

```sh
web3 rc TX_HASH
```
