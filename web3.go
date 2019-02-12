package web3

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"time"

	"github.com/gochain-io/gochain/v3/accounts/abi"
	"github.com/gochain-io/gochain/v3/common"
	"github.com/gochain-io/gochain/v3/common/hexutil"
	"github.com/gochain-io/gochain/v3/core/types"
	"github.com/gochain-io/gochain/v3/crypto"
	"github.com/gochain-io/gochain/v3/rlp"
)

var NotFoundErr = errors.New("not found")

const (
	testnetURL = "https://testnet-rpc.gochain.io"
	mainnetURL = "https://rpc.gochain.io"
)

var Networks = map[string]Network{
	"testnet": {
		URL:  testnetURL,
		Unit: "GO",
	},
	"mainnet": {
		URL:  mainnetURL,
		Unit: "GO",
	},
	"localhost": {
		URL:  "http://localhost:8545",
		Unit: "GO",
	},
	"ethereum": {
		URL:  "https://main-rpc.linkpool.io",
		Unit: "ETH",
	},
	"ropsten": {
		URL:  "https://ropsten-rpc.linkpool.io",
		Unit: "ETH",
	},
}

type Network struct {
	URL  string
	Unit string
	//TODO net_id, chain_id
}

var (
	weiPerGO   = big.NewInt(1e18)
	weiPerGwei = big.NewInt(1e9)
)

// WeiAsBase converts w wei in to the base unit, and formats it as a decimal fraction with full precision (up to 18 decimals).
func WeiAsBase(w *big.Int) string {
	return new(big.Rat).SetFrac(w, weiPerGO).FloatString(18)
}

// WeiAsGwei converts w wei in to gwei, and formats it as a decimal fraction with full precision (up to 9 decimals).
func WeiAsGwei(w *big.Int) string {
	return new(big.Rat).SetFrac(w, weiPerGwei).FloatString(9)
}

func convertOutputParameter(t abi.Argument) interface{} {
	switch t.Type.T {
	case abi.BoolTy:
		return new(bool)
	case abi.UintTy, abi.IntTy:
		return new(big.Int)
	case abi.StringTy:
		return new(string)
	case abi.AddressTy:
		return new(common.Address)
	case abi.BytesTy, abi.FixedBytesTy:
		return new([]byte)
	default:
		return new(string)
	}
}

// CallConstantFunction executes a contract function call without submitting a transaction.
func CallConstantFunction(ctx context.Context, client Client, myabi abi.ABI, address, functionName string, parameters ...interface{}) (interface{}, error) {
	var out []interface{}
	for _, t := range myabi.Methods[functionName].Outputs {
		out = append(out, convertOutputParameter(t))
	}
	if len(myabi.Methods[functionName].Inputs) != len(parameters) {
		return nil, errors.New("Wrong number of arguments expected:" + strconv.Itoa(len(myabi.Methods[functionName].Inputs)) + " given:" + strconv.Itoa(len(parameters)))
	}

	input, err := myabi.Pack(functionName, convertParameters(myabi.Methods[functionName], parameters)...)
	if err != nil {
		return nil, err
	}

	toAddress := common.HexToAddress(address)

	res, err := client.Call(ctx, CallMsg{Data: input, To: &toAddress})
	if err != nil {
		return nil, err

	}
	if len(myabi.Methods[functionName].Outputs) > 1 {
		err = myabi.Unpack(&out, functionName, res)
		if err != nil {
			return nil, err
		}
		for i, o := range out {
			out[i] = reflect.ValueOf(o).Elem().Interface()
		}
		return out, nil
	}
	err = myabi.Unpack(&out[0], functionName, res)
	if err != nil {
		return nil, err
	}
	return out[0], nil
}

// CallTransactFunction submits a transaction to execute a smart contract function call.
func CallTransactFunction(ctx context.Context, client Client, myabi abi.ABI, address, privateKeyHex, functionName string, amount int, parameters ...interface{}) (*Transaction, error) {

	if len(myabi.Methods[functionName].Inputs) != len(parameters) {
		return nil, errors.New("Wrong number of arguments expected:" + strconv.Itoa(len(myabi.Methods[functionName].Inputs)) + " given:" + strconv.Itoa(len(parameters)))
	}

	input, err := myabi.Pack(functionName, convertParameters(myabi.Methods[functionName], parameters)...)
	if err != nil {
		return nil, err
	}
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}
	gasPrice, err := client.GetGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get gas price: %v", err)
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
	toAddress := common.HexToAddress(address)
	tx := types.NewTransaction(nonce, toAddress, big.NewInt(int64(amount)), 20000000, gasPrice, input)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, privateKey)
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

// DeployContract submits a contract creation transaction.
func DeployContract(ctx context.Context, client Client, privateKeyHex string, contractData string) (*Transaction, error) {
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	gasPrice, err := client.GetGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get gas price: %v", err)
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
	decodedContractData, err := hexutil.Decode(contractData)
	if err != nil {
		return nil, fmt.Errorf("cannot decode contract data: %v", err)
	}
	//TODO try to use web3.Transaction only; can't sign currently
	tx := types.NewContractCreation(nonce, big.NewInt(0), 2000000, gasPrice, decodedContractData)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, privateKey)
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

func convertTx(tx *types.Transaction, from common.Address) *Transaction {
	rtx := &Transaction{}
	rtx.Nonce = tx.Nonce()
	rtx.GasPrice = tx.GasPrice()
	rtx.GasLimit = tx.Gas()
	rtx.To = tx.To()
	rtx.Value = tx.Value()
	rtx.Input = tx.Data()
	rtx.Hash = tx.Hash()
	rtx.From = from
	rtx.V, rtx.R, rtx.S = tx.RawSignatureValues()
	return rtx
}

func convertParameters(method abi.Method, inputParams []interface{}) []interface{} {
	var convertedParams []interface{}
	for i, input := range method.Inputs {
		switch input.Type.T {
		case abi.BoolTy:
			val, _ := strconv.ParseBool(inputParams[i].(string))
			convertedParams = append(convertedParams, val)
		case abi.UintTy:
			val := new(big.Int)
			fmt.Sscan(inputParams[i].(string), val)
			convertedParams = append(convertedParams, val)
		case abi.AddressTy:
			val := common.HexToAddress(inputParams[i].(string))
			convertedParams = append(convertedParams, val)
		default:
			val := inputParams[i].(string)
			convertedParams = append(convertedParams, val)
		}

	}
	return convertedParams
}

// WaitForReceipt polls for a transaction receipt until it is available, or ctx is cancelled.
func WaitForReceipt(ctx context.Context, client Client, hash common.Hash) (*Receipt, error) {
	for i := 0; ; i++ {
		receipt, err := client.GetTransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if i >= (5) {
			return nil, fmt.Errorf("cannot get the receipt: %v", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
