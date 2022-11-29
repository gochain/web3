package web3_actions

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/rs/zerolog/log"
	"github.com/zeus-fyi/gochain/web3/assets"
)

func (w *Web3Actions) PauseContract(ctx context.Context, contractAddress string, amount *big.Int, timeoutInSeconds uint64) error {
	w.Dial()
	defer w.Close()
	err := w.GetAndSetChainID(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Web3Actions: GetAndSetChainID")
		return err
	}
	myabi, err := abi.JSON(strings.NewReader(assets.UpgradeableProxyABI))
	if err != nil {
		err = errors.New("cannot initialize ABI")
		log.Ctx(ctx).Err(err).Msg("Web3Actions: PauseContract")
		return err
	}
	gp := GasPriceLimits{
		GasPrice: nil,
		GasLimit: 70000,
	}
	payload := SendContractTxPayload{
		SmartContractAddr: contractAddress,
		MethodName:        Pause,
		SendTxPayload: SendTxPayload{
			TransferArgs: TransferArgs{
				Amount:    amount,
				ToAddress: common.Address{},
			},
			GasPriceLimits: gp,
		},
	}
	tx, err := w.CallTransactFunction(ctx, myabi, payload)
	if err != nil {
		err = errors.New("cannot pause the contract")
		log.Ctx(ctx).Err(err).Msg("Web3Actions: PauseContract")
	}
	ctx, cancelFn := context.WithTimeout(ctx, time.Duration(timeoutInSeconds)*time.Second)
	defer cancelFn()
	receipt, err := w.WaitForReceipt(ctx, tx.Hash)
	if err != nil {
		err = errors.New("cannot get the receipt for transaction with hash")
		log.Ctx(ctx).Err(err).Interface("txHash", tx.Hash).Msg("Web3Actions: WaitForReceipt")
	}
	fmt.Println("Transaction address:", receipt.TxHash.Hex())
	return err
}
