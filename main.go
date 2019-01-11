package main

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
)

func main() {
	var network, rpcUrl, function, contractAddress string
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	app := cli.NewApp()
	app.Name = "web3-cli"
	app.Version = "0.0.1"
	app.Usage = "web3 cli tool"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "network",
			Usage:       "The name of the network (gochain-testnet/gochain-mainnet/ethereum-mainnet/localhost)",
			Value:       "gochain-testnet",
			Destination: &network,
			EnvVar:      "NETWORK",
			Hidden:      false},
		cli.StringFlag{
			Name:        "rpc-url",
			Usage:       "The network RPC URL",
			Destination: &rpcUrl,
			EnvVar:      "RPC_URL",
			Hidden:      false},
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
	}
	app.Run(os.Args)
}

func getRPCURL(network, rpcURL string) string {

	if rpcURL != "" {
		return rpcURL
	}

	switch network {
	case "gochain-testnet":
		return "https://testnet-rpc.gochain.io"
	case "gochain-mainnet":
		return "https://rpc.gochain.io"
	case "localhost":
		return "http://localhost:8545"
	case "ethereum-mainnet":
		return "https://main-rpc.linkpool.io"
	default:
		log.Fatal().Str("Network", network).Msg("Cannot recognize the network")
		return ""
	}
}
func parseBigInt(value string) (*big.Int, error) {
	i := big.Int{}
	_, err := fmt.Sscan(value, &i)
	return &i, err
}

func GetBlockDetails(network, rpcURL, blockNumber string) {
	client := GetClient(getRPCURL(network, rpcURL))
	blockN, err := parseBigInt(blockNumber)
	if err != nil {
		log.Fatal().Err(err).Str("Block number", blockNumber).Msg("Wrong format of the block number, please use number")
	}
	block, err := client.GetBlockByNumber(blockN)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot get block details from the network")
	}
	log.Info().Interface("Block", block.Header()).Msg("Block details")
}
func GetTransactionDetails(network, rpcURL, txhash string) {
	client := GetClient(getRPCURL(network, rpcURL))
	tx, isPending, err := client.GetTransactionByHash(txhash)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot get transaction details from the network")
	}
	log.Info().Interface("Transaction", tx).Bool("Pending", isPending).Msg("Transaction details")
}
func GetAddressDetails(network, rpcURL, addrHash string) {
	client := GetClient(getRPCURL(network, rpcURL))
	addr, err := client.GetBalance(addrHash, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot get address balance from the network")
	}
	code, err := client.GetCode(addrHash, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot get address code from the network")
	}
	log.Info().Int64("Balance", addr.Int64()).Msg("Address details")
	if len(code) > 0 {
		log.Info().Str("Code", string(code[:])).Msg("Address details")
	}
}

func BuildSol(filename string) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Info().Err(err).Msg("Building Sol")
	}
	str := string(b) // convert content to a 'string'
	log.Info().Str("File", str).Msg("Building Sol")
	compileData, err := CompileSolidityString(str)
	log.Info().Err(err).Msg("Building Sol")
	log.Info().Interface("Compiled", compileData).Msg("Building Sol")
}
