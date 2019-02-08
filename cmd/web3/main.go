package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gochain-io/gochain/v3/accounts/abi"
	"github.com/gochain-io/gochain/v3/common"
	"github.com/urfave/cli"

	"github.com/gochain-io/web3"
)

// Flags
var (
	verbose bool
	format  string
)

func main() {
	// Interrupt cancellation.
	ctx, cancelFn := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	defer close(sigCh)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for range sigCh {
			cancelFn()
		}
	}()

	// Flags
	var netName, rpcUrl, function, contractAddress, contractFile, privateKey string
	var amount int
	var testnet bool

	app := cli.NewApp()
	app.Name = "web3"
	app.Version = Version
	app.Usage = "web3 cli tool"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "network, n",
			Usage:       "The name of the network (mainnet/testnet/ethereum/ropsten/localhost). Default is mainnet.",
			Destination: &netName,
			EnvVar:      "WEB3_NETWORK",
			Hidden:      false},
		cli.BoolFlag{
			Name:        "testnet",
			Usage:       "Shorthand for '-network testnet'.",
			Destination: &testnet,
			Hidden:      false},
		cli.StringFlag{
			Name:        "rpc-url",
			Usage:       "The network RPC URL",
			Destination: &rpcUrl,
			EnvVar:      "WEB3_RPC_URL",
			Hidden:      false},
		cli.BoolFlag{
			Name:        "verbose",
			Usage:       "Enable verbose logging",
			Destination: &verbose,
			Hidden:      false},
		cli.StringFlag{
			Name:        "format, f",
			Usage:       "Output format (json). Default is human readable log lines.",
			Destination: &format,
			Hidden:      false},
	}
	var network web3.Network
	app.Before = func(*cli.Context) error {
		network = getNetwork(netName, rpcUrl, testnet)
		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:    "block",
			Usage:   "Block details for a block number (decimal integer) or hash (hexadecimal with 0x prefix). Omit for latest.",
			Aliases: []string{"bl"},
			Action: func(c *cli.Context) {
				GetBlockDetails(ctx, network.URL, c.Args().First())
			},
		},
		{
			Name:    "transaction",
			Aliases: []string{"tx"},
			Usage:   "Transaction details for a tx hash",
			Action: func(c *cli.Context) {
				GetTransactionDetails(ctx, network, c.Args().First())
			},
		},
		{
			Name:    "receipt",
			Aliases: []string{"rc"},
			Usage:   "Transaction receipt for a tx hash",
			Action: func(c *cli.Context) {
				GetTransactionReceipt(ctx, network.URL, c.Args().First())
			},
		},
		{
			Name:    "address",
			Aliases: []string{"addr"},
			Usage:   "Address details",
			Action: func(c *cli.Context) {
				GetAddressDetails(ctx, network, c.Args().First())
			},
		},
		{
			Name:    "contract",
			Aliases: []string{"c"},
			Usage:   "Contract operations",
			Subcommands: []cli.Command{
				{
					Name:  "build",
					Usage: "Build the specified contract",
					Action: func(c *cli.Context) {
						BuildSol(ctx, c.Args().First())
					},
				},
				{
					Name:  "deploy",
					Usage: "Build and deploy the specified contract to the network",
					Action: func(c *cli.Context) {
						DeploySol(ctx, network.URL, privateKey, c.Args().First())
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "private-key",
							Usage:       "The private key",
							EnvVar:      "WEB3_PRIVATE_KEY",
							Destination: &privateKey,
							Hidden:      false},
					},
				},
				{
					Name:  "list",
					Usage: "List contract functions",
					Action: func(c *cli.Context) {
						ListContract(contractFile)
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "abi",
							Destination: &contractFile,
							Usage:       "The abi file of the deployed contract",
							Hidden:      false},
					},
				},
				{
					Name:  "call",
					Usage: "Call contract function",
					Action: func(c *cli.Context) {
						args := make([]interface{}, len(c.Args()))
						for i, v := range c.Args() {
							args[i] = v
						}
						CallContract(ctx, network.URL, privateKey, contractAddress, contractFile, function, amount, args...)
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "function",
							Usage:       "Target function name",
							Destination: &function,
							Hidden:      false},
						cli.StringFlag{
							Name:        "address",
							Destination: &contractAddress,
							Usage:       "Deployed contract address",
							Hidden:      false},
						cli.StringFlag{
							Name:        "abi",
							Destination: &contractFile,
							Usage:       "ABI file matching deployed contract",
							Hidden:      false},
						cli.IntFlag{
							Name:        "amount",
							Destination: &amount,
							Usage:       "Amount in wei that you want to send to the transaction",
							Hidden:      false},
						cli.StringFlag{
							Name:        "private-key",
							Usage:       "Private key",
							EnvVar:      "WEB3_PRIVATE_KEY",
							Destination: &privateKey,
							Hidden:      false},
					},
				},
			},
		},
		{
			Name:    "snapshot",
			Aliases: []string{"sn"},
			Usage:   "Clique snapshot",
			Action: func(c *cli.Context) {
				GetSnapshot(ctx, network.URL)
			},
		},
		{
			Name:    "id",
			Aliases: []string{"id"},
			Usage:   "Network/Chain information",
			Action: func(c *cli.Context) {
				GetID(ctx, network.URL)
			},
		},
	}
	app.Run(os.Args)
}

// getNetwork resolves the rpcUrl from the user specified options, or quits if an illegal combination or value is found.
func getNetwork(name, rpcURL string, testnet bool) web3.Network {
	var network web3.Network
	if rpcURL != "" {
		if name != "" {
			log.Fatalf("Cannot set both rpcURL %q and network %q", rpcURL, network)
		}
		if testnet {
			log.Fatalf("Cannot set both rpcURL %q and testnet", rpcURL)
		}
		network.URL = rpcURL
		network.Unit = "GO"
	} else {
		if testnet {
			if name != "" {
				log.Fatalf("Cannot set both network %q and testnet", network)
			}
			name = "testnet"
		} else if name == "" {
			name = "mainnet"
		}
		var ok bool
		network, ok = web3.Networks[name]
		if !ok {
			log.Fatal("Unrecognized network:", name)
		}
		if verbose {
			log.Printf("Network: %v", name)
		}
	}
	if verbose {
		log.Println("Network Info:", network)
	}
	return network
}

func parseBigInt(value string) (*big.Int, error) {
	if value == "" {
		return nil, nil
	}
	i, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, errors.New("failed to parse integer")
	}
	return i, nil
}

func GetBlockDetails(ctx context.Context, rpcURL, numberOrHash string) {
	client, err := web3.NewClient(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", rpcURL, err)
	}
	defer client.Close()
	var block *web3.Block
	if strings.HasPrefix(numberOrHash, "0x") {
		var err error
		block, err = client.GetBlockByHash(ctx, numberOrHash, false)
		if err != nil {
			log.Fatalf("Cannot get block details from the network: %v", err)
		}
	} else {
		blockN, err := parseBigInt(numberOrHash)
		if err != nil {
			log.Fatalf("Block argument must be a number (decimal integer) or hash (hexadecimal with 0x prefix) %q: %v", numberOrHash, err)
		}
		block, err = client.GetBlockByNumber(ctx, blockN, false)
		if err != nil {
			log.Fatalf("Cannot get block details from the network: %v", err)
		}
	}
	if verbose {
		log.Println("Block details:")
	}
	switch format {
	case "json":
		fmt.Println(marshalJSON(block))
		return
	}

	fmt.Println("Number:", block.Number)
	fmt.Println("Time:", block.Timestamp.Format(time.RFC3339))
	fmt.Println("Transactions:", len(block.Txs))
	gasPct := big.NewRat(int64(block.GasUsed), int64(block.GasLimit))
	gasPct = gasPct.Mul(gasPct, big.NewRat(100, 1))
	fmt.Printf("Gas Used: %d/%d (%s%%)\n", block.GasUsed, block.GasLimit, gasPct.FloatString(2))
	fmt.Println("Difficulty:", block.Difficulty)
	fmt.Println("Total Difficulty:", block.TotalDifficulty)
	fmt.Println("Hash:", block.Hash.String())
	fmt.Println("Vanity:", block.ExtraVanity())
	fmt.Println("Coinbase:", block.Miner.String())
	fmt.Println("ParentHash:", block.ParentHash.String())
	fmt.Println("UncleHash:", block.Sha3Uncles.String())
	fmt.Println("Nonce:", block.Nonce.Uint64())
	fmt.Println("Root:", block.StateRoot.String())
	fmt.Println("TxHash:", block.TxsRoot.String())
	fmt.Println("ReceiptHash:", block.ReceiptsRoot.String())
	fmt.Println("Bloom:", "0x"+common.Bytes2Hex(block.LogsBloom.Bytes()))
	fmt.Println("MixDigest:", block.MixHash.String())
	if len(block.Signers) > 0 {
		fmt.Println("Signers:", fmtAddresses(block.Signers).String())
	}
	if len(block.Voters) > 0 {
		fmt.Println("Voters:", fmtAddresses(block.Voters).String())
	}
	if len(block.Signer) > 0 {
		fmt.Println("Signer:", "0x"+common.Bytes2Hex(block.Signer))
	}
}

type fmtAddresses []common.Address

func (fa fmtAddresses) String() string {
	var b bytes.Buffer
	fmt.Fprint(&b,"[")
	for i, a := range fa {
		if i > 0 {
			fmt.Fprint(&b, ", ")
		}
		fmt.Fprint(&b, a.Hex())
	}
	fmt.Fprint(&b,"]")
	return b.String()
}

func GetTransactionDetails(ctx context.Context, network web3.Network, txhash string) {
	client, err := web3.NewClient(network.URL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", network.URL, err)
	}
	defer client.Close()
	tx, err := client.GetTransactionByHash(ctx, common.HexToHash(txhash))
	if err != nil {
		log.Fatalf("Cannot get transaction details from the network: %v", err)
	}
	if verbose {
		fmt.Println("Transaction details:")
	}

	switch format {
	case "json":
		fmt.Println(marshalJSON(tx))
		return
	}

	fmt.Println("Hash:", tx.Hash.String())
	fmt.Println("From:", tx.From.String())
	fmt.Println("To:", tx.To.String())
	fmt.Println("Value:", web3.WeiAsBase(tx.Value), network.Unit)
	fmt.Println("Nonce:", uint64(tx.Nonce))
	fmt.Println("Gas Limit:", tx.GasLimit)
	fmt.Println("Gas Price:", web3.WeiAsGwei(tx.GasPrice), "gwei")
	if tx.BlockHash == (common.Hash{}) {
		fmt.Println("Pending: true")
	} else {
		fmt.Println("Block Number:", tx.BlockNumber)
		fmt.Println("Block Hash:", tx.BlockHash.String())
	}
}

func GetTransactionReceipt(ctx context.Context, rpcURL, txhash string) {
	client, err := web3.NewClient(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", rpcURL, err)
	}
	defer client.Close()
	r, err := client.GetTransactionReceipt(ctx, common.HexToHash(txhash))
	if err != nil {
		log.Fatalf("Failed to get transaction receipt: %v", err)
	}
	if verbose {
		fmt.Println("Transaction Receipt Details:")
	}

	switch format {
	case "json":
		fmt.Println(marshalJSON(r))
		return
	}

	fmt.Println("TxHash:", r.TxHash.String())
	if r.ContractAddress != (common.Address{}) {
		fmt.Println("Contract Address:", r.ContractAddress.String())
	}
	fmt.Println("GasUsed:", r.GasUsed)
	fmt.Println("Cumulative Gas Used:", r.CumulativeGasUsed)
	fmt.Println("Status:", r.Status)
	fmt.Println("Post State:", "0x"+common.Bytes2Hex(r.PostState))
	fmt.Println("Bloom:", "0x"+common.Bytes2Hex(r.Bloom.Bytes()))
	fmt.Println("Logs:", r.Logs)
}

func GetAddressDetails(ctx context.Context, network web3.Network, addrHash string) {
	client, err := web3.NewClient(network.URL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", network.URL, err)
	}
	defer client.Close()
	bal, err := client.GetBalance(ctx, addrHash, nil)
	if err != nil {
		log.Fatalf("Cannot get address balance from the network: %v", err)
	}
	code, err := client.GetCode(ctx, addrHash, nil)
	if err != nil {
		log.Fatalf("Cannot get address code from the network: %v", err)
	}
	if verbose {
		log.Println("Address details:")
	}

	switch format {
	case "json":
		data := struct {
			Balance *big.Int `json:"balance"`
			Code    *string  `json:"code"`
		}{Balance: bal}
		if len(code) > 0 {
			sc := string(code)
			data.Code = &sc
		}
		fmt.Println(marshalJSON(&data))
		return
	}

	fmt.Println("Balance:", web3.WeiAsBase(bal), network.Unit)
	if len(code) > 0 {
		fmt.Println("Code:", string(code))
	}
}

func GetSnapshot(ctx context.Context, rpcURL string) {
	client, err := web3.NewClient(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", rpcURL, err)
	}
	defer client.Close()
	s, err := client.GetSnapshot(ctx)
	if err != nil {
		log.Fatalf("Cannot get snapshot from the network: %v", err)
	}
	if verbose {
		log.Println("Snapshot details:")
	}

	switch format {
	case "json":
		fmt.Println(marshalJSON(s))
		return
	}

	fmt.Println("Latest Number:", s.Number)
	fmt.Println("Latest Hash:", s.Hash.String())
	fmt.Println("Signers:")
	type signer struct {
		addr common.Address
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
		//TODO mark signers which have fallen behind
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
			fmt.Printf("\t%d: signer %s voted to %sauthorize %s", vote.Block, vote.Signer, pre, vote.Address)
		}
		fmt.Println("Tally:", s.Tally)
		for addr, tally := range s.Tally {
			str := "unauthorize"
			if tally.Authorize {
				str = str[2:]
			}
			fmt.Println("", addr.String(), str, tally.Votes)
		}
	}
}

func GetID(ctx context.Context, rpcURL string) {
	client, err := web3.NewClient(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", rpcURL, err)
	}
	defer client.Close()
	id, err := client.GetID(ctx)
	if err != nil {
		log.Fatalf("Cannot get id info from the network: %v", err)
	}
	if verbose {
		log.Println("Snapshot details:")
	}
	switch format {
	case "json":
		fmt.Println(marshalJSON(id))
		return
	}
	fmt.Println("Network ID:", id.NetworkID)
	fmt.Println("Chain ID:", id.ChainID)
	fmt.Println("Genesis Hash:", id.GenesisHash.String())
}

func BuildSol(ctx context.Context, filename string) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read file %q: %v", filename, err)
	}
	str := string(b) // convert content to a 'string'
	if verbose {
		log.Println("Building Sol:", str)
	}
	compileData, err := web3.CompileSolidityString(ctx, str)
	if err != nil {
		log.Fatalf("Failed to compile %q: %v", filename, err)
	}
	if verbose {
		log.Println("Compiled Sol Details:", marshalJSON(compileData))
	}

	var filenames []string
	for contractName, v := range compileData {
		filename := contractName[8:]
		err := ioutil.WriteFile(filename+".bin", []byte(v.Code), 0600)
		if err != nil {
			log.Fatalf("Cannot write the bin file: %v", err)
		}
		err = ioutil.WriteFile(filename+".abi", []byte(marshalJSON(v.Info.AbiDefinition)), 0600)
		if err != nil {
			log.Fatalf("Cannot write the abi file: %v", err)
		}
		filenames = append(filenames, filename)
	}

	switch format {
	case "json":
		data := struct {
			Bin []string `json:"bin"`
			ABI []string `json:"abi"`
		}{}
		for _, f := range filenames {
			data.Bin = append(data.Bin, f+".bin")
			data.ABI = append(data.ABI, f+".abi")
		}
		fmt.Println(marshalJSON(data))
		return
	}

	fmt.Println("Successfully compiled contracts and wrote the following files:")
	for _, filename := range filenames {
		fmt.Println("", filename+".bin,", filename+".abi")
	}
}

func DeploySol(ctx context.Context, rpcURL, privateKey, contractName string) {
	client, err := web3.NewClient(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", rpcURL, err)
	}
	defer client.Close()
	if _, err := os.Stat(contractName); os.IsNotExist(err) {
		log.Fatalf("Cannot find the bin file: %v", err)
	}
	dat, err := ioutil.ReadFile(contractName)
	if err != nil {
		log.Fatalf("Cannot read the bin file: %v", err)
	}
	tx, err := web3.DeployContract(ctx, client, privateKey, string(dat))
	if err != nil {
		log.Fatalf("Cannot deploy the contract: %v", err)
	}
	receipt, err := web3.WaitForReceipt(ctx, client, tx)
	if err != nil {
		log.Fatalf("Cannot get the receipt: %v", err)
	}

	switch format {
	case "json":
		fmt.Println(marshalJSON(receipt))
		return
	}

	fmt.Println("Contract has been successfully deployed with transaction:", tx.Hash.Hex())
	fmt.Println("Contract address is:", receipt.ContractAddress.Hex())
}
func ListContract(contractFile string) {
	if _, err := os.Stat(contractFile); os.IsNotExist(err) {
		log.Fatalf("Cannot find the abi file: %v", err)
	}
	jsonReader, err := os.Open(contractFile)
	if err != nil {
		log.Fatalf("Cannot read the abi file: %v", err)
	}
	myabi, err := abi.JSON(jsonReader)
	if err != nil {
		log.Fatalf("Cannot initialize ABI: %v", err)
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

func CallContract(ctx context.Context, rpcURL, privateKey, contractAddress, contractFile, functionName string, amount int, parameters ...interface{}) {
	client, err := web3.NewClient(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", rpcURL, err)
	}
	defer client.Close()
	if _, err := os.Stat(contractFile); os.IsNotExist(err) {
		log.Fatalf("Cannot find the abi file: %v", err)
	}
	jsonReader, err := os.Open(contractFile)
	if err != nil {
		log.Fatalf("Cannot read the abi file: %v", err)
	}
	myabi, err := abi.JSON(jsonReader)
	if err != nil {
		log.Fatalf("Cannot initialize ABI: %v", err)
	}
	if _, ok := myabi.Methods[functionName]; ok {
		if myabi.Methods[functionName].Const {
			res, err := web3.CallConstantFunction(ctx, client, myabi, contractAddress, functionName, parameters...)
			if err != nil {
				log.Fatalf("Cannot call the contract: %v", err)
			}
			switch format {
			case "json":
				m := make(map[string]interface{})
				m["response"] = res
				fmt.Println(marshalJSON(m))
				return
			}
			fmt.Println("Call results:", res)
		} else {
			tx, err := web3.CallTransactFunction(ctx, client, myabi, contractAddress, privateKey, functionName, amount, parameters...)
			if err != nil {
				log.Fatalf("Cannot call the contract: %v", err)
			}
			receipt, err := web3.WaitForReceipt(ctx, client, tx)
			if err != nil {
				log.Fatalf("Cannot get the receipt: %v", err)
			}
			fmt.Println("Transaction address:", receipt.TxHash.Hex())
		}

	} else {
		fmt.Println("There is no such function:", functionName)
	}
}

func marshalJSON(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("Cannot marshal json: %v", err)
	}
	return string(b)
}
