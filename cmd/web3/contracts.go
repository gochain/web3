package main

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/gochain/gochain/v3/accounts/abi"
	"github.com/gochain/web3"
)

func ListContract(contractFile string) {

	myabi, err := web3.GetABI(contractFile)
	if err != nil {
		fatalExit(err)
	}
	switch format {
	case "json":
		fmt.Println(marshalJSON(myabi.Methods))
		return
	}

	for _, method := range myabi.Methods {
		fmt.Println(method)
	}

}

func GetContractConst(ctx context.Context, rpcURL, contractAddress, contractFile, functionName string, parameters ...interface{}) ([]interface{}, error) {
	client, err := web3.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to %q: %v", rpcURL, err)
	}
	defer client.Close()
	myabi, err := web3.GetABI(contractFile)
	if err != nil {
		return nil, err
	}
	fn, ok := myabi.Methods[functionName]
	if !ok {
		return nil, fmt.Errorf("There is no such function: %v", functionName)
	}
	if !fn.Const {
		return nil, err
	}
	res, err := web3.CallConstantFunction(ctx, client, *myabi, contractAddress, functionName, parameters...)
	if err != nil {
		return nil, fmt.Errorf("Error calling constant function: %v", err)
	}
	return res, nil
}

func callContract(ctx context.Context, rpcURL, privateKey, contractAddress, contractFile, functionName string,
	amount *big.Int, gasLimit uint64, waitForReceipt, toString bool, parameters ...interface{}) {
	client, err := web3.Dial(rpcURL)
	if err != nil {
		fatalExit(fmt.Errorf("Failed to connect to %q: %v", rpcURL, err))
	}
	defer client.Close()
	myabi, err := web3.GetABI(contractFile)
	if err != nil {
		fatalExit(err)
	}
	m, ok := myabi.Methods[functionName]
	if !ok {
		fmt.Println("There is no such function:", functionName)
		return
	}
	if m.Const || m.StateMutability == abi.MutabilityView || m.StateMutability == abi.MutabilityPure { // view and pure are both read only: https://ethereum.stackexchange.com/a/57424/9815
		res, err := web3.CallConstantFunction(ctx, client, *myabi, contractAddress, functionName, parameters...)
		if err != nil {
			fatalExit(fmt.Errorf("Error calling constant function: %v", err))
		}
		switch format {
		case "json":
			m := make(map[string]interface{})
			if len(res) == 1 {
				m["response"] = res[0]
			} else {
				m["response"] = res
			}
			fmt.Println(marshalJSON(m))
			return
		}
		if toString {
			for i := range res {
				fmt.Printf("%s\n", res[i])
			}
			return
		}
		for _, r := range res {
			// These explicit checks ensure we get hex encoded output.
			if s, ok := r.(fmt.Stringer); ok {
				r = s.String()
			}
			fmt.Println(r)
		}
		return
	}
	tx, err := web3.CallTransactFunction(ctx, client, *myabi, contractAddress, privateKey, functionName, amount, gasLimit, parameters...)
	if err != nil {
		fatalExit(fmt.Errorf("Error calling contract: %v", err))
	}
	fmt.Println("Transaction hash:", tx.Hash.Hex())
	if !waitForReceipt {
		return
	}
	fmt.Println("Waiting for receipt...")
	ctx, _ = context.WithTimeout(ctx, 60*time.Second)
	receipt, err := web3.WaitForReceipt(ctx, client, tx.Hash)
	if err != nil {
		fatalExit(fmt.Errorf("getting receipt: %v", err))
	}
	printReceiptDetails(receipt, myabi)
}
