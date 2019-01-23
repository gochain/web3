package web3

import (
	"time"

	"github.com/gochain-io/gochain/v3/common"
	"github.com/gochain-io/gochain/v3/common/hexutil"
	"github.com/gochain-io/gochain/v3/core/types"
)

type rpcBlock struct {
	ParentHash      common.Hash      `json:"parentHash"`
	Sha3Uncles      common.Hash      `json:"sha3Uncles"`
	Miner           common.Address   `json:"miner"`
	Signers         []common.Address `json:"signers,omitempty"`
	Voters          []common.Address `json:"voters,omitempty"`
	Signer          hexutil.Bytes    `json:"signer,omitempty"`
	StateRoot       common.Hash      `json:"stateRoot"`
	TxsRoot         common.Hash      `json:"transactionsRoot"`
	ReceiptsRoot    common.Hash      `json:"receiptsRoot"`
	LogsBloom       *types.Bloom     `json:"logsBloom"`
	Difficulty      hexutil.Big      `json:"difficulty"`
	TotalDifficulty hexutil.Big      `json:"totalDifficulty"`
	Number          hexutil.Big      `json:"number"`
	GasLimit        hexutil.Uint64   `json:"gasLimit"`
	GasUsed         hexutil.Uint64   `json:"gasUsed"`
	Timestamp       hexutil.Uint64   `json:"timestamp"`
	ExtraData       hexutil.Bytes    `json:"extraData"`
	MixHash         common.Hash      `json:"mixHash"`
	Nonce           types.BlockNonce `json:"nonce"`
	Hash            common.Hash      `json:"hash"`

	// TODO support full Transactions
	Txs    []common.Hash `json:"transactions,omitempty"`
	Uncles []common.Hash `json:"uncles"`
}

// copyTo copies the fields from r to b.
func (r *rpcBlock) copyTo(b *Block) {
	b.ParentHash = r.ParentHash
	b.Sha3Uncles = r.Sha3Uncles
	b.Miner = r.Miner
	b.Signers = r.Signers
	b.Voters = r.Voters
	b.Signer = r.Signer
	b.StateRoot = r.StateRoot
	b.TxsRoot = r.TxsRoot
	b.ReceiptsRoot = r.ReceiptsRoot
	b.LogsBloom = r.LogsBloom
	b.Difficulty = r.Difficulty.ToInt()
	b.TotalDifficulty = r.TotalDifficulty.ToInt()
	b.Number = r.Number.ToInt()
	b.GasLimit = uint64(r.GasLimit)
	b.GasUsed = uint64(r.GasUsed)
	b.Timestamp = time.Unix(int64(r.Timestamp), 0).UTC()
	b.ExtraData = r.ExtraData
	b.MixHash = r.MixHash
	b.Nonce = r.Nonce
	b.Hash = r.Hash
	b.Txs = r.Txs
	b.Uncles = r.Uncles
}

// copyFrom copies the fields from b to r.
func (r *rpcBlock) copyFrom(b *Block) {
	r.ParentHash = b.ParentHash
	r.Sha3Uncles = b.Sha3Uncles
	r.Miner = b.Miner
	r.Signers = b.Signers
	r.Voters = b.Voters
	r.Signer = b.Signer
	r.StateRoot = b.StateRoot
	r.TxsRoot = b.TxsRoot
	r.ReceiptsRoot = b.ReceiptsRoot
	r.LogsBloom = b.LogsBloom
	r.Difficulty = (hexutil.Big)(*b.Difficulty)
	r.TotalDifficulty = (hexutil.Big)(*b.TotalDifficulty)
	r.Number = (hexutil.Big)(*b.Number)
	r.GasLimit = hexutil.Uint64(b.GasLimit)
	r.GasUsed = hexutil.Uint64(b.GasUsed)
	r.Timestamp = hexutil.Uint64(b.Timestamp.Unix())
	r.ExtraData = b.ExtraData
	r.MixHash = b.MixHash
	r.Nonce = b.Nonce
	r.Hash = b.Hash
	r.Txs = b.Txs
	r.Uncles = b.Uncles
}

type rpcTransaction struct {
	Nonce            hexutil.Uint64 `json:"nonce"`
	GasPrice         hexutil.Big    `json:"gasPrice"`
	GasLimit         hexutil.Uint64 `json:"gas"`
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

// copyTo copies the fields from r to t.
func (r *rpcTransaction) copyTo(t *Transaction) {
	t.Nonce = uint64(r.Nonce)
	t.GasPrice = r.GasPrice.ToInt()
	t.GasLimit = uint64(r.GasLimit)
	t.To = r.To
	t.Value = r.Value.ToInt()
	t.Input = r.Input
	t.Hash = r.Hash
	t.BlockNumber = r.BlockNumber.ToInt()
	t.BlockHash = r.BlockHash
	t.From = r.From
	t.TransactionIndex = uint64(r.TransactionIndex)
	t.V = r.V.ToInt()
	t.R = r.R
	t.S = r.S
}

// copyFrom copies the fields from t to r.
func (r *rpcTransaction) copyFrom(t *Transaction) {
	r.Nonce = hexutil.Uint64(t.Nonce)
	r.GasPrice = hexutil.Big(*t.GasPrice)
	r.GasLimit = hexutil.Uint64(t.GasLimit)
	r.To = t.To
	r.Value = hexutil.Big(*t.Value)
	r.Input = t.Input
	r.Hash = t.Hash
	r.BlockNumber = hexutil.Big(*t.BlockNumber)
	r.BlockHash = t.BlockHash
	r.From = t.From
	r.TransactionIndex = hexutil.Uint64(t.TransactionIndex)
	r.V = hexutil.Big(*t.V)
	r.R = t.R
	r.S = t.S
}
