package web3

import "math/big"

const (
	testnetExplorerURL = "https://testnet-explorer.gochain.io/api"
	mainnetExplorerURL = "https://explorer.gochain.io/api"
	testnetURL         = "https://testnet-rpc.gochain.io"
	mainnetURL         = "https://rpc.gochain.io"
)

var Networks = map[string]Network{
	"localhost": {
		Name: "localhost",
		URL:  "http://localhost:8545",
		Unit: "ETH",
	},
	"goerli": {
		Name:        "goerli",
		URL:         "https://ethereum-goerli.publicnode.com",
		Unit:        "ETH",
		ExplorerURL: "https://goerli.etherscan.io",
	},
	"sepolia": {
		Name:        "sepolia",
		URL:         "https://endpoints.omniatech.io/v1/eth/sepolia/public",
		ChainID:     big.NewInt(11155111),
		Unit:        "SepoliaETH",
		ExplorerURL: "https://sepolia.etherscan.io",
	},
	"ethereum": {
		Name: "ethereum",
		// change the URL
		URL:         "https://mainnet.infura.io/v3/bc5b0e5cfd9b4385befb69a68a9400c3",
		ChainID:     big.NewInt(1),
		Unit:        "ETH",
		ExplorerURL: "https://etherscan.io",
	},
	"ropsten": {
		Name:    "ropsten",
		URL:     "https://ropsten-rpc.linkpool.io",
		ChainID: big.NewInt(3),
		Unit:    "ETH",
	},
}

type Network struct {
	Name        string
	URL         string
	ExplorerURL string
	Unit        string
	ChainID     *big.Int
}
