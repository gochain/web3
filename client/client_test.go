package web3_client

import (
	"context"
	"fmt"
	"math/big"
)

func ExampleRPCClient_GetBlockByNumber() {
	for _, network := range []string{MainnetURL, TestnetURL} {
		exampleRPCClient_GetBlockByNumber(network)
	}
	// Output:
	// Got ID.
	// Got latest block.
	// Got genesis block.
	// Got latest snapshot.
	// Got initial alloc balance.
	// Got ID.
	// Got latest block.
	// Got genesis block.
	// Got latest snapshot.
	// Got initial alloc balance.
}

func exampleRPCClient_GetBlockByNumber(url string) {
	c, err := Dial(url)
	if err != nil {
		fmt.Printf("Failed to connect to network %q: %v\n", url, err)
		return
	}
	defer c.Close()

	ctx := context.Background()
	id, err := c.GetID(ctx)
	if err != nil {
		fmt.Printf("Failed to get id: %v\n", err)
	}
	if id == nil {
		fmt.Println("ID nil.")
	} else {
		fmt.Println("Got ID.")
	}

	bl, err := c.GetBlockByNumber(ctx, nil, false)
	if err != nil {
		fmt.Printf("Failed to get latest block: %v", err)
	}
	if bl == nil {
		fmt.Println("Latest block nil.")
	} else {
		fmt.Println("Got latest block.")
	}

	bl, err = c.GetBlockByNumber(ctx, big.NewInt(0), false)
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
	bal, err := c.GetBalance(ctx, testAddr(url), big.NewInt(0))
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

func testAddr(network string) string {
	switch network {
	case MainnetURL:
		return "0xf75b6e2d2d69da07f2940e239e25229350f8103f"
	case TestnetURL:
		return "0x2fe70f1df222c85ad6dd24a3376eb5ac32136978"
	default:
		panic("unsupported network: " + network)
	}
}
