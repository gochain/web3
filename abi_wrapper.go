package web3

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gochain/gochain/v3/accounts/abi"
	"github.com/gochain/web3/assets"
)

// GetABI accepts either built in contracts (erc20, erc721), a file location or a URL
func GetABI(contractLocation string) (*abi.ABI, error) {
	abi, err := ABIBuiltIn(contractLocation)
	if err != nil {
		return nil, fmt.Errorf("Cannot get ABI from the bundled storage: %v", err)
	}
	if abi != nil {
		return abi, nil
	}
	abi, err = ABIOpenFile(contractLocation)
	if err == nil {
		return abi, nil
	}
	// else most likely just not found, log it?

	abi, err = ABIOpenURL(contractLocation)
	if err == nil {
		return abi, nil
	}
	return nil, err
}

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

func ABIOpenURL(contractURL string) (*abi.ABI, error) {
	resp, err := http.Get(contractURL)
	if err != nil {
		return nil, fmt.Errorf("error getting ABI: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error getting ABI %v: %v", resp.StatusCode, string(bodyBytes))
	}
	return readAbi(resp.Body)
}

func readAbi(reader io.Reader) (*abi.ABI, error) {
	abi, err := abi.JSON(reader)
	if err != nil {
		return nil, err
	}
	return &abi, nil
}

var bundledContracts = map[string]string{
	"erc20":  assets.ERC20ABI,
	"erc721": assets.ERC721ABI}
