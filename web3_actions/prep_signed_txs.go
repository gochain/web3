package web3_actions

import (
	"context"
	"fmt"

	"github.com/gochain/gochain/v4/core/types"
	"github.com/rs/zerolog/log"
)

func (w *Web3Actions) GetSignedSendTx(ctx context.Context, params SendEtherPayload) (*types.Transaction, error) {
	w.Dial()
	defer w.Close()
	err := w.SetGasPriceAndLimit(ctx, &params.GasPriceLimits)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Send: SignTx")
		return nil, fmt.Errorf("cannot sign transaction: %v", err)
	}
	nonce, err := w.GetNonce(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Send: GetNonce")
		return nil, err
	}
	chainID, err := w.GetChainID(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Send: GetChainID")
		return nil, fmt.Errorf("couldn't get chain ID: %v", err)
	}
	log.Ctx(ctx).Info().Interface("chainID", chainID).Msg("GetSignedSendTx")
	tx := types.NewTransaction(nonce, params.ToAddress, params.Amount, params.GasLimit, params.GasPrice, nil)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), w.EcdsaPrivateKey())
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Send: SignTx")
		return nil, fmt.Errorf("cannot sign transaction: %v", err)
	}
	return signedTx, err
}
