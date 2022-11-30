package web3_actions

import (
	"context"
	"fmt"
	"math/big"

	"github.com/rs/zerolog/log"
)

func (w *Web3Actions) GetAndSetChainID(ctx context.Context) error {
	w.Dial()
	defer w.Close()
	chainID, err := w.GetChainID(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Transfer: GetChainID")
		return fmt.Errorf("couldn't get chain ID: %v", err)
	}
	w.SetChainID(chainID)
	return err
}

func (w *Web3Actions) TransferToken(ctx context.Context, payload SendContractTxPayload, wait bool, timeoutInSeconds uint64) error {
	w.Dial()
	defer w.Close()
	err := w.GetAndSetChainID(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Web3Actions: GetAndSetChainID")
		return err
	}
	err = w.SetGasPriceAndLimit(ctx, &payload.GasPriceLimits)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Web3Actions: Transfer: SetGasPriceAndLimit")
		return err
	}
	if payload.SmartContractAddr != "" {
		payload.MethodName = Transfer
		return w.transferToken(ctx, &payload, wait, timeoutInSeconds)
	}
	return err
}

func (w *Web3Actions) transferToken(ctx context.Context, payload *SendContractTxPayload, wait bool, timeoutInSeconds uint64) error {
	payload.ContractFile = ERC20
	payload.MethodName = Transfer
	payload.Params = []interface{}{payload.ToAddress, payload.Amount}
	payload.Amount = &big.Int{}
	err := w.CallContract(ctx, payload, wait, nil, timeoutInSeconds)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Transfer: CallContract")
		return err
	}
	return err

}
