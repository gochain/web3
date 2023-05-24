package web3_actions

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"github.com/gochain/gochain/v4/common/hexutil"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/gochain/gochain/v4/crypto"
	"github.com/rs/zerolog/log"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

// DeployContract submits a contract creation transaction.
// abiJSON is only required when including params for the constructor.
func (w *Web3Actions) DeployContract(ctx context.Context, binHex string, payload SendContractTxPayload) (*web3_types.Transaction, error) {
	w.Dial()
	defer w.Close()
	var err error

	signedTx, err := w.GetSignedDeployTxToCallFunctionWithArgs(ctx, binHex, &payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("DeployContract: Decode")
		return nil, fmt.Errorf("cannot decode contract data: %v", err)
	}
	tx, err := w.SubmitSignedTxAndReturnTxData(ctx, signedTx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// DeployBin will deploy a bin file to the network
func (w *Web3Actions) DeployBin(ctx context.Context, binFilename, abiFilename string,
	gasPrice *big.Int, gasLimit uint64, constructorArgs ...interface{}) (*web3_types.Transaction, error) {
	w.Dial()
	defer w.Close()
	var bin []byte
	var err error
	if isValidUrl(binFilename) {
		bin, err = downloadFile(ctx, binFilename)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("DeployBin: downloadFile")
			return nil, fmt.Errorf("cannot download the bin file %q: %v", binFilename, err)
		}
	} else {
		bin, err = os.ReadFile(binFilename)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("DeployBin: ReadFile")
			return nil, fmt.Errorf("cannot read the bin file %q: %v", binFilename, err)
		}
	}
	var abi []byte
	if len(constructorArgs) > 0 {
		if isValidUrl(abiFilename) {
			abi, err = downloadFile(ctx, abiFilename)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("DeployBin: downloadFile")
				return nil, fmt.Errorf("cannot download the abi file %q: %v", abiFilename, err)
			}
		} else {
			abi, err = os.ReadFile(abiFilename)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("DeployBin: ReadFile")
				return nil, fmt.Errorf("cannot read the abi file %q: %v", abiFilename, err)
			}
		}
	}

	params := SendContractTxPayload{
		SmartContractAddr: "",
		SendEtherPayload:  SendEtherPayload{},
		ContractFile:      string(abi),
		ContractABI:       nil,
		MethodName:        "",
		Params:            nil,
	}

	return w.DeployContract(ctx, string(bin), params)
}

// GetSignedDeployTxToCallFunctionWithArgs prepares the tx for broadcast
func (w *Web3Actions) GetSignedDeployTxToCallFunctionWithArgs(ctx context.Context, binHex string, payload *SendContractTxPayload) (*types.Transaction, error) {
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
	binData, err := hexutil.Decode(binHex)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("DeployContract: Decode")
		return nil, fmt.Errorf("cannot decode contract data: %v", err)
	}
	if len(payload.Params) > 0 {
		goParams, cerr := web3_types.ConvertArguments(payload.ContractABI.Constructor.Inputs, payload.Params)
		if cerr != nil {
			log.Ctx(ctx).Err(cerr).Msg("DeployContract: ConvertArguments")
			return nil, cerr
		}
		input, perr := payload.ContractABI.Pack("", goParams...)
		if perr != nil {
			perr = fmt.Errorf("cannot pack parameters: %v", perr)
			log.Ctx(ctx).Err(perr).Msg("DeployContract: ConvertArguments")
			return nil, perr
		}
		binData = append(binData, input...)
	}
	signedTx, err := w.GetSignedTxToDeploySmartContract(ctx, payload, binData)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: GetSignedDeployTxToCallFunctionWithArgs")
		return nil, err
	}
	return signedTx, err
}

// GetSignedTxToDeploySmartContract prepares the tx for deploy
func (w *Web3Actions) GetSignedTxToDeploySmartContract(ctx context.Context, payload *SendContractTxPayload, data []byte) (*types.Transaction, error) {
	var err error
	w.Dial()
	defer w.Close()

	err = w.SetGasPriceAndLimit(ctx, &payload.GasPriceLimits)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("GetSignedTxToCallFunctionWithData: SetGasPriceAndLimit")
		return nil, err
	}
	from := w.Address()
	est, err := w.GetGasPriceEstimateForTx(ctx, web3_types.CallMsg{
		From:     &from,
		To:       nil,
		GasPrice: payload.GasPrice,
		Data:     data,
	})
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("GetSignedTxToCallFunctionWithData: SetGasPriceAndLimit")
		return nil, err
	}
	payload.GasLimit = est.Uint64()
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
	amount := new(big.Int).SetUint64(0)
	tx := types.NewContractCreation(nonce, amount, payload.GasLimit, payload.GasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), w.EcdsaPrivateKey())
	if err != nil {
		err = fmt.Errorf("cannot sign transaction: %v", err)
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: SignTx")
		return nil, err
	}
	return signedTx, err
}
