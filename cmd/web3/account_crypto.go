package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gochain/web3"
	"github.com/urfave/cli/v2"
)

func myAddressCommand() *cli.Command {
	return &cli.Command{
		Name:  "myaddress",
		Usage: fmt.Sprintf("Print the EIP-55 address derived from %s", pkVarName),
		Flags: []cli.Flag{pkFlag(false)},
		Action: func(c *cli.Context) error {
			pk := strings.TrimSpace(c.String("private-key"))
			if pk == "" {
				return fmt.Errorf("%s not set", pkVarName)
			}
			acc, err := web3.ParsePrivateKey(pk)
			if err != nil {
				return err
			}
			if isJSONFormat() {
				fmt.Println(marshalJSON(map[string]string{"address": acc.PublicKey()}))
				return nil
			}
			fmt.Println(acc.PublicKey())
			return nil
		},
	}
}

func accountCommand() *cli.Command {
	return &cli.Command{
		Name:    "account",
		Aliases: []string{"a"},
		Usage:   "Create accounts and manage Web3 Secret Storage V3 keystores",
		Subcommands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Generate a new secp256k1 account (optionally encrypted into a keystore file)",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "keystore", Usage: "Optional file path to write encrypted V3 keystore JSON"},
					&cli.StringFlag{Name: "password", Usage: "Password for --keystore encryption"},
				},
				Action: func(c *cli.Context) error {
					acc, err := web3.CreateAccount()
					if err != nil {
						return err
					}
					ksPath := strings.TrimSpace(c.String("keystore"))
					if ksPath != "" {
						ksJSON, err := web3.EncryptKeystore(acc, c.String("password"), 1<<14, 1)
						if err != nil {
							return err
						}
						if dir := filepath.Dir(ksPath); dir != "" && dir != "." {
							if err := os.MkdirAll(dir, 0700); err != nil {
								return err
							}
						}
						if err := os.WriteFile(ksPath, ksJSON, 0600); err != nil {
							return err
						}
					}
					if isJSONFormat() {
						out := map[string]string{
							"private_key": acc.PrivateKey(),
							"address":     acc.PublicKey(),
						}
						if ksPath != "" {
							out["keystore"] = ksPath
						}
						fmt.Println(marshalJSON(out))
						return nil
					}
					fmt.Printf("Private key: %s\n", acc.PrivateKey())
					fmt.Printf("Public address: %s\n", acc.PublicKey())
					if ksPath != "" {
						fmt.Printf("Keystore written to: %s\n", ksPath)
					}
					return nil
				},
			},
			{
				Name:  "extract",
				Usage: "Decrypt and extract the private key from a V3 keystore file",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "keyfile", Required: true, Usage: "Path to keystore JSON file"},
					&cli.StringFlag{Name: "password", Usage: "Keystore password"},
				},
				Action: func(c *cli.Context) error {
					f := c.String("keyfile")
					kbytes, err := os.ReadFile(f)
					if err != nil {
						return fmt.Errorf("failed to read %q: %w", f, err)
					}
					acc, err := web3.DecryptKeystore(kbytes, c.String("password"))
					if err != nil {
						return err
					}
					if isJSONFormat() {
						fmt.Println(marshalJSON(map[string]string{
							"private_key": acc.PrivateKey(),
							"address":     acc.PublicKey(),
						}))
						return nil
					}
					fmt.Printf("Private key: %s\n", acc.PrivateKey())
					fmt.Printf("Public address: %s\n", acc.PublicKey())
					return nil
				},
			},
		},
	}
}

func signCommand() *cli.Command {
	return &cli.Command{
		Name:      "sign",
		Usage:     "Sign a message using EIP-191 personal_sign",
		ArgsUsage: "<message>",
		Flags: []cli.Flag{
			pkFlag(true),
			&cli.BoolFlag{Name: "hex", Usage: "Interpret message as 0x-prefixed hex bytes instead of UTF-8 text"},
		},
		Action: func(c *cli.Context) error {
			if c.Args().Len() == 0 {
				return errors.New("missing message argument")
			}
			msgStr := strings.Join(c.Args().Slice(), " ")
			var msgBytes []byte
			var err error
			if c.Bool("hex") {
				msgBytes, err = web3.DecodeHex(msgStr)
				if err != nil {
					return fmt.Errorf("invalid --hex message: %w", err)
				}
			} else {
				msgBytes = []byte(msgStr)
			}
			acc, err := web3.ParsePrivateKey(c.String("private-key"))
			if err != nil {
				return err
			}
			sig, err := web3.PersonalSign(msgBytes, acc.Key())
			if err != nil {
				return err
			}
			sigHex := web3.EncodeHex(sig)
			if isJSONFormat() {
				fmt.Println(marshalJSON(map[string]string{
					"address":      acc.PublicKey(),
					"message_hash": web3.PersonalMessageHash(msgBytes).Hex(),
					"signature":    sigHex,
				}))
				return nil
			}
			fmt.Println(sigHex)
			return nil
		},
	}
}

func verifySigCommand() *cli.Command {
	return &cli.Command{
		Name:  "verify-sig",
		Usage: "Recover signer (and optionally verify address) for an EIP-191 personal_sign signature",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "message", Aliases: []string{"m"}, Required: true, Usage: "Signed message"},
			&cli.StringFlag{Name: "signature", Aliases: []string{"s"}, Required: true, Usage: "65-byte 0x signature"},
			&cli.StringFlag{Name: "address", Aliases: []string{"a"}, Usage: "Expected signer address (optional)"},
			&cli.BoolFlag{Name: "hex", Usage: "Interpret --message as hex bytes"},
		},
		Action: func(c *cli.Context) error {
			var msgBytes []byte
			var err error
			if c.Bool("hex") {
				msgBytes, err = web3.DecodeHex(c.String("message"))
				if err != nil {
					return err
				}
			} else {
				msgBytes = []byte(c.String("message"))
			}
			sigBytes, err := web3.DecodeHex(c.String("signature"))
			if err != nil {
				return fmt.Errorf("invalid --signature hex: %w", err)
			}
			recovered, err := web3.PersonalRecover(msgBytes, sigBytes)
			if err != nil {
				return err
			}
			expected := strings.TrimSpace(c.String("address"))
			valid := true
			if expected != "" {
				if !web3.IsHexAddress(expected) {
					return fmt.Errorf("invalid --address %q", expected)
				}
				valid = recovered == web3.HexToAddress(expected)
			}
			if isJSONFormat() {
				out := map[string]interface{}{
					"signer": recovered.Hex(),
					"valid":  valid,
				}
				fmt.Println(marshalJSON(out))
				if !valid {
					os.Exit(1)
				}
				return nil
			}
			if expected != "" && !valid {
				return fmt.Errorf("signature mismatch: recovered %s, expected %s", recovered.Hex(), web3.HexToAddress(expected).Hex())
			}
			fmt.Println(recovered.Hex())
			return nil
		},
	}
}

func keccakCommand() *cli.Command {
	return &cli.Command{
		Name:      "keccak",
		Aliases:   []string{"hash"},
		Usage:     "Compute the Keccak-256 hash of UTF-8 text or --hex bytes",
		ArgsUsage: "<input>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "hex", Usage: "Interpret input as hex bytes"},
		},
		Action: func(c *cli.Context) error {
			if c.Args().Len() == 0 {
				return errors.New("missing input argument")
			}
			raw := strings.Join(c.Args().Slice(), " ")
			var b []byte
			var err error
			if c.Bool("hex") {
				b, err = web3.DecodeHex(raw)
				if err != nil {
					return err
				}
			} else {
				b = []byte(raw)
			}
			h := web3.Keccak256Hash(b)
			if isJSONFormat() {
				fmt.Println(marshalJSON(map[string]string{"hash": h.Hex()}))
				return nil
			}
			fmt.Println(h.Hex())
			return nil
		},
	}
}

func encodeCommand() *cli.Command {
	return &cli.Command{
		Name:      "encode",
		Aliases:   []string{"calldata"},
		Usage:     "ABI-encode function calldata offline (e.g. `web3 encode \"transfer(address,uint256)\" 0x... 100`)",
		ArgsUsage: "[signature] [args...]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "abi", Usage: "Optional ABI file/URL/bundled name instead of inline signature"},
			&cli.StringFlag{Name: "function", Usage: "Function name when --abi is used"},
		},
		Action: func(c *cli.Context) error {
			abiSpec := strings.TrimSpace(c.String("abi"))
			fnName := strings.TrimSpace(c.String("function"))
			args := c.Args().Slice()

			var packed []byte
			if abiSpec != "" {
				myabi, err := web3.GetABI(abiSpec)
				if err != nil {
					return err
				}
				params := make([]interface{}, len(args))
				for i, v := range args {
					params[i] = v
				}
				packed, err = myabi.Pack(fnName, params...)
				if err != nil {
					return err
				}
			} else {
				if len(args) == 0 {
					return errors.New("expected function signature argument, e.g. `web3 encode \"transfer(address,uint256)\" 0x... 100`")
				}
				m, err := web3.ParseHumanSignature(args[0])
				if err != nil {
					return err
				}
				params := make([]interface{}, len(args)-1)
				for i, v := range args[1:] {
					params[i] = v
				}
				body, err := m.Inputs.Pack(params...)
				if err != nil {
					return err
				}
				packed = append(append([]byte{}, m.ID...), body...)
			}
			hexData := web3.EncodeHex(packed)
			if isJSONFormat() {
				fmt.Println(marshalJSON(map[string]string{"data": hexData}))
				return nil
			}
			fmt.Println(hexData)
			return nil
		},
	}
}

func decodeCommand() *cli.Command {
	return &cli.Command{
		Name:      "decode",
		Usage:     "ABI-decode function calldata, return values (--output), or Solidity reverts (--revert)",
		ArgsUsage: "[signature] <0x-data>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "output", Usage: "Decode return values instead of input calldata"},
			&cli.BoolFlag{Name: "revert", Usage: "Decode Error(string), Panic(uint256), or custom ABI error"},
			&cli.StringFlag{Name: "abi", Usage: "Optional ABI file/URL/bundled name"},
		},
		Action: func(c *cli.Context) error {
			args := c.Args().Slice()
			if c.Bool("revert") {
				if len(args) < 1 {
					return errors.New("missing 0x revert data argument")
				}
				raw, err := web3.DecodeHex(args[len(args)-1])
				if err != nil {
					return err
				}
				var myabi *web3.ABI
				if abiSpec := strings.TrimSpace(c.String("abi")); abiSpec != "" {
					myabi, err = web3.GetABI(abiSpec)
					if err != nil {
						return err
					}
				}
				msg, ok := web3.DecodeRevert(raw, myabi)
				if !ok {
					return errors.New("empty revert data")
				}
				if isJSONFormat() {
					fmt.Println(marshalJSON(map[string]string{"revert": msg}))
					return nil
				}
				fmt.Println(msg)
				return nil
			}

			if len(args) < 2 {
				return errors.New("usage: web3 decode [--output] \"func(type1,type2):(retType)\" 0x...")
			}
			m, err := web3.ParseHumanSignature(args[0])
			if err != nil {
				return err
			}
			raw, err := web3.DecodeHex(args[1])
			if err != nil {
				return err
			}
			var vals []interface{}
			if c.Bool("output") {
				vals, err = m.Outputs.UnpackValues(raw)
			} else {
				if len(raw)%32 == 4 && len(raw) >= 4 {
					raw = raw[4:]
				}
				vals, err = m.Inputs.UnpackValues(raw)
			}
			if err != nil {
				return err
			}
			if isJSONFormat() {
				fmt.Println(marshalJSON(vals))
				return nil
			}
			for _, v := range vals {
				fmt.Println(v)
			}
			return nil
		},
	}
}

func unitsCommand() *cli.Command {
	return &cli.Command{
		Name:      "units",
		Usage:     "Convert an amount (e.g. `1.5eth`, `25gwei`, `1000000000wei`) across wei, gwei, and eth/go",
		ArgsUsage: "<amount>",
		Action: func(c *cli.Context) error {
			if c.Args().Len() == 0 {
				return errors.New("missing amount argument (e.g. `web3 units 1.5eth` or `web3 units 25gwei`)")
			}
			wei, err := web3.ParseAmount(strings.Join(c.Args().Slice(), ""))
			if err != nil {
				return err
			}
			gwei := web3.IntToDec(wei, 9).String()
			eth := web3.IntToDec(wei, 18).String()
			if isJSONFormat() {
				fmt.Println(marshalJSON(map[string]string{
					"wei":  wei.String(),
					"gwei": gwei,
					"eth":  eth,
				}))
				return nil
			}
			fmt.Println("Wei: ", wei.String())
			fmt.Println("Gwei:", gwei)
			fmt.Println("ETH: ", eth)
			return nil
		},
	}
}
