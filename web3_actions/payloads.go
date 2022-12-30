package web3_actions

import (
	"math/big"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
)

type SendContractTxPayload struct {
	SmartContractAddr string
	SendEtherPayload  // payable would be an amount, otherwise for tokens use the params field
	ContractFile      string
	ContractABI       *abi.ABI // this has first priority, if nil will check default contracts using contract file
	MethodName        string   // name of the smart contract function
	Params            []interface{}
}

type SendEtherPayload struct {
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
type CallMsg struct {
	From     *common.Address // the sender of the 'transaction'
	To       *common.Address // the destination contract (nil for contract creation)
	Gas      uint64          // if 0, the call executes with near-infinite gas
	GasPrice *big.Int        // wei <-> gas exchange ratio
	Value    *big.Int        // amount of wei sent along with the call
	Data     []byte          // input data, usually an ABI-encoded contract method invocation
}
