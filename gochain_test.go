package web3

import (
	"context"
	"fmt"
	"math/big"
)

func ExampleRPCClient_GetBlockByNumber() {
	ctx := context.Background()
	for _, network := range []string{"mainnet", "testnet"} {
		c := GetClient(NetworkURL(network))

		bl, err := c.GetBlockByNumber(ctx, nil)
		if err != nil {
			fmt.Printf("Failed to get latest block: %v", err)
		}
		if bl == nil {
			fmt.Println("Latest block nil.")
		} else {
			fmt.Println("Got latest block.")
		}

		bl, err = c.GetBlockByNumber(ctx, big.NewInt(0))
		if err != nil {
			fmt.Printf("Failed to get genesis block: %v", err)
		}
		if bl == nil {
			fmt.Println("Genesis block nil.")
		} else {
			fmt.Println("Got genesis block.")
		}

		sn, err := c.GetSnapshot(ctx)
		if err != nil {
			fmt.Printf("Failed to get snapshot: %v\n", err)
		}
		if sn == nil {
			fmt.Println("Latest snapshot nil.")
		} else {
			fmt.Println("Got latest snapshot.")
		}

		initAlloc, ok := new(big.Int).SetString("1000000000000000000000000000", 10)
		if !ok {
			panic("failed to parse big.Int string")
		}
		bal, err := c.GetBalance(ctx, testAddr(network), big.NewInt(0))
		if err != nil {
			fmt.Printf("Failed to get balance: %v\n", err)
		}
		if bal == nil {
			fmt.Println("Initial alloc balance nil.")
		} else if bal.Cmp(initAlloc) != 0 {
			fmt.Printf("Unexpected initial alloc balance %s, wanted %s\n", bal, initAlloc)
		} else {
			fmt.Println("Got initial alloc balance.")
		}
	}
	// Output:
	// Got latest block.
	// Got genesis block.
	// Got latest snapshot.
	// Got initial alloc balance.
	// Got latest block.
	// Got genesis block.
	// Got latest snapshot.
	// Got initial alloc balance.
}

func testAddr(network string) string {
	switch network {
	case "mainnet":
		return "0xf75b6e2d2d69da07f2940e239e25229350f8103f"
	case "testnet":
		return "0x2fe70f1df222c85ad6dd24a3376eb5ac32136978"
	default:
		panic("unsupported network: " + network)
	}
}
