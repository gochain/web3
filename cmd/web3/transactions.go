package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/gochain/gochain/v3/accounts/abi"
	"github.com/gochain/gochain/v3/common"
	"github.com/gochain/gochain/v3/core/types"
	"github.com/gochain/web3"
	"github.com/shopspring/decimal"
)

func IncreaseGas(ctx context.Context, privateKey string, network web3.Network, txHash string, amountGwei string) {
	client, err := web3.Dial(network.URL)
	if err != nil {
		fatalExit(fmt.Errorf("Failed to connect to %q: %v", network.URL, err))
	}
	defer client.Close()
	// then we'll clone the original and increase gas
	txOrig, err := client.GetTransactionByHash(ctx, common.HexToHash(txHash))
	if err != nil {
		fatalExit(fmt.Errorf("error on GetTransactionByHash: %v", err))
	}
	if txOrig.BlockNumber != nil {
		fmt.Printf("tx isn't pending, so can't increase gas")
		return
	}
	amount, err := web3.ParseGwei(amountGwei)
	if err != nil {
		fmt.Printf("failed to parse amount %q: %v", amountGwei, err)
		return
	}
	newPrice := new(big.Int).Add(txOrig.GasPrice, amount)
	_ = ReplaceTx(ctx, privateKey, network, txOrig.Nonce, *txOrig.To, txOrig.Value, txOrig.GasLimit, newPrice, txOrig.Input)
	fmt.Printf("Increased gas price to %v\n", newPrice)
}

func ReplaceTx(ctx context.Context, privateKey string, network web3.Network, nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *types.Transaction {
	client, err := web3.Dial(network.URL)
	if err != nil {
		fatalExit(fmt.Errorf("Failed to connect to %q: %v", network.URL, err))
	}
	defer client.Close()
	if gasPrice == nil {
		gasPrice, err = client.GetGasPrice(ctx)
		if err != nil {
			fatalExit(fmt.Errorf("couldn't get suggested gas price: %v", err))
		}
		fmt.Printf("Using suggested gas price: %v\n", gasPrice)
	}
	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, data)
	acct, err := web3.ParsePrivateKey(privateKey)
	if err != nil {
		fatalExit(err)
	}
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, acct.Key())
	if err != nil {
		fatalExit(fmt.Errorf("couldn't sign tx: %v", err))
	}

	err = web3.SendTransaction(ctx, client, signedTx)
	if err != nil {
		fatalExit(fmt.Errorf("error sending transaction: %v", err))
	}
	fmt.Printf("Replaced transaction. New transaction: %s\n", signedTx.Hash().Hex())
	return tx
}

func Transfer(ctx context.Context, rpcURL, privateKey, contractAddress string, wait, toString bool, tail []string) {
	if len(tail) < 3 {
		fatalExit(errors.New("Invalid arguments. Format is: `transfer X to ADDRESS`"))
	}

	// TODO: change this to shopspring/decimal
	amountS := tail[0]
	amountD, err := decimal.NewFromString(amountS)
	if err != nil {
		fatalExit(fmt.Errorf("invalid amount %v", amountS))
	}
	toAddress := tail[2]

	if contractAddress != "" {
		decimals, err := GetContractConst(ctx, rpcURL, contractAddress, "erc20", "decimals")
		if err != nil {
			fatalExit(err)
		}
		// decimals are uint8
		// fmt.Println("DECIMALS:", decimals, reflect.TypeOf(decimals))
		// todo: could get symbol here to display
		amount := web3.DecToInt(amountD, int32(decimals[0].(uint8)))
		callContract(ctx, rpcURL, privateKey, contractAddress, "erc20", "transfer", &big.Int{}, 70000, wait, toString, toAddress, amount)
		return
	}

	amount := web3.DecToInt(amountD, 18)
	client, err := web3.Dial(rpcURL)
	if err != nil {
		fatalExit(fmt.Errorf("Failed to connect to %q: %v", rpcURL, err))
	}
	defer client.Close()
	if toAddress == "" {
		fatalExit(errors.New("The recepient address cannot be empty"))
	}
	if !common.IsHexAddress(toAddress) {
		fatalExit(fmt.Errorf("Invalid to 'address': %s", toAddress))
	}
	address := common.HexToAddress(toAddress)
	tx, err := web3.Send(ctx, client, privateKey, address, amount)
	if err != nil {
		fatalExit(fmt.Errorf("Cannot create transaction: %v", err))
	}
	fmt.Println("Transaction address:", tx.Hash.Hex())
}

func printReceiptDetails(r *web3.Receipt, myabi *abi.ABI) {
	var logs []web3.Event
	var err error
	if myabi != nil {
		logs, err = web3.ParseLogs(*myabi, r.Logs)
		r.ParsedLogs = logs
		if err != nil {
			fmt.Printf("ERROR: Cannot parse the receipt logs: %v\ncontinuing...\n", err)
		}
	}
	switch format {
	case "json":
		fmt.Println(marshalJSON(r))
		return
	}

	fmt.Println("Transaction receipt address:", r.TxHash.Hex())
	fmt.Printf("Block: #%d %s\n", r.BlockNumber, r.BlockHash.Hex())
	fmt.Println("Tx Index:", r.TxIndex)
	fmt.Println("Tx Hash:", r.TxHash.String())
	fmt.Println("From:", r.From.Hex())
	if r.To != nil {
		fmt.Println("To:", r.To.Hex())
	}
	if r.ContractAddress != (common.Address{}) {
		fmt.Println("Contract Address:", r.ContractAddress.String())
	}
	fmt.Println("Gas Used:", r.GasUsed)
	fmt.Println("Cumulative Gas Used:", r.CumulativeGasUsed)
	var status string
	switch r.Status {
	case types.ReceiptStatusFailed:
		status = "Failed"
	case types.ReceiptStatusSuccessful:
		status = "Successful"
	default:
		status = fmt.Sprintf("%d (unrecognized status)", r.Status)
	}
	fmt.Println("Status:", status)
	fmt.Println("Post State:", "0x"+common.Bytes2Hex(r.PostState))
	fmt.Println("Bloom:", "0x"+common.Bytes2Hex(r.Bloom.Bytes()))
	fmt.Println("Logs:", r.Logs)
	if myabi != nil {
		fmt.Println("Parsed Logs:", marshalJSON(r.ParsedLogs))
	}
}

func GetTransactionReceipt(ctx context.Context, rpcURL, txhash, contractFile string) {
	var myabi *abi.ABI
	client, err := web3.Dial(rpcURL)
	if err != nil {
		fatalExit(fmt.Errorf("Failed to connect to %q: %v", rpcURL, err))
	}
	defer client.Close()
	if contractFile != "" {
		myabi, err = web3.GetABI(contractFile)
		if err != nil {
			fatalExit(err)
		}
	}
	r, err := client.GetTransactionReceipt(ctx, common.HexToHash(txhash))
	if err != nil {
		fatalExit(fmt.Errorf("Failed to get transaction receipt: %v", err))
	}
	if verbose {
		fmt.Println("Transaction Receipt Details:")
	}

	printReceiptDetails(r, myabi)
}
