package web3_actions

import (
	"math/big"

	"github.com/gochain/gochain/v4/common"
)

type SendContractTxPayload struct {
	SmartContractAddr string
	SendTxPayload
	ContractFile string
	MethodName   string
	Params       []interface{}
}

type SendTxPayload struct {
	TransferArgs
	GasPriceLimits
}

type TransferArgs struct {
	Amount    *big.Int
	ToAddress common.Address
}

type GasPriceLimits struct {
	GasPrice *big.Int
	GasLimit uint64
}
