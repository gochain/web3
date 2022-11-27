package web3_actions

import (
	"context"
	"fmt"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/rs/zerolog/log"
	"github.com/zeus-fyi/gochain/web3"
)

func GetTransactionReceipt(ctx context.Context, rpcURL, txhash, contractFile string) error {
	var myabi *abi.ABI
	client, err := web3.Dial(rpcURL)
	if err != nil {
		err = fmt.Errorf("failed to connect to %q: %v", rpcURL, err)
		log.Ctx(ctx).Err(err).Msg("GetTransactionReceipt: Dial")
		return err
	}
	defer client.Close()
	if contractFile != "" {
		myabi, err = web3.GetABI(contractFile)
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
