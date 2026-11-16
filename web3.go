package web3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gochain/web3/assets"
)

var NotFoundErr = errors.New("not found")

// TxOptions configures gas fees, gas limit, value, and nonce when submitting a transaction.
type TxOptions struct {
	Value                *big.Int
	GasPrice             *big.Int // Explicit legacy gasPrice (wei)
	MaxFeePerGas         *big.Int // Explicit EIP-1559 maxFeePerGas (wei)
	MaxPriorityFeePerGas *big.Int // Explicit EIP-1559 maxPriorityFeePerGas (wei)
	GasLimit             uint64   // 0 = auto-estimate via eth_estimateGas
	Nonce                *uint64  // nil = auto-fetch pending nonce
}

// CallConstantFunction executes a read-only contract call (eth_call) and unpacks the outputs.
func CallConstantFunction(ctx context.Context, client Client, myabi ABI, address string, functionName string, params ...interface{}) ([]interface{}, error) {
	if !IsHexAddress(address) {
		return nil, fmt.Errorf("invalid contract address %q", address)
	}
	fn, ok := myabi.ResolveMethod(functionName, len(params))
	if !ok {
		return nil, fmt.Errorf("function %q not found in ABI", functionName)
	}
	input, err := myabi.Pack(fn.Sig(), params...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack values: %w", err)
	}
	toAddress := HexToAddress(address)
	res, err := client.Call(ctx, CallMsg{Data: input, To: &toAddress})
	if err != nil {
		return nil, FormatRPCError(err, &myabi)
	}
	if len(res) == 0 {
		code, codeErr := client.GetCode(ctx, address, nil)
		if codeErr == nil && len(code) == 0 {
			return nil, fmt.Errorf("no contract code at address %s", toAddress.Hex())
		}
	}
	vals, err := fn.Outputs.UnpackValues(res)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack values from %s: %w", EncodeHex(res), err)
	}
	return vals, nil
}

// CallTransactFunction submits a transaction to execute a smart contract function call.
// Deprecated: prefer CallFunctionWithArgs or CallFunctionWithOpts.
func CallTransactFunction(ctx context.Context, client Client, myabi ABI, address, privateKeyHex, functionName string,
	amount *big.Int, gasPrice *big.Int, gasLimit uint64, params ...interface{}) (*Transaction, error) {
	return CallFunctionWithArgs(ctx, client, privateKeyHex, address, amount, gasPrice, gasLimit, myabi, functionName, params...)
}

// CallFunctionWithArgs submits a contract transaction with legacy gasPrice/gasLimit parameters (0 gasLimit auto-estimates).
func CallFunctionWithArgs(ctx context.Context, client Client, privateKeyHex, address string,
	amount *big.Int, gasPrice *big.Int, gasLimit uint64, myabi ABI, functionName string, params ...interface{}) (*Transaction, error) {
	return CallFunctionWithOpts(ctx, client, privateKeyHex, address, TxOptions{
		Value:    amount,
		GasPrice: gasPrice,
		GasLimit: gasLimit,
	}, myabi, functionName, params...)
}

// CallFunctionWithOpts submits a contract transaction supporting EIP-1559 fees, auto gas estimation, and revert decoding.
func CallFunctionWithOpts(ctx context.Context, client Client, privateKeyHex, address string,
	opts TxOptions, myabi ABI, functionName string, params ...interface{}) (*Transaction, error) {
	if !IsHexAddress(address) {
		return nil, fmt.Errorf("invalid contract address %q", address)
	}
	fn, ok := myabi.ResolveMethod(functionName, len(params))
	if !ok {
		return nil, fmt.Errorf("function %q not found in ABI", functionName)
	}
	data, err := myabi.Pack(fn.Sig(), params...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack values: %w", err)
	}
	toAddr := HexToAddress(address)
	tx, err := submitTx(ctx, client, privateKeyHex, &toAddr, data, opts, &myabi)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// CallFunctionWithData submits pre-encoded calldata to address.
func CallFunctionWithData(ctx context.Context, client Client, privateKeyHex, address string,
	amount *big.Int, gasPrice *big.Int, gasLimit uint64, data []byte) (*Transaction, error) {
	return CallFunctionWithDataOpts(ctx, client, privateKeyHex, address, TxOptions{
		Value:    amount,
		GasPrice: gasPrice,
		GasLimit: gasLimit,
	}, data)
}

// CallFunctionWithDataOpts submits pre-encoded calldata to address with full TxOptions.
func CallFunctionWithDataOpts(ctx context.Context, client Client, privateKeyHex, address string, opts TxOptions, data []byte) (*Transaction, error) {
	if !IsHexAddress(address) {
		return nil, fmt.Errorf("invalid contract address %q", address)
	}
	toAddr := HexToAddress(address)
	return submitTx(ctx, client, privateKeyHex, &toAddr, data, opts, nil)
}

// DeployBin deploys a .bin file (local path or URL) with optional .abi file and constructor arguments.
func DeployBin(ctx context.Context, client Client, privateKeyHex, binFilename, abiFilename string,
	gasPrice *big.Int, gasLimit uint64, constructorArgs ...interface{}) (*Transaction, error) {
	bin, err := loadFileOrURL(binFilename)
	if err != nil {
		return nil, fmt.Errorf("cannot load bin %q: %w", binFilename, err)
	}
	var abiBytes []byte
	if len(constructorArgs) > 0 {
		abiBytes, err = loadFileOrURL(abiFilename)
		if err != nil {
			return nil, fmt.Errorf("cannot load abi %q: %w", abiFilename, err)
		}
	}
	return DeployContract(ctx, client, privateKeyHex, string(bin), string(abiBytes), gasPrice, gasLimit, constructorArgs...)
}

// DeployContract submits a contract creation transaction (0 gasLimit auto-estimates).
func DeployContract(ctx context.Context, client Client, privateKeyHex, binHex, abiJSON string, gasPrice *big.Int, gasLimit uint64, constructorArgs ...interface{}) (*Transaction, error) {
	return DeployContractWithOpts(ctx, client, privateKeyHex, binHex, abiJSON, TxOptions{
		GasPrice: gasPrice,
		GasLimit: gasLimit,
	}, constructorArgs...)
}

// DeployContractWithOpts submits a contract creation transaction with full TxOptions.
func DeployContractWithOpts(ctx context.Context, client Client, privateKeyHex, binHex, abiJSON string, opts TxOptions, constructorArgs ...interface{}) (*Transaction, error) {
	binData, err := DecodeHex(strings.TrimSpace(binHex))
	if err != nil {
		return nil, fmt.Errorf("cannot decode contract bytecode: %w", err)
	}
	var parsedABI *ABI
	if abiJSON != "" {
		a, err := ParseABI(abiJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ABI: %w", err)
		}
		parsedABI = &a
	}
	if len(constructorArgs) > 0 {
		if parsedABI == nil {
			return nil, errors.New("ABI is required when passing constructor arguments")
		}
		input, err := parsedABI.Pack("", constructorArgs...)
		if err != nil {
			return nil, fmt.Errorf("cannot pack constructor parameters: %w", err)
		}
		binData = append(binData, input...)
	}
	return submitTx(ctx, client, privateKeyHex, nil, binData, opts, parsedABI)
}

// DeployEIP1967Proxy deploys a standard EIP-1967 upgradeable proxy pointing at implementationAddr with optional initData.
func DeployEIP1967Proxy(ctx context.Context, client Client, privateKeyHex string, implementationAddr, adminAddr Address, initData []byte, opts TxOptions) (*Transaction, error) {
	if initData == nil {
		initData = []byte{}
	}
	return DeployContractWithOpts(ctx, client, privateKeyHex, assets.EIP1967ProxyBin, assets.EIP1967ProxyABI, opts, implementationAddr, adminAddr, initData)
}

// Send performs a native coin transfer (GO/ETH).
func Send(ctx context.Context, client Client, privateKeyHex string, to Address, amount *big.Int, gasPrice *big.Int, gasLimit uint64) (*Transaction, error) {
	return SendWithOpts(ctx, client, privateKeyHex, to, TxOptions{
		Value:    amount,
		GasPrice: gasPrice,
		GasLimit: gasLimit,
	})
}

// SendWithOpts performs a native coin transfer supporting EIP-1559 and auto gas estimation.
func SendWithOpts(ctx context.Context, client Client, privateKeyHex string, to Address, opts TxOptions) (*Transaction, error) {
	return submitTx(ctx, client, privateKeyHex, &to, nil, opts, nil)
}

// SendTransaction encodes and broadcasts a pre-signed Transaction.
func SendTransaction(ctx context.Context, client Client, signedTx *Transaction) error {
	raw, err := EncodeRawTransaction(signedTx)
	if err != nil {
		return err
	}
	return client.SendRawTransaction(ctx, raw)
}

func submitTx(ctx context.Context, client Client, privateKeyHex string, to *Address, data []byte, opts TxOptions, myabi *ABI) (*Transaction, error) {
	acc, err := ParsePrivateKey(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	fromAddr := acc.Address()

	chainID, err := client.GetChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't get chain ID: %w", err)
	}

	var nonce uint64
	if opts.Nonce != nil {
		nonce = *opts.Nonce
	} else {
		nonce, err = client.GetPendingTransactionCount(ctx, fromAddr)
		if err != nil {
			return nil, fmt.Errorf("cannot get nonce: %w", err)
		}
	}

	val := opts.Value
	if val == nil {
		val = big.NewInt(0)
	}

	tx := &Transaction{
		ChainID: chainID,
		Nonce:   nonce,
		To:      to,
		Value:   val,
		Input:   data,
		From:    fromAddr,
	}

	if err := populateTxFees(ctx, client, tx, opts); err != nil {
		return nil, err
	}

	gasLimit := opts.GasLimit
	if gasLimit == 0 {
		if to != nil && len(data) == 0 {
			// Check if destination is an EOA; if contract, estimate gas for receive/fallback
			if code, codeErr := client.GetCode(ctx, to.Hex(), nil); codeErr == nil && len(code) == 0 {
				gasLimit = 21000
			}
		}
		if gasLimit == 0 {
			msg := CallMsg{
				From:                 &fromAddr,
				To:                   to,
				Value:                val,
				Data:                 data,
				GasPrice:             tx.GasPrice,
				MaxFeePerGas:         tx.MaxFeePerGas,
				MaxPriorityFeePerGas: tx.MaxPriorityFeePerGas,
			}
			est, estErr := client.EstimateGas(ctx, msg)
			if estErr != nil {
				return nil, fmt.Errorf("gas estimation failed: %w", FormatRPCError(estErr, myabi))
			}
			// Add 20% safety buffer for state changes
			gasLimit = est + est/5
		}
	}
	tx.GasLimit = gasLimit

	raw, err := SignTransaction(tx, chainID, acc.Key())
	if err != nil {
		return nil, fmt.Errorf("cannot sign transaction: %w", err)
	}
	if err := client.SendRawTransaction(ctx, raw); err != nil {
		return nil, fmt.Errorf("cannot send transaction: %w", FormatRPCError(err, myabi))
	}
	return tx, nil
}

func populateTxFees(ctx context.Context, client Client, tx *Transaction, opts TxOptions) error {
	// 1. Explicit EIP-1559 fees requested
	if opts.MaxFeePerGas != nil || opts.MaxPriorityFeePerGas != nil {
		tx.Type = TxTypeDynamicFee
		tip := opts.MaxPriorityFeePerGas
		if tip == nil {
			var err error
			tip, err = client.MaxPriorityFeePerGas(ctx)
			if err != nil {
				tip = Gwei(1)
			}
		}
		maxFee := opts.MaxFeePerGas
		if maxFee == nil {
			blk, err := client.GetBlockByNumber(ctx, nil, false)
			if err == nil && blk.BaseFeePerGas != nil {
				maxFee = new(big.Int).Mul(blk.BaseFeePerGas, big.NewInt(2))
				maxFee.Add(maxFee, tip)
			} else {
				gp, gpErr := client.GetGasPrice(ctx)
				if gpErr != nil {
					return fmt.Errorf("cannot determine maxFeePerGas: %w", gpErr)
				}
				maxFee = new(big.Int).Add(gp, tip)
			}
		}
		if maxFee.Cmp(tip) < 0 {
			maxFee = new(big.Int).Set(tip)
		}
		tx.MaxPriorityFeePerGas = tip
		tx.MaxFeePerGas = maxFee
		return nil
	}

	// 2. Explicit Legacy gasPrice requested
	if opts.GasPrice != nil && opts.GasPrice.Sign() > 0 {
		tx.Type = TxTypeLegacy
		tx.GasPrice = opts.GasPrice
		return nil
	}

	// 3. Auto-detect EIP-1559 support from latest block header's baseFeePerGas
	if blk, err := client.GetBlockByNumber(ctx, nil, false); err == nil && blk != nil && blk.BaseFeePerGas != nil {
		tip, tipErr := client.MaxPriorityFeePerGas(ctx)
		if tipErr != nil || tip == nil {
			tip = Gwei(1)
		}
		maxFee := new(big.Int).Mul(blk.BaseFeePerGas, big.NewInt(2))
		maxFee.Add(maxFee, tip)
		tx.Type = TxTypeDynamicFee
		tx.MaxPriorityFeePerGas = tip
		tx.MaxFeePerGas = maxFee
		return nil
	}

	// 4. Fallback to Legacy EIP-155 gasPrice (e.g. GoChain)
	gp, err := client.GetGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("cannot get gas price: %w", err)
	}
	tx.Type = TxTypeLegacy
	tx.GasPrice = gp
	return nil
}

// WaitForReceipt polls for a transaction receipt until available or ctx is cancelled.
func WaitForReceipt(ctx context.Context, client Client, hash Hash) (*Receipt, error) {
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()
	for {
		receipt, err := client.GetTransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if !errors.Is(err, NotFoundErr) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func isValidURL(toTest string) bool {
	u, err := url.Parse(toTest)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func loadFileOrURL(loc string) ([]byte, error) {
	if isValidURL(loc) {
		return downloadFile(loc)
	}
	return os.ReadFile(loc)
}

func downloadFile(rawURL string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d fetching %s", resp.StatusCode, rawURL)
	}
	var dst bytes.Buffer
	if _, err := io.Copy(&dst, resp.Body); err != nil {
		return nil, err
	}
	return dst.Bytes(), nil
}
