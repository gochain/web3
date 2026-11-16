package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/gochain/web3"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"
)

func transferCommand(ctx context.Context) *cli.Command {
	flags := []cli.Flag{
		pkFlag(false),
		&cli.BoolFlag{Name: "erc20", Usage: "Transfer ERC20 tokens instead of native coin"},
		addressFlag("ERC20 token contract address (when --erc20 is set)"),
		&cli.BoolFlag{Name: "wait", Usage: "Wait for transaction receipt"},
		&cli.BoolFlag{Name: "to-string", Usage: "Convert byte results to UTF-8 strings"},
		timeoutFlag(),
	}
	flags = append(flags, txGasFlags(0)...)

	return &cli.Command{
		Name:      "transfer",
		Aliases:   []string{"send"},
		Usage:     "Transfer native coins (GO/ETH) or ERC20 tokens (e.g. `web3 transfer 1.5 to 0xADDRESS`)",
		ArgsUsage: "<amount> [to] <address>",
		Flags:     flags,
		Action: func(c *cli.Context) error {
			args := c.Args().Slice()
			var amountS, toAddress string
			if len(args) == 2 {
				amountS, toAddress = args[0], args[1]
			} else if len(args) >= 3 && strings.EqualFold(args[1], "to") {
				amountS, toAddress = args[0], args[2]
			} else {
				return errors.New("invalid arguments: expected `transfer <amount> to <address>`")
			}

			amountD, err := decimal.NewFromString(amountS)
			if err != nil {
				return fmt.Errorf("invalid amount %q: %w", amountS, err)
			}
			if !web3.IsHexAddress(toAddress) {
				return fmt.Errorf("invalid recipient address %q", toAddress)
			}

			opts, err := parseTxOptions(c)
			if err != nil {
				return err
			}
			pk := strings.TrimSpace(c.String("private-key"))
			if pk == "" {
				return fmt.Errorf("private key required (pass --private-key or set %s)", pkVarName)
			}

			client, _, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()

			if c.Bool("erc20") {
				tokenAddr := strings.TrimSpace(c.String("address"))
				if tokenAddr == "" {
					return errors.New("must specify ERC20 contract --address when using --erc20")
				}
				erc20ABI, err := web3.GetABI("erc20")
				if err != nil {
					return err
				}
				decRes, err := web3.CallConstantFunction(ctx, client, *erc20ABI, tokenAddr, "decimals")
				if err != nil {
					return err
				}
				scaled := web3.DecToInt(amountD, int32(decRes[0].(uint8)))
				return executeContractCall(ctx, client, pk, tokenAddr, "erc20", "transfer", opts,
					c.Bool("wait"), c.Bool("to-string"), nil, c.Uint64("timeout"), toAddress, scaled)
			}

			opts.Value = web3.DecToInt(amountD, 18)
			tx, err := web3.SendWithOpts(ctx, client, pk, web3.HexToAddress(toAddress), opts)
			if err != nil {
				return fmt.Errorf("transfer failed: %w", err)
			}

			if !c.Bool("wait") {
				if isJSONFormat() {
					fmt.Println(marshalJSON(map[string]interface{}{"tx_hash": tx.Hash.Hex()}))
				} else {
					fmt.Println("Transaction hash:", tx.Hash.Hex())
				}
				return nil
			}

			waitCtx, cancel := context.WithTimeout(ctx, time.Duration(c.Uint64("timeout"))*time.Second)
			defer cancel()
			receipt, err := web3.WaitForReceipt(waitCtx, client, tx.Hash)
			if err != nil {
				return fmt.Errorf("waiting for receipt (%s): %w", tx.Hash.Hex(), err)
			}
			printReceiptDetails(receipt, nil)
			return nil
		},
	}
}

func increaseGasCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "increasegas",
		Usage: "Speed up a pending transaction by bumping its gas price / priority fee",
		Flags: []cli.Flag{
			pkFlag(true),
			&cli.StringFlag{
				Name:     "tx",
				Usage:    "Hash of the pending transaction",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "amount",
				Usage: "Amount in GWEI to add to gas price / max fees (decimals allowed, e.g. 1.5)",
				Value: "1",
			},
		},
		Action: func(c *cli.Context) error {
			bumpWei, err := web3.ParseGwei(c.String("amount"))
			if err != nil {
				return fmt.Errorf("invalid --amount %q: %w", c.String("amount"), err)
			}
			client, _, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()

			txOrig, err := client.GetTransactionByHash(ctx, web3.HexToHash(c.String("tx")))
			if err != nil {
				return fmt.Errorf("failed to fetch transaction: %w", err)
			}
			if txOrig.BlockNumber != nil {
				return errors.New("transaction is already mined (not pending)")
			}

			nonce := txOrig.Nonce
			opts := web3.TxOptions{
				Value:    txOrig.Value,
				GasLimit: txOrig.GasLimit,
				Nonce:    &nonce,
			}
			if txOrig.Type == web3.TxTypeDynamicFee && txOrig.MaxFeePerGas != nil {
				tip := txOrig.MaxPriorityFeePerGas
				if tip == nil {
					tip = big.NewInt(0)
				}
				opts.MaxPriorityFeePerGas = bumpAtLeast10Pct(tip, bumpWei)
				opts.MaxFeePerGas = bumpAtLeast10Pct(txOrig.MaxFeePerGas, bumpWei)
			} else {
				base := txOrig.GasPrice
				if base == nil {
					base = big.NewInt(0)
				}
				opts.GasPrice = bumpAtLeast10Pct(base, bumpWei)
			}

			newTx, err := broadcastReplacement(ctx, client, c.String("private-key"), txOrig.To, txOrig.Input, opts)
			if err != nil {
				return err
			}
			if isJSONFormat() {
				fmt.Println(marshalJSON(newTx))
				return nil
			}
			fmt.Printf("Replaced pending transaction (nonce %d). New tx hash: %s\n", newTx.Nonce, newTx.Hash.Hex())
			return nil
		},
	}
}

func cancelCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "cancel",
		Usage: "Cancel a pending transaction by sending a 0-value self-transfer at the same nonce with higher gas",
		Flags: []cli.Flag{
			pkFlag(true),
			&cli.StringFlag{
				Name:  "tx",
				Usage: "Pending transaction hash to cancel (auto-reads nonce & fees)",
			},
			&cli.Uint64Flag{
				Name:  "nonce",
				Usage: "Explicit nonce to cancel (when --tx is not provided)",
			},
			&cli.StringFlag{
				Name:  "amount",
				Usage: "GWEI bump to add above pending transaction's fee",
				Value: "2",
			},
		},
		Action: func(c *cli.Context) error {
			pk := c.String("private-key")
			acc, err := web3.ParsePrivateKey(pk)
			if err != nil {
				return err
			}
			selfAddr := acc.Address()

			bumpWei, err := web3.ParseGwei(c.String("amount"))
			if err != nil {
				return fmt.Errorf("invalid --amount %q: %w", c.String("amount"), err)
			}

			client, _, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()

			var opts web3.TxOptions
			opts.Value = big.NewInt(0)
			opts.GasLimit = 21000

			if txHash := strings.TrimSpace(c.String("tx")); txHash != "" {
				txOrig, err := client.GetTransactionByHash(ctx, web3.HexToHash(txHash))
				if err != nil {
					return fmt.Errorf("failed to fetch transaction: %w", err)
				}
				if txOrig.BlockNumber != nil {
					return errors.New("transaction is already mined and cannot be cancelled")
				}
				nonce := txOrig.Nonce
				opts.Nonce = &nonce
				if txOrig.Type == web3.TxTypeDynamicFee && txOrig.MaxFeePerGas != nil {
					tip := txOrig.MaxPriorityFeePerGas
					if tip == nil {
						tip = big.NewInt(0)
					}
					opts.MaxPriorityFeePerGas = bumpAtLeast10Pct(tip, bumpWei)
					opts.MaxFeePerGas = bumpAtLeast10Pct(txOrig.MaxFeePerGas, bumpWei)
				} else if txOrig.GasPrice != nil {
					opts.GasPrice = bumpAtLeast10Pct(txOrig.GasPrice, bumpWei)
				}
			} else if c.IsSet("nonce") {
				nonce := c.Uint64("nonce")
				opts.Nonce = &nonce
			} else {
				return errors.New("must specify either --tx <pending-hash> or --nonce <n>")
			}

			newTx, err := broadcastReplacement(ctx, client, pk, &selfAddr, nil, opts)
			if err != nil {
				return err
			}
			if isJSONFormat() {
				fmt.Println(marshalJSON(newTx))
				return nil
			}
			fmt.Printf("Submitted cancellation transaction (nonce %d). Tx hash: %s\n", newTx.Nonce, newTx.Hash.Hex())
			return nil
		},
	}
}

func replaceCommand(ctx context.Context) *cli.Command {
	flags := []cli.Flag{
		pkFlag(true),
		&cli.Uint64Flag{
			Name:     "nonce",
			Usage:    "Nonce of the pending transaction to replace",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "to",
			Usage:    "Destination address",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "amountd",
			Usage: "Amount of native coin (GO/ETH) in decimal format",
		},
		&cli.StringFlag{
			Name:  "data",
			Usage: "Hex input calldata (optional)",
		},
	}
	flags = append(flags, txGasFlags(21000)...)

	return &cli.Command{
		Name:  "replace",
		Usage: "Replace a pending transaction at a specific nonce",
		Flags: flags,
		Action: func(c *cli.Context) error {
			toS := strings.TrimSpace(c.String("to"))
			if !web3.IsHexAddress(toS) {
				return fmt.Errorf("invalid --to address %q", toS)
			}
			toAddr := web3.HexToAddress(toS)

			opts, err := parseTxOptions(c)
			if err != nil {
				return err
			}
			nonce := c.Uint64("nonce")
			opts.Nonce = &nonce

			if ad := strings.TrimSpace(c.String("amountd")); ad != "" {
				amountD, err := decimal.NewFromString(ad)
				if err != nil {
					return fmt.Errorf("invalid --amountd %q: %w", ad, err)
				}
				opts.Value = web3.DecToInt(amountD, 18)
			}

			var dataB []byte
			if d := strings.TrimSpace(c.String("data")); d != "" {
				dataB, err = hex.DecodeString(strings.TrimPrefix(strings.TrimPrefix(d, "0x"), "0X"))
				if err != nil {
					return fmt.Errorf("invalid --data hex: %w", err)
				}
			}

			client, _, err := dialNetwork(c)
			if err != nil {
				return err
			}
			defer client.Close()

			newTx, err := broadcastReplacement(ctx, client, c.String("private-key"), &toAddr, dataB, opts)
			if err != nil {
				return err
			}
			if isJSONFormat() {
				fmt.Println(marshalJSON(newTx))
				return nil
			}
			fmt.Printf("Replaced transaction (nonce %d). New transaction: %s\n", newTx.Nonce, newTx.Hash.Hex())
			return nil
		},
	}
}

func bumpAtLeast10Pct(orig, add *big.Int) *big.Int {
	plusAdd := new(big.Int).Add(orig, add)
	// Mempools generally require >= 10% bump; compute orig * 112 / 100 as floor.
	minPct := new(big.Int).Mul(orig, big.NewInt(112))
	minPct.Div(minPct, big.NewInt(100))
	if plusAdd.Cmp(minPct) >= 0 {
		return plusAdd
	}
	return minPct
}

func broadcastReplacement(ctx context.Context, client web3.Client, pk string, to *web3.Address, data []byte, opts web3.TxOptions) (*web3.Transaction, error) {
	if to == nil {
		return web3.DeployContractWithOpts(ctx, client, pk, web3.EncodeHex(data), "", opts)
	}
	return web3.CallFunctionWithDataOpts(ctx, client, pk, to.Hex(), opts, data)
}
