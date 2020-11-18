package web3

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/gochain/gochain/v3/common"
	"github.com/gochain/gochain/v3/core/types"
)

type CallMsg struct {
	From     *common.Address  // the sender of the 'transaction'
	To       *common.Address // the destination contract (nil for contract creation)
	Gas      uint64          // if 0, the call executes with near-infinite gas
	GasPrice *big.Int        // wei <-> gas exchange ratio
	Value    *big.Int        // amount of wei sent along with the call
	Data     []byte          // input data, usually an ABI-encoded contract method invocation
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

type ID struct {
	NetworkID   *big.Int    `json:"network_id"`
	ChainID     *big.Int    `json:"chain_id"`
	GenesisHash common.Hash `json:"genesis_hash"`
}

type Receipt struct {
	PostState         []byte
	Status            uint64
	CumulativeGasUsed uint64
	Bloom             types.Bloom
	Logs              []*types.Log
	TxHash            common.Hash
	TxIndex           uint64
	ContractAddress   common.Address
	GasUsed           uint64
	ParsedLogs        []Event
	BlockHash         common.Hash
	BlockNumber       uint64
	From              common.Address
	To                *common.Address
}

func (r *Receipt) UnmarshalJSON(data []byte) error {
	var rr rpcReceipt
	err := json.Unmarshal(data, &rr)
	if err != nil {
		return err
	}
	return rr.copyTo(r)
}

func (r *Receipt) MarshalJSON() ([]byte, error) {
	var rr rpcReceipt
	rr.copyFrom(r)
	return json.Marshal(&rr)
}

type Block struct {
	ParentHash      common.Hash
	Sha3Uncles      common.Hash
	Miner           common.Address
	Signers         []common.Address
	Voters          []common.Address
	Signer          []byte
	StateRoot       common.Hash
	TxsRoot         common.Hash
	ReceiptsRoot    common.Hash
	LogsBloom       *types.Bloom
	Difficulty      *big.Int
	TotalDifficulty *big.Int
	Number          *big.Int
	GasLimit        uint64
	GasUsed         uint64
	Timestamp       time.Time
	ExtraData       []byte
	MixHash         common.Hash
	Nonce           types.BlockNonce
	Hash            common.Hash

	// Only one of TxHashes or TxDetails will be populated.
	TxHashes  []common.Hash
	TxDetails []*Transaction

	Uncles []common.Hash
}

func (b *Block) UnmarshalJSON(data []byte) error {
	var r rpcBlock
	err := json.Unmarshal(data, &r)
	if err != nil {
		return err
	}
	return r.copyTo(b)
}

func (b *Block) MarshalJSON() ([]byte, error) {
	var r rpcBlock
	if err := r.copyFrom(b); err != nil {
		return nil, err
	}
	return json.Marshal(&r)
}

func (b *Block) ExtraVanity() string {
	l := len(b.ExtraData)
	if l > 32 {
		l = 32
	}
	return string(b.ExtraData[:l])
}

func (b *Block) TxCount() int {
	if b.TxHashes != nil {
		return len(b.TxHashes)
	}
	return len(b.TxDetails)
}

type Transaction struct {
	Nonce    uint64
	GasPrice *big.Int // wei
	GasLimit uint64
	To       *common.Address
	Value    *big.Int // wei
	Input    []byte
	From     common.Address
	V        *big.Int
	R        *big.Int
	S        *big.Int
	Hash     common.Hash

	BlockNumber      *big.Int
	BlockHash        common.Hash
	TransactionIndex uint64
}
type Event struct {
	Name   string                 `json:"name"`
	Fields map[string]interface{} `json:"fields"`
}

func (t *Transaction) UnmarshalJSON(data []byte) error {
	var r rpcTransaction
	err := json.Unmarshal(data, &r)
	if err != nil {
		return err
	}
	return r.copyTo(t)
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	var r rpcTransaction
	r.copyFrom(t)
	return json.Marshal(&r)
}
