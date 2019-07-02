package gost

import (
	"fmt"
	"math/big"

	"github.com/gochain-io/gochain/v3/common"
)

func ExampleTransferEventHash() {
	fmt.Println("TransferEvent ID:", transferEventID.Hex())
	fmt.Println("TransferEventHashes:")
	source := common.HexToAddress("0x1234")
	for _, e := range []struct{ addr, amount string }{
		{"0x1", "1"},
		{"0xF", "10"},
		{"0xA", "1000"},
		{"0x9", "1000000000000000000"},
	} {
		addr := common.HexToAddress(e.addr)
		amount, ok := new(big.Int).SetString(e.amount, 10)
		if !ok {
			fmt.Println("Failed to parse:", e.amount)
			return
		}
		hash, err := TransferEventHash(source, addr, amount)
		if err != nil {
			fmt.Println("Failed to hash:", source, addr.Hex(), e.amount)
		}
		fmt.Printf("%s %s: %s\n", addr.Hex(), e.amount, hash.Hex())
	}

	// Output:
	//TransferEvent ID: 0xb98a26c1d0427d0e3492e749861c7795bed8f6e7599a65143b5903942e611bb0
	//TransferEventHashes:
	//0x0000000000000000000000000000000000000001 1: 0xb0ac7d1bcc67772d396f1ef33b61e468a6af7bba40e9ee94fa3bb0f11762e033
	//0x000000000000000000000000000000000000000F 10: 0xe7c9c57479a7a81885e5d86e214b54bbe5ff61229ddfc03c367163aa34a1ebe5
	//0x000000000000000000000000000000000000000A 1000: 0x8ae9bcae78fe3be388b42965ef220c45cec3891757bbfc4c1f1b003ddd4e0b7c
	//0x0000000000000000000000000000000000000009 1000000000000000000: 0x42d355b46a5cc012e09789a9bd092e436ad2b5b62851c1a548c58dcdb35ac68f
}
