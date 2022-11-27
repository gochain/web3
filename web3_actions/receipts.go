package web3_actions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/rs/zerolog/log"
	"github.com/zeus-fyi/gochain/web3/client"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

var NotFoundErr = errors.New("not found")

func GetTransactionReceipt(ctx context.Context, rpcURL, txhash, contractFile string) error {
	var myabi *abi.ABI
	client, err := client.Dial(rpcURL)
	if err != nil {
		err = fmt.Errorf("failed to connect to %q: %v", rpcURL, err)
		log.Ctx(ctx).Err(err).Msg("GetTransactionReceipt: Dial")
		return err
	}
	defer client.Close()
	if contractFile != "" {
		myabi, err = web3_types.GetABI(contractFile)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("GetTransactionReceipt: GetABI")
			return err
		}
	}
	r, err := client.GetTransactionReceipt(ctx, common.HexToHash(txhash))
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
func WaitForReceipt(ctx context.Context, client client.Client, hash common.Hash) (*web3_types.Receipt, error) {
	for {
		receipt, err := client.GetTransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if err != NotFoundErr {
			log.Ctx(ctx).Err(err).Msg("WaitForReceipt")
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
