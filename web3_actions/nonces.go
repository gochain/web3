package web3_actions

import (
	"context"
	"fmt"

	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/crypto"
	"github.com/rs/zerolog/log"
)

func (w *Web3Actions) GetNonce(ctx context.Context) (uint64, error) {
	w.Dial()
	defer w.Close()
	publicKeyECDSA := w.EcdsaPublicKey()
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	return w.GetNonceByPublicAddress(ctx, fromAddress)
}

func (w *Web3Actions) GetNonceByPublicAddress(ctx context.Context, fromAddress common.Address) (uint64, error) {
	w.Dial()
	defer w.Close()
	nonce, err := w.GetPendingTransactionCount(ctx, fromAddress)
	if err != nil {
		log.Ctx(ctx).Err(err).Interface("fromAddress", fromAddress).Msg("Web3Actions: GetNonceByPublicAddress")
		return 0, fmt.Errorf("cannot get nonce: %v", err)
	}
	return nonce, err
}
