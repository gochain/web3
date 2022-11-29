package web3_actions

import (
	"context"
	"fmt"
	"math/big"

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

	// checks that gas limit is not less than gas price, else makes them equal
	glBigInt := big.Int{}
	glBigInt.SetUint64(params.GasLimit)
	b := big.Int{}
	diff := b.Sub(&glBigInt, params.GasPrice)
	if diff.Sign() == -1 {
		glBigInt.SetUint64(params.GasPrice.Uint64())
		params.GasLimit = glBigInt.Uint64()
	}
	return nil
}
