package web3

import (
	"math/big"
	"sort"
)

const (
	testnetExplorerURL = "https://testnet-explorer.gochain.io/api"
	mainnetExplorerURL = "https://explorer.gochain.io/api"
	testnetURL         = "https://testnet-rpc.gochain.io"
	mainnetURL         = "https://rpc.gochain.io"
)

var Networks = map[string]Network{
	"gochain": {
		Name:        "gochain",
		URL:         mainnetURL,
		ChainID:     big.NewInt(60),
		Unit:        "GO",
		ExplorerURL: mainnetExplorerURL,
	},
	"testnet": {
		Name:        "testnet",
		URL:         testnetURL,
		ChainID:     big.NewInt(31337),
		Unit:        "GO",
		ExplorerURL: testnetExplorerURL,
	},
	"localhost": {
		Name: "localhost",
		URL:  "http://localhost:8545",
		Unit: "ETH",
	},
	"ethereum": {
		Name:        "ethereum",
		URL:         "https://ethereum-rpc.publicnode.com",
		ChainID:     big.NewInt(1),
		Unit:        "ETH",
		ExplorerURL: "https://etherscan.io",
	},
	"sepolia": {
		Name:        "sepolia",
		URL:         "https://ethereum-sepolia-rpc.publicnode.com",
		ChainID:     big.NewInt(11155111),
		Unit:        "ETH",
		ExplorerURL: "https://sepolia.etherscan.io",
	},
	"holesky": {
		Name:        "holesky",
		URL:         "https://ethereum-holesky-rpc.publicnode.com",
		ChainID:     big.NewInt(17000),
		Unit:        "ETH",
		ExplorerURL: "https://holesky.etherscan.io",
	},
	"base": {
		Name:        "base",
		URL:         "https://mainnet.base.org",
		ChainID:     big.NewInt(8453),
		Unit:        "ETH",
		ExplorerURL: "https://basescan.org",
	},
	"base-sepolia": {
		Name:        "base-sepolia",
		URL:         "https://sepolia.base.org",
		ChainID:     big.NewInt(84532),
		Unit:        "ETH",
		ExplorerURL: "https://sepolia.basescan.org",
	},
	"arbitrum": {
		Name:        "arbitrum",
		URL:         "https://arb1.arbitrum.io/rpc",
		ChainID:     big.NewInt(42161),
		Unit:        "ETH",
		ExplorerURL: "https://arbiscan.io",
	},
	"optimism": {
		Name:        "optimism",
		URL:         "https://mainnet.optimism.io",
		ChainID:     big.NewInt(10),
		Unit:        "ETH",
		ExplorerURL: "https://optimistic.etherscan.io",
	},
	"polygon": {
		Name:        "polygon",
		URL:         "https://polygon-rpc.com",
		ChainID:     big.NewInt(137),
		Unit:        "POL",
		ExplorerURL: "https://polygonscan.com",
	},
	"bsc": {
		Name:        "bsc",
		URL:         "https://bsc-dataseed.binance.org",
		ChainID:     big.NewInt(56),
		Unit:        "BNB",
		ExplorerURL: "https://bscscan.com",
	},
	"avalanche": {
		Name:        "avalanche",
		URL:         "https://api.avax.network/ext/bc/C/rpc",
		ChainID:     big.NewInt(43114),
		Unit:        "AVAX",
		ExplorerURL: "https://snowtrace.io",
	},
}

type Network struct {
	Name        string
	URL         string
	ExplorerURL string
	Unit        string
	ChainID     *big.Int
}

// NetworkNames returns sorted built-in network names.
func NetworkNames() []string {
	names := make([]string, 0, len(Networks))
	for k := range Networks {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
