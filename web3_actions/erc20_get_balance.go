package web3_actions

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func (w *Web3Actions) ReadERC20TokenBalance(ctx context.Context, contractAddress, addrHash string) (*big.Int, error) {
	w.Dial()
	defer w.Close()
	payload := SendContractTxPayload{
		SmartContractAddr: contractAddress,
		ContractFile:      ERC20,
		SendEtherPayload:  SendEtherPayload{},
		MethodName:        Decimals,
	}
	payload.MethodName = BalanceOf
	addrString := common.HexToAddress(addrHash).String()
	payload.Params = []interface{}{addrString}
	balance, err := w.GetContractConst(ctx, &payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("ReadERC20TokenBalance")
		return new(big.Int), err
	}
	return balance[0].(*big.Int), err
}

func (w *Web3Actions) ReadERC20TokenDecimals(ctx context.Context, payload SendContractTxPayload) (int32, error) {
	w.Dial()
	defer w.Close()
	payload.Params = []interface{}{}
	decimals, err := w.GetContractConst(ctx, &payload)
	if err != nil {
		return 0, err
	}
	return int32(decimals[0].(uint8)), err
}
