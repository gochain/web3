package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gochain-io/gochain/v3/accounts/abi"
	"github.com/gochain-io/gochain/v3/common"
	"github.com/gochain-io/gochain/v3/common/hexutil"
	"github.com/gochain-io/gochain/v3/core/types"
	"github.com/gochain-io/web3"
	"github.com/gochain-io/web3/assets"
	"github.com/urfave/cli"
)

// Flags
var (
	verbose bool
	format  string
)

const (
	asciiLogo = `  ___  _____  ___  _   _    __    ____  _  _ 
 / __)(  _  )/ __)( )_( )  /__\  (_  _)( \( )
( (_-. )(_)(( (__  ) _ (  /(__)\  _)(_  )  ( 
 \___/(_____)\___)(_) (_)(__)(__)(____)(_)\_)`

	pkVarName      = "WEB3_PRIVATE_KEY"
	addrVarName    = "WEB3_ADDRESS"
	networkVarName = "WEB3_NETWORK"
	rpcURLVarName  = "WEB3_RPC_URL"
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
	var netName, rpcUrl, function, contractAddress, contractFile, privateKey, txFormat, txInputFormat, recepientAddress string
	var amount int
	var testnet, waitForReceipt bool

	app := cli.NewApp()
	app.Name = "web3"
	app.Version = Version
	app.Usage = "web3 cli tool"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "network, n",
			Usage:       `The name of the network. Options: gochain/testnet/ethereum/ropsten/localhost. (default: "gochain")`,
			Destination: &netName,
			EnvVar:      networkVarName,
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
			EnvVar:      rpcURLVarName,
			Hidden:      false},
		cli.BoolFlag{
			Name:        "verbose",
			Usage:       "Enable verbose logging",
			Destination: &verbose,
			Hidden:      false},
		cli.StringFlag{
			Name:        "format, f",
			Usage:       "Output format. Options: json. Default: human readable output.",
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
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "tx",
					Usage:       "Transaction format: count/hash/detail",
					Destination: &txFormat,
					Value:       "count",
				},
				cli.StringFlag{
					Name:        "input",
					Usage:       "Transaction input data format: len/hex/utf8",
					Destination: &txInputFormat,
					Value:       "len",
				},
			},
			Action: func(c *cli.Context) {
				GetBlockDetails(ctx, network, c.Args().First(), txFormat, txInputFormat)
			},
		},
		{
			Name:    "transaction",
			Aliases: []string{"tx"},
			Usage:   "Transaction details for a tx hash",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "input",
					Usage:       "Transaction input data format: len/hex/utf8",
					Destination: &txInputFormat,
					Value:       "len",
				},
			},
			Action: func(c *cli.Context) {
				GetTransactionDetails(ctx, network, c.Args().First(), txInputFormat)
			},
		},
		{
			Name:    "receipt",
			Aliases: []string{"rc"},
			Usage:   "Transaction receipt for a tx hash",
			Action: func(c *cli.Context) {
				GetTransactionReceipt(ctx, network.URL, c.Args().First(), contractFile)
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "abi",
					Destination: &contractFile,
					Usage:       "ABI file matching deployed contract",
					Hidden:      false},
			},
		},
		{
			Name:    "address",
			Aliases: []string{"addr"},
			Usage:   "Account details for a specific address, or the one corresponding to the private key.",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "private-key, pk",
					Usage:       "The private key",
					EnvVar:      pkVarName,
					Destination: &privateKey,
					Hidden:      false},
			},
			Action: func(c *cli.Context) {
				GetAddressDetails(ctx, network, c.Args().First(), privateKey)
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
							Name:        "private-key, pk",
							Usage:       "The private key",
							EnvVar:      pkVarName,
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
						CallContract(ctx, network.URL, privateKey, contractAddress, contractFile, function, amount, waitForReceipt, args...)
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "function",
							Usage:       "Target function name",
							Destination: &function,
							Hidden:      false},
						cli.StringFlag{
							Name:        "address",
							EnvVar:      addrVarName,
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
							Name:        "private-key, pk",
							Usage:       "Private key",
							EnvVar:      pkVarName,
							Destination: &privateKey,
							Hidden:      false},
						cli.BoolFlag{
							Name:        "wait",
							Usage:       "Wait for the receipt for transact functions",
							Destination: &waitForReceipt,
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
		{
			Name:  "start",
			Usage: "Start a local GoChain development node",
			Flags: []cli.Flag{
				cli.BoolTFlag{
					Name:  "detach, d",
					Usage: "Run container in background.",
				},
				cli.StringFlag{
					Name:  "env-file",
					Usage: "Path to custom configuration file.",
				},
				cli.StringFlag{
					Name:   "private-key,pk",
					Usage:  "Private key",
					EnvVar: pkVarName,
				},
			},
			Action: func(c *cli.Context) error {
				return start(ctx, c)
			},
		},
		{
			Name:  "myaddress",
			Usage: fmt.Sprintf("Returns the address associated with %v", pkVarName),
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "private-key,pk",
					Usage:  "Private key",
					EnvVar: pkVarName,
				},
			},
			Action: func(c *cli.Context) {
				pk := c.String("private-key")
				if pk == "" {
					fmt.Printf("%v not set", pkVarName)
					return
				}
				acc, err := web3.ParsePrivateKey(pk)
				if err != nil {
					log.Fatalln(err)
				}
				fmt.Print(acc.PublicKey())
			},
		},
		{
			Name:    "account",
			Aliases: []string{"a"},
			Usage:   "Account operations",
			Subcommands: []cli.Command{
				{
					Name:  "create",
					Usage: "Create a new account",
					Action: func(c *cli.Context) {
						acc, err := web3.CreateAccount()
						if err != nil {
							log.Fatalln(err)
						}
						fmt.Printf("Private key: %v\n", acc.PrivateKey())
						fmt.Printf("Public address: %v\n", acc.PublicKey())
					},
				},
			},
		},
		{
			Name:    "send",
			Usage:   fmt.Sprintf("Transfer GO to an account (web3 send -to 0xb 10go/eth/nanogo/gwei/attogo/wei)"),
			Aliases: []string{"transfer"},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "private-key,pk",
					Usage:       "Private key",
					EnvVar:      pkVarName,
					Destination: &privateKey,
					Hidden:      false,
				},
				cli.StringFlag{
					Name:        "to",
					EnvVar:      addrVarName,
					Destination: &recepientAddress,
					Usage:       "The recepient address",
					Hidden:      false},
			},
			Action: func(c *cli.Context) {
				Send(ctx, network.URL, privateKey, recepientAddress, c.Args().First())
			},
		},
		{
			Name:  "env",
			Usage: "List environment variables",
			Action: func(c *cli.Context) {
				varNames := []string{addrVarName, pkVarName, networkVarName, rpcURLVarName}
				sort.Strings(varNames)
				for _, name := range varNames {
					fmt.Printf("%s=%s\n", name, os.Getenv(name))
				}
			},
		},
		{
			Name:    "generate",
			Usage:   "Generate an ABI code",
			Aliases: []string{"g"},
			Subcommands: []cli.Command{
				{
					Name:    "contract",
					Usage:   "Generate a contract",
					Aliases: []string{"c"},
					Subcommands: []cli.Command{
						{
							Name:  "erc20",
							Usage: "Generate a erc20 contract",
							Flags: []cli.Flag{
								cli.BoolFlag{
									Name:  "pausable, p",
									Usage: "Pausable contract.",
								},
								cli.BoolFlag{
									Name:  "mintable, m",
									Usage: "Mintable contract.",
								},
								cli.BoolFlag{
									Name:  "burnable, b",
									Usage: "Burnable contract.",
								},
								cli.StringFlag{
									Name:  "symbol, s",
									Usage: "Token Symbol.",
								},
								cli.StringFlag{
									Name:  "name, n",
									Usage: "Token Name",
								},
								cli.StringFlag{
									Name:  "capped, c",
									Usage: "Cap, total supply(in GO/ETH)",
								},
								cli.IntFlag{
									Name:  "decimals, d",
									Usage: "Decimals",
									Value: 18,
								},
							},
							Action: func(c *cli.Context) {
								GenerateContract(ctx, "erc20", c)
							},
						},
						{
							Name:  "erc721",
							Usage: "Generate a erc721 contract",
							Flags: []cli.Flag{
								cli.BoolFlag{
									Name:  "pausable, p",
									Usage: "Pausable contract.",
								},
								cli.BoolFlag{
									Name:  "mintable, m",
									Usage: "Mintable contract.",
								},
								cli.BoolFlag{
									Name:  "burnable, b",
									Usage: "Burnable contract.",
								},
								cli.BoolFlag{
									Name:  "metadata-mintable, mm",
									Usage: "Contract with a mintable metadata.",
								},
								cli.StringFlag{
									Name:  "symbol, s",
									Usage: "Token Symbol.",
								},
								cli.StringFlag{
									Name:  "name, n",
									Usage: "Token Name",
								},
							},
							Action: func(c *cli.Context) {
								GenerateContract(ctx, "erc721", c)
							},
						},
					},
				},
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
				log.Fatalf("Cannot set both network %q and testnet", name)
			}
			name = "testnet"
		} else if name == "" {
			name = "gochain"
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

func GetBlockDetails(ctx context.Context, network web3.Network, numberOrHash string, txFormat, txInputFormat string) {
	client, err := web3.NewClient(network.URL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", network.URL, err)
	}
	defer client.Close()
	var block *web3.Block
	var includeTxs bool
	switch txFormat {
	case "detail":
		includeTxs = true
	case "count", "hash":
	default:
		log.Fatalf(`Unrecognized transaction format %q: must be "count", "hash", or "detail"`, txFormat)
	}
	if strings.HasPrefix(numberOrHash, "0x") {
		var err error
		block, err = client.GetBlockByHash(ctx, numberOrHash, includeTxs)
		if err != nil {
			log.Fatalf("Cannot get block details from the network: %v", err)
		}
	} else {
		blockN, err := web3.ParseBigInt(numberOrHash)
		if err != nil {
			log.Fatalf("Block argument must be a number (decimal integer) or hash (hexadecimal with 0x prefix) %q: %v", numberOrHash, err)
		}
		block, err = client.GetBlockByNumber(ctx, blockN, includeTxs)
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
	fmt.Println("Transactions:", block.TxCount())
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
				fmt.Printf("\t%d\t", i)
				fmt.Print("Hash: ", tx.Hash.Hex())
				fmt.Print(" From: ", tx.From.Hex())
				fmt.Print(" To: ", tx.To.Hex())
				fmt.Print(" Value: ", web3.WeiAsBase(tx.Value), " ", network.Unit)
				fmt.Print(" Nonce: ", tx.Nonce)
				fmt.Print(" Gas Limit: ", tx.GasLimit)
				fmt.Print(" Gas Price: ", web3.WeiAsGwei(tx.GasPrice), " gwei")
				fmt.Print(" ")
				printInputData(tx.Input, txInputFormat)
				fmt.Println()
			}
		}
	}
}

type fmtAddresses []common.Address

func (fa fmtAddresses) String() string {
	var b bytes.Buffer
	fmt.Fprint(&b, "[")
	for i, a := range fa {
		if i > 0 {
			fmt.Fprint(&b, ", ")
		}
		fmt.Fprint(&b, a.Hex())
	}
	fmt.Fprint(&b, "]")
	return b.String()
}

func GetTransactionDetails(ctx context.Context, network web3.Network, txhash, inputFormat string) {
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
	if tx.To != nil {
		fmt.Println("To:", tx.To.String())
	}
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
	printInputData(tx.Input, inputFormat)
	fmt.Println()
}

func printInputData(data []byte, format string) {
	switch format {
	case "len":
		fmt.Print("Input Length: ", len(data), " bytes")
	case "hex":
		fmt.Print("Input: ", hexutil.Encode(data))
	case "utf8":
		fmt.Print("Input: ", string(data))
	default:
		log.Fatalf(`unrecognized input data format %q: expected "len", "hex", or "utf8"`, format)
	}
}

func GetTransactionReceipt(ctx context.Context, rpcURL, txhash, contractFile string) {
	var myabi *abi.ABI
	client, err := web3.NewClient(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", rpcURL, err)
	}
	defer client.Close()
	if contractFile != "" {
		myabi = getAbi(contractFile)
	}
	r, err := client.GetTransactionReceipt(ctx, common.HexToHash(txhash))
	if err != nil {
		log.Fatalf("Failed to get transaction receipt: %v", err)
	}
	if verbose {
		fmt.Println("Transaction Receipt Details:")
	}

	printReceiptDetails(r, myabi)
}

func GetAddressDetails(ctx context.Context, network web3.Network, addrHash, privateKey string) {
	if addrHash == "" {
		if privateKey == "" {
			log.Fatalf("Missing address. Must be specified as only argument, or implied from a private key.")
		}
		acct, err := web3.ParsePrivateKey(privateKey)
		if err != nil {
			log.Fatalln(err)
		}
		addrHash = acct.PublicKey()
	}
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
		fileparts := strings.Split(contractName, ":")
		if fileparts[0] != "<stdin>" {
			continue
		}
		err := ioutil.WriteFile(fileparts[1]+".bin", []byte(v.Code), 0600)
		if err != nil {
			log.Fatalf("Cannot write the bin file: %v", err)
		}
		err = ioutil.WriteFile(fileparts[1]+".abi", []byte(marshalJSON(v.Info.AbiDefinition)), 0600)
		if err != nil {
			log.Fatalf("Cannot write the abi file: %v", err)
		}
		filenames = append(filenames, fileparts[1])
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
	receipt, err := web3.WaitForReceipt(ctx, client, tx.Hash)
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

	myabi := getAbi(contractFile)

	switch format {
	case "json":
		fmt.Println(marshalJSON(myabi.Methods))
		return
	}

	for _, method := range myabi.Methods {
		fmt.Println(method)
	}

}

func CallContract(ctx context.Context, rpcURL, privateKey, contractAddress, contractFile, functionName string, amount int, waitForReceipt bool, parameters ...interface{}) {
	client, err := web3.NewClient(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", rpcURL, err)
	}
	defer client.Close()
	myabi := getAbi(contractFile)
	if _, ok := myabi.Methods[functionName]; !ok {
		fmt.Println("There is no such function:", functionName)
		return
	}
	if myabi.Methods[functionName].Const {
		res, err := web3.CallConstantFunction(ctx, client, *myabi, contractAddress, functionName, parameters...)
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
		return
	}
	tx, err := web3.CallTransactFunction(ctx, client, *myabi, contractAddress, privateKey, functionName, amount, parameters...)
	if err != nil {
		log.Fatalf("Cannot call the contract: %v", err)
	}
	if !waitForReceipt {
		fmt.Println("Transaction address:", tx.Hash.Hex())
		return
	}
	receipt, err := web3.WaitForReceipt(ctx, client, tx.Hash)
	if err != nil {
		log.Fatalf("Cannot get the receipt: %v", err)
	}
	printReceiptDetails(receipt, myabi)

}

func Send(ctx context.Context, rpcURL, privateKey, toAddress, amount string) {
	client, err := web3.NewClient(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", rpcURL, err)
	}
	defer client.Close()
	nAmount, err := web3.ParseAmount(amount)
	if err != nil {
		log.Fatalf("Cannot parse amount: %v", err)
	}
	if toAddress == "" {
		log.Fatalln("The recepient address cannot be empty")
	}
	address := common.HexToAddress(toAddress)
	tx, err := web3.Send(ctx, client, privateKey, address, nAmount)
	if err != nil {
		log.Fatalf("Cannot create transaction: %v", err)
	}
	fmt.Println("Transaction address:", tx.Hash.Hex())
}

func GenerateContract(ctx context.Context, contractType string, c *cli.Context) {
	if _, err := os.Stat("lib/oz"); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", "--depth", "1", "--branch", "master", "https://github.com/OpenZeppelin/openzeppelin-solidity", "lib/oz")
		log.Printf("Cloning OpenZeppeling repo...")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			log.Fatalf("Cloning finished with error: %v", err)
		}
	}
	if contractType == "erc20" {
		var capped *big.Int
		decimals := c.Int("decimals")
		if decimals <= 0 {
			log.Fatalln("Decimals should be greater than 0")
		}
		if c.String("capped") != "" {
			var ok bool
			capped, ok = new(big.Int).SetString(c.String("capped"), 10)
			if !ok {
				log.Fatalln("Cannot parse capped value")
			}
			if capped.Cmp(big.NewInt(0)) < 1 {
				log.Fatalln("Capped should be greater than 0")
			}
			capped.Mul(capped, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
		}
		if c.String("symbol") == "" {
			log.Fatalln("Symbol is required")
		}
		if c.String("name") == "" {
			log.Fatalln("Name is required")
		}

		params := assets.Erc20Params{
			Symbol:    c.String("symbol"),
			TokenName: c.String("name"),
			Cap:       capped,
			Pausable:  c.Bool("pausable"),
			Mintable:  c.Bool("mintable"),
			Burnable:  c.Bool("burnable"),
			Decimals:  decimals,
		}
		processTemplate(params, params.TokenName, assets.ERC20Template)
	} else if contractType == "erc721" {
		if c.String("symbol") == "" {
			log.Fatalln("Symbol is required")
		}
		if c.String("name") == "" {
			log.Fatalln("Name is required")
		}

		params := assets.Erc721Params{
			Symbol:           c.String("symbol"),
			TokenName:        c.String("name"),
			Pausable:         c.Bool("pausable"),
			Mintable:         c.Bool("mintable"),
			Burnable:         c.Bool("burnable"),
			MetadataMintable: c.Bool("metadata-mintable"),
		}
		processTemplate(params, params.TokenName, assets.ERC721Template)
	}
}

func processTemplate(params interface{}, fileName, contractTemplate string) {
	tmpl, err := template.New("contract").Parse(contractTemplate)
	if err != nil {
		log.Fatalf("Cannot parse the template: %v", err)
	}
	f, err := os.Create(fileName + ".sol")
	if err != nil {
		log.Fatalf("Cannot create the file: %v", err)
		return
	}
	err = tmpl.Execute(f, params)
	if err != nil {
		log.Fatalf("Cannot execute the template: %v", err)
		return
	}
	fmt.Println("The sample contract has been successfully written to", fileName+".sol", "file")
}
func printReceiptDetails(r *web3.Receipt, myabi *abi.ABI) {
	var logs []web3.Event
	var err error
	if myabi != nil {
		logs, err = web3.ParseLogs(*myabi, r.Logs)
		r.ParsedLogs = logs
		if err != nil {
			log.Fatalf("Cannot parse the receipt logs: %v", err)
		}
	}
	switch format {
	case "json":
		fmt.Println(marshalJSON(r))
		return
	}

	fmt.Println("Transaction receipt address:", r.TxHash.Hex())
	fmt.Println("TxHash:", r.TxHash.String())
	if r.ContractAddress != (common.Address{}) {
		fmt.Println("Contract Address:", r.ContractAddress.String())
	}
	fmt.Println("GasUsed:", r.GasUsed)
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
		fmt.Println("Logs of the receipt:", marshalJSON(r.ParsedLogs))
	}
}
func marshalJSON(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("Cannot marshal json: %v", err)
	}
	return string(b)
}

func getAbi(contractFile string) *abi.ABI {
	abi, err := web3.ABIBuiltIn(contractFile)
	if err != nil {
		log.Fatalf("Cannot get ABI from the bundled storage: %v", err)
	}
	if abi == nil {
		abi, err = web3.ABIOpenFile(contractFile)
		if err != nil {
			log.Fatalf("Cannot get ABI: %v", err)
		}
	}
	return abi
}
