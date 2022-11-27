package web3_actions

import (
	"context"
	"fmt"
	"math/big"
	"reflect"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/common/hexutil"
	"github.com/gochain/gochain/v4/crypto"
	"github.com/gochain/gochain/v4/rlp"
	"github.com/rs/zerolog/log"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

// CallFunctionWithArgs submits a transaction to execute a smart contract function call.
func (w *Web3Actions) CallFunctionWithArgs(ctx context.Context, address string,
	amount *big.Int, gasPrice *big.Int, gasLimit uint64, myabi abi.ABI, functionName string, params ...interface{}) (*web3_types.Transaction, error) {
	signedTx, err := w.GetSignedTxToCallFunctionWithArgs(ctx, address, amount, gasPrice, gasLimit, myabi, functionName, params...)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: GetSignedTxToCallFunctionWithArgs")
		return nil, err
	}
	return w.SubmitSignedTxAndReturnTxData(ctx, signedTx)
}

// CallFunctionWithData if you already have the encoded function data, then use this
func (w *Web3Actions) CallFunctionWithData(ctx context.Context, address string,
	amount *big.Int, gasPrice *big.Int, gasLimit uint64, data []byte) (*web3_types.Transaction, error) {
	signedTx, err := w.GetSignedTxToCallFunctionWithData(ctx, address, amount, gasPrice, gasLimit, data)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallFunctionWithData: GetSignedTxToCallFunctionWithData")
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
	publicKeyECDSA := w.EcdsaPublicKey()
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
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
