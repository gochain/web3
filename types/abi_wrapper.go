package web3_types

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/rs/zerolog/log"
	"github.com/zeus-fyi/gochain/web3/assets"
)

// GetABI accepts either built in contracts (erc20, erc721), a file location or a URL
func GetABI(abiFile string) (*abi.ABI, error) {
	abiIn, err := ABIBuiltIn(abiFile)
	if err != nil {
		log.Err(err).Msg("GetABI: ABIBuiltIn")
		return nil, fmt.Errorf("cannot get ABI from the bundled storage: %v", err)
	}
	if abiIn != nil {
		return abiIn, nil
	}
	abiIn, err = ABIOpenFile(abiFile)
	if err == nil {
		log.Err(err).Msg("GetABI: ABIOpenFile")
		return abiIn, nil
	}
	// else most likely just not found, log it?
	abiIn, err = ABIOpenURL(abiFile)
	if err == nil {
		log.Err(err).Msg("GetABI: ABIOpenURL")
		return abiIn, nil
	}
	return nil, err
}

func ABIBuiltIn(abiFile string) (*abi.ABI, error) {
	if val, ok := bundledContracts[abiFile]; ok {
		return readAbi(strings.NewReader(val))
	}
	return nil, nil
}

func ABIOpenFile(abiFile string) (*abi.ABI, error) {
	jsonReader, err := os.Open(abiFile)
	if err != nil {
		log.Err(err).Msg("GetABI: ABIOpenFile")
		return nil, err
	}
	return readAbi(jsonReader)
}

func ABIOpenURL(abiFile string) (*abi.ABI, error) {
	resp, err := http.Get(abiFile)
	if err != nil {
		log.Err(err).Msg("ABIOpenURL: http.Get")
		return nil, fmt.Errorf("error getting ABI: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, berr := io.ReadAll(resp.Body)
		if berr != nil {
			log.Err(berr).Msg("ABIOpenURL: ReadAll")
			return nil, berr
		}
		return nil, fmt.Errorf("error getting ABI %v: %v", resp.StatusCode, string(bodyBytes))
	}
	return readAbi(resp.Body)
}

func readAbi(reader io.Reader) (*abi.ABI, error) {
	abiIn, err := abi.JSON(reader)
	if err != nil {
		log.Err(err).Msg("readAbi:  abi.JSON")
		return nil, err
	}
	return &abiIn, nil
}

var bundledContracts = map[string]string{
	"erc20":  assets.ERC20ABI,
	"erc721": assets.ERC721ABI}
