package web3

import (
	"context"
	"math/big"

	"github.com/gochain-io/gochain/consensus/clique"
	"github.com/gochain-io/gochain/core/types"
)

//TODO return pure response types from rpc instead of types.*
type Client interface {
	GetBalance(ctx context.Context, address string, blockNumber *big.Int) (*big.Int, error)
	GetCode(ctx context.Context, address string, blockNumber *big.Int) ([]byte, error)
	GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	GetTransactionByHash(ctx context.Context, hash string) (*types.Transaction, bool, error)
	GetSnapshot(ctx context.Context) (*clique.Snapshot, error)
	GetID(ctx context.Context) (*ID, error)
	DeployContract(ctx context.Context, privateKeyHex string, contractData string) (*types.Transaction, error)
	WaitForReceipt(ctx context.Context, tx *types.Transaction) (*types.Receipt, error)
}
