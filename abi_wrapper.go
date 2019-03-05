package web3

import (
	"io"
	"os"
	"strings"

	abis "github.com/gochain-io/web3/abi"

	"github.com/gochain-io/gochain/v3/accounts/abi"
)

func ABIBuiltIn(contractFile string) (*abi.ABI, error) {
	if val, ok := bundledContracts[contractFile]; ok {
		return readAbi(strings.NewReader(val))
	}
	return nil, nil
}

func ABIOpenFile(contractFile string) (*abi.ABI, error) {
	jsonReader, err := os.Open(contractFile)
	if err != nil {
		return nil, err
	}
	return readAbi(jsonReader)
}

func readAbi(reader io.Reader) (*abi.ABI, error) {
	abi, err := abi.JSON(reader)
	if err != nil {
		return nil, err
	}
	return &abi, nil
}

var bundledContracts = map[string]string{
	"erc20":  abis.ERC20,
	"erc721": abis.ERC721}
