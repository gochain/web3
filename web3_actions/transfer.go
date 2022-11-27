package web3_actions

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/gochain/gochain/v4/common"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"github.com/zeus-fyi/gochain/web3/types"
)

func (w *Web3Actions) Transfer(ctx context.Context, chainID *big.Int, contractAddress string, gasPrice *big.Int, gasLimit uint64, wait, toString bool, timeoutInSeconds uint64, tail []string) error {
	if len(tail) < 3 {
		err := errors.New("invalid arguments. format is: `transfer X to ADDRESS`")
		log.Ctx(ctx).Err(err).Msg("ReplaceTx: SendTransaction")
		return err
	}
	w.Dial()
	defer w.Close()
	amountS := tail[0]
	amountD, err := decimal.NewFromString(amountS)
	if err != nil {
		err = fmt.Errorf("invalid amount %v", amountS)
		log.Ctx(ctx).Err(err).Msg("Transfer: decimal.NewFromString")
		return err
	}
	toAddress := tail[2]
	w.SetChainID(chainID)

	if contractAddress != "" {
		decimals, derr := w.GetContractConst(ctx, contractAddress, "erc20", "decimals")
		if derr != nil {
			log.Ctx(ctx).Err(derr).Msg("Transfer: GetContractConst")
			return derr
		}
		amount := web3_types.DecToInt(amountD, int32(decimals[0].(uint8)))
		err = w.CallContract(ctx, contractAddress, "erc20", "transfer", &big.Int{}, nil, 70000, wait, toString, nil, timeoutInSeconds, toAddress, amount)
		if err != nil {
			log.Ctx(ctx).Err(derr).Msg("Transfer: CallContract")
			return err
		}
		return err
	}

	amount := web3_types.DecToInt(amountD, 18)
	if toAddress == "" {
		err = errors.New("the recipient address cannot be empty")
		log.Ctx(ctx).Err(err).Msg("Transfer: toAddress")
		return err
	}
	if !common.IsHexAddress(toAddress) {
		err = fmt.Errorf("invalid to 'address': %s", toAddress)
		log.Ctx(ctx).Err(err).Msg("Transfer: IsHexAddress")
		return err
	}
	address := common.HexToAddress(toAddress)
	params := SendEtherPayload{
		Amount:    amount,
		ToAddress: address,
		GasPriceLimits: GasPriceLimits{
			GasPrice: gasPrice,
			GasLimit: gasLimit,
		},
	}
	tx, err := w.Send(ctx, params)
	if err != nil {
		err = fmt.Errorf("cannot create transaction: %v", err)
		log.Ctx(ctx).Err(err).Msg("Transfer: Send")
		return err
	}
	fmt.Println("Transaction address:", tx.Hash.Hex())
	return err
}
