package web3

import (
	"context"
	"fmt"
	"math/big"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/common/hexutil"
)

// CallTransactFunction submits a transaction to execute a smart contract function call.
// @Deprecated use CallFunctionWithArgs, better signature
func CallTransactFunction(ctx context.Context, client Client, myabi abi.ABI, address, privateKeyHex, functionName string,
	amount *big.Int, gasPrice *big.Int, gasLimit uint64, params ...interface{}) (*Transaction, error) {
	return CallFunctionWithArgs(ctx, client, privateKeyHex, address, amount, gasPrice, gasLimit, myabi, functionName, params...)
}

// TODO Deprecated: prefer built-in UnpackValues() func and convertOutputParams.
func convertOutputParameter(t abi.Argument) (interface{}, error) {
	switch t.Type.T {
	case abi.BoolTy:
		return new(bool), nil
	case abi.UintTy:
		switch size := t.Type.Size; {
		case size > 64:
			i := new(big.Int)
			return &i, nil
		case size > 32:
			return new(uint64), nil
		case size > 16:
			return new(uint32), nil
		case size > 8:
			return new(uint16), nil
		default:
			return new(uint8), nil
		}
	case abi.IntTy:
		switch size := t.Type.Size; {
		case size > 64:
			i := new(big.Int)
			return &i, nil
		case size > 32:
			return new(int64), nil
		case size > 16:
			return new(int32), nil
		case size > 8:
			return new(int16), nil
		default:
			return new(int8), nil
		}
	case abi.StringTy:
		return new(string), nil
	case abi.AddressTy:
		return new(common.Address), nil
	case abi.BytesTy:
		return new(hexutil.Bytes), nil
	case abi.FixedBytesTy:
		switch size := t.Type.Size; {
		case size == 32:
			return new(common.Hash), nil
		default:
			return nil, fmt.Errorf("unsupported output byte array size %v", size)
		}
	default:
		return nil, fmt.Errorf("unsupported output type %v", t.Type.T)
	}
}
