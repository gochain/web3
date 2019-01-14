package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/gochain-io/gochain/common/hexutil"

	"github.com/gochain-io/gochain/common"
	"github.com/gochain-io/gochain/consensus/clique"
	"github.com/gochain-io/gochain/core/types"
	"github.com/gochain-io/gochain/crypto"
	"github.com/gochain-io/gochain/goclient"
)

type RPCClient struct {
	url    string
	client *goclient.Client
}

func GetClient(rpcURL string) *RPCClient {
	client, err := goclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("Cannot connect to the network %q: %v", rpcURL, err)
	}
	rpc := &RPCClient{
		url:    rpcURL,
		client: client,
	}
	return rpc
}

func (rpc *RPCClient) GetBalance(address string, blockNumber *big.Int) (*big.Int, error) {
	return rpc.client.BalanceAt(context.Background(), common.HexToAddress(address), blockNumber)
}

func (rpc *RPCClient) GetCode(address string, blockNumber *big.Int) ([]byte, error) {
	return rpc.client.CodeAt(context.Background(), common.HexToAddress(address), blockNumber)
}

func (rpc *RPCClient) GetBlockByNumber(number *big.Int) (*types.Block, error) {
	return rpc.client.BlockByNumber(context.Background(), number)
}

func (rpc *RPCClient) GetTransactionByHash(hash string) (*types.Transaction, bool, error) {
	return rpc.client.TransactionByHash(context.Background(), common.HexToHash(hash))
}

func (rpc *RPCClient) GetSnapshot() (*clique.Snapshot, error) {
	return rpc.client.SnapshotAt(context.Background(), nil)
}

func (rpc *RPCClient) DeployContract(privateKeyHex string, contractData string) (*types.Transaction, error) {
	if privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Wrong private key:%s", err))
	}

	gasPrice, err := rpc.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot get gas price:%s", err))
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := rpc.client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot get nonce:%s", err))
	}
	decodedContractData, err := hexutil.Decode(contractData)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot decode contract data:%s", err))
	}
	tx := types.NewContractCreation(nonce, big.NewInt(0), 2000000, gasPrice, decodedContractData)
	signedTx, _ := types.SignTx(tx, types.HomesteadSigner{}, privateKey)

	err = rpc.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot send transaction:%s", err))
	}

	return signedTx, nil
}
func (rpc *RPCClient) WaitForReceipt(tx *types.Transaction) (*types.Receipt, error) {
	for i := 0; ; i++ {
		receipt, err := rpc.client.TransactionReceipt(context.Background(), tx.Hash())
		if err == nil {
			return receipt, nil
		}
		if i >= (5) {
			return nil, errors.New(fmt.Sprintf("Cannot get the receipt:%s", err))
		}
		time.Sleep(2 * time.Second)
	}
}
