package web3_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync/atomic"

	zlog "github.com/rs/zerolog/log"
	web3_types "github.com/zeus-fyi/gochain/web3/types"

	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/common/hexutil"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/gochain/gochain/v4/rpc"
)

var NotFoundErr = errors.New("not found")

// Client is an interface for the web3 RPC API.
type Client interface {
	// GetBalance returns the balance for an address at the given block number (nil for latest).
	GetBalance(ctx context.Context, address string, blockNumber *big.Int) (*big.Int, error)
	// GetCode returns the code for an address at the given block number (nil for latest).
	GetCode(ctx context.Context, address string, blockNumber *big.Int) ([]byte, error)
	// GetBlockByNumber returns block details by number (nil for latest), optionally including full txs.
	GetBlockByNumber(ctx context.Context, number *big.Int, includeTxs bool) (*web3_types.Block, error)
	// GetBlockByHash returns block details for the given hash, optionally include full transaction details.
	GetBlockByHash(ctx context.Context, hash string, includeTxs bool) (*web3_types.Block, error)
	// GetTransactionByHash returns transaction details for a hash.
	GetTransactionByHash(ctx context.Context, hash common.Hash) (*web3_types.Transaction, error)
	// GetSnapshot returns the latest clique snapshot.
	GetSnapshot(ctx context.Context) (*web3_types.Snapshot, error)
	// GetID returns unique identifying information for the network.
	GetID(ctx context.Context) (*web3_types.ID, error)
	// GetTransactionReceipt returns the receipt for a transaction hash.
	GetTransactionReceipt(ctx context.Context, hash common.Hash) (*web3_types.Receipt, error)
	// GetChainID returns the chain id for the network.
	GetChainID(ctx context.Context) (*big.Int, error)
	// GetNetworkID returns the network id.
	GetNetworkID(ctx context.Context) (*big.Int, error)
	// GetGasPriceEstimateForTx returns the estimated gas cost for a given transcation
	GetGasPriceEstimateForTx(ctx context.Context, msg web3_types.CallMsg) (*big.Int, error)
	// GetGasPrice returns a suggested gas price.
	GetGasPrice(ctx context.Context) (*big.Int, error)
	// GetPendingTransactionCount returns the transaction count including pending txs.
	// This value is also the next legal nonce.
	GetPendingTransactionCount(ctx context.Context, account common.Address) (uint64, error)
	// SendRawTransaction sends the signed raw transaction bytes.
	SendRawTransaction(ctx context.Context, tx []byte) error
	// Call executes a call without submitting a transaction.
	Call(ctx context.Context, msg web3_types.CallMsg) ([]byte, error)
	Close()
	SetChainID(*big.Int)
}

// Dial returns a new client backed by dialing url (supported schemes "http", "https", "ws" and "wss").
func Dial(url string) (Client, error) {
	r, err := rpc.Dial(url)
	if err != nil {
		zlog.Err(err).Msg("Dial")
		return nil, err
	}
	return NewClient(r), nil
}

// NewClient returns a new client backed by an existing rpc.Client.
func NewClient(r *rpc.Client) Client {
	return &client{r: r}
}

type client struct {
	r       *rpc.Client
	chainID atomic.Value
}

func (c *client) Close() {
	c.r.Close()
}

func (c *client) Call(ctx context.Context, msg web3_types.CallMsg) ([]byte, error) {
	var result hexutil.Bytes
	err := c.r.CallContext(ctx, &result, "eth_call", toCallArg(msg), "latest")
	if err != nil {
		zlog.Err(err).Msg("client: Call: CallContext")
		return nil, err
	}
	return result, err
}

func (c *client) GetBalance(ctx context.Context, address string, blockNumber *big.Int) (*big.Int, error) {
	var result hexutil.Big
	err := c.r.CallContext(ctx, &result, "eth_getBalance", common.HexToAddress(address), toBlockNumArg(blockNumber))
	return (*big.Int)(&result), err
}

func (c *client) GetCode(ctx context.Context, address string, blockNumber *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	err := c.r.CallContext(ctx, &result, "eth_getCode", common.HexToAddress(address), toBlockNumArg(blockNumber))
	if err != nil {
		zlog.Err(err).Msg("GetCode: CallContext")
		return result, err
	}
	return result, err
}

func (c *client) GetBlockByNumber(ctx context.Context, number *big.Int, includeTxs bool) (*web3_types.Block, error) {
	return c.getBlock(ctx, "eth_getBlockByNumber", toBlockNumArg(number), includeTxs)
}

func (c *client) GetBlockByHash(ctx context.Context, hash string, includeTxs bool) (*web3_types.Block, error) {
	return c.getBlock(ctx, "eth_getBlockByHash", hash, includeTxs)
}

func (c *client) GetTransactionByHash(ctx context.Context, hash common.Hash) (*web3_types.Transaction, error) {
	var tx *web3_types.Transaction
	err := c.r.CallContext(ctx, &tx, "eth_getTransactionByHash", hash.String())
	if err != nil {
		zlog.Err(err).Msg("GetTransactionByHash: CallContext")
		return nil, err
	} else if tx == nil {
		zlog.Err(NotFoundErr).Msg("GetTransactionByHash: NotFoundErr")
		return nil, NotFoundErr
	} else if tx.R == nil {
		zlog.Err(err).Msg("GetTransactionByHash: tx.R == nil")
		return nil, fmt.Errorf("server returned transaction without signature")
	}
	return tx, nil
}

func (c *client) GetSnapshot(ctx context.Context) (*web3_types.Snapshot, error) {
	var s web3_types.Snapshot
	err := c.r.CallContext(ctx, &s, "clique_getSnapshot", "latest")
	if err != nil {
		zlog.Err(err).Msg("GetSnapshot: CallContext")
		return nil, err
	}
	return &s, nil
}

func (c *client) GetID(ctx context.Context) (*web3_types.ID, error) {
	var block web3_types.Block
	var netIDStr string
	chainID := new(hexutil.Big)
	batch := []rpc.BatchElem{
		{Method: "eth_getBlockByNumber", Args: []interface{}{"0x0", false}, Result: &block},
		{Method: "net_version", Result: &netIDStr},
		{Method: "eth_chainId", Result: chainID},
	}
	if err := c.r.BatchCallContext(ctx, batch); err != nil {
		return nil, err
	}
	for _, e := range batch {
		if e.Error != nil {
			zlog.Err(e.Error).Msg("GetID: BatchCallContext")
			log.Printf("Method %q failed: %v\n", e.Method, e.Error)
		}
	}
	netID := new(big.Int)
	if _, ok := netID.SetString(netIDStr, 10); !ok {
		err := fmt.Errorf("invalid net_version result %q", netIDStr)
		zlog.Err(err).Msg("GetID: netID.SetString(netIDStr, 10)")
		return nil, err
	}
	return &web3_types.ID{NetworkID: netID, ChainID: (*big.Int)(chainID), GenesisHash: block.Hash}, nil
}

func (c *client) GetNetworkID(ctx context.Context) (*big.Int, error) {
	version := new(big.Int)
	var ver string
	if err := c.r.CallContext(ctx, &ver, "net_version"); err != nil {
		zlog.Err(err).Msg("GetNetworkID: CallContext")
		return nil, err
	}
	if _, ok := version.SetString(ver, 10); !ok {
		err := fmt.Errorf("invalid net_version result %q", ver)
		zlog.Err(err).Msg("GetNetworkID: CallContext")
		return nil, err
	}
	return version, nil
}

func (c *client) SetChainID(chainID *big.Int) {
	c.chainID.Store(chainID)
}

func (c *client) GetChainID(ctx context.Context) (*big.Int, error) {
	if l := c.chainID.Load(); l != nil {
		if i := l.(*big.Int); i != nil {
			return i, nil
		}
	}
	var result hexutil.Big
	err := c.r.CallContext(ctx, &result, "eth_chainId")
	i := (*big.Int)(&result)
	c.SetChainID(i)
	return i, err
}

func (c *client) GetTransactionReceipt(ctx context.Context, hash common.Hash) (*web3_types.Receipt, error) {
	var r *web3_types.Receipt
	err := c.r.CallContext(ctx, &r, "eth_getTransactionReceipt", hash)
	if err == nil {
		if r == nil {
			zlog.Err(NotFoundErr).Msg("GetTransactionReceipt: NotFoundErr")
			return nil, NotFoundErr
		}
	}
	return r, err
}

// SuggestGasTipCap retrieves the currently suggested gas tip cap after 1559 to
// allow a timely execution of a transaction.
func (c *client) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	var hex hexutil.Big
	if err := c.r.CallContext(ctx, &hex, "eth_maxPriorityFeePerGas"); err != nil {
		return nil, err
	}
	return (*big.Int)(&hex), nil
}

func (c *client) GetGasPriceEstimateForTx(ctx context.Context, msg web3_types.CallMsg) (*big.Int, error) {
	var hex hexutil.Big
	if err := c.r.CallContext(ctx, &hex, "eth_estimateGas", toCallArg(msg)); err != nil {
		zlog.Err(err).Msg("GetGasPriceEstimateForTx: CallContext")
		return nil, err
	}
	return (*big.Int)(&hex), nil
}

func (c *client) GetGasPrice(ctx context.Context) (*big.Int, error) {
	var hex hexutil.Big
	if err := c.r.CallContext(ctx, &hex, "eth_gasPrice"); err != nil {
		zlog.Err(err).Msg("GetGasPrice: CallContext")
		return nil, err
	}
	return (*big.Int)(&hex), nil
}

func (c *client) GetPendingTransactionCount(ctx context.Context, account common.Address) (uint64, error) {
	return c.getTransactionCount(ctx, account, "pending")
}

func (c *client) getTransactionCount(ctx context.Context, account common.Address, blockNumArg string) (uint64, error) {
	var result hexutil.Uint64
	err := c.r.CallContext(ctx, &result, "eth_getTransactionCount", account, blockNumArg)
	if err != nil {
		zlog.Err(err).Msg("client: getTransactionCount")
	}
	return uint64(result), err
}

func (c *client) SendRawTransaction(ctx context.Context, tx []byte) error {
	return c.r.CallContext(ctx, nil, "eth_sendRawTransaction", hexutil.Encode(tx))
}

func (c *client) getBlock(ctx context.Context, method string, hashOrNum string, includeTxs bool) (*web3_types.Block, error) {
	var raw json.RawMessage
	err := c.r.CallContext(ctx, &raw, method, hashOrNum, includeTxs)
	if err != nil {
		zlog.Err(err).Msg("client: getBlock")
		return nil, err
	} else if len(raw) == 0 {
		zlog.Err(NotFoundErr).Msg("client: NotFoundErr")
		return nil, NotFoundErr
	}
	var block web3_types.Block
	if err = json.Unmarshal(raw, &block); err != nil {
		err = fmt.Errorf("failed to unmarshal json response: %v", err)
		zlog.Err(err).Msg("client: getBlock")
		return nil, err
	}
	// Quick-verify transaction and uncle lists. This mostly helps with debugging the server.
	if block.Sha3Uncles == types.EmptyUncleHash && len(block.Uncles) > 0 {
		err = fmt.Errorf("server returned non-empty uncle list but block header indicates no uncles")
		zlog.Err(err).Msg("client: getBlock")
		return nil, err
	}
	if block.Sha3Uncles != types.EmptyUncleHash && len(block.Uncles) == 0 {
		err = fmt.Errorf("server returned empty uncle list but block header indicates uncles")
		zlog.Err(err).Msg("client: getBlock")
		return nil, err
	}
	if block.TxsRoot == types.EmptyRootHash && block.TxCount() > 0 {
		err = fmt.Errorf("server returned non-empty transaction list but block header indicates no transactions")
		zlog.Err(err).Msg("client: getBlock")
		return nil, err
	}
	if block.TxsRoot != types.EmptyRootHash && len(block.TxsRoot) == 0 {
		return nil, fmt.Errorf("server returned empty transaction list but block header indicates transactions")
	}
	// Load uncles because they are not included in the block response.
	var uncles []*types.Header
	if len(block.Uncles) > 0 {
		uncles = make([]*types.Header, len(block.Uncles))
		reqs := make([]rpc.BatchElem, len(block.Uncles))
		for i := range reqs {
			reqs[i] = rpc.BatchElem{
				Method: "eth_getUncleByBlockHashAndIndex",
				Args:   []interface{}{block.Hash, hexutil.EncodeUint64(uint64(i))},
				Result: &uncles[i],
			}
		}
		if err = c.r.BatchCallContext(ctx, reqs); err != nil {
			zlog.Err(err).Msg("client: getBlock, BatchCallContext")
			return nil, err
		}
		for i := range reqs {
			if reqs[i].Error != nil {
				zlog.Err(reqs[i].Error).Msg("client: getBlock")
				return nil, reqs[i].Error
			}
			if uncles[i] == nil {
				err = fmt.Errorf("got null header for uncle %d of block %x", i, block.Hash[:])
				zlog.Err(err).Msg("client: getBlock")
				return nil, err
			}
		}
	}
	return &block, nil
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	return hexutil.EncodeBig(number)
}

func toCallArg(msg web3_types.CallMsg) interface{} {
	arg := map[string]interface{}{
		"to": msg.To,
	}
	if msg.From != nil {
		arg["from"] = msg.From
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}
