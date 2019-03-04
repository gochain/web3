package web3

import (
	"io"
	"log"
	"os"

	rice "github.com/GeertJohan/go.rice"
	"github.com/gochain-io/gochain/v3/accounts/abi"
)

func getBox() *rice.Box {
	abiBox, err := rice.FindBox("abi")
	if err != nil {
		log.Fatal("Cannot open the embedded storage", err)
	}
	return abiBox
}

func GetAbi(contractFile string) *abi.ABI {
	var reader io.Reader
	if _, err := os.Stat(contractFile); os.IsNotExist(err) {
		reader, err = getBox().Open(contractFile)
		if err != nil {
			log.Fatalf("Cannot find the abi file: %v", err)
		}
	} else {
		reader, err = os.Open(contractFile)
		if err != nil {
			log.Fatalf("Cannot read the abi file: %v", err)
		}
	}

	abi, err := abi.JSON(reader)
	if err != nil {
		log.Fatalf("Cannot initialize ABI: %v", err)
	}
	return &abi
}
