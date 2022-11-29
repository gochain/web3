package web3_actions

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common/hexutil"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/gochain/gochain/v4/crypto"
	"github.com/gochain/gochain/v4/rlp"
	"github.com/rs/zerolog/log"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

// DeployContract submits a contract creation transaction.
// abiJSON is only required when including params for the constructor.
func (w *Web3Actions) DeployContract(ctx context.Context, binHex, abiJSON string, gasPrice *big.Int, gasLimit uint64, constructorArgs ...interface{}) (*web3_types.Transaction, error) {
	w.Dial()
	defer w.Close()
	var err error
	if gasPrice == nil || gasPrice.Int64() == 0 {
		gasPrice, err = w.GetGasPrice(ctx)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("DeployContract: GetGasPrice")
			return nil, fmt.Errorf("cannot get gas price: %v", err)
		}
	}
	chainID, err := w.GetChainID(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("DeployContract: GetChainID")
		return nil, fmt.Errorf("couldn't get chain ID: %v", err)
	}

	publicKeyECDSA := w.EcdsaPublicKey()
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := w.GetPendingTransactionCount(ctx, fromAddress)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("DeployContract: GetPendingTransactionCount")
		return nil, fmt.Errorf("cannot get nonce: %v", err)
	}
	binData, err := hexutil.Decode(binHex)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("DeployContract: Decode")
		return nil, fmt.Errorf("cannot decode contract data: %v", err)
	}
	if len(constructorArgs) > 0 {
		abiData, abiErr := abi.JSON(strings.NewReader(abiJSON))
		if abiErr != nil {
			log.Ctx(ctx).Err(abiErr).Msg("DeployContract: abi.JSON")
			return nil, fmt.Errorf("failed to parse ABI: %v", abiErr)
		}
		goParams, cerr := web3_types.ConvertArguments(abiData.Constructor.Inputs, constructorArgs)
		if cerr != nil {
			log.Ctx(ctx).Err(cerr).Msg("DeployContract: ConvertArguments")
			return nil, cerr
		}
		input, perr := abiData.Pack("", goParams...)
		if perr != nil {
			perr = fmt.Errorf("cannot pack parameters: %v", perr)
			log.Ctx(ctx).Err(perr).Msg("DeployContract: ConvertArguments")
			return nil, perr
		}
		binData = append(binData, input...)
	}
	//TODO try to use web3.Transaction only; can't sign currently
	tx := types.NewContractCreation(nonce, big.NewInt(0), gasLimit, gasPrice, binData)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), w.EcdsaPrivateKey())
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("DeployContract: types.SignTx")
		return nil, fmt.Errorf("cannot sign transaction: %v", err)
	}
	raw, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("DeployContract: rlp.EncodeToBytes")
		return nil, err
	}
	err = w.SendRawTransaction(ctx, raw)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("DeployContract: SendRawTransaction")
		return nil, fmt.Errorf("cannot send transaction: %v", err)
	}

	return ConvertTx(signedTx, fromAddress), nil
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

	return w.DeployContract(ctx, string(bin), string(abi), gasPrice, gasLimit, constructorArgs...)
}
