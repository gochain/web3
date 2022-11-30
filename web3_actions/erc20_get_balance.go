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
		SendEtherPayload:  SendEtherPayload{},
		MethodName:        Decimals,
	}
	decimals, err := w.ReadERC20TokenDecimals(ctx, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("ReadERC20TokenBalance")
		return decimal.Decimal{}, err
	}
	// todo: could get symbol here to display
	payload.MethodName = BalanceOf
	payload.Params = []interface{}{addrHash}
	balance, err := w.GetContractConst(ctx, &payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("ReadERC20TokenBalance")
		return decimal.Decimal{}, err
	}
	fmt.Println(web3_types.IntToDec(balance[0].(*big.Int), decimals))

	log.Ctx(ctx).Info().Msgf("BALANCE:", web3_types.IntToDec(balance[0].(*big.Int), decimals))
	return web3_types.IntToDec(balance[0].(*big.Int), decimals), err
}

func (w *Web3Actions) ReadERC20TokenDecimals(ctx context.Context, payload SendContractTxPayload) (int32, error) {
	payload.Params = []interface{}{}
	decimals, err := w.GetContractConst(ctx, &payload)
	if err != nil {
		return 0, err
	}
	return int32(decimals[0].(uint8)), err
}
