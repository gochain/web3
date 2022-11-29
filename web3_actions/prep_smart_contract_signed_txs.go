package web3_actions

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/gochain/gochain/v4/crypto"
	"github.com/rs/zerolog/log"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

// GetSignedTxToCallFunctionWithData prepares the tx for broadcast
func (w *Web3Actions) GetSignedTxToCallFunctionWithData(ctx context.Context, address string,
	amount *big.Int, gasPrice *big.Int, gasLimit uint64, data []byte) (*types.Transaction, error) {
	if address == "" {
		return nil, errors.New("no contract address specified")
	}
	var err error
	w.Dial()
	defer w.Close()

	gp := GasPriceLimits{
		GasPrice: gasPrice,
		GasLimit: gasLimit,
	}
	err = w.SetGasPriceAndLimit(ctx, &gp)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("GetSignedTxToCallFunctionWithData: SetGasPriceAndLimit")
		return nil, err
	}
	chainID, err := w.GetChainID(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: GetChainID")
		return nil, fmt.Errorf("couldn't get chain ID: %v", err)
	}
	publicKeyECDSA := w.EcdsaPublicKey()
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := w.GetPendingTransactionCount(ctx, fromAddress)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: GetPendingTransactionCount")
		return nil, fmt.Errorf("cannot get nonce: %v", err)
	}
	toAddress := common.HexToAddress(address)
	tx := types.NewTransaction(nonce, toAddress, amount, gp.GasLimit, gp.GasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), w.EcdsaPrivateKey())
	if err != nil {
		err = fmt.Errorf("cannot sign transaction: %v", err)
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: SignTx")
		return nil, err
	}
	return signedTx, err
}

// GetSignedTxToCallFunctionWithArgs prepares the tx for broadcast
func (w *Web3Actions) GetSignedTxToCallFunctionWithArgs(ctx context.Context, address string,
	amount *big.Int, gasPrice *big.Int, gasLimit uint64, myabi abi.ABI, functionName string, params ...interface{}) (*types.Transaction, error) {
	fn := myabi.Methods[functionName]
	goParams, err := web3_types.ConvertArguments(fn.Inputs, params)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithArgs")
		return nil, err
	}
	data, err := myabi.Pack(functionName, goParams...)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithArgs")
		return nil, fmt.Errorf("failed to pack values: %v", err)
	}
	signedTx, err := w.GetSignedTxToCallFunctionWithData(ctx, address, amount, gasPrice, gasLimit, data)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: GetSignedTxToCallFunctionWithData")
		return nil, err
	}
	return signedTx, err
}
