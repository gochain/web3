package web3_actions

import (
	"context"
	"fmt"

	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/gochain/gochain/v4/crypto"
	"github.com/rs/zerolog/log"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

// GetSignedTxToCallFunctionWithData prepares the tx for broadcast
func (w *Web3Actions) GetSignedTxToCallFunctionWithData(ctx context.Context, payload *SendContractTxPayload, data []byte) (*types.Transaction, error) {
	var err error
	w.Dial()
	defer w.Close()

	err = w.SetGasPriceAndLimit(ctx, &payload.GasPriceLimits)
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
	tx := types.NewTransaction(nonce, common.HexToAddress(payload.SmartContractAddr), payload.Amount, payload.GasLimit, payload.GasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), w.EcdsaPrivateKey())
	if err != nil {
		err = fmt.Errorf("cannot sign transaction: %v", err)
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: SignTx")
		return nil, err
	}
	return signedTx, err
}

// GetSignedTxToCallFunctionWithArgs prepares the tx for broadcast
func (w *Web3Actions) GetSignedTxToCallFunctionWithArgs(ctx context.Context, payload *SendContractTxPayload) (*types.Transaction, error) {
	w.Dial()
	defer w.Close()
	err := w.GetAndSetChainID(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Web3Actions: GetAndSetChainID")
		return nil, err
	}
	err = w.SetGasPriceAndLimit(ctx, &payload.GasPriceLimits)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Web3Actions: Transfer: SetGasPriceAndLimit")
		return nil, err
	}

	myabi := payload.ContractABI
	if myabi == nil {
		abiInternal, aerr := web3_types.GetABI(payload.ContractFile)
		if aerr != nil {
			log.Ctx(ctx).Err(aerr).Msg("CallContract: GetABI")
			return nil, aerr
		}
		myabi = abiInternal
	}

	fn := myabi.Methods[payload.MethodName]
	goParams, err := web3_types.ConvertArguments(fn.Inputs, payload.Params)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithArgs")
		return nil, err
	}
	data, err := myabi.Pack(payload.MethodName, goParams...)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithArgs")
		return nil, fmt.Errorf("failed to pack values: %v", err)
	}
	signedTx, err := w.GetSignedTxToCallFunctionWithData(ctx, payload, data)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: GetSignedTxToCallFunctionWithData")
		return nil, err
	}
	return signedTx, err
}
