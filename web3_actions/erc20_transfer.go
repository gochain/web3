package web3_actions

import (
	"context"
	"math/big"

	"github.com/rs/zerolog/log"
)

func (w *Web3Actions) TransferERC20Token(ctx context.Context, payload SendContractTxPayload, wait bool, timeoutInSeconds uint64) error {
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
	return w.transferToken(ctx, &payload, wait, timeoutInSeconds)
}

// transferToken requires you to place the amounts in the params, payload amount otherwise is payable
func (w *Web3Actions) transferToken(ctx context.Context, payload *SendContractTxPayload, wait bool, timeoutInSeconds uint64) error {
	payload.ContractFile = ERC20
	payload.MethodName = Transfer
	payload.Amount = &big.Int{}
	err := w.CallContract(ctx, payload, wait, nil, timeoutInSeconds)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Transfer: CallContract")
		return err
	}
	return err
}
