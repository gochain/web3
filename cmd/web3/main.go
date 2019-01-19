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
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/gochain-io/gochain/common"
	"github.com/urfave/cli"

	"github.com/gochain-io/web3"
)

// Flags
var (
	verbose bool
	format  string
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
	var network, rpcUrl, function, contractAddress, privateKey string
	var testnet bool

	app := cli.NewApp()
	app.Name = "web3"
	app.Version = "0.0.3"
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
		cli.StringFlag{
			Name:        "format",
			Usage:       "Output format",
			Destination: &format,
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
		{
			Name:    "id",
			Aliases: []string{"id"},
			Usage:   "Show chain id information",
			Action: func(c *cli.Context) {
				GetID(ctx, rpcUrl)
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
		rpcURL = web3.NetworkURL(network)
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

func parseBigInt(value string) (*big.Int, error) {
	if value == "" {
		return nil, nil
	}
	i := big.Int{}
	_, err := fmt.Sscan(value, &i)
	return &i, err
}

func GetBlockDetails(ctx context.Context, rpcURL, blockNumber string) {
	client, err := web3.NewClient(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", rpcURL, err)
	}
	defer client.Close()
	blockN, err := parseBigInt(blockNumber)
	if err != nil {
		log.Fatalf("block number must be integer %q: %v", blockNumber, err)
	}
	block, err := client.GetBlockByNumber(ctx, blockN, false)
	if err != nil {
		log.Fatalf("Cannot get block details from the network: %v", err)
	}
	if verbose {
		log.Println("Block details:")
	}
	switch format {
	case "json":
		fmt.Println(marshalJSON(block))
		return
	}

	n, err := block.NumberInt64()
	if err != nil {
		log.Fatalf("Failed to parse number %q: %v", block.Difficulty, err)
	}
	fmt.Println("Number:", n)
	ts, err := block.TimestampUnix()
	if err != nil {
		log.Fatalf("Failed to parse unix timestamp %q: %v", block.Timestamp, err)
	}
	fmt.Println("Time:", time.Unix(ts, 0).In(time.UTC).Format(time.RFC3339))
	fmt.Println("Transactions:", len(block.Txs))
	gasUsed, err := block.GasUsedInt64()
	if err != nil {
		log.Fatalf("Failed to parse gas used %q: %v", block.GasUsed, err)
	}
	gasLimit, err := block.GasLimitInt64()
	if err != nil {
		log.Fatalf("Failed to parse gas limit %q: %v", block.GasLimit, err)
	}
	gasPct := big.NewRat(gasUsed, gasLimit)
	gasPct = gasPct.Mul(gasPct, big.NewRat(100, 1))
	fmt.Printf("Gas Used: %d/%d (%s%%)\n", gasUsed, gasLimit, gasPct.FloatString(2))
	d, err := strconv.ParseInt(block.Difficulty, 0, 64)
	if err != nil {
		log.Fatalf("Failed to parse difficulty %q: %v", block.Difficulty, err)
	}
	fmt.Println("Difficulty:", d)
	td, err := strconv.ParseInt(block.TotalDifficulty, 0, 64)
	if err != nil {
		log.Fatalf("Failed to parse total difficulty %q: %v", block.TotalDifficulty, err)
	}
	fmt.Println("Total Difficulty:", td)
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
		fmt.Println("Signers:", block.Signers)
	}
	if len(block.Voters) > 0 {
		fmt.Println("Voters:", block.Voters)
	}
	if len(block.Signer) > 0 {
		fmt.Println("Signer:", block.Signer)
	}
}

var (
	weiPerGO   = big.NewInt(1000000000000000000)
	weiPerGwei = big.NewInt(1000000000)
)

func GetTransactionDetails(ctx context.Context, rpcURL, txhash string) {
	client, err := web3.NewClient(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", rpcURL, err)
	}
	defer client.Close()
	tx, err := client.GetTransactionByHash(ctx, txhash)
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
	fmt.Println("To:", tx.To.String())
	amt := new(big.Rat).SetFrac((*big.Int)(&tx.Value), weiPerGO)
	fmt.Println("Value:", amt.FloatString(18), "GO")
	fmt.Println("Nonce:", uint64(tx.Nonce))
	fmt.Println("Gas Limit:", (*big.Int)(&tx.GasLimit))
	gp := new(big.Rat).SetFrac((*big.Int)(&tx.GasPrice), weiPerGwei)
	fmt.Println("Gas Price:", gp.FloatString(18), "gwei")
	if tx.BlockHash == (common.Hash{}) {
		fmt.Println("Pending: true")
	} else {
		fmt.Println("Block Number:", (*big.Int)(&tx.BlockNumber))
		fmt.Println("Block Hash:", tx.BlockHash.String())
	}
}

func GetAddressDetails(ctx context.Context, rpcURL, addrHash string) {
	client, err := web3.NewClient(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to %q: %v", rpcURL, err)
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

	fmt.Println("Balance:", bal)
	fmt.Println("Code:", string(code))
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
		filename := contractName[8:]
		err := ioutil.WriteFile(filename+".bin", []byte(v.RuntimeCode), 0600)
		if err != nil {
			log.Fatalf("Cannot write the bin file: %v", err)
		}
		err = ioutil.WriteFile(filename+".abi", []byte(marshalJSON(v.Info.AbiDefinition)), 0600)
		if err != nil {
			log.Fatalf("Cannot write the abi file: %v", err)
		}
		filenames = append(filenames, filename)
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
	tx, err := client.DeployContract(ctx, privateKey, string(dat))
	if err != nil {
		log.Fatalf("Cannot deploy the contract: %v", err)
	}
	receipt, err := client.WaitForReceipt(ctx, tx)
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

func marshalJSON(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("Cannot marshal json: %v", err)
	}
	return string(b)
}
