package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gochain-io/web3"
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

func GetContractConst(ctx context.Context, rpcURL, contractAddress, contractFile, functionName string, parameters ...interface{}) (interface{}, error) {
	client, err := web3.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to %q: %v", rpcURL, err)
	}
	defer client.Close()
	myabi, err := web3.GetABI(contractFile)
	if err != nil {
		return "", err
	}
	if _, ok := myabi.Methods[functionName]; !ok {
		return nil, fmt.Errorf("There is no such function: %v", functionName)
	}
	if !myabi.Methods[functionName].Const {
		return nil, err
	}
	res, err := web3.CallConstantFunction(ctx, client, *myabi, contractAddress, functionName, parameters...)
	if err != nil {
		return nil, fmt.Errorf("Cannot call the contract: %v", err)
	}
	return res, nil
}

func CallContract(ctx context.Context, rpcURL, privateKey, contractAddress, contractFile, functionName string,
	amount int, waitForReceipt bool, parameters ...interface{}) {
	client, err := web3.Dial(rpcURL)
	if err != nil {
		fatalExit(fmt.Errorf("Failed to connect to %q: %v", rpcURL, err))
	}
	defer client.Close()
	myabi, err := web3.GetABI(contractFile)
	if err != nil {
		fatalExit(err)
	}
	if _, ok := myabi.Methods[functionName]; !ok {
		fmt.Println("There is no such function:", functionName)
		return
	}
	if myabi.Methods[functionName].Const {
		res, err := web3.CallConstantFunction(ctx, client, *myabi, contractAddress, functionName, parameters...)
		if err != nil {
			fatalExit(fmt.Errorf("Cannot call the contract: %v", err))
		}
		switch format {
		case "json":
			m := make(map[string]interface{})
			m["response"] = res
			fmt.Println(marshalJSON(m))
			return
		}
		fmt.Println(res)
		return
	}
	tx, err := web3.CallTransactFunction(ctx, client, *myabi, contractAddress, privateKey, functionName, amount, parameters...)
	if err != nil {
		fatalExit(fmt.Errorf("Cannot call the contract: %v", err))
	}
	if !waitForReceipt {
		fmt.Println("Transaction address:", tx.Hash.Hex())
		return
	}
	ctx, _ = context.WithTimeout(ctx, 10*time.Second)
	receipt, err := web3.WaitForReceipt(ctx, client, tx.Hash)
	if err != nil {
		fatalExit(fmt.Errorf("Cannot get the receipt: %v", err))
	}
	printReceiptDetails(receipt, myabi)

}
