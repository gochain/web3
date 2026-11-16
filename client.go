package web3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"
)

// Client is the interface for interacting with an EVM JSON-RPC node.
type Client interface {
	GetBalance(ctx context.Context, address string, blockNumber *big.Int) (*big.Int, error)
	GetCode(ctx context.Context, address string, blockNumber *big.Int) ([]byte, error)
	GetStorageAt(ctx context.Context, address Address, slot Hash, blockNumber *big.Int) (Hash, error)
	GetBlockNumber(ctx context.Context) (uint64, error)
	GetBlockByNumber(ctx context.Context, number *big.Int, includeTxs bool) (*Block, error)
	GetBlockByHash(ctx context.Context, hash string, includeTxs bool) (*Block, error)
	GetTransactionByHash(ctx context.Context, hash Hash) (*Transaction, error)
	GetTransactionReceipt(ctx context.Context, hash Hash) (*Receipt, error)
	GetLogs(ctx context.Context, q FilterQuery) ([]Log, error)
	GetSnapshot(ctx context.Context) (*Snapshot, error)
	GetID(ctx context.Context) (*ID, error)
	GetChainID(ctx context.Context) (*big.Int, error)
	GetNetworkID(ctx context.Context) (*big.Int, error)
	GetGasPrice(ctx context.Context) (*big.Int, error)
	MaxPriorityFeePerGas(ctx context.Context) (*big.Int, error)
	EstimateGas(ctx context.Context, msg CallMsg) (uint64, error)
	GetPendingTransactionCount(ctx context.Context, account Address) (uint64, error)
	GetTransactionCount(ctx context.Context, account Address, blockNumber *big.Int) (uint64, error)
	SendRawTransaction(ctx context.Context, tx []byte) error
	Call(ctx context.Context, msg CallMsg) ([]byte, error)
	Close()
	SetChainID(*big.Int)
}

// Dial returns a new Client connected to url ("http", "https", "ws", or "wss").
func Dial(url string) (Client, error) {
	r, err := DialRPC(url)
	if err != nil {
		return nil, err
	}
	return NewClient(r), nil
}

// NewClient wraps an *RPCClient in a web3 Client.
func NewClient(r *RPCClient) Client {
	return &client{r: r}
}

type client struct {
	r       *RPCClient
	chainID atomic.Value
}

func (c *client) Close() {
	c.r.Close()
}

func (c *client) Call(ctx context.Context, msg CallMsg) ([]byte, error) {
	var result HexBytes
	if err := c.r.CallContext(ctx, &result, "eth_call", toCallArg(msg), "latest"); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) EstimateGas(ctx context.Context, msg CallMsg) (uint64, error) {
	var result HexUint64
	if err := c.r.CallContext(ctx, &result, "eth_estimateGas", toCallArg(msg)); err != nil {
		return 0, err
	}
	return uint64(result), nil
}

func (c *client) GetBalance(ctx context.Context, address string, blockNumber *big.Int) (*big.Int, error) {
	var result HexBig
	err := c.r.CallContext(ctx, &result, "eth_getBalance", HexToAddress(address), toBlockNumArg(blockNumber))
	if err != nil {
		return nil, err
	}
	return result.ToInt(), nil
}

func (c *client) GetCode(ctx context.Context, address string, blockNumber *big.Int) ([]byte, error) {
	var result HexBytes
	err := c.r.CallContext(ctx, &result, "eth_getCode", HexToAddress(address), toBlockNumArg(blockNumber))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) GetStorageAt(ctx context.Context, address Address, slot Hash, blockNumber *big.Int) (Hash, error) {
	var result HexBytes
	if err := c.r.CallContext(ctx, &result, "eth_getStorageAt", address, slot, toBlockNumArg(blockNumber)); err != nil {
		return ZeroHash, err
	}
	return BytesToHash(result), nil
}

func (c *client) GetBlockNumber(ctx context.Context) (uint64, error) {
	var result HexUint64
	if err := c.r.CallContext(ctx, &result, "eth_blockNumber"); err != nil {
		return 0, err
	}
	return uint64(result), nil
}

func (c *client) GetBlockByNumber(ctx context.Context, number *big.Int, includeTxs bool) (*Block, error) {
	return c.getBlock(ctx, "eth_getBlockByNumber", toBlockNumArg(number), includeTxs)
}

func (c *client) GetBlockByHash(ctx context.Context, hash string, includeTxs bool) (*Block, error) {
	return c.getBlock(ctx, "eth_getBlockByHash", hash, includeTxs)
}

func (c *client) GetTransactionByHash(ctx context.Context, hash Hash) (*Transaction, error) {
	var tx *Transaction
	err := c.r.CallContext(ctx, &tx, "eth_getTransactionByHash", hash.String())
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, NotFoundErr
	}
	return tx, nil
}

func (c *client) GetLogs(ctx context.Context, q FilterQuery) ([]Log, error) {
	arg, err := toFilterArg(q)
	if err != nil {
		return nil, err
	}
	var logs []Log
	if err := c.r.CallContext(ctx, &logs, "eth_getLogs", arg); err != nil {
		return nil, err
	}
	return logs, nil
}

func (c *client) GetSnapshot(ctx context.Context) (*Snapshot, error) {
	var s Snapshot
	if err := c.r.CallContext(ctx, &s, "clique_getSnapshot", "latest"); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *client) GetID(ctx context.Context) (*ID, error) {
	var block Block
	var netIDStr string
	chainID := new(HexBig)
	batch := []BatchElem{
		{Method: "eth_getBlockByNumber", Args: []interface{}{"0x0", false}, Result: &block},
		{Method: "net_version", Result: &netIDStr},
		{Method: "eth_chainId", Result: chainID},
	}
	if err := c.r.BatchCallContext(ctx, batch); err != nil {
		return nil, err
	}
	for _, e := range batch {
		if e.Error != nil {
			return nil, fmt.Errorf("method %q failed: %w", e.Method, e.Error)
		}
	}
	netID := new(big.Int)
	if _, ok := netID.SetString(netIDStr, 10); !ok {
		return nil, fmt.Errorf("invalid net_version result %q", netIDStr)
	}
	return &ID{NetworkID: netID, ChainID: chainID.ToInt(), GenesisHash: block.Hash}, nil
}

func (c *client) GetNetworkID(ctx context.Context) (*big.Int, error) {
	var ver string
	if err := c.r.CallContext(ctx, &ver, "net_version"); err != nil {
		return nil, err
	}
	version, ok := new(big.Int).SetString(ver, 10)
	if !ok {
		return nil, fmt.Errorf("invalid net_version result %q", ver)
	}
	return version, nil
}

func (c *client) SetChainID(chainID *big.Int) {
	if chainID != nil {
		c.chainID.Store(new(big.Int).Set(chainID))
	}
}

func (c *client) GetChainID(ctx context.Context) (*big.Int, error) {
	if l := c.chainID.Load(); l != nil {
		if i, ok := l.(*big.Int); ok && i != nil {
			return new(big.Int).Set(i), nil
		}
	}
	var result HexBig
	if err := c.r.CallContext(ctx, &result, "eth_chainId"); err != nil {
		return nil, err
	}
	i := result.ToInt()
	c.SetChainID(i)
	return i, nil
}

func (c *client) GetTransactionReceipt(ctx context.Context, hash Hash) (*Receipt, error) {
	var r *Receipt
	err := c.r.CallContext(ctx, &r, "eth_getTransactionReceipt", hash)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, NotFoundErr
	}
	return r, nil
}

func (c *client) GetGasPrice(ctx context.Context) (*big.Int, error) {
	var hex HexBig
	if err := c.r.CallContext(ctx, &hex, "eth_gasPrice"); err != nil {
		return nil, err
	}
	return hex.ToInt(), nil
}

func (c *client) MaxPriorityFeePerGas(ctx context.Context) (*big.Int, error) {
	var hex HexBig
	if err := c.r.CallContext(ctx, &hex, "eth_maxPriorityFeePerGas"); err != nil {
		return nil, err
	}
	return hex.ToInt(), nil
}

func (c *client) GetPendingTransactionCount(ctx context.Context, account Address) (uint64, error) {
	return c.getTransactionCount(ctx, account, "pending")
}

func (c *client) GetTransactionCount(ctx context.Context, account Address, blockNumber *big.Int) (uint64, error) {
	return c.getTransactionCount(ctx, account, toBlockNumArg(blockNumber))
}

func (c *client) getTransactionCount(ctx context.Context, account Address, blockNumArg string) (uint64, error) {
	var result HexUint64
	err := c.r.CallContext(ctx, &result, "eth_getTransactionCount", account, blockNumArg)
	return uint64(result), err
}

func (c *client) SendRawTransaction(ctx context.Context, tx []byte) error {
	return c.r.CallContext(ctx, nil, "eth_sendRawTransaction", EncodeHex(tx))
}

func (c *client) getBlock(ctx context.Context, method string, hashOrNum string, includeTxs bool) (*Block, error) {
	var raw json.RawMessage
	err := c.r.CallContext(ctx, &raw, method, hashOrNum, includeTxs)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, NotFoundErr
	}
	var block Block
	if err := json.Unmarshal(raw, &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json response: %w", err)
	}
	if block.Sha3Uncles == EmptyUncleHash && len(block.Uncles) > 0 {
		return nil, errors.New("server returned non-empty uncle list but block header indicates no uncles")
	}
	if !block.Sha3Uncles.IsZero() && block.Sha3Uncles != EmptyUncleHash && len(block.Uncles) == 0 {
		return nil, errors.New("server returned empty uncle list but block header indicates uncles")
	}
	if block.TxsRoot == EmptyRootHash && block.TxCount() > 0 {
		return nil, errors.New("server returned non-empty transaction list but block header indicates no transactions")
	}
	if !block.TxsRoot.IsZero() && block.TxsRoot != EmptyRootHash && block.TxCount() == 0 {
		return nil, errors.New("server returned empty transaction list but block header indicates transactions")
	}
	return &block, nil
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	if number.Sign() < 0 {
		return "latest"
	}
	return EncodeBig(number)
}

func toCallArg(msg CallMsg) interface{} {
	arg := make(map[string]interface{})
	if msg.From != nil {
		arg["from"] = msg.From
	}
	if msg.To != nil {
		arg["to"] = msg.To
	}
	if len(msg.Data) > 0 {
		arg["input"] = HexBytes(msg.Data)
		arg["data"] = HexBytes(msg.Data)
	}
	if msg.Value != nil && msg.Value.Sign() > 0 {
		arg["value"] = (*HexBig)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = HexUint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*HexBig)(msg.GasPrice)
	}
	if msg.MaxFeePerGas != nil {
		arg["maxFeePerGas"] = (*HexBig)(msg.MaxFeePerGas)
	}
	if msg.MaxPriorityFeePerGas != nil {
		arg["maxPriorityFeePerGas"] = (*HexBig)(msg.MaxPriorityFeePerGas)
	}
	return arg
}

func toFilterArg(q FilterQuery) (interface{}, error) {
	arg := make(map[string]interface{})
	if len(q.Addresses) == 1 {
		arg["address"] = q.Addresses[0]
	} else if len(q.Addresses) > 1 {
		arg["address"] = q.Addresses
	}
	if len(q.Topics) > 0 {
		topics := make([]interface{}, len(q.Topics))
		for i, group := range q.Topics {
			if len(group) == 0 {
				topics[i] = nil
			} else if len(group) == 1 {
				topics[i] = group[0]
			} else {
				topics[i] = group
			}
		}
		arg["topics"] = topics
	}
	if q.BlockHash != nil {
		if q.FromBlock != nil || q.ToBlock != nil {
			return nil, errors.New("cannot specify both BlockHash and FromBlock/ToBlock")
		}
		arg["blockHash"] = *q.BlockHash
		return arg, nil
	}
	arg["fromBlock"] = toBlockNumArg(q.FromBlock)
	arg["toBlock"] = toBlockNumArg(q.ToBlock)
	return arg, nil
}
