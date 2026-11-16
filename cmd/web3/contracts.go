package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gochain/web3"
	"github.com/gochain/web3/assets"
	"github.com/urfave/cli/v2"
)

func contractCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:    "contract",
		Aliases: []string{"c"},
		Usage:   "Smart contract build, deploy, call, verify, events, and EIP-1967 proxy operations",
		Subcommands: []*cli.Command{
			contractFlattenCmd(ctx),
			contractBuildCmd(ctx),
			contractDeployCmd(ctx),
			contractVerifyCmd(ctx),
			contractListCmd(),
			contractCallCmd(ctx),
			contractEventsCmd(ctx),
			contractUpgradeCmd(ctx),
			contractTargetCmd(ctx),
			contractAdminCmd(ctx),
			contractPauseCmd(ctx),
			contractResumeCmd(ctx),
		},
	}
}

func contractFlattenCmd(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:      "flatten",
		Usage:     "Flatten a Solidity source file and its imports into a single file",
		ArgsUsage: "<Contract.sol>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output file (default: <name>_flatten.sol)"},
		},
		Action: func(c *cli.Context) error {
			src := strings.TrimSpace(c.Args().First())
			if src == "" {
				return errors.New("missing Solidity source file argument")
			}
			_, outFile, err := FlattenSourceFile(ctx, src, c.String("output"))
			if err != nil {
				return fmt.Errorf("cannot flatten %q: %w", src, err)
			}
			if isJSONFormat() {
				fmt.Println(marshalJSON(map[string]string{"output": outFile}))
				return nil
			}
			fmt.Println("Flattened contract:", outFile)
			return nil
		},
	}
}

func contractBuildCmd(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:      "build",
		Usage:     "Compile a Solidity contract into .bin and .abi files",
		ArgsUsage: "<Contract.sol>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "solc-version", Aliases: []string{"c"}, Usage: "Solc version (uses local solc if matching, else ethereum/solc Docker tag)"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output directory"},
			&cli.StringFlag{Name: "evm-version", Usage: "Optional target EVM version (e.g. paris, shanghai, cancun)"},
			&cli.BoolFlag{Name: "optimize", Usage: "Enable Solidity optimizer", Value: true},
		},
		Action: func(c *cli.Context) error {
			filename := strings.TrimSpace(c.Args().First())
			if filename == "" {
				return errors.New("missing Solidity file argument")
			}
			return buildSol(ctx, filename, c.String("solc-version"), c.String("evm-version"), c.Bool("optimize"), c.String("output"))
		},
	}
}

func buildSol(ctx context.Context, filename, solcVersion, evmVersion string, optimize bool, output string) error {
	flatOut := ""
	if output != "" {
		if err := os.MkdirAll(output, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		basename := filepath.Base(filename)
		oName := strings.TrimSuffix(basename, filepath.Ext(basename)) + "_flatten.sol"
		flatOut = filepath.Join(output, oName)
	}
	name, sourceFile, err := FlattenSourceFile(ctx, filename, flatOut)
	if err != nil {
		return fmt.Errorf("cannot flatten source: %w", err)
	}
	b, err := os.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to read %q: %w", sourceFile, err)
	}

	compileData, err := web3.CompileSolidityString(ctx, string(b), solcVersion, evmVersion, optimize)
	if err != nil {
		return fmt.Errorf("failed to compile %q: %w", sourceFile, err)
	}

	var filenames []string
	for contractKey, v := range compileData {
		parts := strings.Split(contractKey, ":")
		shortContract := parts[len(parts)-1]
		if len(parts) > 1 && parts[0] != "<stdin>" {
			continue
		}
		if name != "" && shortContract != name {
			continue
		}
		outPrefix := filepath.Join(output, shortContract)
		if err := os.WriteFile(outPrefix+".bin", []byte(v.Code), 0644); err != nil {
			return fmt.Errorf("cannot write bin file: %w", err)
		}
		if err := os.WriteFile(outPrefix+".abi", []byte(marshalJSON(v.Info.AbiDefinition)), 0644); err != nil {
			return fmt.Errorf("cannot write abi file: %w", err)
		}
		filenames = append(filenames, shortContract)
	}
	sort.Strings(filenames)

	if isJSONFormat() {
		var bins, abis []string
		for _, f := range filenames {
			bins = append(bins, filepath.Join(output, f+".bin"))
			abis = append(abis, filepath.Join(output, f+".abi"))
		}
		fmt.Println(marshalJSON(map[string]interface{}{
			"source": sourceFile,
			"bin":    bins,
			"abi":    abis,
		}))
		return nil
	}

	fmt.Println("Successfully compiled contracts and wrote the following files:")
	fmt.Println("Source file:", sourceFile)
	for _, f := range filenames {
		fmt.Printf("  %s.bin, %s.abi\n", f, f)
	}
	return nil
}

func contractDeployCmd(ctx context.Context) *cli.Command {
	flags := []cli.Flag{
		pkFlag(false),
		&cli.BoolFlag{Name: "upgradeable", Usage: "Deploy behind a standard EIP-1967 upgradeable proxy"},
		&cli.StringFlag{Name: "init-data", Usage: "Optional 0x hex initializer calldata executed atomically on EIP-1967 proxy deploy"},
		&cli.StringFlag{Name: "verify", Usage: "Solidity source file to verify on explorer after deployment"},
		&cli.StringFlag{Name: "explorer-api", Usage: "Explorer API URL"},
		&cli.StringFlag{Name: "solc-version", Aliases: []string{"c"}, Usage: "Compiler version for explorer verification"},
		&cli.StringFlag{Name: "evm-version", Usage: "Target EVM version for explorer verification"},
		&cli.BoolFlag{Name: "optimize", Usage: "Whether contract was compiled with optimization", Value: true},
		timeoutFlag(),
	}
	flags = append(flags, txGasFlags(0)...)

	return &cli.Command{
		Name:      "deploy",
		Usage:     "Deploy a compiled .bin contract to the network",
		ArgsUsage: "<Contract.bin> [constructorArgs...]",
		Flags:     flags,
		Action: func(c *cli.Context) error {
			binFile := strings.TrimSpace(c.Args().First())
			if binFile == "" {
				return errors.New("missing .bin file argument")
			}
			pk := strings.TrimSpace(c.String("private-key"))
			if pk == "" {
				return fmt.Errorf("private key required (pass --private-key or set %s)", pkVarName)
			}
			opts, err := parseTxOptions(c)
			if err != nil {
				return err
			}
			tail := c.Args().Tail()
			ctorArgs := make([]interface{}, len(tail))
			for i, v := range tail {
				ctorArgs[i] = v
			}

			client, network, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()

			abiFile := ""
			if len(ctorArgs) > 0 {
				abiFile = strings.TrimSuffix(binFile, ".bin") + ".abi"
			}
			binData, err := readLocalOrURL(binFile)
			if err != nil {
				return fmt.Errorf("cannot read %q: %w", binFile, err)
			}
			var abiStr string
			if abiFile != "" {
				b, err := readLocalOrURL(abiFile)
				if err != nil {
					return fmt.Errorf("cannot read %q: %w", abiFile, err)
				}
				abiStr = string(b)
			}

			tx, err := web3.DeployContractWithOpts(ctx, client, pk, string(binData), abiStr, opts, ctorArgs...)
			if err != nil {
				return fmt.Errorf("error deploying contract: %w", err)
			}

			timeout := time.Duration(c.Uint64("timeout")) * time.Second
			waitCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			receipt, err := web3.WaitForReceipt(waitCtx, client, tx.Hash)
			if err != nil {
				return fmt.Errorf("cannot get receipt for %s: %w", tx.Hash.Hex(), err)
			}
			if receipt.Status != web3.ReceiptStatusSuccessful {
				return fmt.Errorf("contract deployment transaction %s reverted on-chain", tx.Hash.Hex())
			}

			if !c.Bool("upgradeable") {
				if isJSONFormat() {
					fmt.Println(marshalJSON(receipt))
					return nil
				}
				fmt.Println("Contract has been successfully deployed with transaction:", tx.Hash.Hex())
				fmt.Println("Contract address is:", receipt.ContractAddress.Hex())
				if verifySrc := strings.TrimSpace(c.String("verify")); verifySrc != "" {
					return verifyContractOnExplorer(ctx, network, c.String("explorer-api"), receipt.ContractAddress.Hex(),
						strings.TrimSuffix(filepath.Base(binFile), ".bin"), verifySrc, c.String("solc-version"), c.String("evm-version"), c.Bool("optimize"))
				}
				return nil
			}

			var initBytes []byte
			if rawInit := strings.TrimSpace(c.String("init-data")); rawInit != "" {
				initBytes, err = web3.DecodeHex(rawInit)
				if err != nil {
					return fmt.Errorf("invalid --init-data hex: %w", err)
				}
			}

			proxyTx, err := web3.DeployEIP1967Proxy(ctx, client, pk, receipt.ContractAddress, web3.ZeroAddress, initBytes, opts)
			if err != nil {
				return fmt.Errorf("cannot deploy EIP-1967 proxy: %w", err)
			}
			pWaitCtx, pCancel := context.WithTimeout(ctx, timeout)
			defer pCancel()
			proxyReceipt, err := web3.WaitForReceipt(pWaitCtx, client, proxyTx.Hash)
			if err != nil {
				return fmt.Errorf("cannot get EIP-1967 proxy receipt: %w", err)
			}
			if proxyReceipt.Status != web3.ReceiptStatusSuccessful {
				return fmt.Errorf("EIP-1967 proxy deployment %s reverted", proxyTx.Hash.Hex())
			}

			if isJSONFormat() {
				fmt.Println(marshalJSON(map[string]interface{}{
					"proxy_address":          proxyReceipt.ContractAddress.Hex(),
					"proxy_tx_hash":          proxyTx.Hash.Hex(),
					"implementation_address": receipt.ContractAddress.Hex(),
					"implementation_tx_hash": tx.Hash.Hex(),
				}))
				return nil
			}
			fmt.Println("EIP-1967 Upgradeable contract has been successfully deployed.")
			fmt.Println("Implementation address:", receipt.ContractAddress.Hex())
			fmt.Println("Proxy transaction:     ", proxyTx.Hash.Hex())
			fmt.Println("Contract address is:   ", proxyReceipt.ContractAddress.Hex())
			return nil
		},
	}
}

func contractVerifyCmd(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:      "verify",
		Usage:     "Verify a deployed contract source file on a block explorer",
		ArgsUsage: "<Contract.sol>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "explorer-api", Usage: "Explorer API URL"},
			&cli.StringFlag{Name: "contract-name", Usage: "Deployed contract name"},
			addressFlag("Deployed contract address"),
			&cli.StringFlag{Name: "solc-version", Aliases: []string{"c"}, Usage: "Compiler version"},
			&cli.StringFlag{Name: "evm-version", Usage: "Target EVM version"},
			&cli.BoolFlag{Name: "optimize", Usage: "Whether optimizer was enabled", Value: true},
		},
		Action: func(c *cli.Context) error {
			network, err := resolveNetwork(c)
			if err != nil {
				return err
			}
			return verifyContractOnExplorer(ctx, network, c.String("explorer-api"), c.String("address"),
				c.String("contract-name"), c.Args().First(), c.String("solc-version"), c.String("evm-version"), c.Bool("optimize"))
		},
	}
}

func verifyContractOnExplorer(ctx context.Context, network web3.Network, explorerURL, contractAddress, contractName, sourceCodeFile, compilerVersion, evmVersion string, optimize bool) error {
	if explorerURL == "" {
		if network.ExplorerURL == "" {
			return errors.New("missing --explorer-api argument")
		}
		explorerURL = network.ExplorerURL
	}
	if !web3.IsHexAddress(contractAddress) {
		return fmt.Errorf("invalid contract --address %q", contractAddress)
	}
	if contractName == "" {
		return errors.New("missing --contract-name argument")
	}
	if sourceCodeFile == "" {
		return errors.New("missing source file argument")
	}
	source, err := os.ReadFile(sourceCodeFile)
	if err != nil {
		return fmt.Errorf("cannot read %q: %w", sourceCodeFile, err)
	}
	if compilerVersion == "" {
		sol, err := web3.SolidityVersion(string(source))
		if err != nil {
			return fmt.Errorf("cannot detect pragma solidity version in %q: %w", sourceCodeFile, err)
		}
		compilerVersion = sol.Version
	}
	message := map[string]interface{}{
		"address":          contractAddress,
		"contract_name":    contractName,
		"compiler_version": compilerVersion,
		"optimization":     optimize,
		"source_code":      string(source),
	}
	if evmVersion != "" {
		message["evm_version"] = evmVersion
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	httpClient := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(explorerURL, "/")+"/verify", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", web3.DefaultUserAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("explorer request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("Your contract is successfully verified!")
		return nil
	}
	var errResp struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &errResp)
	if errResp.Error.Message != "" {
		return fmt.Errorf("contract verification failed (HTTP %d): %s", resp.StatusCode, errResp.Error.Message)
	}
	return fmt.Errorf("contract verification failed (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func contractListCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List functions, events, and errors in an ABI",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "abi", Required: true, Usage: "ABI file, URL, or bundled name (erc20/erc721/erc1155/proxy)"},
		},
		Action: func(c *cli.Context) error {
			myabi, err := web3.GetABI(c.String("abi"))
			if err != nil {
				return err
			}
			var keys []string
			for k, m := range myabi.Methods {
				if k == m.Sig() {
					continue
				}
				keys = append(keys, k)
			}
			sort.Strings(keys)
			if isJSONFormat() {
				methods := make([]web3.ABIMethod, 0, len(keys))
				for _, k := range keys {
					methods = append(methods, myabi.Methods[k])
				}
				fmt.Println(marshalJSON(methods))
				return nil
			}
			for _, k := range keys {
				fmt.Println(myabi.Methods[k].String())
			}
			return nil
		},
	}
}

func contractCallCmd(ctx context.Context) *cli.Command {
	flags := []cli.Flag{
		&cli.StringFlag{Name: "function", Usage: "Function name or signature"},
		addressFlag("Deployed contract address"),
		&cli.StringFlag{Name: "abi", Usage: "ABI file, URL, inline signature, or bundled name (erc20/erc721/erc1155/proxy)"},
		&cli.StringFlag{Name: "amount", Usage: "Value to send with transaction (wei, or with suffix like 0.1eth / 10gwei)"},
		pkFlag(false),
		&cli.BoolFlag{Name: "wait", Usage: "Wait for receipt on state-changing transactions"},
		&cli.BoolFlag{Name: "to-string", Usage: "Convert byte outputs to UTF-8 strings"},
		&cli.StringFlag{Name: "data", Usage: "Raw 0x hex calldata (bypasses --abi / --function)"},
		timeoutFlag(),
	}
	flags = append(flags, txGasFlags(0)...)

	return &cli.Command{
		Name:      "call",
		Usage:     "Call a read-only (view/pure) or state-changing contract function",
		ArgsUsage: "[args...]",
		Flags:     flags,
		Action: func(c *cli.Context) error {
			opts, err := parseTxOptions(c)
			if err != nil {
				return err
			}
			if amtStr := strings.TrimSpace(c.String("amount")); amtStr != "" {
				opts.Value, err = parseWeiOrUnitAmount(amtStr)
				if err != nil {
					return err
				}
			}

			var dataB []byte
			if d := strings.TrimSpace(c.String("data")); d != "" {
				dataB, err = hex.DecodeString(strings.TrimPrefix(strings.TrimPrefix(d, "0x"), "0X"))
				if err != nil {
					return fmt.Errorf("invalid --data hex: %w", err)
				}
			}

			rawArgs := c.Args().Slice()
			params := make([]interface{}, len(rawArgs))
			for i, v := range rawArgs {
				params[i] = v
			}

			client, _, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()

			return executeContractCall(ctx, client, c.String("private-key"), c.String("address"),
				c.String("abi"), c.String("function"), opts, c.Bool("wait"), c.Bool("to-string"), dataB, c.Uint64("timeout"), params...)
		},
	}
}

func executeContractCall(ctx context.Context, client web3.Client, privateKey, contractAddress, abiSpec, functionName string,
	opts web3.TxOptions, waitForReceipt, toString bool, rawData []byte, timeoutSec uint64, params ...interface{}) error {
	if !web3.IsHexAddress(contractAddress) {
		return fmt.Errorf("invalid or missing contract --address %q", contractAddress)
	}

	var tx *web3.Transaction
	var myabi *web3.ABI
	var err error

	if len(rawData) > 0 {
		tx, err = web3.CallFunctionWithDataOpts(ctx, client, privateKey, contractAddress, opts, rawData)
	} else {
		if abiSpec == "" {
			// Allow passing full human signature directly in --function, e.g. --function "balanceOf(address):(uint256)"
			if strings.Contains(functionName, "(") {
				abiSpec = functionName
			} else {
				return errors.New("must specify --abi (file, URL, bundled name, or inline signature)")
			}
		}
		myabi, err = web3.GetABI(abiSpec)
		if err != nil {
			return err
		}
		fnName := functionName
		if strings.Contains(fnName, "(") {
			if parsedFn, pErr := web3.ParseHumanSignature(fnName); pErr == nil {
				fnName = parsedFn.Sig()
			}
		}
		m, ok := myabi.ResolveMethod(fnName, len(params))
		if !ok {
			return fmt.Errorf("function %q not found in ABI", functionName)
		}

		if m.IsConstant() {
			res, err := web3.CallConstantFunction(ctx, client, *myabi, contractAddress, m.Sig(), params...)
			if err != nil {
				return fmt.Errorf("error calling constant function: %w", err)
			}
			if isJSONFormat() {
				out := map[string]interface{}{}
				if len(res) == 1 {
					out["response"] = res[0]
				} else {
					out["response"] = res
				}
				fmt.Println(marshalJSON(out))
				return nil
			}
			if toString {
				for _, r := range res {
					switch v := r.(type) {
					case web3.HexBytes:
						fmt.Println(strings.TrimRight(string(v), "\x00"))
					case web3.Hash:
						fmt.Println(strings.TrimRight(string(v[:]), "\x00"))
					default:
						fmt.Printf("%s\n", r)
					}
				}
				return nil
			}
			for _, r := range res {
				if s, ok := r.(fmt.Stringer); ok {
					fmt.Println(s.String())
				} else {
					fmt.Println(r)
				}
			}
			return nil
		}

		tx, err = web3.CallFunctionWithOpts(ctx, client, privateKey, contractAddress, opts, *myabi, m.Sig(), params...)
	}
	if err != nil {
		return fmt.Errorf("error calling contract: %w", err)
	}

	if !waitForReceipt {
		if isJSONFormat() {
			fmt.Println(marshalJSON(map[string]interface{}{"tx_hash": tx.Hash.Hex()}))
		} else {
			fmt.Println("Transaction hash:", tx.Hash.Hex())
		}
		return nil
	}

	if !isJSONFormat() {
		fmt.Println("Transaction hash:", tx.Hash.Hex())
		fmt.Println("Waiting for receipt...")
	}
	waitCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()
	receipt, err := web3.WaitForReceipt(waitCtx, client, tx.Hash)
	if err != nil {
		return fmt.Errorf("getting receipt: %w", err)
	}
	printReceiptDetails(receipt, myabi)
	return nil
}

func contractEventsCmd(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "events",
		Usage: "Query and decode events emitted by a contract",
		Flags: []cli.Flag{
			addressFlag("Deployed contract address"),
			&cli.StringFlag{Name: "abi", Required: true, Usage: "ABI file, URL, or bundled name (erc20/erc721/erc1155/proxy)"},
			&cli.StringFlag{Name: "event", Usage: "Specific event name to filter"},
			&cli.StringFlag{Name: "from-block", Usage: "Start block number (decimal)"},
			&cli.StringFlag{Name: "to-block", Usage: "End block number (decimal)"},
			&cli.StringSliceFlag{Name: "topic", Usage: "Additional topic filter"},
		},
		Action: func(c *cli.Context) error {
			return queryAndPrintLogs(ctx, c)
		},
	}
}

func contractTargetCmd(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "target",
		Usage: "Read implementation address of an EIP-1967 (or legacy GoChain) upgradeable proxy",
		Flags: []cli.Flag{
			addressFlag("Proxy contract address"),
		},
		Action: func(c *cli.Context) error {
			addrStr := firstNonEmpty(c.Args().First(), c.String("address"))
			if !web3.IsHexAddress(addrStr) {
				return fmt.Errorf("invalid or missing proxy --address %q", addrStr)
			}
			proxyAddr := web3.HexToAddress(addrStr)

			client, _, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()

			// 1. Standard EIP-1967 implementation storage slot (works on all OZ, UUPS, Transparent, and web3 proxies!)
			implSlot := web3.HexToHash(assets.EIP1967ImplementationSlot)
			slotVal, err := client.GetStorageAt(ctx, proxyAddr, implSlot, nil)
			if err == nil && !slotVal.IsZero() {
				impl := web3.BytesToAddress(slotVal[12:])
				printAddressResult("implementation", impl)
				return nil
			}

			// 2. Fallback: legacy GoChain proxy slot keccak256("gochain.proxy.target")
			legacySlot := web3.Keccak256Hash([]byte("gochain.proxy.target"))
			legacyVal, err := client.GetStorageAt(ctx, proxyAddr, legacySlot, nil)
			if err == nil && !legacyVal.IsZero() {
				impl := web3.BytesToAddress(legacyVal[12:])
				printAddressResult("implementation", impl)
				return nil
			}

			// 3. Fallback: call legacy target() view function if non-EIP-1967 proxy
			legacyTargetABI, err := web3.GetABI("target():(address)")
			if err != nil {
				return err
			}
			res, err := web3.CallConstantFunction(ctx, client, *legacyTargetABI, proxyAddr.Hex(), "target")
			if err != nil {
				return fmt.Errorf("could not read proxy implementation slot or target(): %w", err)
			}
			printAddressResult("implementation", res[0].(web3.Address))
			return nil
		},
	}
}

func contractAdminCmd(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "admin",
		Usage: "Read admin address of an EIP-1967 upgradeable proxy",
		Flags: []cli.Flag{
			addressFlag("Proxy contract address"),
		},
		Action: func(c *cli.Context) error {
			addrStr := firstNonEmpty(c.Args().First(), c.String("address"))
			if !web3.IsHexAddress(addrStr) {
				return fmt.Errorf("invalid or missing proxy --address %q", addrStr)
			}
			proxyAddr := web3.HexToAddress(addrStr)

			client, _, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()

			adminSlot := web3.HexToHash(assets.EIP1967AdminSlot)
			slotVal, err := client.GetStorageAt(ctx, proxyAddr, adminSlot, nil)
			if err != nil {
				return err
			}
			printAddressResult("admin", web3.BytesToAddress(slotVal[12:]))
			return nil
		},
	}
}

func contractUpgradeCmd(ctx context.Context) *cli.Command {
	flags := []cli.Flag{
		addressFlag("Proxy contract address"),
		&cli.StringFlag{Name: "to", Required: true, Usage: "New implementation contract address"},
		&cli.StringFlag{Name: "init-data", Usage: "Optional 0x hex calldata to execute via upgradeToAndCall"},
		&cli.StringFlag{Name: "amount", Usage: "Optional wei/unit value to send with upgradeToAndCall"},
		pkFlag(true),
		timeoutFlag(),
	}
	flags = append(flags, txGasFlags(0)...)

	return &cli.Command{
		Name:  "upgrade",
		Usage: "Upgrade an upgradeable proxy to a new implementation address",
		Flags: flags,
		Action: func(c *cli.Context) error {
			proxyAddr := strings.TrimSpace(c.String("address"))
			targetAddr := strings.TrimSpace(c.String("to"))
			if !web3.IsHexAddress(proxyAddr) {
				return fmt.Errorf("invalid proxy --address %q", proxyAddr)
			}
			if !web3.IsHexAddress(targetAddr) {
				return fmt.Errorf("invalid --to implementation address %q", targetAddr)
			}
			opts, err := parseTxOptions(c)
			if err != nil {
				return err
			}
			if amt := strings.TrimSpace(c.String("amount")); amt != "" {
				opts.Value, err = parseWeiOrUnitAmount(amt)
				if err != nil {
					return err
				}
			}

			client, _, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()

			proxyABI, err := web3.ParseABI(assets.EIP1967ProxyABI)
			if err != nil {
				return err
			}
			var tx *web3.Transaction
			if initHex := strings.TrimSpace(c.String("init-data")); initHex != "" {
				initData, err := web3.DecodeHex(initHex)
				if err != nil {
					return fmt.Errorf("invalid --init-data: %w", err)
				}
				tx, err = web3.CallFunctionWithOpts(ctx, client, c.String("private-key"), proxyAddr, opts, proxyABI, "upgradeToAndCall", targetAddr, initData)
			} else {
				tx, err = web3.CallFunctionWithOpts(ctx, client, c.String("private-key"), proxyAddr, opts, proxyABI, "upgradeTo", targetAddr)
			}
			if err != nil {
				return fmt.Errorf("cannot upgrade proxy: %w", err)
			}
			return waitAndReportProxyTx(ctx, client, tx.Hash, c.Uint64("timeout"))
		},
	}
}

func contractPauseCmd(ctx context.Context) *cli.Command {
	return proxyActionCmd(ctx, "pause", "Pause an upgradeable proxy contract")
}

func contractResumeCmd(ctx context.Context) *cli.Command {
	return proxyActionCmd(ctx, "resume", "Resume a paused upgradeable proxy contract")
}

func proxyActionCmd(ctx context.Context, method, usage string) *cli.Command {
	flags := []cli.Flag{
		addressFlag("Proxy contract address"),
		pkFlag(true),
		timeoutFlag(),
	}
	flags = append(flags, txGasFlags(0)...)

	return &cli.Command{
		Name:  method,
		Usage: usage,
		Flags: flags,
		Action: func(c *cli.Context) error {
			addrStr := firstNonEmpty(c.Args().First(), c.String("address"))
			if !web3.IsHexAddress(addrStr) {
				return fmt.Errorf("invalid or missing proxy --address %q", addrStr)
			}
			opts, err := parseTxOptions(c)
			if err != nil {
				return err
			}
			client, _, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()

			proxyABI, err := web3.ParseABI(assets.EIP1967ProxyABI)
			if err != nil {
				return err
			}
			tx, err := web3.CallFunctionWithOpts(ctx, client, c.String("private-key"), addrStr, opts, proxyABI, method)
			if err != nil {
				return fmt.Errorf("cannot %s contract: %w", method, err)
			}
			return waitAndReportProxyTx(ctx, client, tx.Hash, c.Uint64("timeout"))
		},
	}
}

func waitAndReportProxyTx(ctx context.Context, client web3.Client, hash web3.Hash, timeoutSec uint64) error {
	waitCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()
	receipt, err := web3.WaitForReceipt(waitCtx, client, hash)
	if err != nil {
		return fmt.Errorf("cannot get receipt for %s: %w", hash.Hex(), err)
	}
	if receipt.Status != web3.ReceiptStatusSuccessful {
		return fmt.Errorf("proxy transaction %s reverted on-chain", hash.Hex())
	}
	if isJSONFormat() {
		fmt.Println(marshalJSON(receipt))
		return nil
	}
	fmt.Println("Transaction hash:", receipt.TxHash.Hex())
	return nil
}

func printAddressResult(field string, addr web3.Address) {
	if isJSONFormat() {
		fmt.Println(marshalJSON(map[string]string{field: addr.Hex()}))
		return
	}
	fmt.Println(addr.Hex())
}

func parseWeiOrUnitAmount(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return big.NewInt(0), nil
	}
	if v, ok := new(big.Int).SetString(s, 10); ok {
		return v, nil
	}
	return web3.ParseAmount(s)
}

func readLocalOrURL(loc string) ([]byte, error) {
	if strings.HasPrefix(loc, "http://") || strings.HasPrefix(loc, "https://") {
		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequest(http.MethodGet, loc, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", web3.DefaultUserAgent)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, loc)
		}
		return io.ReadAll(resp.Body)
	}
	return os.ReadFile(loc)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
