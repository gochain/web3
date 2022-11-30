package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gochain/gochain/v4/accounts/keystore"
	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/common/hexutil"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/gochain/gochain/v4/crypto"
	"github.com/shopspring/decimal"
	"github.com/treeder/gotils/v2"
	"github.com/urfave/cli"
	"github.com/zeus-fyi/gochain/web3/accounts"
	"github.com/zeus-fyi/gochain/web3/assets"
	web3_client "github.com/zeus-fyi/gochain/web3/client"
	"github.com/zeus-fyi/gochain/web3/types"
	"github.com/zeus-fyi/gochain/web3/web3_actions"
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

	pkVarName          = "WEB3_PRIVATE_KEY"
	addrVarName        = "WEB3_ADDRESS"
	networkVarName     = "WEB3_NETWORK"
	rpcURLVarName      = "WEB3_RPC_URL"
	didRegistryVarName = "WEB3_DID_REGISTRY"
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
	var netName, rpcUrl, function, contractAddress, toContractAddress, abiFile, privateKey, txFormat, txInputFormat string
	var testnet, waitForReceipt, upgradeable bool

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
	var network web3_client.Network
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
				ac := web3_actions.NewWeb3ActionsClient(network.URL)
				ac.Dial()
				defer ac.Close()
				err := ac.GetTxReceipt(ctx, c.Args().First(), abiFile)
				if err != nil {
					fatalExit(err)
				}
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "abi",
					Destination: &abiFile,
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
				cli.StringFlag{
					Name:   "block",
					Usage:  "Block number",
					Hidden: false},
			},
			Action: func(c *cli.Context) {
				GetAddressDetails(ctx, network, c.Args().First(), privateKey, false, "", c.String("block"))
			},
		},
		{
			Name:  "balance",
			Usage: "Get balance for your private key or an address passed in. eg: `balance 0xABC123`",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "private-key, pk",
					Usage:       "The private key",
					EnvVar:      pkVarName,
					Destination: &privateKey,
					Hidden:      false},
				cli.BoolFlag{
					Name:   "erc20",
					Usage:  "set if using erc20 tokens",
					Hidden: false},
				cli.StringFlag{
					Name:   "address",
					EnvVar: addrVarName,
					Usage:  "Contract address",
					Hidden: false},
				cli.StringFlag{
					Name:   "block",
					Usage:  "Block number",
					Hidden: false},
			},
			Action: func(c *cli.Context) {
				contractAddress = ""
				if c.Bool("erc20") {
					contractAddress = c.String("address")
					if contractAddress == "" {
						fatalExit(errors.New("you must set ERC20 contract address"))
					}
				}
				GetAddressDetails(ctx, network, c.Args().First(), privateKey, true, contractAddress, c.String("block"))
			},
		},
		{
			Name:  "increasegas",
			Usage: "Increase gas for a transaction. Useful if a tx is taking too long and you want it to go faster.",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "private-key, pk",
					Usage:       "The private key",
					EnvVar:      pkVarName,
					Destination: &privateKey,
					Required:    true},
				cli.StringFlag{
					Name:     "tx",
					Usage:    "The transaction hash of the pending transaction.",
					Required: true},
				cli.IntFlag{
					Name:  "amount",
					Usage: "The amount in GWEI to increase the price. 1 would add 1 more GWEI. Decimal values allowed. (default: 1)",
					Value: 1,
				},
			},
			Action: func(c *cli.Context) {
				ac := web3_actions.NewWeb3ActionsClient(network.URL)
				ac.Dial()
				defer ac.Close()
				acc, err := accounts.ParsePrivateKey(privateKey)
				if err != nil {
					fatalExit(err)
				}
				ac.Account = acc
				err = ac.IncreaseGas(ctx, network, c.String("tx"), c.String("amount"))
				if err != nil {
					fatalExit(err)
				}
			},
		},
		{
			Name:  "replace",
			Usage: "Replace transaction. If a transaction is still pending, you can attempt to replace it.",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "private-key, pk",
					Usage:       "The private key",
					EnvVar:      pkVarName,
					Destination: &privateKey,
					Required:    true},
				cli.StringFlag{
					Name:     "nonce",
					Usage:    "The nonce to replace.",
					Required: true},
				cli.StringFlag{
					Name:     "to",
					Usage:    "to address",
					Required: true,
				},
				cli.StringFlag{
					Name:  "amountd", // adding a d for backwards compatibility. If d, then it's decimal, otherwise, the old stuff.
					Usage: "The amount of GO or ETH in decimal format",
				},
				cli.Uint64Flag{
					Name:  "gas-limit",
					Usage: "Gas limit (multiplied by price for total gas)",
					Value: 21000,
				},
				cli.StringFlag{
					Name:  "gas-price",
					Usage: "Gas price to use, if left blank, will use suggested gas price.",
				},
				cli.StringFlag{
					Name:  "gas-price-gwei",
					Usage: "Gas price to use in GWEI, if left blank, will use suggested gas price.",
				},
				cli.StringFlag{
					Name:  "data",
					Usage: "Data for smart contract call in hex (can copy from etherscan and other explorers)",
				},
			},
			Action: func(c *cli.Context) {
				toS := c.String("to")
				if toS == "" {
					fatalExit(errors.New("to address not set"))
				}
				var amount *big.Int
				ad := c.String("amountd")
				if ad != "" {
					amountd, err := decimal.NewFromString(ad)
					if err != nil {
						fatalExit(err)
					}
					amount = web3_types.DecToInt(amountd, 18)
				} else {
					amount = nil
				}
				price, limit := parseGasPriceAndLimit(c)
				to := common.HexToAddress(toS)
				dataB, err := hex.DecodeString(strings.TrimPrefix(c.String("data"), "0x"))
				if err != nil {
					fatalExit(err)
				}
				ac := web3_actions.NewWeb3ActionsClient(network.URL)
				ac.Dial()
				defer ac.Close()
				acc, err := accounts.ParsePrivateKey(privateKey)
				if err != nil {
					fatalExit(err)
				}
				ac.Account = acc
				_, err = ac.ReplaceTx(ctx, network, c.Uint64("nonce"), to, amount, price, limit, dataB)
				if err != nil {
					fatalExit(err)
				}
			},
		},
		{
			Name:    "contract",
			Aliases: []string{"c"},
			Usage:   "Contract operations",
			Subcommands: []cli.Command{
				{
					Name:  "flatten",
					Usage: "Make the specified contract flat",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "output, o",
							Usage: "The output file, by default it will add _flatten postfix",
						},
					},
					Action: func(c *cli.Context) {
						FlattenSol(ctx, c.Args().First(), c.String("output"))
					},
				},
				{
					Name:  "build",
					Usage: "Build the specified contract",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "solc-version, c",
							Usage: "The version of the solc compiler(a tag of the ethereum/solc docker image)",
						},
						cli.StringFlag{
							Name:  "output, o",
							Usage: "Output directory.",
						},
						cli.StringFlag{
							Name:  "evm-version",
							Usage: "Solidity EVM version",
							Value: "petersburg",
						},
						cli.BoolTFlag{
							Name:  "optimize",
							Usage: "Solidity optimization",
						},
					},
					Action: func(c *cli.Context) {
						BuildSol(ctx, c.Args().First(), c.String("solc-version"), c.String("evm-version"),
							c.BoolT("optimize"), c.String("output"))
					},
				},
				{
					Name:  "deploy",
					Usage: "Deploy the specified contract to the network. eg: web3 contract deploy MyContract.bin",
					Action: func(c *cli.Context) {
						binFile := c.Args().First()
						tail := c.Args().Tail()
						args := make([]interface{}, len(tail))
						for i, v := range c.Args().Tail() {
							args[i] = v
						}
						price, limit := parseGasPriceAndLimit(c)
						DeploySol(ctx, network, privateKey, binFile, c.String("verify"),
							c.String("solc-version"), c.String("evm-version"), c.BoolT("optimize"),
							c.String("explorer-api"), price, limit, upgradeable, c.Uint64("timeout"), args...)
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "private-key, pk",
							Usage:       "The private key",
							EnvVar:      pkVarName,
							Destination: &privateKey,
							Hidden:      false},
						cli.BoolFlag{
							Name:        "upgradeable",
							Usage:       "Allow contract to be upgraded",
							Destination: &upgradeable,
							Hidden:      false},
						cli.StringFlag{
							Name:  "verify",
							Usage: "Source code of the contract",
						},
						cli.StringFlag{
							Name:  "explorer-api",
							Usage: "Explorer API URL. Optional for GoChain networks, which use `{testnet-}explorer.gochain.io` by default",
						},
						cli.StringFlag{
							Name:  "solc-version, c",
							Usage: "The version of the solc compiler(a tag of the ethereum/solc docker image)",
						}, cli.StringFlag{
							Name:  "evm-version",
							Usage: "Solidity EVM version",
							Value: "petersburg",
						},
						cli.BoolTFlag{
							Name:  "optimize",
							Usage: "Solidity optimization",
						},
						cli.Uint64Flag{
							Name:  "gas-limit",
							Value: 4000000,
						},
						cli.StringFlag{
							Name:  "gas-price",
							Usage: "Gas price to use, if left blank, will use suggested gas price.",
						},
						cli.StringFlag{
							Name:  "gas-price-gwei",
							Usage: "Gas price to use in GWEI, if left blank, will use suggested gas price.",
						},
						cli.UintFlag{
							Name:  "timeout",
							Usage: "Timeout in seconds (default: 60).",
							Value: 60,
						},
					},
				},
				{
					Name:  "verify",
					Usage: "Verify the specified contract which is already deployed to the network",
					Action: func(c *cli.Context) {
						VerifyContract(ctx, network, c.String("explorer-api"), contractAddress,
							c.String("contract-name"), c.Args().First(), c.String("solc-version"),
							c.String("evm-version"), c.BoolT("optimize"))
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "explorer-api",
							Usage: "Explorer API URL"},
						cli.StringFlag{
							Name:  "contract-name",
							Usage: "Deployed contract name"},
						cli.StringFlag{
							Name:        "address",
							EnvVar:      addrVarName,
							Destination: &contractAddress,
							Usage:       "Deployed contract address",
							Hidden:      false},
						cli.StringFlag{
							Name:  "solc-version, c",
							Usage: "The version of the solc compiler(a tag of the ethereum/solc docker image)"},
						cli.StringFlag{
							Name:  "evm-version",
							Usage: "Solidity EVM version",
							Value: "petersburg"},
						cli.BoolTFlag{
							Name:  "optimize",
							Usage: "Solidity optimization"},
					},
				},
				{
					Name:  "list",
					Usage: "List contract functions",
					Action: func(c *cli.Context) {
						err := web3_actions.ListContract(ctx, abiFile)
						if err != nil {
							fatalExit(err)
						}
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "abi",
							Destination: &abiFile,
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
						amount := toAmountBig(c.String("amount"))
						price, limit := parseGasPriceAndLimit(c)
						ac := web3_actions.NewWeb3ActionsClient(network.URL)
						ac.Dial()
						defer ac.Close()
						ac.SetChainID(network.ChainID)
						var dataB []byte
						var err error
						if c.String("data") != "" {
							dataB, err = hex.DecodeString(strings.TrimPrefix(c.String("data"), "0x"))
							if err != nil {
								fatalExit(err)
							}
						}
						gp := web3_actions.GasPriceLimits{
							GasPrice: price,
							GasLimit: limit,
						}

						payload := web3_actions.SendContractTxPayload{
							SmartContractAddr: contractAddress,
							SendEtherPayload: web3_actions.SendEtherPayload{
								TransferArgs: web3_actions.TransferArgs{
									Amount:    amount,
									ToAddress: common.Address{},
								},
								GasPriceLimits: gp,
							},
						}
						err = ac.CallContract(ctx, &payload, waitForReceipt, dataB, c.Uint64("timeout"))
						if err != nil {
							fatalExit(err)
						}
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
							Destination: &abiFile,
							Usage:       "ABI file matching deployed contract",
							Hidden:      false},
						cli.StringFlag{
							Name:   "amount",
							Usage:  "Amount in wei that you want to send to the transaction",
							Hidden: false},
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
						cli.BoolFlag{
							Name:  "to-string",
							Usage: "Convert result to a string, useful if using byte arrays that are strings and you want to see the string value.",
						},
						cli.Uint64Flag{
							Name:  "gas-limit",
							Value: 70000,
						},
						cli.StringFlag{
							Name:  "gas-price",
							Usage: "Gas price to use, if left blank, will use suggested gas price.",
						},
						cli.StringFlag{
							Name:  "gas-price-gwei",
							Usage: "Gas price to use in GWEI, if left blank, will use suggested gas price.",
						},
						cli.StringFlag{
							Name:  "data",
							Usage: "Data for smart contract call in hex (can copy from etherscan and other explorers)",
						},
						cli.UintFlag{
							Name:  "timeout",
							Usage: "Timeout in seconds (default: 60).",
							Value: 60,
						},
					},
				},
				{
					Name:  "upgrade",
					Usage: "Upgrade contract to new address",
					Action: func(c *cli.Context) {
						amount := toAmountBig(c.String("amount"))
						ac := web3_actions.NewWeb3ActionsClient(network.URL)
						err := ac.UpgradeContract(ctx, contractAddress, toContractAddress, amount, c.Uint64("timeout"))
						if err != nil {
							fatalExit(fmt.Errorf("cannot upgrade contract to new address:%v", err))
						}
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "address",
							EnvVar:      addrVarName,
							Destination: &contractAddress,
							Usage:       "Proxy contract address",
							Hidden:      false},
						cli.StringFlag{
							Name:        "to",
							Destination: &toContractAddress,
							Usage:       "Contract address to upgrade to",
							Hidden:      false},
						cli.StringFlag{
							Name:   "amount",
							Usage:  "Amount in wei that you want to send to the transaction",
							Hidden: false},
						cli.StringFlag{
							Name:        "private-key",
							Usage:       "Private key",
							EnvVar:      "WEB3_PRIVATE_KEY",
							Destination: &privateKey,
							Hidden:      false},
						cli.UintFlag{
							Name:  "timeout",
							Usage: "Timeout in seconds (default: 60).",
							Value: 60,
						},
					},
				},
				{
					Name:  "target",
					Usage: "Return target address of upgradeable proxy",
					Action: func(c *cli.Context) {
						ac := web3_actions.NewWeb3ActionsClient(network.URL)
						addr, err := ac.GetTargetContract(ctx, network.URL, contractAddress)
						if err != nil {
							fatalExit(fmt.Errorf("cannot upgrade contract to new address:%v", err))
						}
						fmt.Println(addr)
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "address",
							EnvVar:      addrVarName,
							Destination: &contractAddress,
							Usage:       "Proxy contract address",
							Hidden:      false},
					},
				},
				{
					Name:  "pause",
					Usage: "Pause an upgradeable contract",
					Action: func(c *cli.Context) {
						address := c.Args().First()
						if address == "" {
							address = contractAddress
						}
						amount := toAmountBig(c.String("amount"))
						ac := web3_actions.NewWeb3ActionsClient(network.URL)
						err := ac.PauseContract(ctx, address, amount, c.Uint64("timeout"))
						if err != nil {
							fatalExit(fmt.Errorf("cannot pause upgradeable contract:%v", err))
						}
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "address",
							EnvVar:      addrVarName,
							Destination: &contractAddress,
							Usage:       "Proxy contract address",
							Hidden:      false},
						cli.StringFlag{
							Name:   "amount",
							Usage:  "Amount in wei that you want to send to the transaction",
							Hidden: false},
						cli.StringFlag{
							Name:        "private-key",
							Usage:       "Private key",
							EnvVar:      "WEB3_PRIVATE_KEY",
							Destination: &privateKey,
							Hidden:      false},
						cli.UintFlag{
							Name:  "timeout",
							Usage: "Timeout in seconds (default: 60).",
							Value: 60,
						},
					},
				},
				{
					Name:  "resume",
					Usage: "Resume a paused upgradeable contract",
					Action: func(c *cli.Context) {
						address := c.Args().First()
						if address == "" {
							address = contractAddress
						}
						amount := toAmountBig(c.String("amount"))
						ac := web3_actions.NewWeb3ActionsClient(network.URL)
						ac.Dial()
						defer ac.Close()
						err := ac.ResumeContract(ctx, address, amount, c.Uint64("timeout"))
						if err != nil {
							fatalExit(fmt.Errorf("cannot resume paused upgradeable contract"))
						}
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "address",
							EnvVar:      addrVarName,
							Destination: &contractAddress,
							Usage:       "Proxy contract address",
							Hidden:      false},
						cli.StringFlag{
							Name:   "amount",
							Usage:  "Amount in wei that you want to send to the transaction",
							Hidden: false},
						cli.StringFlag{
							Name:        "private-key",
							Usage:       "Private key",
							EnvVar:      "WEB3_PRIVATE_KEY",
							Destination: &privateKey,
							Hidden:      false},
						cli.UintFlag{
							Name:  "timeout",
							Usage: "Timeout in seconds (default: 60).",
							Value: 60,
						},
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
				acc, err := accounts.ParsePrivateKey(pk)
				if err != nil {
					fatalExit(err)
				}
				fmt.Println(acc.PublicKey())
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
						acc, err := accounts.CreateAccount()
						if err != nil {
							fatalExit(err)
						}
						fmt.Printf("Private key: %v\n", acc.PrivateKey())
						fmt.Printf("Public address: %v\n", acc.PublicKey())
					},
				},
				{
					Name:  "extract",
					Usage: "Extract private key from keystore file",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:   "keyfile",
							Usage:  "filename of keystore file",
							Hidden: false,
						},
						cli.StringFlag{
							Name:   "password",
							Usage:  "password for keystore",
							Hidden: false,
						},
					},
					Action: func(c *cli.Context) {
						f := c.String("keyfile")
						kbytes, err := ioutil.ReadFile(f)
						if err != nil {
							fatalExit(fmt.Errorf("Failed to read file %q: %v", f, err))
						}
						key, err := keystore.DecryptKey(kbytes, c.String("password"))
						if err != nil {
							fatalExit(err)
						}
						fmt.Printf("Private key: %v\n", "0x"+hex.EncodeToString(crypto.FromECDSA(key.PrivateKey)))
						fmt.Printf("Public address: %v\n", key.Address.Hex())
					},
				},
			},
		},
		{
			Name:    "transfer",
			Usage:   fmt.Sprintf("Transfer GO/ETH or ERC20 tokens to another account. eg: `web3 transfer 10.1 to 0xADDRESS`"),
			Aliases: []string{"send"},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "private-key,pk",
					Usage:       "Private key",
					EnvVar:      pkVarName,
					Destination: &privateKey,
					Hidden:      false,
				},
				cli.BoolFlag{
					Name:   "erc20",
					Usage:  "Set if transferring ERC20 tokens",
					Hidden: false},
				cli.StringFlag{
					Name:   "address",
					EnvVar: addrVarName,
					Usage:  "Contract address if this is an ERC20",
					Hidden: false},
				cli.BoolFlag{
					Name:        "wait",
					Usage:       "Wait for the receipt of this transaction",
					Destination: &waitForReceipt,
					Hidden:      false},
				cli.BoolFlag{
					Name:  "to-string",
					Usage: "Convert result to a string, useful if using byte arrays that are strings and you want to see the string value.",
				},
				cli.Uint64Flag{
					Name:  "gas-limit",
					Usage: "Gas limit (multiplied by price for total gas)",
					Value: 21000,
				},
				cli.StringFlag{
					Name:  "gas-price",
					Usage: "Gas price to use, if left blank, will use suggested gas price.",
				},
				cli.StringFlag{
					Name:  "gas-price-gwei",
					Usage: "Gas price to use in GWEI, if left blank, will use suggested gas price.",
				},
			},
			Action: func(c *cli.Context) {
				contractAddress = ""
				if c.Bool("erc20") {
					contractAddress = c.String("address")
					if contractAddress == "" {
						fatalExit(errors.New("you must set ERC20 contract address"))
					}
				}
				price, limit := parseGasPriceAndLimit(c)
				ac := web3_actions.NewWeb3ActionsClient(network.URL)
				gp := web3_actions.GasPriceLimits{
					GasPrice: price,
					GasLimit: limit,
				}
				argsIn, err := web3_actions.ConvertTailForTransfer(ctx, c.Args())
				if err != nil {
					fatalExit(err)
				}
				payload := web3_actions.SendContractTxPayload{
					SmartContractAddr: contractAddress,
					SendEtherPayload: web3_actions.SendEtherPayload{
						TransferArgs:   argsIn,
						GasPriceLimits: gp,
					},
				}
				err = ac.TransferERC20Token(ctx, payload, c.Bool("wait"), c.Uint64("timeout"))
				if err != nil {
					fatalExit(err)
				}
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
			Usage:   "Generate a contract",
			Aliases: []string{"g"},
			Subcommands: []cli.Command{
				{
					Name:  "contract",
					Usage: "Generate a contract",
					Subcommands: []cli.Command{
						{
							Name:  "erc20",
							Usage: "Generate an ERC20 contract",
							Flags: []cli.Flag{
								// cli.BoolTFlag{
								// 	Name:  "pausable, p",
								// 	Usage: "Pausable contract. Default: true",
								// },
								// cli.BoolTFlag{
								// 	Name:  "mintable, m",
								// 	Usage: "Mintable contract. Default: true",
								// },
								// cli.BoolTFlag{
								// 	Name:  "burnable, b",
								// 	Usage: "Burnable contract. Default: true",
								// },
								cli.StringFlag{
									Name:     "symbol, s",
									Usage:    "Token Symbol.",
									Required: true,
								},
								cli.StringFlag{
									Name:     "name, n",
									Usage:    "Token Name",
									Required: true,
								},
								// cli.StringFlag{
								// 	Name:  "capped, c",
								// 	Usage: "Cap, total supply(in GO/ETH)",
								// },
								// cli.IntFlag{
								// 	Name:  "decimals, d",
								// 	Usage: "Decimals",
								// 	Value: 18,
								// },
							},
							Action: func(c *cli.Context) {
								GenerateContract(ctx, "erc20", c)
							},
						},
						{
							Name:  "erc721",
							Usage: "Generate an ERC721 contract",
							Flags: []cli.Flag{
								// cli.BoolTFlag{
								// 	Name:  "pausable, p",
								// 	Usage: "Pausable contract. Default: true",
								// },
								// cli.BoolTFlag{
								// 	Name:  "mintable, m",
								// 	Usage: "Mintable contract. Default: true",
								// },
								// cli.BoolTFlag{
								// 	Name:  "burnable, b",
								// 	Usage: "Burnable contract. Default: true",
								// },
								cli.StringFlag{
									Name:     "symbol, s",
									Usage:    "Token Symbol.",
									Required: true,
								},
								cli.StringFlag{
									Name:     "name, n",
									Usage:    "Token Name",
									Required: true,
								},
								cli.StringFlag{
									Name:     "base-uri",
									Usage:    "Base URI for fetching token metadata",
									Required: true,
								},
							},
							Action: func(c *cli.Context) {
								GenerateContract(ctx, "erc721", c)
							},
						},
					},
				},
				{
					Name:  "code",
					Usage: "Generate code bindings",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "abi, a",
							Usage: "Path to the contract ABI json to bind",
						},
						cli.StringFlag{
							Name:  "lang, l",
							Usage: "Destination language for the bindings (go, java, objc)",
							Value: "go",
						},
						cli.StringFlag{
							Name:  "pkg, p",
							Usage: "Package name to generate the binding into.",
							Value: "main",
						},
						cli.StringFlag{
							Name:  "out, o",
							Usage: "Output file for the generated binding (default = main.go).",
							Value: "out.go",
						},
					},
					Action: func(c *cli.Context) {
						GenerateCode(ctx, c)
					},
				},
			},
		},
		{
			Name:  "did",
			Usage: "Distributed identity operations",
			Subcommands: []cli.Command{
				{
					Name:  "create",
					Usage: "Create a new DID",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "private-key,pk",
							Usage:       "Private key",
							EnvVar:      pkVarName,
							Destination: &privateKey,
						},
						cli.StringFlag{
							Name:   "registry",
							Usage:  "Registry contract address",
							EnvVar: didRegistryVarName,
						},
						cli.UintFlag{
							Name:  "timeout",
							Usage: "Timeout in seconds (default: 60).",
							Value: 60,
						},
					},
					Action: func(c *cli.Context) {
						CreateDID(ctx, network.URL, network.ChainID, privateKey, c.Args().First(), c.String("registry"), c.Uint64("timeout"))
					},
				},
				{
					Name:  "owner",
					Usage: "Display DID owner address",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "private-key,pk",
							Usage:       "Private key",
							EnvVar:      pkVarName,
							Destination: &privateKey,
						},
						cli.StringFlag{
							Name:   "registry",
							Usage:  "Registry contract address",
							EnvVar: didRegistryVarName,
						},
					},
					Action: func(c *cli.Context) {
						DIDOwner(ctx, network.URL, privateKey, c.Args().First(), c.String("registry"))
					},
				},
				{
					Name:  "hash",
					Usage: "Display DID document IPFS hash",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "private-key,pk",
							Usage:       "Private key",
							EnvVar:      pkVarName,
							Destination: &privateKey,
						},
						cli.StringFlag{
							Name:   "registry",
							Usage:  "Registry contract address",
							EnvVar: didRegistryVarName,
						},
					},
					Action: func(c *cli.Context) {
						DIDHash(ctx, network.URL, privateKey, c.Args().First(), c.String("registry"))
					},
				},
				{
					Name:  "show",
					Usage: "Display DID document",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "private-key,pk",
							Usage:       "Private key",
							EnvVar:      pkVarName,
							Destination: &privateKey,
						},
						cli.StringFlag{
							Name:   "registry",
							Usage:  "Registry contract address",
							EnvVar: didRegistryVarName,
						},
					},
					Action: func(c *cli.Context) {
						ShowDID(ctx, network.URL, privateKey, c.Args().First(), c.String("registry"))
					},
				},
			},
		},

		{
			Name:  "claim",
			Usage: "Verifiable claims operations",
			Subcommands: []cli.Command{
				{
					Name:  "sign",
					Usage: "Sign a verifiable claim",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "private-key,pk",
							Usage:       "Private key",
							EnvVar:      pkVarName,
							Destination: &privateKey,
						},
						cli.StringFlag{
							Name:  "id",
							Usage: "Credential ID",
						},
						cli.StringFlag{
							Name:  "type",
							Usage: "Credential type",
						},
						cli.StringFlag{
							Name:  "issuer",
							Usage: "Credential issuer DID",
						},
						cli.StringFlag{
							Name:  "subject",
							Usage: "Credential subject DID",
						},
						cli.StringFlag{
							Name:  "data",
							Usage: "Credential subject JSON object",
						},
					},
					Action: func(c *cli.Context) {
						SignClaim(ctx, network.URL, privateKey, c.String("id"), c.String("type"), c.String("issuer"), c.String("subject"), c.String("data"))
					},
				},
				{
					Name:  "verify",
					Usage: "Verify a signed claim",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "private-key,pk",
							Usage:       "Private key",
							EnvVar:      pkVarName,
							Destination: &privateKey,
						},
						cli.StringFlag{
							Name:   "registry",
							Usage:  "Registry contract address",
							EnvVar: didRegistryVarName,
						},
					},
					Action: func(c *cli.Context) {
						VerifyClaim(ctx, network.URL, privateKey, c.String("registry"), c.Args().First())
					},
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println("ERROR:", err)
	}
}

func toAmountBig(a string) *big.Int {
	var amount *big.Int
	if a != "" {
		var ok bool
		amount, ok = new(big.Int).SetString(a, 10)
		if !ok {
			fatalExit(fmt.Errorf("invalid amount %v", a))
		}
	} else {
		amount = &big.Int{}
	}
	return amount
}

// getNetwork resolves the rpcUrl from the user specified options, or quits if an illegal combination or value is found.
func getNetwork(name, rpcURL string, testnet bool) web3_client.Network {
	var network web3_client.Network
	if rpcURL != "" {
		if name != "" {
			fatalExit(fmt.Errorf("Cannot set both rpcURL %q and network %q", rpcURL, network))
		}
		if testnet {
			fatalExit(fmt.Errorf("Cannot set both rpcURL %q and testnet", rpcURL))
		}
		network.URL = rpcURL
		network.Unit = "GO"
	} else {
		if testnet {
			if name != "" {
				fatalExit(fmt.Errorf("Cannot set both network %q and testnet", name))
			}
			name = "testnet"
		} else if name == "" {
			name = "gochain"
		}
		var ok bool
		network, ok = web3_client.Networks[name]
		if !ok {
			fatalExit(fmt.Errorf("Unrecognized network %q", name))
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

func parseGasPriceAndLimit(c *cli.Context) (*big.Int, uint64) {
	gasLimit := c.Uint64("gas-limit")
	gp := c.String("gas-price")
	var price *big.Int
	var ok bool
	if gp != "" {
		price, ok = new(big.Int).SetString(gp, 10)
		if !ok {
			fatalExit(fmt.Errorf("invalid price %v", gp))
		}
	}
	gp = c.String("gas-price-gwei")
	if gp != "" {
		price, ok = new(big.Int).SetString(gp, 10)
		if !ok {
			fatalExit(fmt.Errorf("invalid price %v", gp))
		}
		price = web3_types.Gwei(price.Int64())
	}
	return price, gasLimit
}

func GetBlockDetails(ctx context.Context, network web3_client.Network, numberOrHash string, txFormat, txInputFormat string) {
	ac := web3_actions.NewWeb3ActionsClient(network.URL)
	ac.Dial()
	defer ac.Close()
	var block *web3_types.Block
	var includeTxs bool
	switch txFormat {
	case "detail":
		includeTxs = true
	case "count", "hash":
	default:
		fatalExit(fmt.Errorf(`Unrecognized transaction format %q: must be "count", "hash", or "detail"`, txFormat))
	}
	var err error

	if strings.HasPrefix(numberOrHash, "0x") {
		block, err = ac.GetBlockByHash(ctx, numberOrHash, includeTxs)
		if err != nil {
			fatalExit(fmt.Errorf("Cannot get block details from the network: %v", err))
		}
	} else {
		var blockN *big.Int
		// Don't try to parse empty string, which means 'latest'.
		if numberOrHash != "" {
			blockN, err = web3_types.ParseBigInt(numberOrHash)
			if err != nil {
				fatalExit(fmt.Errorf("block argument must be a number (decimal integer) %q: %v", numberOrHash, err))
			}
		}
		block, err = ac.GetBlockByNumber(ctx, blockN, includeTxs)
		if err != nil {
			fatalExit(fmt.Errorf("Cannot get block details from the network: %v", err))
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
				fmt.Print(" Value: ", web3_types.WeiAsBase(tx.Value), " ", network.Unit)
				fmt.Print(" Nonce: ", tx.Nonce)
				fmt.Print(" Gas Limit: ", tx.GasLimit)
				fmt.Print(" Gas Price: ", web3_types.WeiAsGwei(tx.GasPrice), " gwei")
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

func GetTransactionDetails(ctx context.Context, network web3_client.Network, txhash, inputFormat string) {
	ac := web3_actions.NewWeb3ActionsClient(network.URL)
	ac.Dial()
	defer ac.Close()
	// fmt.Println(network.URL)
	tx, err := ac.GetTransactionByHash(ctx, common.HexToHash(txhash))
	if err != nil {
		fatalExit(fmt.Errorf("Cannot get transaction details from %v network: %v", network.Name, err))
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
	fmt.Println("Value:", web3_types.WeiAsBase(tx.Value), network.Unit)
	fmt.Println("Nonce:", uint64(tx.Nonce))
	fmt.Println("Gas Limit:", tx.GasLimit)
	fmt.Println("Gas Price:", web3_types.WeiAsGwei(tx.GasPrice), "gwei")
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
		fatalExit(fmt.Errorf(`unrecognized input data format %q: expected "len", "hex", or "utf8"`, format))

	}
}

func GetAddressDetails(ctx context.Context, network web3_client.Network, addrHash, privateKey string, onlyBalance bool,
	contractAddress string, blockNumber string) {
	if addrHash == "" {
		if privateKey == "" {
			fatalExit(errors.New("missing address. Must be specified as only argument, or implied from a private key"))
		}
		acct, err := accounts.ParsePrivateKey(privateKey)
		if err != nil {
			fatalExit(err)
		}
		addrHash = acct.PublicKey()
	}

	ac := web3_actions.NewWeb3ActionsClient(network.URL)
	ac.Dial()
	defer ac.Close()

	if contractAddress != "" {
		_, err := ac.ReadERC20TokenBalance(ctx, contractAddress, addrHash)
		if err != nil {
			fatalExit(err)
		}
		return
	}

	var blockN *big.Int
	var err error
	// Don't try to parse empty string, which means 'latest'.
	if blockNumber != "" {
		blockN, err = web3_types.ParseBigInt(blockNumber)
		if err != nil {
			fatalExit(fmt.Errorf("block argument must be a number (decimal integer) %q: %v", blockNumber, err))
		}
	}

	bal, err := ac.GetBalance(ctx, addrHash, blockN)
	if err != nil {
		fatalExit(fmt.Errorf("Cannot get address balance from the network: %v", err))
	}
	code, err := ac.GetCode(ctx, addrHash, nil)
	if err != nil {
		fatalExit(fmt.Errorf("Cannot get address code from the network: %v", err))
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

	if onlyBalance {
		fmt.Println(web3_types.IntToDec(bal, 18))
	} else {
		fmt.Println("Balance:", web3_types.IntToDec(bal, 18))
		if len(code) > 0 {
			fmt.Println("Code:", string(code))
		}
	}
}

func GetSnapshot(ctx context.Context, rpcURL string) {
	ac := web3_actions.NewWeb3ActionsClient(rpcURL)
	ac.Dial()
	defer ac.Close()
	s, err := ac.GetSnapshot(ctx)
	if err != nil {
		fatalExit(fmt.Errorf("cannot get snapshot from the network: %v", err))
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
	ac := web3_actions.NewWeb3ActionsClient(rpcURL)
	ac.Dial()
	defer ac.Close()
	id, err := ac.GetID(ctx)
	if err != nil {
		fatalExit(fmt.Errorf("cannot get id info from the network: %v", err))
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

// BuildSol builds a contract. Generated files will be under output, or the current directory.
func BuildSol(ctx context.Context, filename, solcVersion, evmVersion string, optimize bool, output string) {
	if filename == "" {
		fatalExit(errors.New("missing file name arg"))
	}
	flatOut := ""
	if output != "" {
		if err := os.MkdirAll(output, 0777); err != nil {
			fatalExit(fmt.Errorf("Failed to create output directory: %v", err))
		}
		basename := filepath.Base(filename)
		oName := strings.TrimSuffix(basename, filepath.Ext(basename)) + "_flatten.sol"
		flatOut = filepath.Join(output, oName)
	}
	name, sourceFile, err := FlattenSourceFile(ctx, filename, flatOut)
	if err != nil {
		fatalExit(fmt.Errorf("Cannot generate flattened file: %v", err))
	}
	b, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		fatalExit(fmt.Errorf("Failed to read file %q: %v", sourceFile, err))
	}
	str := string(b) // convert content to a 'string'
	if verbose {
		log.Println("Building Sol:", str)
	}
	compileData, err := web3_types.CompileSolidityString(ctx, str, solcVersion, evmVersion, optimize)
	if err != nil {
		fatalExit(fmt.Errorf("Failed to compile %q: %v", sourceFile, err))
	}
	if verbose {
		log.Println("Compiled Sol Details:", marshalJSON(compileData))
	}
	fmt.Println("NAME:", name)
	var filenames []string
	for contractName, v := range compileData {
		fmt.Println("contractName:", contractName)
		fileparts := strings.Split(contractName, ":")
		if fileparts[0] != "<stdin>" {
			continue
		}
		if name != "" && fileparts[1] != name {
			// this will skip all the little contract files that it used to litter the the directory with
			continue
		}
		path := filepath.Join(output, fileparts[1])
		err = os.WriteFile(path+".bin", []byte(v.Code), 0600)
		if err != nil {
			fatalExit(fmt.Errorf("Cannot write the bin file: %v", err))
		}
		err = os.WriteFile(path+".abi", []byte(marshalJSON(v.Info.AbiDefinition)), 0600)
		if err != nil {
			fatalExit(fmt.Errorf("Cannot write the abi file: %v", err))
		}
		filenames = append(filenames, fileparts[1])
	}

	switch format {
	case "json":
		data := struct {
			Source string   `json:"source"`
			Bin    []string `json:"bin"`
			ABI    []string `json:"abi"`
		}{}
		data.Source = sourceFile
		for _, f := range filenames {
			data.Bin = append(data.Bin, f+".bin")
			data.ABI = append(data.ABI, f+".abi")
		}
		fmt.Println(marshalJSON(data))
		return
	}

	fmt.Println("Successfully compiled contracts and wrote the following files:")
	fmt.Println("Source file", sourceFile)
	for _, fn := range filenames {
		fmt.Println("", fn+".bin,", fn+".abi")
	}
}

func FlattenSol(ctx context.Context, iFile, oFile string) {
	if iFile == "" {
		fatalExit(errors.New("missing file name arg"))
	}
	_, oFile, err := FlattenSourceFile(ctx, iFile, oFile)
	if err != nil {
		fatalExit(fmt.Errorf("Cannot generate flattened file: %v", err))
	}
	switch format {
	case "json":
		fmt.Println(marshalJSON(oFile))
		return
	}
	fmt.Println("Flattened contract:", oFile)
}

func DeploySol(ctx context.Context, network web3_client.Network,
	privateKey, binFile, contractSource, solcVersion, evmVersion string, optimize bool, explorerURL string,
	gasPrice *big.Int, gasLimit uint64, upgradeable bool, timeoutInSeconds uint64, params ...interface{}) {

	ac := web3_actions.NewWeb3ActionsClient(network.URL)
	ac.Dial()
	defer ac.Close()
	if binFile == "" {
		fatalExit(errors.New("missing contract name arg."))
	}
	ac.SetChainID(network.ChainID)
	// get file
	var bin []byte
	var err error
	if strings.HasPrefix(binFile, "http") {
		bin, err = gotils.GetBytes(binFile)
	} else {
		bin, err = os.ReadFile(binFile)
	}
	if err != nil {
		fatalExit(fmt.Errorf("Cannot read bin file %q: %v", binFile, err))
	}
	var abi string
	if len(params) > 0 {
		abiFile := strings.TrimSuffix(binFile, ".bin") + ".abi"
		var b []byte
		if strings.HasPrefix(binFile, "http") {
			b, err = gotils.GetBytes(abiFile)
		} else {
			b, err = os.ReadFile(abiFile)
		}
		if err != nil {
			fatalExit(fmt.Errorf("Cannot read abi file %q: %v", abiFile, err))
		}
		abi = string(b)
	}
	acc, err := accounts.ParsePrivateKey(privateKey)
	if err != nil {
		fatalExit(err)
	}
	ac.Account = acc
	tx, err := ac.DeployContract(ctx, string(bin), abi, gasPrice, gasLimit, params...)
	if err != nil {
		fatalExit(fmt.Errorf("Error deploying contract: %v", err))
	}
	waitCtx, cancelFn := context.WithTimeout(ctx, time.Duration(timeoutInSeconds)*time.Second)
	defer cancelFn()
	receipt, err := ac.WaitForReceipt(waitCtx, tx.Hash)
	if err != nil {
		fatalExit(fmt.Errorf("cannot get the receipt for transaction with hash '%v': %v", tx.Hash.Hex(), err))
	}

	switch format {
	case "json":
		fmt.Println(marshalJSON(receipt))
		return
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		fatalExit(fmt.Errorf("Contract deploy tx failed: %s. Did you pass in the correct constructor arguments?", tx.Hash.Hex()))
	}

	// Exit early if contract is static.
	if !upgradeable {
		fmt.Println("Contract has been successfully deployed with transaction:", tx.Hash.Hex())
		fmt.Println("Contract address is:", receipt.ContractAddress.Hex())
		if contractSource != "" {
			VerifyContract(ctx, network, explorerURL, receipt.ContractAddress.Hex(),
				strings.TrimSuffix(binFile, ".bin"), contractSource, solcVersion, evmVersion, optimize)
		}
		return
	}

	ac.Account = acc
	proxyTx, err := ac.DeployContract(ctx, assets.OwnerUpgradeableProxyCode(receipt.ContractAddress), "", gasPrice, gasLimit)
	if err != nil {
		log.Fatalf("Cannot deploy the upgradeable proxy contract: %v", err)
	}
	waitCtx, cancelFn2 := context.WithTimeout(ctx, time.Duration(timeoutInSeconds)*time.Second)
	defer cancelFn2()
	proxyReceipt, err := ac.WaitForReceipt(waitCtx, proxyTx.Hash)
	if err != nil {
		log.Fatalf("Cannot get the upgradeable proxy receipt: %v", err)
	}

	if proxyReceipt.Status != types.ReceiptStatusSuccessful {
		fatalExit(fmt.Errorf("upgradeable proxy contract deploy tx failed: %s", proxyTx.Hash.Hex()))
	}

	fmt.Println("Upgradeable contract has been successfully deployed.")
	fmt.Println("Contract has been successfully deployed with transaction:", proxyTx.Hash.Hex())
	fmt.Println("Contract address is:", proxyReceipt.ContractAddress.Hex())
}

func VerifyContract(ctx context.Context, network web3_client.Network, explorerURL, contractAddress, contractName,
	sourceCodeFile, compilerVersion, evmVersion string, optimize bool) {
	if explorerURL == "" {
		if network.ExplorerURL == "" {
			fatalExit(errors.New("missing explorer-api arg"))
		} else {
			explorerURL = network.ExplorerURL
		}
	}
	if contractAddress == "" {
		fatalExit(errors.New("missing address arg"))
	}
	if !common.IsHexAddress(contractAddress) {
		fatalExit(fmt.Errorf("invalid contract 'address': %s", contractAddress))
	}
	if contractName == "" {
		fatalExit(errors.New("missing contract name arg"))
	}
	if sourceCodeFile == "" {
		fatalExit(errors.New("missing source file"))
	}
	source, err := os.ReadFile(sourceCodeFile)
	if err != nil {
		fatalExit(fmt.Errorf("cannot read the source code file %q: %v", sourceCodeFile, err))
	}
	if compilerVersion == "" {
		sol, serr := web3_types.SolidityVersion(string(source))
		if serr != nil {
			fatalExit(fmt.Errorf("cannot parse the version from the source code file %q: %v", sourceCodeFile, err))
		}
		compilerVersion = sol.Version
	}
	message := map[string]interface{}{
		"address":          contractAddress,
		"contract_name":    contractName,
		"compiler_version": compilerVersion,
		"optimization":     optimize,
		"source_code":      string(source),
	}
	if evmVersion != "" {
		message["evm_version"] = evmVersion
	}

	bytesRepresentation, err := json.Marshal(message)
	if err != nil {
		fatalExit(fmt.Errorf("cannot convert the message:%v", err))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", explorerURL+"/verify", bytes.NewBuffer(bytesRepresentation))
	if err != nil {
		fatalExit(fmt.Errorf("cannot create the request:%v", err))
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fatalExit(fmt.Errorf("cannot make the request:%v", err))
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fatalExit(fmt.Errorf("cannot parse the response:%v", err))
	}

	if resp.StatusCode == 202 {
		fmt.Println("Your contract is successfully verified!")
		return
	}

	var errResp struct {
		Error struct {
			Message string
		}
	}
	err = json.Unmarshal(body, &errResp)
	if err != nil {
		fatalExit(fmt.Errorf("cannot parse the error message: %v", err))
	}
	fatalExit(fmt.Errorf("Cannot verify the contract: %s, error code: %v", errResp.Error.Message, resp.StatusCode))
}

func marshalJSON(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fatalExit(fmt.Errorf("Cannot marshal json: %v", err))
	}
	return string(b)
}

func fatalExit(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	os.Exit(1)
}

// IPFSUpload uploads data to IPFS with a given filename.
func IPFSUpload(ctx context.Context, name string, data []byte) (string, error) {
	// Build multi-part request body.
	var body bytes.Buffer
	mpw := multipart.NewWriter(&body)
	if part, err := mpw.CreateFormFile("file", name); err != nil {
		return "", err
	} else if _, err := part.Write(data); err != nil {
		return "", err
	} else if err := mpw.Close(); err != nil {
		return "", err
	}

	// Execute POST against Infura API.
	resp, err := http.Post("https://ipfs.infura.io:5001/api/v0/add?pin=true", mpw.FormDataContentType(), &body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Unmarshal into data structure to extract hash.
	var jsonResp struct {
		Name string
		Hash string
		Size string
	}
	if err = json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return "", err
	}
	return jsonResp.Hash, nil
}
