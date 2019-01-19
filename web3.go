package web3

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/gochain-io/gochain/common"
	"github.com/gochain-io/gochain/common/hexutil"
	"github.com/gochain-io/gochain/core/types"
	"github.com/gochain-io/gochain/crypto"
	"github.com/gochain-io/gochain/rlp"
	"github.com/gochain-io/gochain/rpc"
)

var NotFoundErr = errors.New("not found")

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

type Client interface {
	GetBalance(ctx context.Context, address string, blockNumber *big.Int) (*big.Int, error)
	GetCode(ctx context.Context, address string, blockNumber *big.Int) ([]byte, error)
	GetBlockByNumber(ctx context.Context, number *big.Int, includeTxs bool) (*Block, error)
	GetTransactionByHash(ctx context.Context, hash string) (*Transaction, error)
	GetSnapshot(ctx context.Context) (*Snapshot, error)
	GetID(ctx context.Context) (*ID, error)
	GetTransactionReceipt(ctx context.Context, hash common.Hash) (*Receipt, error)
	GetChainID(ctx context.Context) (*big.Int, error)
	GetNetworkID(ctx context.Context) (*big.Int, error)
	// GetGasPrice returns a suggested gas price.
	GetGasPrice(ctx context.Context) (*big.Int, error)
	// GetPendingTransactionCount returns the transaction count including pending txs.
	// This value is also the next legal nonce.
	GetPendingTransactionCount(ctx context.Context, account common.Address) (uint64, error)
	SendTransaction(ctx context.Context, tx *Transaction) error
	Close()
}

func NewClient(url string) (Client, error) {
	r, err := rpc.Dial(url)
	if err != nil {
		return nil, err
	}
	return &client{r: r}, nil
}

type client struct {
	r *rpc.Client
}

func (c *client) Close() {
	c.r.Close()
}

func (c *client) GetBalance(ctx context.Context, address string, blockNumber *big.Int) (*big.Int, error) {
	var result hexutil.Big
	err := c.r.CallContext(ctx, &result, "eth_getBalance", common.HexToAddress(address), toBlockNumArg(blockNumber))
	return (*big.Int)(&result), err
}

func (c *client) GetCode(ctx context.Context, address string, blockNumber *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	err := c.r.CallContext(ctx, &result, "eth_getCode", common.HexToAddress(address), toBlockNumArg(blockNumber))
	return result, err
}

type Block struct {
	ParentHash      common.Hash      `json:"parentHash"`
	Sha3Uncles      common.Hash      `json:"sha3Uncles"`
	Miner           common.Address   `json:"miner"`
	Signers         []common.Address `json:"signers"`
	Voters          []common.Address `json:"voters"`
	Signer          hexutil.Bytes    `json:"signer"`
	StateRoot       common.Hash      `json:"stateRoot"`
	TxsRoot         common.Hash      `json:"transactionsRoot"`
	ReceiptsRoot    common.Hash      `json:"receiptsRoot"`
	LogsBloom       *types.Bloom     `json:"logsBloom"`
	Difficulty      string           `json:"difficulty"`
	TotalDifficulty string           `json:"totalDifficulty"`
	Number          string           `json:"number"`
	GasLimit        string           `json:"gasLimit"`
	GasUsed         string           `json:"gasUsed"`
	Timestamp       string           `json:"timestamp"`
	ExtraData       hexutil.Bytes    `json:"extraData"`
	MixHash         common.Hash      `json:"mixHash"`
	Nonce           types.BlockNonce `json:"nonce"`
	Hash            common.Hash      `json:"hash"`

	// TODO support full Transactions
	Txs    []common.Hash `json:"transactions,omitempty"`
	Uncles []common.Hash `json:"uncles"`
}

func (b *Block) DifficultyInt64() (int64, error) {
	return strconv.ParseInt(b.Difficulty, 0, 64)
}

func (b *Block) NumberInt64() (int64, error) {
	return strconv.ParseInt(b.Number, 0, 64)
}

func (b *Block) GasLimitInt64() (int64, error) {
	return strconv.ParseInt(b.GasLimit, 0, 64)
}

func (b *Block) GasUsedInt64() (int64, error) {
	return strconv.ParseInt(b.GasUsed, 0, 64)
}

func (b *Block) TimestampUnix() (int64, error) {
	return strconv.ParseInt(b.Timestamp, 0, 64)
}

func (b *Block) ExtraVanity() string {
	l := len(b.ExtraData)
	if l > 32 {
		l = 32
	}
	return string(b.ExtraData[:l])
}

func (c *client) GetBlockByNumber(ctx context.Context, number *big.Int, includeTxs bool) (*Block, error) {
	return c.getBlock(ctx, "eth_getBlockByNumber", toBlockNumArg(number), includeTxs)
}

type Transaction struct {
	Nonce            hexutil.Uint64 `json:"nonce"`
	GasPrice         hexutil.Big    `json:"gasPrice"`
	GasLimit         hexutil.Big    `json:"gas"`
	To               common.Address `json:"to"`
	Value            hexutil.Big    `json:"value"`
	Input            hexutil.Bytes  `json:"input"`
	Hash             common.Hash    `json:"hash"`
	BlockNumber      hexutil.Big    `json:"blockNumber"`
	BlockHash        common.Hash    `json:"blockHash"`
	From             common.Address `json:"from"`
	TransactionIndex hexutil.Uint64 `json:"transactionIndex"`
	V                hexutil.Big    `json:"v"`
	R                common.Hash    `json:"r"`
	S                common.Hash    `json:"s"`
}

func (c *client) GetTransactionByHash(ctx context.Context, hash string) (*Transaction, error) {
	var tx *Transaction
	err := c.r.CallContext(ctx, &tx, "eth_getTransactionByHash", hash)
	if err != nil {
		return nil, err
	} else if tx == nil {
		return nil, NotFoundErr
	} else if tx.R == (common.Hash{}) {
		return nil, fmt.Errorf("server returned transaction without signature")
	}
	return tx, nil
}

type Snapshot struct {
	Number  uint64                      `json:"number"`
	Hash    common.Hash                 `json:"hash"`
	Signers map[common.Address]uint64   `json:"signers"`
	Voters  map[common.Address]struct{} `json:"voters"`
	Votes   []*Vote                     `json:"votes"`
	Tally   map[common.Address]Tally    `json:"tally"`
}

type Vote struct {
	Signer    common.Address `json:"signer"`
	Block     uint64         `json:"block"`
	Address   common.Address `json:"address"`
	Authorize bool           `json:"authorize"`
}

type Tally struct {
	Authorize bool `json:"authorize"`
	Votes     int  `json:"votes"`
}

func (c *client) GetSnapshot(ctx context.Context) (*Snapshot, error) {
	var s Snapshot
	err := c.r.CallContext(ctx, &s, "clique_getSnapshot", "latest")
	if err != nil {
		return nil, err
	}
	return &s, nil
}

type ID struct {
	NetworkID   *big.Int    `json:"network_id"`
	ChainID     *big.Int    `json:"chain_id"`
	GenesisHash common.Hash `json:"genesis_hash"`
}

func (c *client) GetID(ctx context.Context) (*ID, error) {
	var block Block
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
			log.Printf("Method %q failed: %v\n", e.Method, e.Error)
		}
	}
	netID := new(big.Int)
	if _, ok := netID.SetString(netIDStr, 10); !ok {
		return nil, fmt.Errorf("invalid net_version result %q", netIDStr)
	}
	return &ID{NetworkID: netID, ChainID: (*big.Int)(chainID), GenesisHash: block.Hash}, nil
}

func (c *client) GetNetworkID(ctx context.Context) (*big.Int, error) {
	version := new(big.Int)
	var ver string
	if err := c.r.CallContext(ctx, &ver, "net_version"); err != nil {
		return nil, err
	}
	if _, ok := version.SetString(ver, 10); !ok {
		return nil, fmt.Errorf("invalid net_version result %q", ver)
	}
	return version, nil
}

func (c *client) GetChainID(ctx context.Context) (*big.Int, error) {
	var result hexutil.Big
	err := c.r.CallContext(ctx, &result, "eth_chainId")
	return (*big.Int)(&result), err
}

type Receipt struct {
	PostState         []byte         `json:"root"`
	Status            uint64         `json:"status"`
	CumulativeGasUsed uint64         `json:"cumulativeGasUsed"`
	Bloom             types.Bloom    `json:"logsBloom"`
	Logs              []*types.Log   `json:"logs"`
	TxHash            common.Hash    `json:"transactionHash"`
	ContractAddress   common.Address `json:"contractAddress"`
	GasUsed           uint64         `json:"gasUsed"`
}

func (c *client) GetTransactionReceipt(ctx context.Context, hash common.Hash) (*Receipt, error) {
	var r *Receipt
	err := c.r.CallContext(ctx, &r, "eth_getTransactionReceipt", hash)
	if err == nil {
		if r == nil {
			return nil, NotFoundErr
		}
	}
	return r, err
}

func (c *client) GetGasPrice(ctx context.Context) (*big.Int, error) {
	var hex hexutil.Big
	if err := c.r.CallContext(ctx, &hex, "eth_gasPrice"); err != nil {
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
	return uint64(result), err
}

func (c *client) SendTransaction(ctx context.Context, tx *Transaction) error {
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}
	return c.r.CallContext(ctx, nil, "eth_sendRawTransaction", common.ToHex(data))
}

func (c *client) getBlock(ctx context.Context, method string, hashOrNum string, includeTxs bool) (*Block, error) {
	var raw json.RawMessage
	err := c.r.CallContext(ctx, &raw, method, hashOrNum, includeTxs)
	if err != nil {
		return nil, err
	} else if len(raw) == 0 {
		return nil, NotFoundErr
	}
	var block Block
	if err := json.Unmarshal(raw, &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json response: %v", err)
	}
	// Quick-verify transaction and uncle lists. This mostly helps with debugging the server.
	if block.Sha3Uncles == types.EmptyUncleHash && len(block.Uncles) > 0 {
		return nil, fmt.Errorf("server returned non-empty uncle list but block header indicates no uncles")
	}
	if block.Sha3Uncles != types.EmptyUncleHash && len(block.Uncles) == 0 {
		return nil, fmt.Errorf("server returned empty uncle list but block header indicates uncles")
	}
	if block.TxsRoot == types.EmptyRootHash && len(block.Txs) > 0 {
		return nil, fmt.Errorf("server returned non-empty transaction list but block header indicates no transactions")
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
		if err := c.r.BatchCallContext(ctx, reqs); err != nil {
			return nil, err
		}
		for i := range reqs {
			if reqs[i].Error != nil {
				return nil, reqs[i].Error
			}
			if uncles[i] == nil {
				return nil, fmt.Errorf("got null header for uncle %d of block %x", i, block.Hash[:])
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
	//TODO try to use native Transaction only; can't sign currently
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
	rtx.Nonce = hexutil.Uint64(tx.Nonce())
	rtx.GasPrice.ToInt().Set(tx.GasPrice())
	rtx.GasLimit.ToInt().SetUint64(tx.Gas())
	rtx.To = *tx.To()
	rtx.Value.ToInt().Set(tx.Value())
	rtx.Input = hexutil.Bytes(tx.Data())
	rtx.Hash = tx.Hash()
	rtx.From = from
	v, r, s := tx.RawSignatureValues()
	rtx.V.ToInt().Set(v)
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
