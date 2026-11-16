package assets

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"strings"
	"text/template"
	"unicode"
)

//go:embed erc20.abi
var ERC20ABI string

type Erc20Params struct {
	Symbol        string
	TokenName     string
	ContractName  string
	Decimals      int
	InitialSupply *big.Int
	Mintable      bool
	Burnable      bool
	Pausable      bool
}

func EscapeName(symbol string) string {
	var b strings.Builder
	for _, r := range symbol {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			if b.Len() == 0 && unicode.IsDigit(r) {
				b.WriteByte('_')
			}
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "Token"
	}
	return b.String()
}

const erc20Template = `// SPDX-License-Identifier: MIT
// @openzeppelin v{{.OZVersion}}
pragma solidity ^0.8.20;

import {ERC20} from "./lib/oz/contracts/token/ERC20/ERC20.sol";
{{- if .Burnable}}
import {ERC20Burnable} from "./lib/oz/contracts/token/ERC20/extensions/ERC20Burnable.sol";
{{- end}}
{{- if .Pausable}}
import {ERC20Pausable} from "./lib/oz/contracts/token/ERC20/extensions/ERC20Pausable.sol";
{{- end}}
{{- if or .Mintable .Pausable}}
import {Ownable} from "./lib/oz/contracts/access/Ownable.sol";
{{- end}}

contract {{.ContractName}} is ERC20{{if .Burnable}}, ERC20Burnable{{end}}{{if .Pausable}}, ERC20Pausable{{end}}{{if or .Mintable .Pausable}}, Ownable{{end}} {
    constructor()
        ERC20({{printf "%q" .TokenName}}, {{printf "%q" .Symbol}})
        {{- if or .Mintable .Pausable}}
        Ownable(msg.sender)
        {{- end}}
    {
        {{- if .HasInitialSupply}}
        _mint(msg.sender, {{.InitialSupply}} * 10 ** decimals());
        {{- end}}
    }
{{- if ne .Decimals 18}}

    function decimals() public pure override returns (uint8) {
        return {{.Decimals}};
    }
{{- end}}
{{- if .Pausable}}

    function pause() public onlyOwner {
        _pause();
    }

    function unpause() public onlyOwner {
        _unpause();
    }
{{- end}}
{{- if .Mintable}}

    function mint(address to, uint256 amount) public onlyOwner {
        _mint(to, amount);
    }
{{- end}}
{{- if .Pausable}}

    function _update(address from, address to, uint256 value) internal override(ERC20, ERC20Pausable) {
        super._update(from, to, value);
    }
{{- end}}
}
`

func GenERC20(_ context.Context, openZeppelinVersion string, params *Erc20Params) (string, error) {
	if params == nil {
		return "", fmt.Errorf("nil ERC20 params")
	}
	decimals := params.Decimals
	if decimals <= 0 {
		decimals = 18
	}
	cname := params.ContractName
	if cname == "" {
		cname = EscapeName(params.Symbol)
	}
	data := struct {
		OZVersion        string
		Symbol           string
		TokenName        string
		ContractName     string
		Decimals         int
		InitialSupply    string
		HasInitialSupply bool
		Mintable         bool
		Burnable         bool
		Pausable         bool
	}{
		OZVersion:        openZeppelinVersion,
		Symbol:           params.Symbol,
		TokenName:        params.TokenName,
		ContractName:     cname,
		Decimals:         decimals,
		HasInitialSupply: params.InitialSupply != nil && params.InitialSupply.Sign() > 0,
		Mintable:         params.Mintable,
		Burnable:         params.Burnable,
		Pausable:         params.Pausable,
	}
	if data.HasInitialSupply {
		data.InitialSupply = params.InitialSupply.String()
	}

	tmpl, err := template.New("erc20").Parse(erc20Template)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
