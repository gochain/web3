package web3_actions

import (
	"context"
	"fmt"
	"math/big"

	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/rs/zerolog/log"
	web3_client "github.com/zeus-fyi/gochain/web3/client"
	"github.com/zeus-fyi/gochain/web3/types"
)

func (w *Web3Actions) IncreaseGas(ctx context.Context, network web3_client.Network, txHash string, amountGwei string) error {
	w.Dial()
	defer w.Close()
	// then we'll clone the original and increase gas
	txOrig, err := w.GetTransactionByHash(ctx, common.HexToHash(txHash))
	if err != nil {
		err = fmt.Errorf("error on GetTransactionByHash: %v", err)
		log.Ctx(ctx).Err(err).Msg("IncreaseGas: Dial")
		return err
	}
	if txOrig.BlockNumber != nil {
		fmt.Printf("tx isn't pending, so can't increase gas")
		return err
	}
	amount, err := web3_types.ParseGwei(amountGwei)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("IncreaseGas: ParseGwei")
		log.Ctx(ctx).Warn().Msgf("IncreaseGas: failed to parse amount %q: %v\n", amountGwei, err)
		return err
	}
	newPrice := new(big.Int).Add(txOrig.GasPrice, amount)
	_, err = w.ReplaceTx(ctx, network, txOrig.Nonce, *txOrig.To, txOrig.Value, newPrice, txOrig.GasLimit, txOrig.Input)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("IncreaseGas: ReplaceTx")
		return err
	}
	log.Ctx(ctx).Info().Msgf("IncreaseGas: Increased gas price to %v\n", newPrice)
	return err
}

func (w *Web3Actions) ReplaceTx(ctx context.Context, network web3_client.Network, nonce uint64, to common.Address, amount *big.Int,
	gasPrice *big.Int, gasLimit uint64, data []byte) (*types.Transaction, error) {
	w.Dial()
	defer w.Close()
	if gasPrice == nil {
		gasPriceFetched, err := w.GetGasPrice(ctx)
		if err != nil {
			err = fmt.Errorf("couldn't get suggested gas price: %v", err)
			log.Ctx(ctx).Err(err).Msg("ReplaceTx: Dial")
			return nil, err
		}
		gasPrice = gasPriceFetched
		fmt.Printf("Using suggested gas price: %v\n", gasPrice)
	}

	chainID := network.ChainID
	if chainID == nil {
		fetchedChainID, err := w.GetChainID(ctx)
		if err != nil {
			err = fmt.Errorf("couldn't get chain ID: %v", err)
			log.Ctx(ctx).Err(err).Msg("ReplaceTx: Dial")
			return nil, err
		}
		chainID = fetchedChainID
	}
	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, data)
	fmt.Printf("Replacing transaction nonce: %v, gasPrice: %v, gasLimit: %v\n", nonce, gasPrice, gasLimit)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), w.EcdsaPrivateKey())
	if err != nil {
		err = fmt.Errorf("couldn't sign tx: %v", err)
		log.Ctx(ctx).Err(err).Msg("ReplaceTx: SignTx")
		return nil, err
	}
	err = w.SendSignedTransaction(ctx, signedTx)
	if err != nil {
		err = fmt.Errorf("error sending transaction: %v", err)
		log.Ctx(ctx).Err(err).Msg("ReplaceTx: SendTransaction")
		return nil, err
	}
	log.Ctx(ctx).Info().Msgf("ReplaceTx: Replaced transaction. New transaction:  %s\n", signedTx.Hash().Hex())
	return tx, err
}
