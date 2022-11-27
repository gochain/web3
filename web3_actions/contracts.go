package web3_actions

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/rs/zerolog/log"
	"github.com/zeus-fyi/gochain/web3"
)

// Flags
var (
	verbose bool
	format  string
)

func marshalJSON(ctx context.Context, data interface{}) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("marshalJSON")
		return "", err
	}
	return string(b), err
}

func ListContract(ctx context.Context, contractFile string) error {
	myabi, err := web3.GetABI(contractFile)
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

func GetContractConst(ctx context.Context, rpcURL, contractAddress, contractFile, functionName string, parameters ...interface{}) ([]interface{}, error) {
	client, err := web3.Dial(rpcURL)
	if err != nil {
		err = fmt.Errorf("failed to connect to %q: %v", rpcURL, err)
		log.Ctx(ctx).Err(err).Msg("GetContractConst: Dial")
		return nil, err
	}
	defer client.Close()
	myabi, err := web3.GetABI(contractFile)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("GetContractConst: GetABI")
		return nil, err
	}
	fn, ok := myabi.Methods[functionName]
	if !ok {
		err = fmt.Errorf("there is no such function: %v", functionName)
		log.Ctx(ctx).Err(err).Msg("GetContractConst: myabi.Methods")
		return nil, err
	}
	if !fn.IsConstant() {
		log.Ctx(ctx).Err(err).Msg("GetContractConst: !IsConstant")
		return nil, err
	}
	res, err := web3.CallConstantFunction(ctx, client, *myabi, contractAddress, functionName, parameters...)
	if err != nil {
		err = fmt.Errorf("error calling constant function: %v", err)
		log.Ctx(ctx).Err(err).Msg("GetContractConst: CallConstantFunction")
		return nil, err
	}
	return res, nil
}

func CallContract(ctx context.Context, client web3.Client, privateKey, contractAddress, abiFile, functionName string,
	amount *big.Int, gasPrice *big.Int, gasLimit uint64, waitForReceipt, toString bool, data []byte, timeoutInSeconds uint64, parameters ...interface{}) error {

	var err error
	var tx *web3.Transaction
	var myabi *abi.ABI
	if len(data) > 0 {
		tx, err = web3.CallFunctionWithData(ctx, client, privateKey, contractAddress, amount, gasPrice, gasLimit, data)
	} else {
		// var m abi.Method
		myabi, err = web3.GetABI(abiFile)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("CallContract: GetABI")
			return err
		}
		m, ok := myabi.Methods[functionName]
		if !ok {
			err = fmt.Errorf("error calling constant function: %v", err)
			log.Ctx(ctx).Err(err).Msg("CallContract: GetABI")
			return err
		}

		if m.IsConstant() {
			res, cerr := web3.CallConstantFunction(ctx, client, *myabi, contractAddress, functionName, parameters...)
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
			if toString {
				for i := range res {
					fmt.Printf("%s\n", res[i])
				}
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
		tx, err = web3.CallTransactFunction(ctx, client, *myabi, contractAddress, privateKey, functionName, amount, gasPrice, gasLimit, parameters...)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("CallContract: CallTransactFunction")
			return err
		}
	}

	fmt.Println("Transaction hash:", tx.Hash.Hex())
	if !waitForReceipt {
		return err
	}
	fmt.Println("Waiting for receipt...")
	ctx, cancelFunc := context.WithTimeout(ctx, time.Duration(timeoutInSeconds)*time.Second)
	defer cancelFunc()
	receipt, err := web3.WaitForReceipt(ctx, client, tx.Hash)
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
