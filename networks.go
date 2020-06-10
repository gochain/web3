package web3

const (
	testnetExplorerURL = "https://testnet-explorer.gochain.io/api"
	mainnetExplorerURL = "https://explorer.gochain.io/api"
	testnetURL         = "https://testnet-rpc.gochain.io"
	mainnetURL         = "https://rpc.gochain.io"
)

var Networks = map[string]Network{
	"testnet": {
		Name:        "testnet",
		URL:         testnetURL,
		Unit:        "GO",
		ExplorerURL: testnetExplorerURL,
	},
	"gochain": {
		Name:        "gochain",
		URL:         mainnetURL,
		Unit:        "GO",
		ExplorerURL: mainnetExplorerURL,
	},
	"localhost": {
		Name: "localhost",
		URL:  "http://localhost:8545",
		Unit: "GO",
	},
	"ethereum": {
		Name: "ethereum",
		URL:  "https://mainnet.infura.io/v3/bc5b0e5cfd9b4385befb69a68a9400c3",
		// URL: "https://cloudflare-eth.com", // these don't worry very well, constant problems
		// URL: "https://main-rpc.linkpool.io",
		Unit:        "ETH",
		ExplorerURL: "https://etherscan.io",
	},
	"ropsten": {
		Name: "ropsten",
		URL:  "https://ropsten-rpc.linkpool.io",
		Unit: "ETH",
	},
}

type Network struct {
	Name        string
	URL         string
	ExplorerURL string
	Unit        string
}
