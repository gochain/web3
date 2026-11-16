package web3

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type rpcBlock struct {
	ParentHash      *Hash       `json:"parentHash"`
	Sha3Uncles      *Hash       `json:"sha3Uncles"`
	Miner           *Address    `json:"miner"`
	Signers         []Address   `json:"signers,omitempty"`
	Voters          []Address   `json:"voters,omitempty"`
	Signer          *HexBytes   `json:"signer,omitempty"`
	StateRoot       *Hash       `json:"stateRoot"`
	TxsRoot         *Hash       `json:"transactionsRoot"`
	ReceiptsRoot    *Hash       `json:"receiptsRoot"`
	LogsBloom       *Bloom      `json:"logsBloom"`
	Difficulty      *HexBig     `json:"difficulty,omitempty"`
	TotalDifficulty *HexBig     `json:"totalDifficulty,omitempty"`
	Number          *HexBig     `json:"number"`
	GasLimit        *HexUint64  `json:"gasLimit"`
	GasUsed         *HexUint64  `json:"gasUsed"`
	BaseFeePerGas   *HexBig     `json:"baseFeePerGas,omitempty"`
	Timestamp       *HexUint64  `json:"timestamp"`
	ExtraData       *HexBytes   `json:"extraData"`
	MixHash         *Hash       `json:"mixHash,omitempty"`
	Nonce           *BlockNonce `json:"nonce,omitempty"`
	Hash            *Hash       `json:"hash"`
	Txs             json.RawMessage `json:"transactions,omitempty"`
	Uncles          []Hash      `json:"uncles"`
}

func (r *rpcBlock) copyTo(b *Block) error {
	if r.ParentHash == nil {
		return errors.New("missing 'parentHash'")
	}
	b.ParentHash = *r.ParentHash
	if r.Sha3Uncles != nil {
		b.Sha3Uncles = *r.Sha3Uncles
	}
	if r.Miner != nil {
		b.Miner = *r.Miner
	}
	b.Signers = r.Signers
	b.Voters = r.Voters
	if r.Signer != nil {
		b.Signer = *r.Signer
	}
	if r.StateRoot != nil {
		b.StateRoot = *r.StateRoot
	}
	if r.TxsRoot != nil {
		b.TxsRoot = *r.TxsRoot
	}
	if r.ReceiptsRoot != nil {
		b.ReceiptsRoot = *r.ReceiptsRoot
	}
	if r.LogsBloom != nil {
		b.LogsBloom = r.LogsBloom
	} else {
		b.LogsBloom = &Bloom{}
	}
	if r.Difficulty != nil {
		b.Difficulty = r.Difficulty.ToInt()
	}
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
	if r.BaseFeePerGas != nil {
		b.BaseFeePerGas = r.BaseFeePerGas.ToInt()
	}
	if r.Timestamp == nil {
		return errors.New("missing 'timestamp'")
	}
	b.Timestamp = time.Unix(int64(*r.Timestamp), 0).UTC()
	if r.ExtraData != nil {
		b.ExtraData = *r.ExtraData
	}
	if r.MixHash != nil {
		b.MixHash = *r.MixHash
	}
	if r.Nonce != nil {
		b.Nonce = *r.Nonce
	}
	if r.Hash != nil {
		b.Hash = *r.Hash
	}

	if len(r.Txs) > 0 && string(r.Txs) != "null" {
		var hashes []Hash
		if err := json.Unmarshal(r.Txs, &hashes); err == nil {
			b.TxHashes = hashes
		} else {
			var details []*Transaction
			if err := json.Unmarshal(r.Txs, &details); err != nil {
				return fmt.Errorf("failed to unmarshal transactions as either hashes or details %q: %s", err, string(r.Txs))
			}
			b.TxDetails = details
		}
	}

	b.Uncles = r.Uncles
	return nil
}

func (r *rpcBlock) copyFrom(b *Block) error {
	r.ParentHash = &b.ParentHash
	r.Sha3Uncles = &b.Sha3Uncles
	r.Miner = &b.Miner
	r.Signers = b.Signers
	r.Voters = b.Voters
	if len(b.Signer) > 0 {
		s := HexBytes(b.Signer)
		r.Signer = &s
	}
	r.StateRoot = &b.StateRoot
	r.TxsRoot = &b.TxsRoot
	r.ReceiptsRoot = &b.ReceiptsRoot
	r.LogsBloom = b.LogsBloom
	r.Difficulty = (*HexBig)(b.Difficulty)
	r.TotalDifficulty = (*HexBig)(b.TotalDifficulty)
	r.Number = (*HexBig)(b.Number)
	r.GasLimit = (*HexUint64)(&b.GasLimit)
	r.GasUsed = (*HexUint64)(&b.GasUsed)
	r.BaseFeePerGas = (*HexBig)(b.BaseFeePerGas)
	t := uint64(b.Timestamp.Unix())
	r.Timestamp = (*HexUint64)(&t)
	extra := HexBytes(b.ExtraData)
	r.ExtraData = &extra
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

type rpcTransaction struct {
	Type                 *HexUint64    `json:"type,omitempty"`
	ChainID              *HexBig       `json:"chainId,omitempty"`
	Nonce                *HexUint64    `json:"nonce"`
	GasPrice             *HexBig       `json:"gasPrice,omitempty"`
	MaxPriorityFeePerGas *HexBig       `json:"maxPriorityFeePerGas,omitempty"`
	MaxFeePerGas         *HexBig       `json:"maxFeePerGas,omitempty"`
	GasLimit             *HexUint64    `json:"gas"`
	To                   *Address      `json:"to"`
	Value                *HexBig       `json:"value"`
	Input                *HexBytes     `json:"input"`
	Data                 *HexBytes     `json:"data,omitempty"`
	AccessList           []AccessTuple `json:"accessList,omitempty"`
	From                 *Address      `json:"from"`
	V                    *HexBig       `json:"v"`
	YParity              *HexBig       `json:"yParity,omitempty"`
	R                    *HexBig       `json:"r"`
	S                    *HexBig       `json:"s"`
	Hash                 *Hash         `json:"hash"`

	BlockNumber      *HexBig    `json:"blockNumber,omitempty"`
	BlockHash        *Hash      `json:"blockHash,omitempty"`
	TransactionIndex *HexUint64 `json:"transactionIndex,omitempty"`
}

func (r *rpcTransaction) copyTo(t *Transaction) error {
	if r.Type != nil {
		t.Type = uint8(*r.Type)
	}
	if r.ChainID != nil {
		t.ChainID = r.ChainID.ToInt()
	}
	if r.Nonce == nil {
		return errors.New("missing 'nonce'")
	}
	t.Nonce = uint64(*r.Nonce)
	if r.GasPrice != nil {
		t.GasPrice = r.GasPrice.ToInt()
	}
	if r.MaxPriorityFeePerGas != nil {
		t.MaxPriorityFeePerGas = r.MaxPriorityFeePerGas.ToInt()
	}
	if r.MaxFeePerGas != nil {
		t.MaxFeePerGas = r.MaxFeePerGas.ToInt()
		if t.GasPrice == nil {
			t.GasPrice = r.MaxFeePerGas.ToInt()
		}
	}
	if r.GasLimit == nil {
		return errors.New("missing 'gas'")
	}
	t.GasLimit = uint64(*r.GasLimit)
	if r.To != nil {
		t.To = r.To
	}
	if r.Value != nil {
		t.Value = r.Value.ToInt()
	}
	if r.Input != nil {
		t.Input = *r.Input
	} else if r.Data != nil {
		t.Input = *r.Data
	}
	t.AccessList = r.AccessList
	if r.V != nil {
		t.V = r.V.ToInt()
	} else if r.YParity != nil {
		t.V = r.YParity.ToInt()
	}
	if r.R != nil {
		t.R = r.R.ToInt()
	}
	if r.S != nil {
		t.S = r.S.ToInt()
	}
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

func (r *rpcTransaction) copyFrom(t *Transaction) {
	typ := HexUint64(t.Type)
	r.Type = &typ
	r.ChainID = (*HexBig)(t.ChainID)
	r.Nonce = (*HexUint64)(&t.Nonce)
	r.GasPrice = (*HexBig)(t.GasPrice)
	r.MaxPriorityFeePerGas = (*HexBig)(t.MaxPriorityFeePerGas)
	r.MaxFeePerGas = (*HexBig)(t.MaxFeePerGas)
	r.GasLimit = (*HexUint64)(&t.GasLimit)
	r.To = t.To
	r.Value = (*HexBig)(t.Value)
	in := HexBytes(t.Input)
	r.Input = &in
	r.AccessList = t.AccessList
	r.Hash = &t.Hash
	r.BlockNumber = (*HexBig)(t.BlockNumber)
	if !t.BlockHash.IsZero() {
		r.BlockHash = &t.BlockHash
	}
	r.From = &t.From
	r.TransactionIndex = (*HexUint64)(&t.TransactionIndex)
	r.V = (*HexBig)(t.V)
	r.R = (*HexBig)(t.R)
	r.S = (*HexBig)(t.S)
}

type rpcReceipt struct {
	Type              *HexUint64 `json:"type,omitempty"`
	PostState         *HexBytes  `json:"root,omitempty"`
	Status            *HexUint64 `json:"status"`
	CumulativeGasUsed *HexUint64 `json:"cumulativeGasUsed"`
	EffectiveGasPrice *HexBig    `json:"effectiveGasPrice,omitempty"`
	Bloom             *Bloom     `json:"logsBloom,omitempty"`
	Logs              []*Log     `json:"logs"`
	TxHash            *Hash      `json:"transactionHash"`
	TxIndex           *HexUint64 `json:"transactionIndex"`
	ContractAddress   *Address   `json:"contractAddress,omitempty"`
	GasUsed           *HexUint64 `json:"gasUsed"`
	ParsedLogs        *[]Event   `json:"parsedLogs,omitempty"`
	BlockHash         *Hash      `json:"blockHash"`
	BlockNumber       *HexUint64 `json:"blockNumber"`
	From              *Address   `json:"from"`
	To                *Address   `json:"to"`
}

func (rr *rpcReceipt) copyTo(r *Receipt) error {
	if rr.Type != nil {
		r.Type = uint8(*rr.Type)
	}
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
	if rr.EffectiveGasPrice != nil {
		r.EffectiveGasPrice = rr.EffectiveGasPrice.ToInt()
	}
	if rr.Bloom != nil {
		r.Bloom = *rr.Bloom
	}
	if rr.Logs == nil {
		r.Logs = []*Log{}
	} else {
		r.Logs = rr.Logs
	}
	if rr.TxHash == nil {
		return errors.New("missing 'transactionHash'")
	}
	r.TxHash = *rr.TxHash
	if rr.TxIndex != nil {
		r.TxIndex = uint64(*rr.TxIndex)
	}
	if rr.ContractAddress != nil {
		r.ContractAddress = *rr.ContractAddress
	}
	if rr.GasUsed == nil {
		return errors.New("missing 'gasUsed'")
	}
	r.GasUsed = uint64(*rr.GasUsed)
	if rr.BlockHash != nil {
		r.BlockHash = *rr.BlockHash
	}
	if rr.BlockNumber != nil {
		r.BlockNumber = uint64(*rr.BlockNumber)
	}
	if rr.From != nil {
		r.From = *rr.From
	}
	if rr.To != nil {
		r.To = rr.To
	}
	return nil
}

func (rr *rpcReceipt) copyFrom(r *Receipt) {
	typ := HexUint64(r.Type)
	rr.Type = &typ
	if len(r.PostState) > 0 {
		ps := HexBytes(r.PostState)
		rr.PostState = &ps
	}
	rr.Status = (*HexUint64)(&r.Status)
	rr.CumulativeGasUsed = (*HexUint64)(&r.CumulativeGasUsed)
	rr.EffectiveGasPrice = (*HexBig)(r.EffectiveGasPrice)
	rr.Bloom = &r.Bloom
	rr.Logs = r.Logs
	rr.TxHash = &r.TxHash
	rr.TxIndex = (*HexUint64)(&r.TxIndex)
	if !r.ContractAddress.IsZero() {
		rr.ContractAddress = &r.ContractAddress
	}
	rr.GasUsed = (*HexUint64)(&r.GasUsed)
	if len(r.ParsedLogs) > 0 {
		rr.ParsedLogs = &r.ParsedLogs
	}
	rr.BlockHash = &r.BlockHash
	rr.BlockNumber = (*HexUint64)(&r.BlockNumber)
	rr.From = &r.From
	rr.To = r.To
}

type rpcLog struct {
	Address     *Address   `json:"address"`
	Topics      []Hash     `json:"topics"`
	Data        *HexBytes  `json:"data"`
	BlockNumber *HexUint64 `json:"blockNumber"`
	TxHash      *Hash      `json:"transactionHash"`
	TxIndex     *HexUint64 `json:"transactionIndex"`
	BlockHash   *Hash      `json:"blockHash"`
	Index       *HexUint64 `json:"logIndex"`
	Removed     bool       `json:"removed"`
}

func (rl *rpcLog) copyTo(l *Log) error {
	if rl.Address == nil {
		return errors.New("missing log 'address'")
	}
	l.Address = *rl.Address
	l.Topics = rl.Topics
	if rl.Data != nil {
		l.Data = *rl.Data
	}
	if rl.BlockNumber != nil {
		l.BlockNumber = uint64(*rl.BlockNumber)
	}
	if rl.TxHash != nil {
		l.TxHash = *rl.TxHash
	}
	if rl.TxIndex != nil {
		l.TxIndex = uint(*rl.TxIndex)
	}
	if rl.BlockHash != nil {
		l.BlockHash = *rl.BlockHash
	}
	if rl.Index != nil {
		l.Index = uint(*rl.Index)
	}
	l.Removed = rl.Removed
	return nil
}

func (rl *rpcLog) copyFrom(l *Log) {
	rl.Address = &l.Address
	rl.Topics = l.Topics
	d := HexBytes(l.Data)
	rl.Data = &d
	bn := HexUint64(l.BlockNumber)
	rl.BlockNumber = &bn
	rl.TxHash = &l.TxHash
	ti := HexUint64(l.TxIndex)
	rl.TxIndex = &ti
	rl.BlockHash = &l.BlockHash
	idx := HexUint64(l.Index)
	rl.Index = &idx
	rl.Removed = l.Removed
}
