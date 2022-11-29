package web3_actions

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/rs/zerolog/log"
	"github.com/zeus-fyi/gochain/web3/assets"
)

func (w *Web3Actions) ResumeContract(ctx context.Context, chainID *big.Int, contractAddress string, amount *big.Int, timeoutInSeconds uint64) error {
	w.Dial()
	defer w.Close()
	w.SetChainID(chainID)
	myabi, err := abi.JSON(strings.NewReader(assets.UpgradeableProxyABI))
	if err != nil {
		err = errors.New("cannot initialize ABI")
		log.Ctx(ctx).Err(err).Msg("Web3Actions: ResumeContract")
		return err
	}
	tx, err := w.CallTransactFunction(ctx, myabi, contractAddress, "resume", amount, nil, 70000)
	if err != nil {
		err = errors.New("cannot resume the contract")
		log.Ctx(ctx).Err(err).Msg("Web3Actions: ResumeContract")
	}
	ctx, cancelFn := context.WithTimeout(ctx, time.Duration(timeoutInSeconds)*time.Second)
	defer cancelFn()
	receipt, err := w.WaitForReceipt(ctx, tx.Hash)
	if err != nil {
		err = errors.New("cannot get the receipt for transaction with hash")
		log.Ctx(ctx).Err(err).Interface("txHash", tx.Hash).Msg("Web3Actions: WaitForReceipt")
		return err
	}
	log.Ctx(ctx).Info().Msgf("Web3Actions: ResumeContract: Transaction address: %s", receipt.TxHash.Hex())
	return err
}
