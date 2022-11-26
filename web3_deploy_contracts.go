package web3

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common/hexutil"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/gochain/gochain/v4/crypto"
	"github.com/gochain/gochain/v4/rlp"
)

// DeployContract submits a contract creation transaction.
// abiJSON is only required when including params for the constructor.
func DeployContract(ctx context.Context, client Client, privateKeyHex string, binHex, abiJSON string, gasPrice *big.Int, gasLimit uint64, constructorArgs ...interface{}) (*Transaction, error) {
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	if gasPrice == nil || gasPrice.Int64() == 0 {
		gasPrice, err = client.GetGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("cannot get gas price: %v", err)
		}
	}
	chainID, err := client.GetChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't get chain ID: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.GetPendingTransactionCount(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("cannot get nonce: %v", err)
	}
	binData, err := hexutil.Decode(binHex)
	if err != nil {
		return nil, fmt.Errorf("cannot decode contract data: %v", err)
	}
	if len(constructorArgs) > 0 {
		abiData, err := abi.JSON(strings.NewReader(abiJSON))
		if err != nil {
			return nil, fmt.Errorf("failed to parse ABI: %v", err)
		}
		goParams, err := ConvertArguments(abiData.Constructor.Inputs, constructorArgs)
		if err != nil {
			return nil, err
		}
		input, err := abiData.Pack("", goParams...)
		if err != nil {
			return nil, fmt.Errorf("cannot pack parameters: %v", err)
		}
		binData = append(binData, input...)
	}
	//TODO try to use web3.Transaction only; can't sign currently
	tx := types.NewContractCreation(nonce, big.NewInt(0), gasLimit, gasPrice, binData)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return nil, fmt.Errorf("cannot sign transaction: %v", err)
	}
	raw, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return nil, err
	}
	err = client.SendRawTransaction(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("cannot send transaction: %v", err)
	}

	return convertTx(signedTx, fromAddress), nil
}

// DeployBin will deploy a bin file to the network
func DeployBin(ctx context.Context, client Client, privateKeyHex, binFilename, abiFilename string,
	gasPrice *big.Int, gasLimit uint64, constructorArgs ...interface{}) (*Transaction, error) {
	var bin []byte
	var err error
	if isValidUrl(binFilename) {
		bin, err = downloadFile(binFilename)
		if err != nil {
			return nil, fmt.Errorf("Cannot download the bin file %q: %v", binFilename, err)
		}
	} else {
		bin, err = ioutil.ReadFile(binFilename)
		if err != nil {
			return nil, fmt.Errorf("Cannot read the bin file %q: %v", binFilename, err)
		}
	}
	var abi []byte
	if len(constructorArgs) > 0 {
		if isValidUrl(abiFilename) {
			abi, err = downloadFile(abiFilename)
			if err != nil {
				return nil, fmt.Errorf("Cannot download the abi file %q: %v", abiFilename, err)
			}
		} else {
			abi, err = ioutil.ReadFile(abiFilename)
			if err != nil {
				return nil, fmt.Errorf("Cannot read the abi file %q: %v", abiFilename, err)
			}
		}
	}

	return DeployContract(ctx, client, privateKeyHex, string(bin), string(abi), gasPrice, gasLimit, constructorArgs...)
}
