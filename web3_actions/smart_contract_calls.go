package web3_actions

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/common/hexutil"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/gochain/gochain/v4/crypto"
	"github.com/gochain/gochain/v4/rlp"
	"github.com/rs/zerolog/log"
	"github.com/zeus-fyi/gochain/web3/client"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

// CallConstantFunction executes a contract function call without submitting a transaction.
func CallConstantFunction(ctx context.Context, client client.Client, myabi abi.ABI, address string, functionName string, params ...interface{}) ([]interface{}, error) {
	if address == "" {
		err := errors.New("no contract address specified")
		log.Ctx(ctx).Err(err).Msg("CallConstantFunction")
		return nil, err
	}
	fn := myabi.Methods[functionName]
	goParams, err := web3_types.ConvertArguments(fn.Inputs, params)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallConstantFunction: ConvertArguments")
		return nil, err
	}
	input, err := myabi.Pack(functionName, goParams...)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallConstantFunction: myabi.Pack")
		return nil, fmt.Errorf("failed to pack values: %v", err)
	}
	toAddress := common.HexToAddress(address)
	res, err := client.Call(ctx, web3_types.CallMsg{Data: input, To: &toAddress})
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallConstantFunction: client.Call")
		return nil, err
	}
	// TODO: calling a function on a contract errors on unpacking, it should probably know it's not a contract before hand if it can
	// fmt.Printf("RESPONSE: %v\n", string(res))
	vals, err := fn.Outputs.UnpackValues(res)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallConstantFunction: UnpackValues")
		return nil, fmt.Errorf("failed to unpack values from %s: %v", hexutil.Encode(res), err)
	}
	return convertOutputParams(vals), nil
}

// CallFunctionWithArgs submits a transaction to execute a smart contract function call.
func CallFunctionWithArgs(ctx context.Context, client client.Client, privateKeyHex, address string,
	amount *big.Int, gasPrice *big.Int, gasLimit uint64, myabi abi.ABI, functionName string, params ...interface{}) (*web3_types.Transaction, error) {

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
	return CallFunctionWithData(ctx, client, privateKeyHex, address, amount, gasPrice, gasLimit, data)
}

// CallFunctionWithData if you already have the encoded function data, then use this
func CallFunctionWithData(ctx context.Context, client client.Client, privateKeyHex, address string,
	amount *big.Int, gasPrice *big.Int, gasLimit uint64, data []byte) (*web3_types.Transaction, error) {
	if address == "" {
		return nil, errors.New("no contract address specified")
	}
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: HexToECDSA")
		return nil, fmt.Errorf("invalid private key: %v", err)
	}
	if gasPrice == nil || gasPrice.Int64() == 0 {
		gasPrice, err = client.GetGasPrice(ctx)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: GetGasPrice")
			return nil, fmt.Errorf("cannot get gas price: %v", err)
		}
	}
	chainID, err := client.GetChainID(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: GetChainID")
		return nil, fmt.Errorf("couldn't get chain ID: %v", err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		err = errors.New("error casting public key to ECDSA")
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData")
		return nil, err
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.GetPendingTransactionCount(ctx, fromAddress)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: GetPendingTransactionCount")
		return nil, fmt.Errorf("cannot get nonce: %v", err)
	}
	toAddress := common.HexToAddress(address)
	// fmt.Println("Price: ", gasPrice)
	tx := types.NewTransaction(nonce, toAddress, amount, gasLimit, gasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		err = fmt.Errorf("cannot sign transaction: %v", err)
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: SignTx")
		return nil, err
	}
	raw, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: EncodeToBytes")
		return nil, err
	}
	err = client.SendRawTransaction(ctx, raw)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: SendRawTransaction")
		return nil, fmt.Errorf("cannot send transaction: %v", err)
	}
	return ConvertTx(signedTx, fromAddress), nil
}

func convertOutputParams(params []interface{}) []interface{} {
	for i := range params {
		p := params[i]
		if h, ok := p.(common.Hash); ok {
			params[i] = h
		} else if a, okAddr := p.(common.Address); okAddr {
			params[i] = a
		} else if b, okBytes := p.(hexutil.Bytes); okBytes {
			params[i] = b
		} else if v := reflect.ValueOf(p); v.Kind() == reflect.Array {
			if t := v.Type(); t.Elem().Kind() == reflect.Uint8 {
				b := make([]byte, t.Len())
				bv := reflect.ValueOf(b)
				// Copy since we can't t.Slice() unaddressable arrays.
				for i := 0; i < t.Len(); i++ {
					bv.Index(i).Set(v.Index(i))
				}
				params[i] = hexutil.Bytes(b)
			}
		}
	}
	return params
}
