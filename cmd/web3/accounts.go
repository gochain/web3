package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/gochain-io/web3"
	"github.com/urfave/cli"
)

func getAddressHash(addrHash, privateKey string) string {
	if addrHash == "" {
		if privateKey == "" {
			fatalExit(errors.New("Missing address. Command requires address or private key argument."))
		}
		acct, err := web3.ParsePrivateKey(privateKey)
		if err != nil {
			fatalExit(err)
		}
		addrHash = acct.PublicKey()
	}
	return addrHash
}
func GetAddressDetails(ctx context.Context, network web3.Network, addrHash, privateKey string) {
	addrHash = getAddressHash(addrHash, privateKey)
	client, err := web3.NewClient(network.URL)
	if err != nil {
		fatalExit(fmt.Errorf("Failed to connect to %q: %v", network.URL, err))
	}
	defer client.Close()
	bal, err := client.GetBalance(ctx, addrHash, nil)
	if err != nil {
		fatalExit(fmt.Errorf("Cannot get address balance from the network: %v", err))
	}
	code, err := client.GetCode(ctx, addrHash, nil)
	if err != nil {
		fatalExit(fmt.Errorf("Cannot get address code from the network: %v", err))
	}
	if verbose {
		log.Println("Address details:")
	}

	switch format {
	case "json":
		data := struct {
			Balance *big.Int `json:"balance"`
			Code    *string  `json:"code"`
		}{Balance: bal}
		if len(code) > 0 {
			sc := string(code)
			data.Code = &sc
		}
		fmt.Println(marshalJSON(&data))
		return
	}

	fmt.Println("Balance:", web3.WeiAsBase(bal), network.Unit)
	if len(code) > 0 {
		fmt.Println("Code:", string(code))
	}
}

func GetBalance(ctx context.Context, c *cli.Context, network web3.Network) {
	addrHash := getAddressHash(c.String("address"), c.String("private-key"))
	token := strings.ToUpper(c.String("token")) // preparing for more
	if token != "GO" {
		fatalExit(fmt.Errorf("Token %v not supported", token))
	}
	client, err := web3.NewClient(network.URL)
	if err != nil {
		fatalExit(fmt.Errorf("Failed to connect to %q: %v", network.URL, err))
	}
	defer client.Close()
	bal, err := client.GetBalance(ctx, addrHash, nil)
	if err != nil {
		fatalExit(fmt.Errorf("Cannot get address balance from the network: %v", err))
	}
	fmt.Printf("%v", web3.WeiAsBase(bal))
}
