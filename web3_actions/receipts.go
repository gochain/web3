package web3_actions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/rs/zerolog/log"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

var NotFoundErr = errors.New("not found")

func (w *Web3Actions) GetTxReceipt(ctx context.Context, txhash, contractFile string) error {
	var myabi *abi.ABI
	w.Dial()
	defer w.Close()
	if contractFile != "" {
		myabiFetched, err := web3_types.GetABI(contractFile)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("GetTxReceipt: GetABI")
			return err
		}
		myabi = myabiFetched
	}
	r, err := w.GetTransactionReceipt(ctx, common.HexToHash(txhash))
	if err != nil {
		err = fmt.Errorf("failed to get transaction receipt: %v", err)
		log.Ctx(ctx).Err(err).Msg("GetTransactionReceipt: GetTransactionReceipt")
		return err
	}
	if verbose {
		fmt.Println("Transaction Receipt Details:")
	}

	err = PrintReceiptDetails(ctx, r, myabi)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("GetTransactionReceipt: PrintReceiptDetails")
		return err
	}
	return err
}

// WaitForReceipt polls for a transaction receipt until it is available, or ctx is cancelled.
func (w *Web3Actions) WaitForReceipt(ctx context.Context, hash common.Hash) (*web3_types.Receipt, error) {
	w.Dial()
	defer w.Close()
	for {
		receipt, err := w.GetTransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if err != NotFoundErr {
			log.Ctx(ctx).Err(err).Msg("WaitForTxReceipt")
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
