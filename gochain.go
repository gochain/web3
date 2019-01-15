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

func (rpc *RPCClient) GetBalance(ctx context.Context, address string, blockNumber *big.Int) (*big.Int, error) {
	return rpc.client.BalanceAt(ctx, common.HexToAddress(address), blockNumber)
}

func (rpc *RPCClient) GetCode(ctx context.Context, address string, blockNumber *big.Int) ([]byte, error) {
	return rpc.client.CodeAt(ctx, common.HexToAddress(address), blockNumber)
}

func (rpc *RPCClient) GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return rpc.client.BlockByNumber(ctx, number)
}

func (rpc *RPCClient) GetTransactionByHash(ctx context.Context, hash string) (*types.Transaction, bool, error) {
	return rpc.client.TransactionByHash(ctx, common.HexToHash(hash))
}

func (rpc *RPCClient) GetSnapshot(ctx context.Context) (*clique.Snapshot, error) {
	return rpc.client.SnapshotAt(ctx, nil)
}

func (rpc *RPCClient) DeployContract(ctx context.Context, privateKeyHex string, contractData string) (*types.Transaction, error) {
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Wrong private key:%s", err))
	}

	gasPrice, err := rpc.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot get gas price:%s", err))
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := rpc.client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot get nonce:%s", err))
	}
	decodedContractData, err := hexutil.Decode(contractData)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot decode contract data:%s", err))
	}
	tx := types.NewContractCreation(nonce, big.NewInt(0), 2000000, gasPrice, decodedContractData)
	signedTx, _ := types.SignTx(tx, types.HomesteadSigner{}, privateKey)

	err = rpc.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot send transaction:%s", err))
	}

	return signedTx, nil
}
func (rpc *RPCClient) WaitForReceipt(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	for i := 0; ; i++ {
		receipt, err := rpc.client.TransactionReceipt(ctx, tx.Hash())
		if err == nil {
			return receipt, nil
		}
		if i >= (5) {
			return nil, errors.New(fmt.Sprintf("Cannot get the receipt:%s", err))
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
