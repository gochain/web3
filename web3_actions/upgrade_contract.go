package web3_actions

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/rs/zerolog/log"
	"github.com/zeus-fyi/gochain/web3/assets"
)

func (w *Web3Actions) UpgradeContract(ctx context.Context, chainID *big.Int, contractAddress, newTargetAddress string, amount *big.Int, timeoutInSeconds uint64) error {
	w.Dial()
	defer w.Close()
	w.SetChainID(chainID)
	myabi, err := abi.JSON(strings.NewReader(assets.UpgradeableProxyABI))
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("UpgradeContract: Cannot initialize ABI: %v", myabi)
		return err
	}
	tx, err := w.CallTransactFunction(ctx, myabi, contractAddress, "upgrade", amount, nil, 100000, newTargetAddress)
	if err != nil {
		log.Ctx(ctx).Err(err).Interface("tx", tx).Msg("UpgradeContract: Cannot upgrade the contract")
		return err
	}
	ctx, cancelFn := context.WithTimeout(ctx, time.Duration(timeoutInSeconds)*time.Second)
	defer cancelFn()
	receipt, err := w.WaitForReceipt(ctx, tx.Hash)
	if err != nil {
		log.Ctx(ctx).Err(err).Interface("tx", tx).Msgf("UpgradeContract: Cannot get the receipt for transaction with hash %s", tx.Hash.Hex())
		return err
	}
	log.Ctx(ctx).Info().Msgf("Transaction address: %s", receipt.TxHash.Hex())
	return err
}
