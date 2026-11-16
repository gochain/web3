package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gochain/web3"
	"github.com/urfave/cli/v2"
)

var (
	verbose bool
	format  string
)

const (
	asciiLogo = `  ___  _____  ___  _   _    __    ____  _  _ 
 / __)(  _  )/ __)( )_( )  /__\  (_  _)( \( )
( (_-. )(_)(( (__  ) _ (  /(__)\  _)(_  )  ( 
 \___/(_____)\___)(_) (_)(__)(__)(____)(_)\_)`

	pkVarName      = "WEB3_PRIVATE_KEY"
	addrVarName    = "WEB3_ADDRESS"
	networkVarName = "WEB3_NETWORK"
	rpcURLVarName  = "WEB3_RPC_URL"
)

func main() {
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for range sigCh {
			cancelFn()
		}
	}()

	app := buildApp(ctx)
	args := normalizeCLIArgs(app, os.Args)
	if err := app.RunContext(ctx, args); err != nil {
		fatalExit(err)
	}
}

func buildApp(ctx context.Context) *cli.App {
	app := &cli.App{
		Name:    "web3",
		Version: Version,
		Usage:   "Modern EVM & Web3 command-line tool",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "network",
				Aliases: []string{"n"},
				Usage:   fmt.Sprintf("Target network (%s). Default: \"gochain\"", strings.Join(web3.NetworkNames(), "/")),
				EnvVars: []string{networkVarName},
			},
			&cli.BoolFlag{
				Name:  "testnet",
				Usage: "Shorthand for '--network testnet'",
			},
			&cli.StringFlag{
				Name:    "rpc-url",
				Usage:   "Custom JSON-RPC HTTP(S) or WS(S) URL",
				EnvVars: []string{rpcURLVarName},
			},
			&cli.BoolFlag{
				Name:        "verbose",
				Usage:       "Enable verbose logging",
				Destination: &verbose,
			},
			&cli.StringFlag{
				Name:        "format",
				Aliases:     []string{"f"},
				Usage:       "Output format: json (default: human-readable)",
				Destination: &format,
			},
		},
		Commands: []*cli.Command{
			blockCommand(ctx),
			transactionCommand(ctx),
			receiptCommand(ctx),
			addressCommand(ctx),
			balanceCommand(ctx),
			storageCommand(ctx),
			logsCommand(ctx),
			transferCommand(ctx),
			increaseGasCommand(ctx),
			cancelCommand(ctx),
			replaceCommand(ctx),
			contractCommand(ctx),
			generateCommand(ctx),
			accountCommand(),
			myAddressCommand(),
			signCommand(),
			verifySigCommand(),
			keccakCommand(),
			encodeCommand(),
			decodeCommand(),
			unitsCommand(),
			snapshotCommand(ctx),
			idCommand(ctx),
			startCommand(ctx),
			envCommand(),
		},
	}
	return app
}

// Shared CLI flag builders (isolated per subcommand — no shared mutable pointers).

func pkFlag(required bool) *cli.StringFlag {
	return &cli.StringFlag{
		Name:     "private-key",
		Aliases:  []string{"pk"},
		Usage:    "Hex private key",
		EnvVars:  []string{pkVarName},
		Required: required,
	}
}

func addressFlag(usage string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:    "address",
		Usage:   usage,
		EnvVars: []string{addrVarName},
	}
}

func timeoutFlag() *cli.Uint64Flag {
	return &cli.Uint64Flag{
		Name:  "timeout",
		Usage: "Timeout in seconds when waiting for transaction receipt",
		Value: 60,
	}
}

func txGasFlags(defaultGasLimit uint64) []cli.Flag {
	return []cli.Flag{
		&cli.Uint64Flag{
			Name:  "gas-limit",
			Usage: "Gas limit (0 = auto-estimate via eth_estimateGas)",
			Value: defaultGasLimit,
		},
		&cli.StringFlag{
			Name:  "gas-price",
			Usage: "Legacy gas price in wei",
		},
		&cli.StringFlag{
			Name:  "gas-price-gwei",
			Usage: "Legacy gas price in gwei (decimals allowed, e.g. 1.5)",
		},
		&cli.StringFlag{
			Name:  "max-fee-gwei",
			Usage: "EIP-1559 maxFeePerGas in gwei (decimals allowed)",
		},
		&cli.StringFlag{
			Name:  "priority-fee-gwei",
			Usage: "EIP-1559 maxPriorityFeePerGas in gwei (decimals allowed)",
		},
	}
}

func parseTxOptions(c *cli.Context) (web3.TxOptions, error) {
	var opts web3.TxOptions
	opts.GasLimit = c.Uint64("gas-limit")

	if gp := strings.TrimSpace(c.String("gas-price")); gp != "" {
		v, ok := new(big.Int).SetString(gp, 10)
		if !ok {
			return opts, fmt.Errorf("invalid --gas-price wei value %q", gp)
		}
		opts.GasPrice = v
	}
	if gpGwei := strings.TrimSpace(c.String("gas-price-gwei")); gpGwei != "" {
		v, err := web3.ParseGwei(gpGwei)
		if err != nil {
			return opts, fmt.Errorf("invalid --gas-price-gwei %q: %w", gpGwei, err)
		}
		opts.GasPrice = v
	}
	if maxFee := strings.TrimSpace(c.String("max-fee-gwei")); maxFee != "" {
		v, err := web3.ParseGwei(maxFee)
		if err != nil {
			return opts, fmt.Errorf("invalid --max-fee-gwei %q: %w", maxFee, err)
		}
		opts.MaxFeePerGas = v
	}
	if tipFee := strings.TrimSpace(c.String("priority-fee-gwei")); tipFee != "" {
		v, err := web3.ParseGwei(tipFee)
		if err != nil {
			return opts, fmt.Errorf("invalid --priority-fee-gwei %q: %w", tipFee, err)
		}
		opts.MaxPriorityFeePerGas = v
	}
	return opts, nil
}

func resolveNetwork(c *cli.Context) (web3.Network, error) {
	name := strings.TrimSpace(c.String("network"))
	rpcURL := strings.TrimSpace(c.String("rpc-url"))
	testnet := c.Bool("testnet")
	return getNetwork(name, rpcURL, testnet)
}

func getNetwork(name, rpcURL string, testnet bool) (web3.Network, error) {
	var network web3.Network
	if rpcURL != "" {
		if name != "" {
			return network, fmt.Errorf("cannot set both --rpc-url %q and --network %q", rpcURL, name)
		}
		if testnet {
			return network, fmt.Errorf("cannot set both --rpc-url %q and --testnet", rpcURL)
		}
		network.Name = "custom"
		network.URL = rpcURL
		network.Unit = "ETH"
	} else {
		if testnet {
			if name != "" {
				return network, fmt.Errorf("cannot set both --network %q and --testnet", name)
			}
			name = "testnet"
		} else if name == "" {
			name = "gochain"
		}
		var ok bool
		network, ok = web3.Networks[strings.ToLower(name)]
		if !ok {
			return network, fmt.Errorf("unrecognized network %q (supported: %s)", name, strings.Join(web3.NetworkNames(), ", "))
		}
	}
	if verbose {
		log.Printf("Network: %s (%s)", network.Name, network.URL)
	}
	return network, nil
}

func dialNetwork(c *cli.Context) (web3.Client, web3.Network, error) {
	net, err := resolveNetwork(c)
	if err != nil {
		return nil, net, err
	}
	client, err := web3.Dial(net.URL)
	if err != nil {
		return nil, net, fmt.Errorf("failed to connect to %q: %w", net.URL, err)
	}
	if net.ChainID != nil {
		client.SetChainID(net.ChainID)
	}
	return client, net, nil
}

func isJSONFormat() bool {
	return strings.EqualFold(strings.TrimSpace(format), "json")
}

func marshalJSON(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fatalExit(fmt.Errorf("cannot marshal json: %w", err))
	}
	return string(b)
}

func fatalExit(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	os.Exit(1)
}

// normalizeCLIArgs allows flags to appear after positional arguments on leaf subcommands
// (e.g., `web3 contract build hello.sol --solc-version 0.8.24`, `web3 units 1.5eth --format json`),
// hoisting global app flags before the first subcommand while keeping negative numbers (like `-5`) intact.
func normalizeCLIArgs(app *cli.App, rawArgs []string) []string {
	if len(rawArgs) <= 2 {
		return rawArgs
	}
	var globalFlags []string
	var cmdPath []string
	rest := rawArgs[1:]

	curCmdFlags := []cli.Flag{}
	var curCmds []*cli.Command = app.Commands

	for len(rest) > 0 {
		tok := rest[0]
		if tok == "--" {
			break
		}
		if isFlagToken(tok) {
			if len(cmdPath) == 0 || isKnownFlag(app.Flags, tok) {
				consumed := consumeFlagTokens(app.Flags, rest)
				globalFlags = append(globalFlags, rest[:consumed]...)
				rest = rest[consumed:]
				continue
			}
			if isKnownFlag(curCmdFlags, tok) {
				break
			}
			break
		}
		matched := findSubcommand(curCmds, tok)
		if matched == nil {
			break
		}
		cmdPath = append(cmdPath, tok)
		rest = rest[1:]
		curCmdFlags = matched.Flags
		curCmds = matched.Subcommands
	}

	// Partition remaining tokens into hoisted global flags, leaf command flags, and positional args
	var leafFlags, leafPos []string
	for len(rest) > 0 {
		tok := rest[0]
		if tok == "--" {
			leafPos = append(leafPos, rest...)
			break
		}
		if tok == "-h" || tok == "--help" {
			leafFlags = append(leafFlags, tok)
			rest = rest[1:]
			continue
		}
		if isFlagToken(tok) {
			if isKnownFlag(curCmdFlags, tok) {
				consumed := consumeFlagTokens(curCmdFlags, rest)
				leafFlags = append(leafFlags, rest[:consumed]...)
				rest = rest[consumed:]
				continue
			}
			if isKnownFlag(app.Flags, tok) {
				consumed := consumeFlagTokens(app.Flags, rest)
				globalFlags = append(globalFlags, rest[:consumed]...)
				rest = rest[consumed:]
				continue
			}
		}
		leafPos = append(leafPos, tok)
		rest = rest[1:]
	}

	out := make([]string, 0, len(rawArgs))
	out = append(out, rawArgs[0])
	out = append(out, globalFlags...)
	out = append(out, cmdPath...)
	out = append(out, leafFlags...)
	out = append(out, leafPos...)
	return out
}

func isFlagToken(tok string) bool {
	if len(tok) < 2 || tok[0] != '-' {
		return false
	}
	// Do not treat negative numbers (e.g. -1, -0.5) as flags
	if tok[1] >= '0' && tok[1] <= '9' {
		return false
	}
	return true
}

func findSubcommand(cmds []*cli.Command, token string) *cli.Command {
	for _, cmd := range cmds {
		if cmd.Name == token {
			return cmd
		}
		for _, alias := range cmd.Aliases {
			if alias == token {
				return cmd
			}
		}
	}
	return nil
}

func isKnownFlag(flags []cli.Flag, tok string) bool {
	name := strings.TrimLeft(tok, "-")
	if eq := strings.IndexByte(name, '='); eq >= 0 {
		name = name[:eq]
	}
	for _, f := range flags {
		for _, n := range f.Names() {
			if n == name {
				return true
			}
		}
	}
	return false
}

func consumeFlagTokens(flags []cli.Flag, args []string) int {
	tok := args[0]
	if strings.Contains(tok, "=") {
		return 1
	}
	name := strings.TrimLeft(tok, "-")
	for _, f := range flags {
		for _, n := range f.Names() {
			if n == name {
				if _, isBool := f.(*cli.BoolFlag); isBool {
					return 1
				}
				if len(args) > 1 {
					return 2
				}
				return 1
			}
		}
	}
	return 1
}
