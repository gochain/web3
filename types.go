package web3

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/gochain-io/gochain/v3/common"
	"github.com/gochain-io/gochain/v3/core/types"
)

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
	PostState         []byte         `json:"root"`
	Status            uint64         `json:"status"`
	CumulativeGasUsed uint64         `json:"cumulativeGasUsed"`
	Bloom             types.Bloom    `json:"logsBloom"`
	Logs              []*types.Log   `json:"logs"`
	TxHash            common.Hash    `json:"transactionHash"`
	ContractAddress   common.Address `json:"contractAddress"`
	GasUsed           uint64         `json:"gasUsed"`
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

	// TODO support full Transactions
	Txs    []common.Hash
	Uncles []common.Hash
}

func (b *Block) UnmarshalJSON(data []byte) error {
	var r rpcBlock
	err := json.Unmarshal(data, &r)
	if err != nil {
		return err
	}
	r.copyTo(b)
	return nil
}

func (b *Block) MarshalJSON() ([]byte, error) {
	var r rpcBlock
	r.copyFrom(b)
	return json.Marshal(&r)
}

func (b *Block) ExtraVanity() string {
	l := len(b.ExtraData)
	if l > 32 {
		l = 32
	}
	return string(b.ExtraData[:l])
}

type Transaction struct {
	Nonce            uint64
	GasPrice         *big.Int // wei
	GasLimit         uint64
	To               common.Address
	Value            *big.Int // wei
	Input            []byte
	Hash             common.Hash
	BlockNumber      *big.Int
	BlockHash        common.Hash
	From             common.Address
	TransactionIndex uint64
	V                *big.Int
	R                common.Hash
	S                common.Hash
}

func (t *Transaction) UnmarshalJSON(data []byte) error {
	var r rpcTransaction
	err := json.Unmarshal(data, &r)
	if err != nil {
		return err
	}
	r.copyFrom(t)
	return nil
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	var r rpcTransaction
	r.copyFrom(t)
	return json.Marshal(&r)
}
