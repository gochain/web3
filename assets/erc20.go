package assets

import (
	"context"
	"math/big"
	"strconv"
	"strings"
)

type Erc20Params struct {
	Symbol    string
	TokenName string
	Cap       *big.Int
	Decimals  int
	Pausable  bool
	Mintable  bool
	Burnable  bool
}

func GenERC20(ctx context.Context, params *Erc20Params) (string, error) {
	var part1, part2, part3 strings.Builder
	part1.WriteString("pragma solidity ^0.5.2;\n\nimport \"./lib/oz/contracts/token/ERC20/ERC20Detailed.sol\";\n")
	part2.WriteString("\ncontract ")
	part2.WriteString(params.Symbol)
	part2.WriteString(" is")
	{
		part3.WriteString("    constructor() ERC20Detailed(\"")
		part3.WriteString(params.TokenName)
		part3.WriteString("\", \"")
		part3.WriteString(params.Symbol)
		part3.WriteString("\", ")
		part3.WriteString(strconv.Itoa(params.Decimals))
		part3.WriteString(")")

	}
	if params.Pausable {
		part1.WriteString("import \"./lib/oz/contracts/token/ERC20/ERC20Pausable.sol\";\n")
		part2.WriteString(" ERC20Pausable,")
	}
	if params.Burnable {
		part1.WriteString("import \"./lib/oz/contracts/token/ERC20/ERC20Burnable.sol\";\n")
		part2.WriteString(" ERC20Burnable,")
	}
	if params.Mintable {
		part1.WriteString("import \"./lib/oz/contracts/token/ERC20/ERC20Mintable.sol\";\n")
		part2.WriteString(" ERC20Mintable,")
	}
	if params.Cap != nil {
		part1.WriteString("import \"./lib/oz/contracts/token/ERC20/ERC20Capped.sol\";\n")
		part2.WriteString(" ERC20Capped,")
		part3.WriteString(" ERC20Capped(")
		part3.WriteString(params.Cap.String())
		part3.WriteString(")")
	}
	part2.WriteString(" ERC20Detailed {\n\n")

	part3.WriteString(" public {}\n\n}\n")

	return part1.String() + part2.String() + part3.String(), nil
}

const ERC20ABI = `[
	{
		"constant": true,
		"inputs": [],
		"name": "name",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "_spender",
				"type": "address"
			},
			{
				"name": "_value",
				"type": "uint256"
			}
		],
		"name": "approve",
		"outputs": [
			{
				"name": "",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "totalSupply",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "_from",
				"type": "address"
			},
			{
				"name": "_to",
				"type": "address"
			},
			{
				"name": "_value",
				"type": "uint256"
			}
		],
		"name": "transferFrom",
		"outputs": [
			{
				"name": "",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "decimals",
		"outputs": [
			{
				"name": "",
				"type": "uint8"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "_owner",
				"type": "address"
			}
		],
		"name": "balanceOf",
		"outputs": [
			{
				"name": "balance",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "symbol",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "_to",
				"type": "address"
			},
			{
				"name": "_value",
				"type": "uint256"
			}
		],
		"name": "transfer",
		"outputs": [
			{
				"name": "",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "_owner",
				"type": "address"
			},
			{
				"name": "_spender",
				"type": "address"
			}
		],
		"name": "allowance",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"payable": true,
		"stateMutability": "payable",
		"type": "fallback"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "owner",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "spender",
				"type": "address"
			},
			{
				"indexed": false,
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "Approval",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "from",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "to",
				"type": "address"
			},
			{
				"indexed": false,
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "Transfer",
		"type": "event"
	}
]
`
