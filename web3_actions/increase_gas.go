package web3_actions

import (
	"context"
	"fmt"
	"math/big"

	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/rs/zerolog/log"
	"github.com/zeus-fyi/gochain/web3"
	"github.com/zeus-fyi/gochain/web3/accounts"
	"github.com/zeus-fyi/gochain/web3/client"
	"github.com/zeus-fyi/gochain/web3/types"
)

func IncreaseGas(ctx context.Context, privateKey string, network client.Network, txHash string, amountGwei string) error {
	client, err := client.Dial(network.URL)
	if err != nil {
		err = fmt.Errorf("failed to connect to %q: %v", network.URL, err)
		log.Ctx(ctx).Err(err).Msg("IncreaseGas: Dial")
		return err
	}
	defer client.Close()
	// then we'll clone the original and increase gas
	txOrig, err := client.GetTransactionByHash(ctx, common.HexToHash(txHash))
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
		fmt.Printf("failed to parse amount %q: %v", amountGwei, err)
		return err
	}
	newPrice := new(big.Int).Add(txOrig.GasPrice, amount)
	_, err = ReplaceTx(ctx, privateKey, network, txOrig.Nonce, *txOrig.To, txOrig.Value, newPrice, txOrig.GasLimit, txOrig.Input)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("IncreaseGas: ReplaceTx")
		return err
	}
	fmt.Printf("Increased gas price to %v\n", newPrice)
	return err
}

func ReplaceTx(ctx context.Context, privateKey string, network client.Network, nonce uint64, to common.Address, amount *big.Int,
	gasPrice *big.Int, gasLimit uint64, data []byte) (*types.Transaction, error) {
	client, err := client.Dial(network.URL)
	if err != nil {
		err = fmt.Errorf("failed to connect to %q: %v", network.URL, err)
		log.Ctx(ctx).Err(err).Msg("ReplaceTx: Dial")
		return nil, err
	}
	defer client.Close()
	if gasPrice == nil {
		gasPrice, err = client.GetGasPrice(ctx)
		if err != nil {
			err = fmt.Errorf("couldn't get suggested gas price: %v", err)
			log.Ctx(ctx).Err(err).Msg("ReplaceTx: Dial")
			return nil, err
		}
		fmt.Printf("Using suggested gas price: %v\n", gasPrice)
	}
	chainID := network.ChainID
	if chainID == nil {
		chainID, err = client.GetChainID(ctx)
		if err != nil {
			err = fmt.Errorf("couldn't get chain ID: %v", err)
			log.Ctx(ctx).Err(err).Msg("ReplaceTx: Dial")
			return nil, err
		}
	}
	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, data)
	acct, err := accounts.ParsePrivateKey(privateKey)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("ReplaceTx: ParsePrivateKey")
		return nil, err
	}
	fmt.Printf("Replacing transaction nonce: %v, gasPrice: %v, gasLimit: %v\n", nonce, gasPrice, gasLimit)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), acct.Key())
	if err != nil {
		err = fmt.Errorf("couldn't sign tx: %v", err)
		log.Ctx(ctx).Err(err).Msg("ReplaceTx: SignTx")
		return nil, err
	}
	err = web3.SendTransaction(ctx, client, signedTx)
	if err != nil {
		err = fmt.Errorf("error sending transaction: %v", err)
		log.Ctx(ctx).Err(err).Msg("ReplaceTx: SendTransaction")
		return nil, err
	}
	fmt.Printf("Replaced transaction. New transaction: %s\n", signedTx.Hash().Hex())
	return tx, err
}
