package assets

import (
	"fmt"
	"strings"

	"github.com/gochain/gochain/v3/common"
)

// Contract only upgradeable by owner.
const OwnerUpgradeableProxyBin = `0x608060405234801561001057600080fd5b50600073eeffeeffeeffeeffeeffeeffeeffeeffeeffeeff905061004281610060640100000000026401000000009004565b5061005b3361013a640100000000026401000000009004565b6101be565b60008061007a61017b640100000000026401000000009004565b91508273ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16141515156100b757600080fd5b60405180807f676f636861696e2e70726f78792e7461726765740000000000000000000000008152506014019050604051809103902090508281558273ffffffffffffffffffffffffffffffffffffffff167fbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b60405160405180910390a2505050565b600060405180807f676f636861696e2e70726f78792e6f776e6572000000000000000000000000008152506013019050604051809103902090508181555050565b60008060405180807f676f636861696e2e70726f78792e746172676574000000000000000000000000815250601401905060405180910390209050805491505090565b610630806101cd6000396000f300608060405260043610610078576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063046f7da2146100ff5780630900f010146101165780635c975abb146101595780638456cb59146101885780638da5cb5b1461019f578063d4b83992146101f6575b60008061008361024d565b91508115151561009257600080fd5b61009a6102a8565b9050600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141515156100d857600080fd5b60405136600082376000803683855af43d806000843e81600081146100fb578184f35b8184fd5b34801561010b57600080fd5b506101146102eb565b005b34801561012257600080fd5b50610157600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610336565b005b34801561016557600080fd5b5061016e61024d565b604051808215151515815260200191505060405180910390f35b34801561019457600080fd5b5061019d610383565b005b3480156101ab57600080fd5b506101b46103ce565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561020257600080fd5b5061020b6102a8565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b600080600060405180807f676f636861696e2e70726f78792e70617573656400000000000000000000000081525060140190506040518091039020915060006001029050815490506000600102816000191614159250505090565b60008060405180807f676f636861696e2e70726f78792e746172676574000000000000000000000000815250601401905060405180910390209050805491505090565b6102f36103ce565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561032c57600080fd5b610334610411565b565b61033e6103ce565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561037757600080fd5b610380816104a5565b50565b61038b6103ce565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156103c457600080fd5b6103cc610570565b565b60008060405180807f676f636861696e2e70726f78792e6f776e657200000000000000000000000000815250601301905060405180910390209050805491505090565b60008060405180807f676f636861696e2e70726f78792e70617573656400000000000000000000000081525060140190506040518091039020915060007f01000000000000000000000000000000000000000000000000000000000000000290508082557f62451d457bc659158be6e6247f56ec1df424a5c7597f71c20c2bc44e0965c8f960405160405180910390a15050565b6000806104b06102a8565b91508273ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16141515156104ed57600080fd5b60405180807f676f636861696e2e70726f78792e7461726765740000000000000000000000008152506014019050604051809103902090508281558273ffffffffffffffffffffffffffffffffffffffff167fbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b60405160405180910390a2505050565b60008060405180807f676f636861696e2e70726f78792e70617573656400000000000000000000000081525060140190506040518091039020915060017f01000000000000000000000000000000000000000000000000000000000000000290508082557f9e87fac88ff661f02d44f95383c817fece4bce600a3dab7a54406878b965e75260405160405180910390a150505600a165627a7a72305820fb83ed4a5dce35fddc4424d2b82ae073f393ec3c109002bcbc397ce64d1ed3f00029`

// OwnerUpgradeableProxyCode returns the code for an owner-upgradeable proxy contract.
func OwnerUpgradeableProxyCode(target common.Address) string {
	code := OwnerUpgradeableProxyBin

	// Replace placeholder addresses for target contract in constructor.
	code = strings.Replace(code, `eeffeeffeeffeeffeeffeeffeeffeeffeeffeeff`, strings.TrimPrefix(target.String(), "0x"), -1)

	// Strip auxdata.
	return TrimContractCodeAuxdata(code)
}

// TrimContractCodeAuxdata removes the auxdata produced at the end of a contract.
// This only used to strip system contract code so it only supports "bzzr0".
func TrimContractCodeAuxdata(code string) string {
	const auxdataLen = 43
	if len(code) < auxdataLen {
		return code
	}
	auxdata := code[len(code)-auxdataLen:]
	if !strings.HasPrefix(auxdata, fmt.Sprintf("a165%08x", "bzzr0")) {
		return code
	}
	return strings.TrimSuffix(code, auxdata)
}

const UpgradeableProxyABI = `[
  {
    "constant": false,
    "inputs": [],
    "name": "resume",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "addr",
        "type": "address"
      }
    ],
    "name": "upgrade",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "paused",
    "outputs": [
      {
        "name": "val",
        "type": "bool"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [],
    "name": "pause",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "target",
    "outputs": [
      {
        "name": "addr",
        "type": "address"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "constructor"
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
        "name": "target",
        "type": "address"
      }
    ],
    "name": "Upgraded",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [],
    "name": "Paused",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [],
    "name": "Resumed",
    "type": "event"
  }
]`

const OwnerUpgradeableProxyABI = `[
  {
    "constant": false,
    "inputs": [],
    "name": "resume",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "target",
        "type": "address"
      }
    ],
    "name": "upgrade",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "paused",
    "outputs": [
      {
        "name": "val",
        "type": "bool"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [],
    "name": "pause",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "owner",
    "outputs": [
      {
        "name": "addr",
        "type": "address"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "target",
    "outputs": [
      {
        "name": "addr",
        "type": "address"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "constructor"
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
        "name": "target",
        "type": "address"
      }
    ],
    "name": "Upgraded",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [],
    "name": "Paused",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [],
    "name": "Resumed",
    "type": "event"
  }
]`
