package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"os/signal"
	"syscall"

	"github.com/gochain-io/gochain/core/types"
	"github.com/urfave/cli"
)

var verbose bool

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
	var network, rpcUrl, function, contractAddress, privateKey string
	var testnet bool

	app := cli.NewApp()
	app.Name = "web3"
	app.Version = "0.0.2"
	app.Usage = "web3 cli tool"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "network",
			Usage:       "The name of the network (mainnet/testnet/ethereum/ropsten/localhost). Default is mainnet.",
			Destination: &network,
			EnvVar:      "NETWORK",
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
			EnvVar:      "RPC_URL",
			Hidden:      false},
		cli.BoolFlag{
			Name:        "verbose",
			Usage:       "Enable verbose logging",
			Destination: &verbose,
			Hidden:      false},
	}
	app.Before = func(*cli.Context) error {
		rpcUrl = getRPCURL(network, rpcUrl, testnet)
		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:    "block",
			Usage:   "Show information about the block",
			Aliases: []string{"bl"},
			Action: func(c *cli.Context) {
				GetBlockDetails(ctx, rpcUrl, c.Args().First())
			},
		},
		{
			Name:    "transaction",
			Aliases: []string{"tx"},
			Usage:   "Show information about the transaction",
			Action: func(c *cli.Context) {
				GetTransactionDetails(ctx, rpcUrl, c.Args().First())
			},
		},
		{
			Name:    "address",
			Aliases: []string{"addr"},
			Usage:   "Show information about the address",
			Action: func(c *cli.Context) {
				GetAddressDetails(ctx, rpcUrl, c.Args().First())
			},
		},
		{
			Name:    "contract",
			Aliases: []string{"c"},
			Usage:   "actions with contracts",
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
						DeploySol(ctx, rpcUrl, privateKey, c.Args().First())
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "private-key",
							Usage:       "The private key",
							EnvVar:      "PRIVATE_KEY",
							Destination: &privateKey,
							Hidden:      true},
					},
				},
				{
					Name:  "call",
					Usage: "Call the specified function of the contract",
					Action: func(c *cli.Context) {
						println("calling the function of the deployed contract")
						fmt.Println("calling the function of the deployed contract from:", network)
					},
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "function",
							Usage:       "The name of the function to call",
							Destination: &function,
							Hidden:      false},
						cli.StringFlag{
							Name:        "contract",
							Destination: &contractAddress,
							Usage:       "The address of the deployed contract",
							Hidden:      false},
					},
				},
			},
		},
		{
			Name:    "snapshot",
			Aliases: []string{"sn"},
			Usage:   "Show the clique snapshot",
			Action: func(c *cli.Context) {
				GetSnapshot(ctx, rpcUrl)
			},
		},
	}
	app.Run(os.Args)
}

// getRPCURL resolves the rpcUrl from the user specified options, or quits if an illegal combination or value is found.
func getRPCURL(network, rpcURL string, testnet bool) string {
	if rpcURL != "" {
		if network != "" {
			log.Fatalf("Cannot set both rpcURL %q and network %q", rpcURL, network)
		}
		if testnet {
			log.Fatalf("Cannot set both rpcURL %q and testnet", rpcURL)
		}
	} else {
		if testnet {
			if network != "" {
				log.Fatalf("Cannot set both network %q and testnet", network)
			}
			network = "testnet"
		}
		rpcURL = networkURL(network)
		if rpcURL == "" {
			log.Fatal("Unrecognized network:", network)
		}
		if verbose {
			log.Println("Network:", network)
		}
	}
	if verbose {
		log.Println("RPC URL:", rpcURL)
	}
	return rpcURL
}

func networkURL(network string) string {
	switch network {
	case "testnet":
		return "https://testnet-rpc.gochain.io"
	case "mainnet", "":
		return "https://rpc.gochain.io"
	case "localhost":
		return "http://localhost:8545"
	case "ethereum":
		return "https://main-rpc.linkpool.io"
	case "ropsten":
		return "https://ropsten-rpc.linkpool.io"
	default:
		return ""
	}
}

func parseBigInt(value string) (*big.Int, error) {
	if value == "" {
		return nil, nil
	}
	i := big.Int{}
	_, err := fmt.Sscan(value, &i)
	return &i, err
}

func GetBlockDetails(ctx context.Context, rpcURL, blockNumber string) {
	client := GetClient(rpcURL)
	blockN, err := parseBigInt(blockNumber)
	if err != nil {
		log.Fatalf("block number must be integer %q: %v", blockNumber, err)
	}
	block, err := client.GetBlockByNumber(ctx, blockN)
	if err != nil {
		log.Fatalf("Cannot get block details from the network: %v", err)
	}
	if verbose {
		log.Println("Block details:")
	}
	fmt.Println(marshalJSON(block.Header()))
}

func GetTransactionDetails(ctx context.Context, rpcURL, txhash string) {
	client := GetClient(rpcURL)
	tx, isPending, err := client.GetTransactionByHash(ctx, txhash)
	if err != nil {
		log.Fatalf("Cannot get transaction details from the network: %v", err)
	}
	if verbose {
		log.Println("Transaction details:")
	}
	type details struct {
		Transaction *types.Transaction
		Pending     bool
	}
	fmt.Println(marshalJSON(details{Transaction: tx, Pending: isPending}))
}

type addressDetails struct {
	Balance *big.Int
	Code    *string
}

func GetAddressDetails(ctx context.Context, rpcURL, addrHash string) {
	client := GetClient(rpcURL)
	addr, err := client.GetBalance(ctx, addrHash, nil)
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
	ad := addressDetails{Balance: addr}
	if len(code) > 0 {
		sc := string(code)
		ad.Code = &sc
	}
	fmt.Println(marshalJSON(&ad))
}

func GetSnapshot(ctx context.Context, rpcUrl string) {
	client := GetClient(rpcUrl)
	s, err := client.GetSnapshot(ctx)
	if err != nil {
		log.Fatalf("Cannot get snapshot from the network: %v", err)
	}
	if verbose {
		log.Println("Snapshot details:")
	}
	fmt.Println(marshalJSON(s))
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
	compileData, err := CompileSolidityString(ctx, str)
	if err != nil {
		log.Fatalf("Failed to compile %q: %v", filename, err)
	}
	if verbose {
		log.Println("Compiled Sol Details:", marshalJSON(compileData))
	}

	for contractName, v := range compileData {
		filename := contractName[8:]
		err := ioutil.WriteFile(filename+".bin", []byte(v.RuntimeCode), 0600)
		if err != nil {
			log.Fatalf("Cannot write the bin file: %v", err)
		}
		err = ioutil.WriteFile(filename+".abi", []byte(marshalJSON(v.Info.AbiDefinition)), 0600)
		if err != nil {
			log.Fatalf("Cannot write the abi file: %v", err)
		}
		fmt.Println("Contract has been successfully compiled and the following files have been written:", filename+".bin,", filename+".abi")
	}
}

func DeploySol(ctx context.Context, rpcUrl, privateKey, contractName string) {
	client := GetClient(rpcUrl)
	if _, err := os.Stat(contractName); os.IsNotExist(err) {
		log.Fatalf("Cannot find the bin file: %v", err)
	}
	dat, err := ioutil.ReadFile(contractName)
	if err != nil {
		log.Fatalf("Cannot read the bin file: %v", err)
	}
	tx, err := client.DeployContract(ctx, privateKey, string(dat))
	if err != nil {
		log.Fatalf("Cannot deploy the contract: %v", err)
	}
	receipt, err := client.WaitForReceipt(ctx, tx)
	if err != nil {
		log.Fatalf("Cannot get the receipt: %v", err)
	}
	fmt.Println("Contract has been successfully deployed with transaction:", tx.Hash().Hex())
	fmt.Println("Contract address is:", receipt.ContractAddress.Hex())
}

func marshalJSON(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("Cannot marshal json: %v", err)
	}
	return string(b)
}
