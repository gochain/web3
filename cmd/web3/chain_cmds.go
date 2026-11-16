package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gochain/web3"
	"github.com/urfave/cli/v2"
)

func blockCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:    "block",
		Aliases: []string{"bl"},
		Usage:   "Block details for a block number (decimal) or hash (0x...). Omit for latest.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "tx",
				Usage: "Transaction format: count/hash/detail",
				Value: "count",
			},
			&cli.StringFlag{
				Name:  "input",
				Usage: "Transaction input data format: len/hex/utf8",
				Value: "len",
			},
		},
		Action: func(c *cli.Context) error {
			client, network, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()
			return getBlockDetails(ctx, client, network, c.Args().First(), c.String("tx"), c.String("input"))
		},
	}
}

func getBlockDetails(ctx context.Context, client web3.Client, network web3.Network, numberOrHash, txFormat, txInputFormat string) error {
	var includeTxs bool
	switch txFormat {
	case "detail":
		includeTxs = true
	case "count", "hash":
	default:
		return fmt.Errorf(`unrecognized transaction format %q: must be "count", "hash", or "detail"`, txFormat)
	}

	var block *web3.Block
	var err error
	if strings.HasPrefix(numberOrHash, "0x") || strings.HasPrefix(numberOrHash, "0X") {
		block, err = client.GetBlockByHash(ctx, numberOrHash, includeTxs)
	} else {
		var blockN *big.Int
		if numberOrHash != "" && !strings.EqualFold(numberOrHash, "latest") {
			blockN, err = web3.ParseBigInt(numberOrHash)
			if err != nil {
				return fmt.Errorf("block argument must be a decimal number or 0x hash %q: %w", numberOrHash, err)
			}
		}
		block, err = client.GetBlockByNumber(ctx, blockN, includeTxs)
	}
	if err != nil {
		return fmt.Errorf("cannot get block details: %w", err)
	}

	if isJSONFormat() {
		fmt.Println(marshalJSON(block))
		return nil
	}

	fmt.Println("Number:", block.Number)
	fmt.Println("Time:", block.Timestamp.Format(time.RFC3339))
	fmt.Println("Transactions:", block.TxCount())
	if block.GasLimit > 0 {
		gasPct := big.NewRat(int64(block.GasUsed), int64(block.GasLimit))
		gasPct.Mul(gasPct, big.NewRat(100, 1))
		fmt.Printf("Gas Used: %d/%d (%s%%)\n", block.GasUsed, block.GasLimit, gasPct.FloatString(2))
	} else {
		fmt.Printf("Gas Used: %d\n", block.GasUsed)
	}
	if block.BaseFeePerGas != nil {
		fmt.Println("Base Fee:", web3.WeiAsGwei(block.BaseFeePerGas), "gwei")
	}
	if block.Difficulty != nil {
		fmt.Println("Difficulty:", block.Difficulty)
	}
	if block.TotalDifficulty != nil {
		fmt.Println("Total Difficulty:", block.TotalDifficulty)
	}
	fmt.Println("Hash:", block.Hash.String())
	fmt.Println("Vanity:", block.ExtraVanity())
	fmt.Println("Coinbase:", block.Miner.String())
	fmt.Println("ParentHash:", block.ParentHash.String())
	fmt.Println("UncleHash:", block.Sha3Uncles.String())
	fmt.Println("Nonce:", block.Nonce.Uint64())
	fmt.Println("Root:", block.StateRoot.String())
	fmt.Println("TxHash:", block.TxsRoot.String())
	fmt.Println("ReceiptHash:", block.ReceiptsRoot.String())
	if block.LogsBloom != nil {
		fmt.Println("Bloom:", block.LogsBloom.Hex())
	}
	fmt.Println("MixDigest:", block.MixHash.String())
	if len(block.Signers) > 0 {
		fmt.Println("Signers:", fmtAddresses(block.Signers).String())
	}
	if len(block.Voters) > 0 {
		fmt.Println("Voters:", fmtAddresses(block.Voters).String())
	}
	if len(block.Signer) > 0 {
		fmt.Println("Signer:", web3.EncodeHex(block.Signer))
	}
	if block.TxCount() > 0 {
		switch txFormat {
		case "hash":
			fmt.Println("Transaction Hashes:")
			for i, hash := range block.TxHashes {
				fmt.Printf("\t%d\t%s\n", i, hash.Hex())
			}
		case "detail":
			fmt.Println("Transaction Details:")
			for i, tx := range block.TxDetails {
				toStr := "<Contract Creation>"
				if tx.To != nil {
					toStr = tx.To.Hex()
				}
				fmt.Printf("\t%d\tHash: %s From: %s To: %s Value: %s %s Nonce: %d Gas Limit: %d ",
					i, tx.Hash.Hex(), tx.From.Hex(), toStr, web3.WeiAsBase(tx.Value), network.Unit, tx.Nonce, tx.GasLimit)
				if tx.GasPrice != nil {
					fmt.Printf("Gas Price: %s gwei ", web3.WeiAsGwei(tx.GasPrice))
				}
				if err := printInputData(tx.Input, txInputFormat); err != nil {
					return err
				}
				fmt.Println()
			}
		}
	}
	return nil
}

type fmtAddresses []web3.Address

func (fa fmtAddresses) String() string {
	var b bytes.Buffer
	b.WriteByte('[')
	for i, a := range fa {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(a.Hex())
	}
	b.WriteByte(']')
	return b.String()
}

func transactionCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:    "transaction",
		Aliases: []string{"tx"},
		Usage:   "Transaction details for a tx hash",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "input",
				Usage: "Transaction input data format: len/hex/utf8",
				Value: "len",
			},
		},
		Action: func(c *cli.Context) error {
			txHash := strings.TrimSpace(c.Args().First())
			if txHash == "" {
				return errors.New("missing transaction hash argument")
			}
			client, network, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()
			tx, err := client.GetTransactionByHash(ctx, web3.HexToHash(txHash))
			if err != nil {
				return fmt.Errorf("cannot get transaction details: %w", err)
			}
			if isJSONFormat() {
				fmt.Println(marshalJSON(tx))
				return nil
			}
			fmt.Println("Hash:", tx.Hash.String())
			fmt.Printf("Type: %d (%s)\n", tx.Type, txTypeLabel(tx.Type))
			if tx.ChainID != nil {
				fmt.Println("Chain ID:", tx.ChainID)
			}
			fmt.Println("From:", tx.From.String())
			if tx.To != nil {
				fmt.Println("To:", tx.To.String())
			} else {
				fmt.Println("To: <Contract Creation>")
			}
			fmt.Println("Value:", web3.WeiAsBase(tx.Value), network.Unit)
			fmt.Println("Nonce:", tx.Nonce)
			fmt.Println("Gas Limit:", tx.GasLimit)
			if tx.Type == web3.TxTypeDynamicFee {
				if tx.MaxFeePerGas != nil {
					fmt.Println("Max Fee:", web3.WeiAsGwei(tx.MaxFeePerGas), "gwei")
				}
				if tx.MaxPriorityFeePerGas != nil {
					fmt.Println("Max Priority Fee:", web3.WeiAsGwei(tx.MaxPriorityFeePerGas), "gwei")
				}
			}
			if tx.GasPrice != nil {
				fmt.Println("Gas Price:", web3.WeiAsGwei(tx.GasPrice), "gwei")
			}
			if tx.BlockHash.IsZero() {
				fmt.Println("Pending: true")
			} else {
				fmt.Println("Block Number:", tx.BlockNumber)
				fmt.Println("Block Hash:", tx.BlockHash.String())
			}
			if err := printInputData(tx.Input, c.String("input")); err != nil {
				return err
			}
			fmt.Println()
			return nil
		},
	}
}

func txTypeLabel(t uint8) string {
	switch t {
	case web3.TxTypeLegacy:
		return "Legacy"
	case web3.TxTypeAccessList:
		return "EIP-2930"
	case web3.TxTypeDynamicFee:
		return "EIP-1559"
	default:
		return "Typed"
	}
}

func printInputData(data []byte, inputFmt string) error {
	switch inputFmt {
	case "len":
		fmt.Print("Input Length: ", len(data), " bytes")
	case "hex":
		fmt.Print("Input: ", web3.EncodeHex(data))
	case "utf8":
		fmt.Print("Input: ", string(data))
	default:
		return fmt.Errorf(`unrecognized input data format %q: expected "len", "hex", or "utf8"`, inputFmt)
	}
	return nil
}

func receiptCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:    "receipt",
		Aliases: []string{"rc"},
		Usage:   "Transaction receipt for a tx hash",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "abi",
				Usage: "ABI file, URL, or bundled name (erc20/erc721/erc1155) to decode logs",
			},
		},
		Action: func(c *cli.Context) error {
			txHash := strings.TrimSpace(c.Args().First())
			if txHash == "" {
				return errors.New("missing transaction hash argument")
			}
			client, _, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()

			var myabi *web3.ABI
			if abiSpec := strings.TrimSpace(c.String("abi")); abiSpec != "" {
				myabi, err = web3.GetABI(abiSpec)
				if err != nil {
					return err
				}
			}
			r, err := client.GetTransactionReceipt(ctx, web3.HexToHash(txHash))
			if err != nil {
				return fmt.Errorf("failed to get transaction receipt: %w", err)
			}
			printReceiptDetails(r, myabi)
			return nil
		},
	}
}

func printReceiptDetails(r *web3.Receipt, myabi *web3.ABI) {
	if myabi != nil && r != nil && len(r.Logs) > 0 {
		logs, err := web3.ParseLogs(*myabi, r.Logs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: cannot parse receipt logs: %v\n", err)
		} else {
			r.ParsedLogs = logs
		}
	}
	if isJSONFormat() {
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
	if !r.ContractAddress.IsZero() {
		fmt.Println("Contract Address:", r.ContractAddress.String())
	}
	fmt.Println("Gas Used:", r.GasUsed)
	fmt.Println("Cumulative Gas Used:", r.CumulativeGasUsed)
	if r.EffectiveGasPrice != nil {
		fmt.Println("Effective Gas Price:", web3.WeiAsGwei(r.EffectiveGasPrice), "gwei")
	}
	var status string
	switch r.Status {
	case web3.ReceiptStatusFailed:
		status = "Failed"
	case web3.ReceiptStatusSuccessful:
		status = "Successful"
	default:
		status = fmt.Sprintf("%d (unrecognized status)", r.Status)
	}
	fmt.Println("Status:", status)
	if len(r.PostState) > 0 {
		fmt.Println("Post State:", web3.EncodeHex(r.PostState))
	}
	fmt.Println("Bloom:", r.Bloom.Hex())
	fmt.Println("Logs:", len(r.Logs))
	if myabi != nil && len(r.ParsedLogs) > 0 {
		fmt.Println("Parsed Logs:", marshalJSON(r.ParsedLogs))
	}
}

func addressCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:    "address",
		Aliases: []string{"addr"},
		Usage:   "Account details for an address or WEB3_PRIVATE_KEY",
		Flags: []cli.Flag{
			pkFlag(false),
			&cli.StringFlag{Name: "block", Usage: "Block number (decimal)"},
		},
		Action: func(c *cli.Context) error {
			return runAddressOrBalance(ctx, c, false)
		},
	}
}

func balanceCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "balance",
		Usage: "Get native or ERC20 token balance for an address or WEB3_PRIVATE_KEY",
		Flags: []cli.Flag{
			pkFlag(false),
			&cli.BoolFlag{Name: "erc20", Usage: "Query ERC20 token balance"},
			addressFlag("ERC20 token contract address (when --erc20 is set)"),
			&cli.StringFlag{Name: "block", Usage: "Block number (decimal)"},
		},
		Action: func(c *cli.Context) error {
			return runAddressOrBalance(ctx, c, true)
		},
	}
}

func runAddressOrBalance(ctx context.Context, c *cli.Context, onlyBalance bool) error {
	addrStr := strings.TrimSpace(c.Args().First())
	if addrStr == "" {
		pk := strings.TrimSpace(c.String("private-key"))
		if pk == "" {
			return errors.New("missing address: pass an address argument or set --private-key / WEB3_PRIVATE_KEY")
		}
		acc, err := web3.ParsePrivateKey(pk)
		if err != nil {
			return err
		}
		addrStr = acc.PublicKey()
	} else if !web3.IsHexAddress(addrStr) {
		return fmt.Errorf("invalid hex address %q", addrStr)
	}
	addr := web3.HexToAddress(addrStr)

	client, network, err := dialNetwork(c)
	if err != nil {
		return err
	}
	defer client.Close()

	if onlyBalance && c.Bool("erc20") {
		tokenAddr := strings.TrimSpace(c.String("address"))
		if !web3.IsHexAddress(tokenAddr) {
			return fmt.Errorf("invalid or missing ERC20 contract --address %q", tokenAddr)
		}
		erc20ABI, err := web3.GetABI("erc20")
		if err != nil {
			return err
		}
		decRes, err := web3.CallConstantFunction(ctx, client, *erc20ABI, tokenAddr, "decimals")
		if err != nil {
			return err
		}
		balRes, err := web3.CallConstantFunction(ctx, client, *erc20ABI, tokenAddr, "balanceOf", addr.Hex())
		if err != nil {
			return err
		}
		decimals := int32(decRes[0].(uint8))
		rawBal := balRes[0].(*big.Int)
		humanBal := web3.IntToDec(rawBal, decimals)

		if isJSONFormat() {
			var sym string
			if symRes, sErr := web3.CallConstantFunction(ctx, client, *erc20ABI, tokenAddr, "symbol"); sErr == nil && len(symRes) == 1 {
				sym, _ = symRes[0].(string)
			}
			fmt.Println(marshalJSON(map[string]interface{}{
				"address":  addr.Hex(),
				"token":    web3.HexToAddress(tokenAddr).Hex(),
				"symbol":   sym,
				"decimals": decimals,
				"raw":      rawBal.String(),
				"balance":  humanBal.String(),
			}))
			return nil
		}
		fmt.Println(humanBal)
		return nil
	}

	var blockN *big.Int
	if bStr := strings.TrimSpace(c.String("block")); bStr != "" && !strings.EqualFold(bStr, "latest") {
		blockN, err = web3.ParseBigInt(bStr)
		if err != nil {
			return fmt.Errorf("invalid --block %q: %w", bStr, err)
		}
	}

	bal, err := client.GetBalance(ctx, addr.Hex(), blockN)
	if err != nil {
		return fmt.Errorf("cannot get balance: %w", err)
	}

	if onlyBalance {
		if isJSONFormat() {
			fmt.Println(marshalJSON(map[string]interface{}{
				"address": addr.Hex(),
				"balance": web3.IntToDec(bal, 18).String(),
				"wei":     bal.String(),
				"unit":    network.Unit,
			}))
			return nil
		}
		fmt.Println(web3.IntToDec(bal, 18))
		return nil
	}

	code, err := client.GetCode(ctx, addr.Hex(), blockN)
	if err != nil {
		return fmt.Errorf("cannot get code: %w", err)
	}
	nonce, err := client.GetTransactionCount(ctx, addr, blockN)
	if err != nil {
		return fmt.Errorf("cannot get nonce: %w", err)
	}

	if isJSONFormat() {
		payload := map[string]interface{}{
			"address":     addr.Hex(),
			"balance":     web3.IntToDec(bal, 18).String(),
			"wei":         bal.String(),
			"unit":        network.Unit,
			"nonce":       nonce,
			"is_contract": len(code) > 0,
		}
		if len(code) > 0 {
			payload["code"] = web3.EncodeHex(code)
		}
		fmt.Println(marshalJSON(payload))
		return nil
	}

	fmt.Println("Address:", addr.Hex())
	fmt.Println("Balance:", web3.IntToDec(bal, 18), network.Unit)
	fmt.Println("Nonce:", nonce)
	if len(code) > 0 {
		fmt.Printf("Code (%d bytes): %s\n", len(code), web3.EncodeHex(code))
	}
	return nil
}

func storageCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:      "storage",
		Usage:     "Read a 32-byte contract storage slot (eth_getStorageAt)",
		ArgsUsage: "<address> <slot>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "block", Usage: "Block number (decimal)"},
		},
		Action: func(c *cli.Context) error {
			if c.Args().Len() < 2 {
				return errors.New("usage: web3 storage <address> <slot>")
			}
			addrStr := c.Args().Get(0)
			slotStr := c.Args().Get(1)
			if !web3.IsHexAddress(addrStr) {
				return fmt.Errorf("invalid address %q", addrStr)
			}
			slotHash, err := parseSlotArg(slotStr)
			if err != nil {
				return err
			}
			var blockN *big.Int
			if bStr := strings.TrimSpace(c.String("block")); bStr != "" && !strings.EqualFold(bStr, "latest") {
				blockN, err = web3.ParseBigInt(bStr)
				if err != nil {
					return err
				}
			}
			client, _, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()

			val, err := client.GetStorageAt(ctx, web3.HexToAddress(addrStr), slotHash, blockN)
			if err != nil {
				return err
			}
			if isJSONFormat() {
				fmt.Println(marshalJSON(map[string]interface{}{
					"address": web3.HexToAddress(addrStr).Hex(),
					"slot":    slotHash.Hex(),
					"value":   val.Hex(),
					"uint256": new(big.Int).SetBytes(val[:]).String(),
				}))
				return nil
			}
			fmt.Println(val.Hex())
			return nil
		},
	}
}

func parseSlotArg(s string) (web3.Hash, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		b, err := web3.DecodeHex(s)
		if err != nil || len(b) > 32 {
			return web3.ZeroHash, fmt.Errorf("invalid storage slot hex %q", s)
		}
		return web3.BytesToHash(b), nil
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok || n.Sign() < 0 {
		return web3.ZeroHash, fmt.Errorf("invalid storage slot %q (expected decimal or 0x hex)", s)
	}
	return web3.BytesToHash(n.Bytes()), nil
}

func logsCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "logs",
		Usage: "Query event logs (eth_getLogs) with optional ABI event decoding",
		Flags: []cli.Flag{
			addressFlag("Filter by contract address"),
			&cli.StringFlag{Name: "from-block", Usage: "Start block number (decimal, default: latest)"},
			&cli.StringFlag{Name: "to-block", Usage: "End block number (decimal, default: latest)"},
			&cli.StringSliceFlag{Name: "topic", Usage: "Filter topic0, topic1, ... (0x hash, repeatable)"},
			&cli.StringFlag{Name: "abi", Usage: "ABI file, URL, or bundled name (erc20/erc721/erc1155) to decode events"},
			&cli.StringFlag{Name: "event", Usage: "Filter by event name from --abi"},
		},
		Action: func(c *cli.Context) error {
			return queryAndPrintLogs(ctx, c)
		},
	}
}

func queryAndPrintLogs(ctx context.Context, c *cli.Context) error {
	var q web3.FilterQuery
	if addrStr := strings.TrimSpace(c.String("address")); addrStr != "" {
		if !web3.IsHexAddress(addrStr) {
			return fmt.Errorf("invalid --address %q", addrStr)
		}
		q.Addresses = []web3.Address{web3.HexToAddress(addrStr)}
	}
	if fromStr := strings.TrimSpace(c.String("from-block")); fromStr != "" && !strings.EqualFold(fromStr, "latest") {
		n, err := web3.ParseBigInt(fromStr)
		if err != nil {
			return fmt.Errorf("invalid --from-block: %w", err)
		}
		q.FromBlock = n
	}
	if toStr := strings.TrimSpace(c.String("to-block")); toStr != "" && !strings.EqualFold(toStr, "latest") {
		n, err := web3.ParseBigInt(toStr)
		if err != nil {
			return fmt.Errorf("invalid --to-block: %w", err)
		}
		q.ToBlock = n
	}

	var myabi *web3.ABI
	if abiSpec := strings.TrimSpace(c.String("abi")); abiSpec != "" {
		var err error
		myabi, err = web3.GetABI(abiSpec)
		if err != nil {
			return err
		}
	}
	if evName := strings.TrimSpace(c.String("event")); evName != "" {
		if myabi == nil {
			return errors.New("--event requires --abi")
		}
		ev, ok := myabi.Events[evName]
		if !ok {
			return fmt.Errorf("event %q not found in ABI", evName)
		}
		q.Topics = append(q.Topics, []web3.Hash{ev.ID})
	}
	for _, t := range c.StringSlice("topic") {
		if t == "" {
			q.Topics = append(q.Topics, nil)
		} else {
			q.Topics = append(q.Topics, []web3.Hash{web3.HexToHash(t)})
		}
	}

	client, _, err := dialNetwork(c)
	if err != nil {
		return err
	}
	defer client.Close()

	logs, err := client.GetLogs(ctx, q)
	if err != nil {
		return fmt.Errorf("eth_getLogs failed: %w", err)
	}

	var decoded []web3.Event
	if myabi != nil && len(logs) > 0 {
		ptrs := make([]*web3.Log, len(logs))
		for i := range logs {
			ptrs[i] = &logs[i]
		}
		decoded, err = web3.ParseLogs(*myabi, ptrs)
		if err != nil {
			return err
		}
	}

	if isJSONFormat() {
		if myabi != nil {
			fmt.Println(marshalJSON(map[string]interface{}{
				"logs":   logs,
				"events": decoded,
			}))
		} else {
			fmt.Println(marshalJSON(logs))
		}
		return nil
	}

	fmt.Printf("Found %d log(s)\n", len(logs))
	for i, l := range logs {
		fmt.Printf("[%d] Block #%d Tx %s Address %s\n", i, l.BlockNumber, l.TxHash.Hex(), l.Address.Hex())
		for ti, top := range l.Topics {
			fmt.Printf("    topic[%d]: %s\n", ti, top.Hex())
		}
		if len(l.Data) > 0 {
			fmt.Printf("    data:     %s\n", web3.EncodeHex(l.Data))
		}
	}
	if len(decoded) > 0 {
		fmt.Println("Decoded Events:")
		fmt.Println(marshalJSON(decoded))
	}
	return nil
}

func snapshotCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:    "snapshot",
		Aliases: []string{"sn"},
		Usage:   "Clique consensus snapshot",
		Action: func(c *cli.Context) error {
			client, _, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()
			s, err := client.GetSnapshot(ctx)
			if err != nil {
				return fmt.Errorf("cannot get snapshot: %w", err)
			}
			if isJSONFormat() {
				fmt.Println(marshalJSON(s))
				return nil
			}
			fmt.Println("Latest Number:", s.Number)
			fmt.Println("Latest Hash:", s.Hash.String())
			fmt.Println("Signers:")
			type signer struct {
				addr web3.Address
				num  uint64
			}
			signers := make([]signer, 0, len(s.Signers))
			for addr, num := range s.Signers {
				signers = append(signers, signer{addr, num})
			}
			sort.Slice(signers, func(i, j int) bool {
				return signers[j].num < signers[i].num
			})
			for _, si := range signers {
				fmt.Println("", si.addr.String(), "signed block", si.num, "-", s.Number-si.num, "blocks ago")
			}
			fmt.Println("Voters:")
			for addr := range s.Voters {
				fmt.Println("", addr.String())
			}
			if len(s.Votes) > 0 {
				fmt.Println("Votes:")
				for _, vote := range s.Votes {
					pre := "un"
					if vote.Authorize {
						pre = ""
					}
					fmt.Printf("\t%d: signer %s voted to %sauthorize %s\n", vote.Block, vote.Signer, pre, vote.Address)
				}
				fmt.Println("Tally:")
				for addr, tally := range s.Tally {
					str := "unauthorize"
					if tally.Authorize {
						str = "authorize"
					}
					fmt.Println("", addr.String(), str, tally.Votes)
				}
			}
			return nil
		},
	}
}

func idCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "id",
		Usage: "Network/Chain ID and genesis hash information",
		Action: func(c *cli.Context) error {
			client, _, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()
			id, err := client.GetID(ctx)
			if err != nil {
				return fmt.Errorf("cannot get network ID info: %w", err)
			}
			if isJSONFormat() {
				fmt.Println(marshalJSON(id))
				return nil
			}
			fmt.Println("Network ID:", id.NetworkID)
			fmt.Println("Chain ID:", id.ChainID)
			fmt.Println("Genesis Hash:", id.GenesisHash.String())
			return nil
		},
	}
}

func envCommand() *cli.Command {
	return &cli.Command{
		Name:  "env",
		Usage: "List WEB3_* environment variables",
		Action: func(c *cli.Context) error {
			varNames := []string{addrVarName, networkVarName, pkVarName, rpcURLVarName}
			sort.Strings(varNames)
			if isJSONFormat() {
				m := make(map[string]string, len(varNames))
				for _, name := range varNames {
					m[name] = os.Getenv(name)
				}
				fmt.Println(marshalJSON(m))
				return nil
			}
			for _, name := range varNames {
				fmt.Printf("%s=%s\n", name, os.Getenv(name))
			}
			return nil
		},
	}
}
