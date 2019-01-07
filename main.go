package main

import (
	"fmt"
	"os"

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
			Usage:       "The name of the network (testnet/mainnet)",
			Value:       "testnet",
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
				fmt.Println("block details: ", c.Args().First())
			},
		},
		{
			Name:    "transaction",
			Aliases: []string{"tx"},
			Flags:   globalFlags,
			Usage:   "Show information about the transaction",
			Action: func(c *cli.Context) {
				fmt.Println("transaction details: ", c.Args().First())
			},
		},
		{
			Name:    "address",
			Aliases: []string{"addr"},
			Flags:   globalFlags,
			Usage:   "Show information about the address",
			Action: func(c *cli.Context) {
				fmt.Println("address details: ", c.Args().First())
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
					Usage: "Deploy the specified contract to the network",
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
