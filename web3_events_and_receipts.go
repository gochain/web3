package web3

import (
	"context"
	"time"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
)

// WaitForReceipt polls for a transaction receipt until it is available, or ctx is cancelled.
func WaitForReceipt(ctx context.Context, client Client, hash common.Hash) (*Receipt, error) {
	for {
		receipt, err := client.GetTransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if err != NotFoundErr {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func FindEventById(abi abi.ABI, id common.Hash) *abi.Event {
	for _, event := range abi.Events {
		if event.ID == id {
			return &event
		}
	}
	return nil
}
