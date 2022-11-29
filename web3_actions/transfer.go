package web3_actions

import (
	"context"
	"fmt"

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

func (w *Web3Actions) Transfer(ctx context.Context, payload SendContractTxPayload, wait bool, timeoutInSeconds uint64) error {
	w.Dial()
	defer w.Close()
	err := w.GetAndSetChainID(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Web3Actions: GetAndSetChainID")
		return err
	}
	if payload.SmartContractAddr != "" {
		payload.MethodName = Transfer
		return w.transferToContract(ctx, payload, wait, timeoutInSeconds)
	}
	tx, err := w.Send(ctx, payload.SendTxPayload)
	if err != nil {
		err = fmt.Errorf("cannot create transaction: %v", err)
		log.Ctx(ctx).Err(err).Msg("Transfer: Send")
		return err
	}
	log.Ctx(ctx).Info().Interface("txHash", tx.Hash.Hex()).Msg("Transfer: txHash")
	return err
}

func (w *Web3Actions) transferToContract(ctx context.Context, payload SendContractTxPayload, wait bool, timeoutInSeconds uint64) error {
	payload.ContractFile = ERC20
	payload.MethodName = Transfer
	err := w.CallContract(ctx, payload, wait, nil, timeoutInSeconds)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Transfer: CallContract")
		return err
	}
	return err

}
