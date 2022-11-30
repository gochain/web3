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
