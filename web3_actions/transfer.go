package web3_actions

import (
	"context"
	"fmt"
	"math/big"

	"github.com/gochain/gochain/v4/common"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"github.com/zeus-fyi/gochain/web3/types"
)

func (w *Web3Actions) Transfer(ctx context.Context, chainID *big.Int, contractAddress string, gasPrice *big.Int, gasLimit uint64, wait, toString bool, timeoutInSeconds uint64, tail []string) error {
	w.Dial()
	defer w.Close()
	amountD, toAddress, err := convertTailForTransfer(ctx, tail)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Web3Actions: convertTailForTransfer")
		return err
	}
	w.SetChainID(chainID)
	if contractAddress != "" {
		return w.transferToContract(ctx, contractAddress, toAddress, amountD, wait, toString, timeoutInSeconds)
	}
	amount := web3_types.DecToInt(amountD, 18)
	err = ValidateToAddress(ctx, toAddress)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Transfer: ValidateToAddress")
		return err
	}
	address := common.HexToAddress(toAddress)
	params := constructSendEtherPayload(amount, address, gasPrice, gasLimit)
	tx, err := w.Send(ctx, params)
	if err != nil {
		err = fmt.Errorf("cannot create transaction: %v", err)
		log.Ctx(ctx).Err(err).Msg("Transfer: Send")
		return err
	}
	log.Ctx(ctx).Info().Interface("txHash", tx.Hash.Hex()).Msg("Transfer: txHash")
	return err
}

func (w *Web3Actions) transferToContract(ctx context.Context, contractAddress, toAddress string, amountD decimal.Decimal, wait, toString bool, timeoutInSeconds uint64) error {
	decimals, derr := w.getContractDecimals(ctx, contractAddress)
	if derr != nil {
		log.Ctx(ctx).Err(derr).Msg("Web3Actions: transferToContract")
		return derr
	}
	relativeAmount := web3_types.DecToInt(amountD, decimals)
	err := w.CallContract(ctx, contractAddress, "erc20", "transfer", &big.Int{}, nil, 70000, wait, toString, nil, timeoutInSeconds, toAddress, relativeAmount)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Transfer: CallContract")
		return err
	}
	return err

}
