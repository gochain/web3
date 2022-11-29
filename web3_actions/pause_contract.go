package web3_actions

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/rs/zerolog/log"
	"github.com/zeus-fyi/gochain/web3/assets"
)

func (w *Web3Actions) PauseContract(ctx context.Context, chainID *big.Int, contractAddress string, amount *big.Int, timeoutInSeconds uint64) error {
	w.Dial()
	defer w.Close()
	w.SetChainID(chainID)
	myabi, err := abi.JSON(strings.NewReader(assets.UpgradeableProxyABI))
	if err != nil {
		err = errors.New("cannot initialize ABI")
		log.Ctx(ctx).Err(err).Msg("Web3Actions: PauseContract")
		return err
	}
	tx, err := w.CallTransactFunction(ctx, myabi, contractAddress, "pause", amount, nil, 70000)
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
