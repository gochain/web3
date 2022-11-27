package web3_actions

import (
	"context"
	"errors"
	"fmt"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/common/hexutil"
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
