package web3_actions

import (
	"context"
	"fmt"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/rs/zerolog/log"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

func PrintReceiptDetails(ctx context.Context, r *web3_types.Receipt, myabi *abi.ABI) error {
	var logs []web3_types.Event
	var err error
	if myabi != nil && r != nil && r.Logs != nil {
		logs, err = ParseLogs(*myabi, r.Logs)
		r.ParsedLogs = logs
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("PrintReceiptDetails: ParseLogs")
			fmt.Printf("ERROR: Cannot parse the receipt logs: %v\ncontinuing...\n", err)
			return err
		}
	}
	switch format {
	case "json":
		s, jerr := marshalJSON(ctx, r)
		if jerr != nil {
			log.Ctx(ctx).Err(jerr).Msg("PrintReceiptDetails: marshalJSON")
			return jerr
		}
		fmt.Println(s)
		return nil
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
		b, merr := marshalJSON(context.Background(), r.ParsedLogs)
		if merr != nil {
			log.Ctx(ctx).Err(merr).Msg("PrintReceiptDetails: marshalJSON")
			return merr
		}
		fmt.Println("Parsed Logs:", b)
	}
	return err
}
