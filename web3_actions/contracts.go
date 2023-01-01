package web3_actions

import (
	"context"
	"fmt"
	"time"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/rs/zerolog/log"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

// Flags
var (
	verbose bool
	format  string
)

const (
	ERC20     = "erc20"
	Transfer  = "transfer"
	Decimals  = "decimals"
	BalanceOf = "balanceOf"
	Pause     = "pause"
	Resume    = "resume"
	Upgrade   = "upgrade"
)

func ListContract(ctx context.Context, contractFile string) error {
	myabi, err := web3_types.GetABI(contractFile)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("ListContract: GetABI")
		return err
	}
	switch format {
	case "json":
		outPut, merr := marshalJSON(ctx, myabi.Methods)
		fmt.Println(outPut)
		return merr
	}

	for _, method := range myabi.Methods {
		fmt.Println(method)
	}
	return err
}

func (w *Web3Actions) GetContractConst(ctx context.Context, payload *SendContractTxPayload) ([]interface{}, error) {
	w.Dial()
	defer w.Close()

	myabi := payload.ContractABI
	if myabi == nil {
		abiInternal, aerr := web3_types.GetABI(payload.ContractFile)
		if aerr != nil {
			log.Ctx(ctx).Err(aerr).Msg("CallContract: GetABI")
			return nil, aerr
		}
		myabi = abiInternal
	}

	var err error
	fn, ok := myabi.Methods[payload.MethodName]
	if !ok {
		err = fmt.Errorf("there is no such function: %v", payload.MethodName)
		log.Ctx(ctx).Err(err).Msg("GetContractConst: myabi.Methods")
		return nil, err
	}
	if !fn.IsConstant() {
		log.Ctx(ctx).Err(err).Msg("GetContractConst: !IsConstant")
		return nil, err
	}

	res, err := w.CallConstantFunction(ctx, payload)
	if err != nil {
		err = fmt.Errorf("error calling constant function: %v", err)
		log.Ctx(ctx).Err(err).Msg("GetContractConst: CallConstantFunction")
		return nil, err
	}
	return res, nil
}

func (w *Web3Actions) CallContract(ctx context.Context,
	payload *SendContractTxPayload, waitForReceipt bool, data []byte, timeoutInSeconds uint64) error {
	w.Dial()
	defer w.Close()
	var err error
	var tx *web3_types.Transaction
	var myabi *abi.ABI
	if len(data) > 0 {
		tx, err = w.CallFunctionWithData(ctx, payload, data)
	} else {
		// var m abi.Method
		myabi = payload.ContractABI
		if myabi == nil {
			abiInternal, aerr := web3_types.GetABI(payload.ContractFile)
			if aerr != nil {
				log.Ctx(ctx).Err(aerr).Msg("CallContract: GetABI")
				return aerr
			}
			myabi = abiInternal
		}
		m, ok := myabi.Methods[payload.MethodName]
		if !ok {
			err = fmt.Errorf("error calling constant function: %v", err)
			log.Ctx(ctx).Err(err).Msg("CallContract: GetABI")
			return err
		}

		if m.IsConstant() {
			res, cerr := w.CallConstantFunction(ctx, payload)
			if cerr != nil {
				cerr = fmt.Errorf("error calling constant function: %v", cerr)
				log.Ctx(ctx).Err(cerr).Msg("CallContract: CallConstantFunction")
				return cerr
			}
			switch format {
			case "json":
				hashMap := make(map[string]interface{})
				if len(res) == 1 {
					hashMap["response"] = res[0]
				} else {
					hashMap["response"] = res
				}
				fmt.Println(marshalJSON(ctx, hashMap))
				return err
			}
			for _, r := range res {
				// These explicit checks ensure we get hex encoded output.
				if s, okay := r.(fmt.Stringer); okay {
					r = s.String()
				}
				fmt.Println(r)
			}
			return err
		}
		tx, err = w.CallTransactFunction(ctx, payload)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("CallContract: CallTransactFunction")
			return err
		}
	}

	fmt.Println("Transaction hash:", tx.Hash.Hex())
	if !waitForReceipt {
		return err
	}

	return w.waitForConfirmation(ctx, myabi, tx.Hash, timeoutInSeconds)
}

func (w *Web3Actions) waitForConfirmation(ctx context.Context, myabi *abi.ABI, tx common.Hash, timeoutInSeconds uint64) error {
	fmt.Println("Waiting for receipt...")
	ctx, cancelFunc := context.WithTimeout(ctx, time.Duration(timeoutInSeconds)*time.Second)
	defer cancelFunc()
	receipt, err := w.WaitForReceipt(ctx, tx)
	if err != nil {
		err = fmt.Errorf("getting receipt: %v", err)
		log.Ctx(ctx).Err(err).Msg("CallContract: CallTransactFunction")
		return err
	}
	err = PrintReceiptDetails(ctx, receipt, myabi)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallContract: PrintReceiptDetails")
		return err
	}
	return err
}

func (w *Web3Actions) CallTransactFunction(ctx context.Context, payload *SendContractTxPayload) (*web3_types.Transaction, error) {
	return w.CallFunctionWithArgs(ctx, payload)
}
