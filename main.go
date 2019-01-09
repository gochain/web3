package main

import (
	"fmt"
	"math/big"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
)

func main() {
	var network, function, contractAddress string
	app := cli.NewApp()
	app.Name = "web3-cli"
	app.Version = "0.0.1"
	app.Usage = "web3 cli tool"
	globalFlags := []cli.Flag{
		cli.StringFlag{
			Name:        "network",
			Usage:       "The name of the network (gochain-testnet/gochain-mainnet/ethereum)",
			Value:       "gochain-testnet",
			Destination: &network,
			Hidden:      false},
	}
	app.Commands = []cli.Command{
		{
			Name:    "block",
			Usage:   "Show information about the block",
			Aliases: []string{"bl"},
			Flags:   globalFlags,
			Action: func(c *cli.Context) {
				GetBlockDetails(network, c.Args().First())
			},
		},
		{
			Name:    "transaction",
			Aliases: []string{"tx"},
			Flags:   globalFlags,
			Usage:   "Show information about the transaction",
			Action: func(c *cli.Context) {
				GetTransactionDetails(network, c.Args().First())
			},
		},
		{
			Name:    "address",
			Aliases: []string{"addr"},
			Flags:   globalFlags,
			Usage:   "Show information about the address",
			Action: func(c *cli.Context) {
				GetAddressDetails(network, c.Args().First())
			},
		},
		{
			Name:    "contract",
			Aliases: []string{"c"},
			Flags:   globalFlags,
			Usage:   "actions with contracts",
			Subcommands: []cli.Command{
				{
					Name:  "build",
					Usage: "Build the specified contract",
					Action: func(c *cli.Context) {
						fmt.Println("building the contract from the file: ", c.Args().First())
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

func getRPCURL(network string) string {

	switch network {
	case "gochain-mainnet":
		return "https://rpc.gochain.io"
	case "ethereum":
		return ""
	default:
		//gochain-testnet
		return "https://testnet-rpc.gochain.io"
	}
}
func parseBigInt(value string) (*big.Int, error) {
	i := big.Int{}
	_, err := fmt.Sscan(value, &i)
	return &i, err
}

func GetBlockDetails(network, blockNumber string) {
	client := GetClient(getRPCURL(network))
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
func GetTransactionDetails(network, txhash string) {
	client := GetClient(getRPCURL(network))
	tx, isPending, err := client.GetTransactionByHash(txhash)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot get transaction details from the network")
	}
	log.Info().Interface("Transaction", tx).Bool("Pending", isPending).Msg("Transaction details")
}
func GetAddressDetails(network, addrHash string) {
	client := GetClient(getRPCURL(network))
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
