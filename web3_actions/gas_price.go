package web3_actions

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

func (w *Web3Actions) SetGasPriceAndLimit(ctx context.Context, params *GasPriceLimits) error {
	w.Dial()
	defer w.Close()

	if params.GasLimit == 0 {
		params.GasLimit = 21000
	}

	if params.GasPrice == nil || params.GasPrice.Int64() == 0 {
		gasPrice, gerr := w.GetGasPrice(ctx)
		if gerr != nil {
			log.Ctx(ctx).Err(gerr).Msg("Send: GetGasPrice")
			return fmt.Errorf("cannot get gas price: %v", gerr)
		}
		params.GasPrice = gasPrice
	}
	return nil
}
