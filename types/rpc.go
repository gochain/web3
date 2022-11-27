package web3_types

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/common/hexutil"
	"github.com/gochain/gochain/v4/core/types"
)

type RpcBlock struct {
	ParentHash      *common.Hash      `json:"parentHash"`
	Sha3Uncles      *common.Hash      `json:"sha3Uncles"`
	Miner           *common.Address   `json:"miner"`
	Signers         []common.Address  `json:"signers,omitempty"`
	Voters          []common.Address  `json:"voters,omitempty"`
	Signer          *hexutil.Bytes    `json:"signer,omitempty"`
	StateRoot       *common.Hash      `json:"stateRoot"`
	TxsRoot         *common.Hash      `json:"transactionsRoot"`
	ReceiptsRoot    *common.Hash      `json:"receiptsRoot"`
	LogsBloom       *types.Bloom      `json:"logsBloom"`
	Difficulty      *hexutil.Big      `json:"difficulty"`
	TotalDifficulty *hexutil.Big      `json:"totalDifficulty"`
	Number          *hexutil.Big      `json:"number"`
	GasLimit        *hexutil.Uint64   `json:"gasLimit"`
	GasUsed         *hexutil.Uint64   `json:"gasUsed"`
	Timestamp       *hexutil.Uint64   `json:"timestamp"`
	ExtraData       *hexutil.Bytes    `json:"extraData"`
	MixHash         *common.Hash      `json:"mixHash"`
	Nonce           *types.BlockNonce `json:"nonce"`
	Hash            *common.Hash      `json:"hash"`
	Txs             json.RawMessage   `json:"transactions,omitempty"`
	Uncles          []common.Hash     `json:"uncles"`
}

// CopyTo copies the fields from r to b.
func (r *RpcBlock) CopyTo(b *Block) error {
	if r.ParentHash == nil {
		return errors.New("missing 'parentHash'")
	}
	b.ParentHash = *r.ParentHash
	if r.Sha3Uncles == nil {
		return errors.New("missing 'sha3Uncles'")
	}
	b.Sha3Uncles = *r.Sha3Uncles
	if r.Miner == nil {
		return errors.New("missing 'miner'")
	}
	b.Miner = *r.Miner
	b.Signers = r.Signers
	b.Voters = r.Voters
	if r.Signer != nil {
		b.Signer = *r.Signer
	}
	if r.StateRoot == nil {
		return errors.New("missing 'stateRoot'")
	}
	b.StateRoot = *r.StateRoot
	if r.TxsRoot == nil {
		return errors.New("missing 'transactionsRoot'")
	}
	b.TxsRoot = *r.TxsRoot
	if r.ReceiptsRoot == nil {
		return errors.New("missing 'receiptsRoot'")
	}
	b.ReceiptsRoot = *r.ReceiptsRoot
	if r.LogsBloom == nil {
		return errors.New("missing 'logsBloom'")
	}
	b.LogsBloom = r.LogsBloom
	if r.Difficulty == nil {
		return errors.New("missing 'difficulty'")
	}
	b.Difficulty = r.Difficulty.ToInt()
	if r.TotalDifficulty != nil {
		b.TotalDifficulty = r.TotalDifficulty.ToInt()
	}
	if r.Number == nil {
		return errors.New("missing 'number'")
	}
	b.Number = r.Number.ToInt()
	if r.GasLimit == nil {
		return errors.New("missing 'gasLimit'")
	}
	b.GasLimit = uint64(*r.GasLimit)
	if r.GasUsed == nil {
		return errors.New("missing 'gasUsed'")
	}
	b.GasUsed = uint64(*r.GasUsed)
	if r.Timestamp == nil {
		return errors.New("missing 'timestamp'")
	}
	b.Timestamp = time.Unix(int64(*r.Timestamp), 0).UTC()
	if r.ExtraData == nil {
		return errors.New("missing 'extraData")
	}
	b.ExtraData = *r.ExtraData
	if r.MixHash == nil {
		return errors.New("missing 'mixHash'")
	}
	b.MixHash = *r.MixHash
	if r.Nonce == nil {
		return errors.New("missing 'nonce'")
	}
	b.Nonce = *r.Nonce
	if r.Hash == nil {
		return errors.New("missing 'hash'")
	}
	b.Hash = *r.Hash

	// Try tx hashes first.
	var hashes []common.Hash
	if err := json.Unmarshal(r.Txs, &hashes); err == nil {
		b.TxHashes = hashes
	} else {
		// Try full transactions.
		var details []*Transaction
		if err := json.Unmarshal(r.Txs, &details); err != nil {
			return fmt.Errorf("failed to unmarshal transactions as either hahes or details %q: %s", err, string(r.Txs))
		}
		b.TxDetails = details
	}

	b.Uncles = r.Uncles
	return nil
}

// CopyFrom copies the fields from b to r.
func (r *RpcBlock) CopyFrom(b *Block) error {
	r.ParentHash = &b.ParentHash
	r.Sha3Uncles = &b.Sha3Uncles
	r.Miner = &b.Miner
	r.Signers = b.Signers
	r.Voters = b.Voters
	r.Signer = (*hexutil.Bytes)(&b.Signer)
	r.StateRoot = &b.StateRoot
	r.TxsRoot = &b.TxsRoot
	r.ReceiptsRoot = &b.ReceiptsRoot
	r.LogsBloom = b.LogsBloom
	r.Difficulty = (*hexutil.Big)(b.Difficulty)
	r.TotalDifficulty = (*hexutil.Big)(b.TotalDifficulty)
	r.Number = (*hexutil.Big)(b.Number)
	r.GasLimit = (*hexutil.Uint64)(&b.GasLimit)
	r.GasUsed = (*hexutil.Uint64)(&b.GasUsed)
	t := uint64(b.Timestamp.Unix())
	r.Timestamp = (*hexutil.Uint64)(&t)
	r.ExtraData = (*hexutil.Bytes)(&b.ExtraData)
	r.MixHash = &b.MixHash
	r.Nonce = &b.Nonce
	r.Hash = &b.Hash
	if b.TxHashes != nil {
		data, err := json.Marshal(b.TxHashes)
		if err != nil {
			return fmt.Errorf("failed to marshal tx hashes to json: %v", err)
		}
		r.Txs = data
	} else {
		data, err := json.Marshal(b.TxDetails)
		if err != nil {
			return fmt.Errorf("failed to marshal tx details to json: %v", err)
		}
		r.Txs = data
	}
	r.Uncles = b.Uncles
	return nil
}

type RpcTransaction struct {
	Nonce    *hexutil.Uint64 `json:"nonce"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	GasLimit *hexutil.Uint64 `json:"gas"`
	To       *common.Address `json:"to"`
	Value    *hexutil.Big    `json:"value"`
	Input    *hexutil.Bytes  `json:"input"`
	From     *common.Address `json:"from"`
	V        *hexutil.Big    `json:"v"`
	R        *hexutil.Big    `json:"r"`
	S        *hexutil.Big    `json:"s"`
	Hash     *common.Hash    `json:"hash"`

	BlockNumber      *hexutil.Big    `json:"blockNumber,omitempty"`
	BlockHash        *common.Hash    `json:"blockHash,omitempty"`
	TransactionIndex *hexutil.Uint64 `json:"transactionIndex,omitempty"`
}

// CopyTo copies the fields from r to t.
func (r *RpcTransaction) CopyTo(t *Transaction) error {
	if r.Nonce == nil {
		return errors.New("missing 'nonce'")
	}
	t.Nonce = uint64(*r.Nonce)
	if r.GasPrice == nil {
		return errors.New("missing 'gasPrice'")
	}
	t.GasPrice = r.GasPrice.ToInt()
	if r.GasLimit == nil {
		return errors.New("missing 'gas'")
	}
	t.GasLimit = uint64(*r.GasLimit)
	if r.To != nil {
		t.To = r.To
	}
	if r.Value == nil {
		return errors.New("missing 'value'")
	}
	t.Value = r.Value.ToInt()
	if r.Input != nil {
		t.Input = *r.Input
	}
	if r.V == nil {
		return errors.New("missing 'v'")
	}
	t.V = r.V.ToInt()
	if r.R == nil {
		return errors.New("missing 'r'")
	}
	t.R = r.R.ToInt()
	if r.S == nil {
		return errors.New("missing 's'")
	}
	t.S = r.S.ToInt()
	if r.Hash != nil {
		t.Hash = *r.Hash
	}

	if r.BlockNumber != nil {
		t.BlockNumber = r.BlockNumber.ToInt()
	}
	if r.BlockHash != nil {
		t.BlockHash = *r.BlockHash
	}
	if r.From != nil {
		t.From = *r.From
	}
	if r.TransactionIndex != nil {
		t.TransactionIndex = uint64(*r.TransactionIndex)
	}
	return nil
}

// CopyFrom copies the fields from t to r.
func (r *RpcTransaction) CopyFrom(t *Transaction) {
	r.Nonce = (*hexutil.Uint64)(&t.Nonce)
	r.GasPrice = (*hexutil.Big)(t.GasPrice)
	r.GasLimit = (*hexutil.Uint64)(&t.GasLimit)
	r.To = t.To
	r.Value = (*hexutil.Big)(t.Value)
	r.Input = (*hexutil.Bytes)(&t.Input)
	r.Hash = &t.Hash
	r.BlockNumber = (*hexutil.Big)(t.BlockNumber)
	r.BlockHash = &t.BlockHash
	r.From = &t.From
	r.TransactionIndex = (*hexutil.Uint64)(&t.TransactionIndex)
	r.V = (*hexutil.Big)(t.V)
	r.R = (*hexutil.Big)(t.R)
	r.S = (*hexutil.Big)(t.S)
}

type RpcReceipt struct {
	PostState         *hexutil.Bytes  `json:"root"`
	Status            *hexutil.Uint64 `json:"status"`
	CumulativeGasUsed *hexutil.Uint64 `json:"cumulativeGasUsed"`
	Bloom             *types.Bloom    `json:"logsBloom"`
	Logs              []*types.Log    `json:"logs"`
	TxHash            *common.Hash    `json:"transactionHash"`
	TxIndex           *hexutil.Uint64 `json:"transactionIndex"`
	ContractAddress   *common.Address `json:"contractAddress"`
	GasUsed           *hexutil.Uint64 `json:"gasUsed"`
	ParsedLogs        *[]Event        `json:"parsedLogs"`
	BlockHash         *common.Hash    `json:"blockHash"`
	BlockNumber       *hexutil.Uint64 `json:"blockNumber"`
	From              *common.Address `json:"from"`
	To                *common.Address `json:"to"`
}

func (rr *RpcReceipt) CopyTo(r *Receipt) error {
	if rr.PostState != nil {
		r.PostState = *rr.PostState
	}
	if rr.Status != nil {
		r.Status = uint64(*rr.Status)
	}
	if rr.CumulativeGasUsed == nil {
		return errors.New("missing 'cumulativeGasUsed'")
	}
	r.CumulativeGasUsed = uint64(*rr.CumulativeGasUsed)
	r.Bloom = *rr.Bloom
	if rr.Logs == nil {
		return errors.New("missing 'logs'")
	}
	r.Logs = rr.Logs
	if rr.TxHash == nil {
		return errors.New("missing 'transactionHash'")
	}
	r.TxHash = *rr.TxHash
	if rr.TxIndex == nil {
		return errors.New("missing 'transactionIndex'")
	}
	r.TxIndex = uint64(*rr.TxIndex)
	if rr.ContractAddress != nil {
		r.ContractAddress = *rr.ContractAddress
	}
	if rr.GasUsed == nil {
		return errors.New("missing 'gasUsed'")
	}
	r.GasUsed = uint64(*rr.GasUsed)
	if rr.BlockHash == nil {
		return errors.New("missing 'blockHash'")
	}
	r.BlockHash = *rr.BlockHash
	if rr.BlockNumber == nil {
		return errors.New("missing 'blockNumber'")
	}
	r.BlockNumber = uint64(*rr.BlockNumber)
	if rr.From == nil {
		return errors.New("missing 'from'")
	}
	r.From = *rr.From
	if rr.To != nil {
		r.To = rr.To
	}
	return nil
}

func (rr *RpcReceipt) CopyFrom(r *Receipt) {
	rr.PostState = (*hexutil.Bytes)(&r.PostState)
	rr.Status = (*hexutil.Uint64)(&r.Status)
	rr.CumulativeGasUsed = (*hexutil.Uint64)(&r.CumulativeGasUsed)
	rr.Bloom = &r.Bloom
	rr.Logs = r.Logs
	rr.TxHash = &r.TxHash
	rr.TxIndex = (*hexutil.Uint64)(&r.TxIndex)
	rr.ContractAddress = &r.ContractAddress
	rr.GasUsed = (*hexutil.Uint64)(&r.GasUsed)
	rr.ParsedLogs = &r.ParsedLogs
	rr.BlockHash = &r.BlockHash
	rr.BlockNumber = (*hexutil.Uint64)(&r.BlockNumber)
	rr.From = &r.From
	rr.To = r.To
}
