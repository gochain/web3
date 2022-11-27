package web3_actions

import (
	"context"
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
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

// CallConstantFunction executes a contract function call without submitting a transaction.
func (w *Web3Actions) CallConstantFunction(ctx context.Context, myabi abi.ABI, address string, functionName string, params ...interface{}) ([]interface{}, error) {
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
	w.Dial()
	defer w.Close()
	res, err := w.Call(ctx, web3_types.CallMsg{Data: input, To: &toAddress})
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
func (w *Web3Actions) CallFunctionWithArgs(ctx context.Context, address string,
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
	return w.CallFunctionWithData(ctx, address, amount, gasPrice, gasLimit, data)
}

// CallFunctionWithData if you already have the encoded function data, then use this
func (w *Web3Actions) CallFunctionWithData(ctx context.Context, address string,
	amount *big.Int, gasPrice *big.Int, gasLimit uint64, data []byte) (*web3_types.Transaction, error) {
	if address == "" {
		return nil, errors.New("no contract address specified")
	}
	var err error
	w.Dial()
	defer w.Close()
	if gasPrice == nil || gasPrice.Int64() == 0 {
		gasPrice, err = w.GetGasPrice(ctx)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: GetGasPrice")
			return nil, fmt.Errorf("cannot get gas price: %v", err)
		}
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
	// fmt.Println("Price: ", gasPrice)
	tx := types.NewTransaction(nonce, toAddress, amount, gasLimit, gasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), w.EcdsaPrivateKey())
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
	err = w.SendRawTransaction(ctx, raw)
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
