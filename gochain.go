package main

import (
	"context"
	"math/big"

	"github.com/gochain-io/gochain/common"
	"github.com/gochain-io/gochain/core/types"
	"github.com/gochain-io/gochain/goclient"
	"github.com/rs/zerolog/log"
)

type RPCClient struct {
	url    string
	client *goclient.Client
}

func GetClient(rpcURL string) *RPCClient {
	client, err := goclient.Dial(rpcURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot connect to the network")
	}
	rpc := &RPCClient{
		url:    rpcURL,
		client: client,
	}
	return rpc
}

func (rpc *RPCClient) GetBalance(address string, blockNumber *big.Int) (*big.Int, error) {
	balance, err := rpc.client.BalanceAt(context.Background(), common.HexToAddress(address), blockNumber)
	if err != nil {
		log.Info().Err(err).Str("Address", address).Msg("Cannot get balance details from the network")
	}
	return balance, err
}

func (rpc *RPCClient) GetCode(address string, blockNumber *big.Int) ([]byte, error) {
	code, err := rpc.client.CodeAt(context.Background(), common.HexToAddress(address), blockNumber)
	if err != nil {
		log.Info().Err(err).Str("Address", address).Msg("Cannot get code details from the network")
	}
	return code, err
}

func (rpc *RPCClient) GetBlockByNumber(number *big.Int) (*types.Block, error) {
	blockEth, err := rpc.client.BlockByNumber(context.Background(), number)
	if err != nil {
		log.Info().Err(err).Int64("blockNumber", number.Int64()).Msg("Cannot get block from the network")
	}
	return blockEth, err
}

func (rpc *RPCClient) GetTransactionByHash(hash string) (*types.Transaction, bool, error) {
	tx, isPending, err := rpc.client.TransactionByHash(context.Background(), common.HexToHash(hash))
	if err != nil {
		log.Info().Err(err).Str("TX hash", hash).Msg("cannot get transaction from the network")
	}
	return tx, isPending, err
}
