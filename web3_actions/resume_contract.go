package web3_actions

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/gochain/gochain/v4/common"
	"github.com/rs/zerolog/log"
)

func (w *Web3Actions) ResumeContract(ctx context.Context, contractAddress string, amount *big.Int, timeoutInSeconds uint64) error {
	w.Dial()
	defer w.Close()
	err := w.GetAndSetChainID(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Web3Actions: GetAndSetChainID")
		return err
	}
	gp := GasPriceLimits{
		GasPrice: nil,
		GasLimit: 70000,
	}
	payload := SendContractTxPayload{
		SmartContractAddr: contractAddress,
		MethodName:        Resume,
		SendEtherPayload: SendEtherPayload{
			TransferArgs: TransferArgs{
				Amount:    amount,
				ToAddress: common.Address{},
			},
			GasPriceLimits: gp,
		},
	}
	tx, err := w.CallTransactFunction(ctx, &payload)
	if err != nil {
		err = errors.New("cannot resume the contract")
		log.Ctx(ctx).Err(err).Msg("Web3Actions: ResumeContract")
	}
	ctx, cancelFn := context.WithTimeout(ctx, time.Duration(timeoutInSeconds)*time.Second)
	defer cancelFn()
	receipt, err := w.WaitForReceipt(ctx, tx.Hash)
	if err != nil {
		err = errors.New("cannot get the receipt for transaction with hash")
		log.Ctx(ctx).Err(err).Interface("txHash", tx.Hash).Msg("Web3Actions: WaitForReceipt")
		return err
	}
	log.Ctx(ctx).Info().Msgf("Web3Actions: ResumeContract: Transaction address: %s", receipt.TxHash.Hex())
	return err
}
