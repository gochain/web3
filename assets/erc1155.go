package assets

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed erc1155.abi
var ERC1155ABI string

type Erc1155Params struct {
	Symbol       string
	URI          string
	ContractName string
	Mintable     bool
	Burnable     bool
	Pausable     bool
}

const erc1155Template = `// SPDX-License-Identifier: MIT
// @openzeppelin v{{.OZVersion}}
pragma solidity ^0.8.20;

import {ERC1155} from "./lib/oz/contracts/token/ERC1155/ERC1155.sol";
{{- if .Burnable}}
import {ERC1155Burnable} from "./lib/oz/contracts/token/ERC1155/extensions/ERC1155Burnable.sol";
{{- end}}
{{- if .Pausable}}
import {ERC1155Pausable} from "./lib/oz/contracts/token/ERC1155/extensions/ERC1155Pausable.sol";
{{- end}}
import {Ownable} from "./lib/oz/contracts/access/Ownable.sol";

contract {{.ContractName}} is ERC1155{{if .Burnable}}, ERC1155Burnable{{end}}{{if .Pausable}}, ERC1155Pausable{{end}}, Ownable {
    constructor() ERC1155({{printf "%q" .URI}}) Ownable(msg.sender) {}

    function setURI(string memory newuri) public onlyOwner {
        _setURI(newuri);
    }
{{- if .Pausable}}

    function pause() public onlyOwner {
        _pause();
    }

    function unpause() public onlyOwner {
        _unpause();
    }
{{- end}}
{{- if .Mintable}}

    function mint(address account, uint256 id, uint256 amount, bytes memory data) public onlyOwner {
        _mint(account, id, amount, data);
    }

    function mintBatch(address to, uint256[] memory ids, uint256[] memory amounts, bytes memory data) public onlyOwner {
        _mintBatch(to, ids, amounts, data);
    }
{{- end}}
{{- if .Pausable}}

    function _update(address from, address to, uint256[] memory ids, uint256[] memory values) internal override(ERC1155, ERC1155Pausable) {
        super._update(from, to, ids, values);
    }
{{- end}}
}
`

func GenERC1155(openZeppelinVersion string, params *Erc1155Params) (string, error) {
	if params == nil {
		return "", fmt.Errorf("nil ERC1155 params")
	}
	cname := params.ContractName
	if cname == "" {
		cname = EscapeName(params.Symbol)
	}
	data := struct {
		OZVersion    string
		ContractName string
		URI          string
		Mintable     bool
		Burnable     bool
		Pausable     bool
	}{
		OZVersion:    openZeppelinVersion,
		ContractName: cname,
		URI:          params.URI,
		Mintable:     params.Mintable,
		Burnable:     params.Burnable,
		Pausable:     params.Pausable,
	}
	tmpl, err := template.New("erc1155").Parse(erc1155Template)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
