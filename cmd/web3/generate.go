package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/exec"
	"strings"

	"github.com/gochain/web3"
	"github.com/gochain/web3/assets"
	"github.com/urfave/cli/v2"
)

const OpenZeppelinVersion = "5.3.0"

func generateCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:    "generate",
		Aliases: []string{"g"},
		Usage:   "Generate OpenZeppelin v5 smart contracts or Go ABI bindings",
		Subcommands: []*cli.Command{
			{
				Name:  "contract",
				Usage: "Generate standard OpenZeppelin v5 contracts (erc20, erc721, erc1155)",
				Subcommands: []*cli.Command{
					{
						Name:  "erc20",
						Usage: "Generate an OpenZeppelin v5 ERC20 token contract",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "symbol", Aliases: []string{"s"}, Usage: "Token symbol", Required: true},
							&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Token name", Required: true},
							&cli.IntFlag{Name: "decimals", Aliases: []string{"d"}, Usage: "Token decimals", Value: 18},
							&cli.StringFlag{Name: "initial-supply", Usage: "Initial whole-token supply minted to deployer"},
							&cli.BoolFlag{Name: "mintable", Aliases: []string{"m"}, Usage: "Include owner-only mint()", Value: true},
							&cli.BoolFlag{Name: "burnable", Aliases: []string{"b"}, Usage: "Include ERC20Burnable", Value: true},
							&cli.BoolFlag{Name: "pausable", Aliases: []string{"p"}, Usage: "Include ERC20Pausable"},
						},
						Action: func(c *cli.Context) error {
							return generateStandardContract(ctx, "erc20", c)
						},
					},
					{
						Name:  "erc721",
						Usage: "Generate an OpenZeppelin v5 ERC721 NFT contract",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "symbol", Aliases: []string{"s"}, Usage: "Collection symbol", Required: true},
							&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Collection name", Required: true},
							&cli.StringFlag{Name: "base-uri", Usage: "Optional base URI for tokenURI metadata"},
							&cli.BoolFlag{Name: "mintable", Aliases: []string{"m"}, Usage: "Include owner-only safeMint()", Value: true},
							&cli.BoolFlag{Name: "burnable", Aliases: []string{"b"}, Usage: "Include ERC721Burnable", Value: true},
							&cli.BoolFlag{Name: "pausable", Aliases: []string{"p"}, Usage: "Include ERC721Pausable"},
						},
						Action: func(c *cli.Context) error {
							return generateStandardContract(ctx, "erc721", c)
						},
					},
					{
						Name:  "erc1155",
						Usage: "Generate an OpenZeppelin v5 ERC1155 multi-token contract",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "symbol", Aliases: []string{"s"}, Usage: "Contract name / symbol", Required: true},
							&cli.StringFlag{Name: "uri", Usage: "Metadata URI template (e.g. https://example.com/{id}.json)"},
							&cli.BoolFlag{Name: "mintable", Aliases: []string{"m"}, Usage: "Include owner-only mint() and mintBatch()", Value: true},
							&cli.BoolFlag{Name: "burnable", Aliases: []string{"b"}, Usage: "Include ERC1155Burnable", Value: true},
							&cli.BoolFlag{Name: "pausable", Aliases: []string{"p"}, Usage: "Include ERC1155Pausable"},
						},
						Action: func(c *cli.Context) error {
							return generateStandardContract(ctx, "erc1155", c)
						},
					},
				},
			},
			{
				Name:  "code",
				Usage: "Generate Go contract bindings from an ABI",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "abi", Aliases: []string{"a"}, Required: true, Usage: "ABI file or bundled name (erc20/erc721/erc1155/proxy)"},
					&cli.StringFlag{Name: "bin", Usage: "Optional .bin file to include Deploy<Type> helper"},
					&cli.StringFlag{Name: "lang", Aliases: []string{"l"}, Usage: "Destination language (go)", Value: "go"},
					&cli.StringFlag{Name: "pkg", Aliases: []string{"p"}, Usage: "Go package name", Value: "main"},
					&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Usage: "Generated Go struct name"},
					&cli.StringFlag{Name: "out", Aliases: []string{"o"}, Usage: "Output file", Value: "out.go"},
				},
				Action: func(c *cli.Context) error {
					return runGenerateCode(c)
				},
			},
		},
	}
}

func generateStandardContract(ctx context.Context, contractType string, c *cli.Context) error {
	symbol := strings.TrimSpace(c.String("symbol"))
	if symbol == "" {
		return errors.New("--symbol is required")
	}

	var src string
	var err error
	switch contractType {
	case "erc20":
		name := strings.TrimSpace(c.String("name"))
		if name == "" {
			return errors.New("--name is required")
		}
		decimals := c.Int("decimals")
		if decimals < 0 || decimals > 77 {
			return errors.New("--decimals must be between 0 and 77")
		}
		var initSupply *big.Int
		if supStr := strings.TrimSpace(c.String("initial-supply")); supStr != "" {
			var ok bool
			initSupply, ok = new(big.Int).SetString(supStr, 10)
			if !ok || initSupply.Sign() < 0 {
				return fmt.Errorf("invalid --initial-supply integer %q", supStr)
			}
		}
		src, err = assets.GenERC20(ctx, OpenZeppelinVersion, &assets.Erc20Params{
			Symbol:        symbol,
			TokenName:     name,
			Decimals:      decimals,
			InitialSupply: initSupply,
			Mintable:      c.Bool("mintable"),
			Burnable:      c.Bool("burnable"),
			Pausable:      c.Bool("pausable"),
		})

	case "erc721":
		name := strings.TrimSpace(c.String("name"))
		if name == "" {
			return errors.New("--name is required")
		}
		src, err = assets.GenERC721(OpenZeppelinVersion, &assets.Erc721Params{
			Symbol:    symbol,
			TokenName: name,
			BaseURI:   c.String("base-uri"),
			Mintable:  c.Bool("mintable"),
			Burnable:  c.Bool("burnable"),
			Pausable:  c.Bool("pausable"),
		})

	case "erc1155":
		src, err = assets.GenERC1155(OpenZeppelinVersion, &assets.Erc1155Params{
			Symbol:   symbol,
			URI:      c.String("uri"),
			Mintable: c.Bool("mintable"),
			Burnable: c.Bool("burnable"),
			Pausable: c.Bool("pausable"),
		})
	default:
		return fmt.Errorf("unsupported contract template %q", contractType)
	}
	if err != nil {
		return err
	}

	if err := getOpenZeppelinLib(ctx, OpenZeppelinVersion); err != nil {
		// Warn rather than fail if offline, since the .sol file itself is still valid and useful
		fmt.Fprintf(os.Stderr, "WARN: could not clone OpenZeppelin v%s into lib/oz (%v)\n", OpenZeppelinVersion, err)
	}

	outFile := symbol + ".sol"
	if err := os.WriteFile(outFile, []byte(src), 0644); err != nil {
		return fmt.Errorf("cannot write %q: %w", outFile, err)
	}
	if isJSONFormat() {
		fmt.Println(marshalJSON(map[string]string{"file": outFile, "openzeppelin_version": OpenZeppelinVersion}))
		return nil
	}
	fmt.Printf("Generated %s contract in %s (OpenZeppelin v%s)\n", strings.ToUpper(contractType), outFile, OpenZeppelinVersion)
	return nil
}

func getOpenZeppelinLib(ctx context.Context, version string) error {
	if _, err := os.Stat("lib/oz"); !os.IsNotExist(err) {
		return nil
	}
	if version == "" {
		version = OpenZeppelinVersion
	}
	if verbose {
		log.Printf("Cloning OpenZeppelin v%s...", version)
	}
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", "v"+version, "https://github.com/OpenZeppelin/openzeppelin-contracts", "lib/oz")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone openzeppelin-contracts v%s failed: %w", version, err)
	}
	_ = os.RemoveAll("lib/oz/.git")
	return nil
}

func runGenerateCode(c *cli.Context) error {
	lang := strings.ToLower(strings.TrimSpace(c.String("lang")))
	if lang != "go" {
		return fmt.Errorf("unsupported destination language %q (supported: go)", lang)
	}
	abiSpec := strings.TrimSpace(c.String("abi"))
	var abiJSON string
	switch strings.ToLower(abiSpec) {
	case "erc20":
		abiJSON = assets.ERC20ABI
	case "erc721":
		abiJSON = assets.ERC721ABI
	case "erc1155":
		abiJSON = assets.ERC1155ABI
	case "proxy", "eip1967":
		abiJSON = assets.EIP1967ProxyABI
	default:
		b, err := readLocalOrURL(abiSpec)
		if err != nil {
			return fmt.Errorf("failed to read ABI %q: %w", abiSpec, err)
		}
		abiJSON = string(b)
	}

	var binHex string
	if binFile := strings.TrimSpace(c.String("bin")); binFile != "" {
		b, err := readLocalOrURL(binFile)
		if err != nil {
			return fmt.Errorf("failed to read bin %q: %w", binFile, err)
		}
		binHex = strings.TrimSpace(string(b))
	}

	code, err := web3.GenerateGoBinding(c.String("pkg"), c.String("type"), abiJSON, binHex)
	if err != nil {
		return fmt.Errorf("failed to generate Go binding: %w", err)
	}
	outFile := c.String("out")
	if err := os.WriteFile(outFile, code, 0644); err != nil {
		return fmt.Errorf("failed to write %q: %w", outFile, err)
	}
	if isJSONFormat() {
		fmt.Println(marshalJSON(map[string]string{"file": outFile}))
		return nil
	}
	fmt.Println("Generated Go binding written to", outFile)
	return nil
}
