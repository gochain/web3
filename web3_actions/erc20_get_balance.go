package web3_actions

import (
	"context"
	"fmt"
	"math/big"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

func (w *Web3Actions) ReadERC20TokenBalance(ctx context.Context, contractAddress, addrHash string) (decimal.Decimal, error) {

	payload := SendContractTxPayload{
		SmartContractAddr: contractAddress,
		ContractFile:      ERC20,
		SendTxPayload:     SendTxPayload{},
		MethodName:        Decimals,
		Params:            []interface{}{addrHash},
	}
	decimals, err := w.GetContractConst(ctx, payload)
	if err != nil {
		return decimal.Decimal{}, err
	}
	// fmt.Println("DECIMALS:", decimals, reflect.TypeOf(decimals))
	// todo: could get symbol here to display
	payload.MethodName = BalanceOf
	balance, err := w.GetContractConst(ctx, payload)
	if err != nil {
		return decimal.Decimal{}, err
	}
	// fmt.Println("BALANCE:", balance, reflect.TypeOf(balance))
	fmt.Println(web3_types.IntToDec(balance[0].(*big.Int), int32(decimals[0].(uint8))))

	log.Ctx(ctx).Info().Msgf("BALANCE:", web3_types.IntToDec(balance[0].(*big.Int), int32(decimals[0].(uint8))))
	return web3_types.IntToDec(balance[0].(*big.Int), int32(decimals[0].(uint8))), err
}
