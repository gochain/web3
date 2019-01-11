package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"log"

	"github.com/gochain-io/gochain/core/types"
	"github.com/urfave/cli"
)

var verbose bool

func main() {
	var network, rpcUrl, function, contractAddress string
	app := cli.NewApp()
	app.Name = "web3-cli"
	app.Version = "0.0.1"
	app.Usage = "web3 cli tool"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "network",
			Usage:       "The name of the network (testnet/mainnet/ethereum/ropsten/localhost)",
			Value:       "mainnet",
			Destination: &network,
			EnvVar:      "NETWORK",
			Hidden:      false},
		cli.StringFlag{
			Name:        "rpc-url",
			Usage:       "The network RPC URL",
			Destination: &rpcUrl,
			EnvVar:      "RPC_URL",
			Hidden:      false},
		cli.BoolFlag{
			Name: "verbose",
			Usage: "Enable verbose logging",
			Destination: &verbose,
			Hidden: false},
	}
	app.Commands = []cli.Command{
		{
			Name:    "block",
			Usage:   "Show information about the block",
			Aliases: []string{"bl"},
			Action: func(c *cli.Context) {
				GetBlockDetails(network, rpcUrl, c.Args().First())
			},
		},
		{
			Name:    "transaction",
			Aliases: []string{"tx"},
			Usage:   "Show information about the transaction",
			Action: func(c *cli.Context) {
				GetTransactionDetails(network, rpcUrl, c.Args().First())
			},
		},
		{
			Name:    "address",
			Aliases: []string{"addr"},
			Usage:   "Show information about the address",
			Action: func(c *cli.Context) {
				GetAddressDetails(network, rpcUrl, c.Args().First())
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
						BuildSol(c.Args().First())
					},
				},
				{
					Name:  "deploy",
					Usage: "Build and deploy the specified contract to the network",
					Action: func(c *cli.Context) {
						fmt.Println("deploying the contract from the file: ", c.Args().First())
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
				GetSnapshot(network, rpcUrl)
			},
		},
	}
	app.Run(os.Args)
}

func getRPCURL(network, rpcURL string) string {

	if rpcURL != "" {
		if network != "" {
			log.Fatalf("Cannot set both rpcURL %q and network %q", rpcURL, network)
		}
	} else {
		switch network {
		case "testnet":
			rpcURL = "https://testnet-rpc.gochain.io"
		case "mainnet":
			rpcURL = "https://rpc.gochain.io"
		case "localhost":
			rpcURL = "http://localhost:8545"
		case "ethereum":
			rpcURL = "https://main-rpc.linkpool.io"
		case "ropsten":
			rpcURL = "https://ropsten-rpc.linkpool.io"
		default:
			log.Fatal("Unrecognized network:", network)
			return ""
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
func parseBigInt(value string) (*big.Int, error) {
	if value == "" {
		return nil, nil
	}
	i := big.Int{}
	_, err := fmt.Sscan(value, &i)
	return &i, err
}

func GetBlockDetails(network, rpcURL, blockNumber string) {
	client := GetClient(getRPCURL(network, rpcURL))
	blockN, err := parseBigInt(blockNumber)
	if err != nil {
		log.Fatalf("block number must be integer %q: %v", blockNumber, err)
	}
	block, err := client.GetBlockByNumber(blockN)
	if err != nil {
		log.Fatalf("Cannot get block details from the network: %v", err)
	}
	if verbose {
		log.Println("Block details:")
	}
	marshalJSON(block.Header())
}

func GetTransactionDetails(network, rpcURL, txhash string) {
	client := GetClient(getRPCURL(network, rpcURL))
	tx, isPending, err := client.GetTransactionByHash(txhash)
	if err != nil {
		log.Fatalf("Cannot get transaction details from the network: %v", err)
	}
	if verbose {
		log.Println("Transaction details:")
	}
	type details struct {
		Transaction *types.Transaction
		Pending bool
	}
	marshalJSON(details{Transaction:tx, Pending: isPending})
}

type addressDetails struct {
	Balance *big.Int
	Code *string
}

func GetAddressDetails(network, rpcURL, addrHash string) {
	client := GetClient(getRPCURL(network, rpcURL))
	addr, err := client.GetBalance(addrHash, nil)
	if err != nil {
		log.Fatalf("Cannot get address balance from the network: %v", err)
	}
	code, err := client.GetCode(addrHash, nil)
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
	marshalJSON(&ad)
}

func GetSnapshot(network, rpcUrl string) {
	client := GetClient(getRPCURL(network, rpcUrl))
	s, err := client.GetSnapshot()
	if err != nil {
		log.Fatalf("Cannot get snapshot from the network: %v", err)
	}
	if verbose {
		log.Println("Snapshot details:")
	}
	marshalJSON(s)
}

func BuildSol(filename string) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read file %q: %v", filename, err)
	}
	str := string(b) // convert content to a 'string'
	if verbose {
		log.Println("Building Sol:", str)
	}
	compileData, err := CompileSolidityString(str)
	if verbose {
		log.Println("Compiled Sol Details:")
	}
	marshalJSON(compileData)
}

func marshalJSON(data interface{}) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("Cannot marshal json: %v", err)
	}
	fmt.Println(string(b))
}