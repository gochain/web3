package assets

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed erc721.abi
var ERC721ABI string

type Erc721Params struct {
	Symbol       string
	TokenName    string
	BaseURI      string
	ContractName string
	Mintable     bool
	Burnable     bool
	Pausable     bool
}

const ERC721Template = `// SPDX-License-Identifier: MIT
// @openzeppelin v{{.OZVersion}}
pragma solidity ^0.8.20;

import {ERC721} from "./lib/oz/contracts/token/ERC721/ERC721.sol";
{{- if .Burnable}}
import {ERC721Burnable} from "./lib/oz/contracts/token/ERC721/extensions/ERC721Burnable.sol";
{{- end}}
{{- if .Pausable}}
import {ERC721Pausable} from "./lib/oz/contracts/token/ERC721/extensions/ERC721Pausable.sol";
{{- end}}
{{- if or .Mintable .Pausable}}
import {Ownable} from "./lib/oz/contracts/access/Ownable.sol";
{{- end}}

contract {{.ContractName}} is ERC721{{if .Burnable}}, ERC721Burnable{{end}}{{if .Pausable}}, ERC721Pausable{{end}}{{if or .Mintable .Pausable}}, Ownable{{end}} {
{{- if .Mintable}}
    uint256 private _nextTokenId;
{{- end}}

    constructor()
        ERC721({{printf "%q" .TokenName}}, {{printf "%q" .Symbol}})
        {{- if or .Mintable .Pausable}}
        Ownable(msg.sender)
        {{- end}}
    {}
{{- if .BaseURI}}

    function _baseURI() internal pure override returns (string memory) {
        return {{printf "%q" .BaseURI}};
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

    function safeMint(address to) public onlyOwner returns (uint256) {
        uint256 tokenId = _nextTokenId++;
        _safeMint(to, tokenId);
        return tokenId;
    }
{{- end}}
{{- if .Pausable}}

    function _update(address to, uint256 tokenId, address auth) internal override(ERC721, ERC721Pausable) returns (address) {
        return super._update(to, tokenId, auth);
    }
{{- end}}
}
`

func GenERC721(openZeppelinVersion string, params *Erc721Params) (string, error) {
	if params == nil {
		return "", fmt.Errorf("nil ERC721 params")
	}
	cname := params.ContractName
	if cname == "" {
		cname = EscapeName(params.Symbol)
	}
	data := struct {
		OZVersion    string
		Symbol       string
		TokenName    string
		BaseURI      string
		ContractName string
		Mintable     bool
		Burnable     bool
		Pausable     bool
	}{
		OZVersion:    openZeppelinVersion,
		Symbol:       params.Symbol,
		TokenName:    params.TokenName,
		BaseURI:      params.BaseURI,
		ContractName: cname,
		Mintable:     params.Mintable,
		Burnable:     params.Burnable,
		Pausable:     params.Pausable,
	}
	tmpl, err := template.New("erc721").Parse(ERC721Template)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
