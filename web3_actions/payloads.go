package web3_actions

import (
	"math/big"

	"github.com/gochain/gochain/v4/common"
)

type SendEtherPayload struct {
	Amount    *big.Int
	ToAddress common.Address
	GasPriceLimits
}

type GasPriceLimits struct {
	GasPrice *big.Int
	GasLimit uint64
}
