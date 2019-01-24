package web3

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/gochain-io/gochain/v3/common"
	"github.com/gochain-io/gochain/v3/common/hexutil"
	"github.com/gochain-io/gochain/v3/core/types"
	"github.com/gochain-io/gochain/v3/crypto"
)

var NotFoundErr = errors.New("not found")

//TODO instead return a rich Network struct w/ netId/chainId/baseUnit
func NetworkURL(network string) string {
	switch network {
	case "testnet":
		return "https://testnet-rpc.gochain.io"
	case "mainnet", "":
		return "https://rpc.gochain.io"
	case "localhost":
		return "http://localhost:8545"
	case "ethereum":
		return "https://main-rpc.linkpool.io"
	case "ropsten":
		return "https://ropsten-rpc.linkpool.io"
	default:
		return ""
	}
}

var (
	weiPerGO   = big.NewInt(1e18)
	weiPerGwei = big.NewInt(1e9)
)

// WeiAsBase converts w wei in to the base unit, and formats it as a decimal fraction with full precision (up to 18 decimals).
func WeiAsBase(w *big.Int) string {
	return new(big.Rat).SetFrac(w, weiPerGO).FloatString(18)
}

// WeiAsGwei converts w wei in to gwei, and formats it as a decimal fraction with full precision (up to 9 decimals).
func WeiAsGwei(w *big.Int) string {
	return new(big.Rat).SetFrac(w, weiPerGwei).FloatString(9)
}

func DeployContract(ctx context.Context, client Client, privateKeyHex string, contractData string) (*Transaction, error) {
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	gasPrice, err := client.GetGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get gas price: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.GetPendingTransactionCount(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("cannot get nonce: %v", err)
	}
	decodedContractData, err := hexutil.Decode(contractData)
	if err != nil {
		return nil, fmt.Errorf("cannot decode contract data: %v", err)
	}
	//TODO try to use web3.Transaction only; can't sign currently
	tx := types.NewContractCreation(nonce, big.NewInt(0), 2000000, gasPrice, decodedContractData)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, privateKey)
	if err != nil {
		return nil, fmt.Errorf("cannot sign transaction: %v", err)
	}

	rtx := convertTx(signedTx, fromAddress)
	err = client.SendTransaction(ctx, rtx)
	if err != nil {
		return nil, fmt.Errorf("cannot send transaction: %v", err)
	}

	return rtx, nil
}

func convertTx(tx *types.Transaction, from common.Address) *Transaction {
	rtx := &Transaction{}
	rtx.Nonce = tx.Nonce()
	rtx.GasPrice = tx.GasPrice()
	rtx.GasLimit = tx.Gas()
	rtx.To = tx.To()
	rtx.Value = tx.Value()
	rtx.Input = tx.Data()
	rtx.Hash = tx.Hash()
	rtx.From = from
	v, r, s := tx.RawSignatureValues()
	rtx.V = v
	rtx.R.SetBytes(r.Bytes())
	rtx.S.SetBytes(s.Bytes())
	return rtx
}

func WaitForReceipt(ctx context.Context, client Client, tx *Transaction) (*Receipt, error) {
	for i := 0; ; i++ {
		receipt, err := client.GetTransactionReceipt(ctx, tx.Hash)
		if err == nil {
			return receipt, nil
		}
		if i >= (5) {
			return nil, fmt.Errorf("cannot get the receipt: %v", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
