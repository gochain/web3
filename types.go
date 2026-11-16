package web3

import (
	"encoding/json"
	"math/big"
	"time"
)

const (
	TxTypeLegacy     uint8 = 0x00
	TxTypeAccessList uint8 = 0x01
	TxTypeDynamicFee uint8 = 0x02

	ReceiptStatusFailed     uint64 = 0
	ReceiptStatusSuccessful uint64 = 1
)

var (
	EmptyRootHash  = HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	EmptyUncleHash = HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347")
)

type CallMsg struct {
	From                 *Address
	To                   *Address
	Gas                  uint64
	GasPrice             *big.Int
	MaxFeePerGas         *big.Int
	MaxPriorityFeePerGas *big.Int
	Value                *big.Int
	Data                 []byte
}

type AccessTuple struct {
	Address     Address `json:"address"`
	StorageKeys []Hash  `json:"storageKeys"`
}

type FilterQuery struct {
	BlockHash *Hash
	FromBlock *big.Int // nil = latest
	ToBlock   *big.Int // nil = latest
	Addresses []Address
	Topics    [][]Hash
}

type Log struct {
	Address     Address
	Topics      []Hash
	Data        []byte
	BlockNumber uint64
	TxHash      Hash
	TxIndex     uint
	BlockHash   Hash
	Index       uint
	Removed     bool
}

func (l *Log) UnmarshalJSON(data []byte) error {
	var rl rpcLog
	if err := json.Unmarshal(data, &rl); err != nil {
		return err
	}
	return rl.copyTo(l)
}

func (l Log) MarshalJSON() ([]byte, error) {
	var rl rpcLog
	rl.copyFrom(&l)
	return json.Marshal(&rl)
}

type Snapshot struct {
	Number  uint64              `json:"number"`
	Hash    Hash                `json:"hash"`
	Signers map[Address]uint64  `json:"signers"`
	Voters  map[Address]struct{} `json:"voters"`
	Votes   []*Vote             `json:"votes"`
	Tally   map[Address]Tally   `json:"tally"`
}

type Vote struct {
	Signer    Address `json:"signer"`
	Block     uint64  `json:"block"`
	Address   Address `json:"address"`
	Authorize bool    `json:"authorize"`
}

type Tally struct {
	Authorize bool `json:"authorize"`
	Votes     int  `json:"votes"`
}

type ID struct {
	NetworkID   *big.Int `json:"network_id"`
	ChainID     *big.Int `json:"chain_id"`
	GenesisHash Hash     `json:"genesis_hash"`
}

type Receipt struct {
	Type              uint8
	PostState         []byte
	Status            uint64
	CumulativeGasUsed uint64
	EffectiveGasPrice *big.Int
	Bloom             Bloom
	Logs              []*Log
	TxHash            Hash
	TxIndex           uint64
	ContractAddress   Address
	GasUsed           uint64
	ParsedLogs        []Event
	BlockHash         Hash
	BlockNumber       uint64
	From              Address
	To                *Address
}

func (r *Receipt) UnmarshalJSON(data []byte) error {
	var rr rpcReceipt
	if err := json.Unmarshal(data, &rr); err != nil {
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
	ParentHash      Hash
	Sha3Uncles      Hash
	Miner           Address
	Signers         []Address
	Voters          []Address
	Signer          []byte
	StateRoot       Hash
	TxsRoot         Hash
	ReceiptsRoot    Hash
	LogsBloom       *Bloom
	Difficulty      *big.Int
	TotalDifficulty *big.Int
	Number          *big.Int
	GasLimit        uint64
	GasUsed         uint64
	BaseFeePerGas   *big.Int
	Timestamp       time.Time
	ExtraData       []byte
	MixHash         Hash
	Nonce           BlockNonce
	Hash            Hash

	// Only one of TxHashes or TxDetails will be populated.
	TxHashes  []Hash
	TxDetails []*Transaction

	Uncles []Hash
}

func (b *Block) UnmarshalJSON(data []byte) error {
	var r rpcBlock
	if err := json.Unmarshal(data, &r); err != nil {
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
	Type                 uint8
	ChainID              *big.Int
	Nonce                uint64
	GasPrice             *big.Int // wei (Legacy / EIP-2930, or effective gasPrice returned by RPC)
	MaxPriorityFeePerGas *big.Int // wei (EIP-1559)
	MaxFeePerGas         *big.Int // wei (EIP-1559)
	GasLimit             uint64
	To                   *Address
	Value                *big.Int // wei
	Input                []byte
	AccessList           []AccessTuple
	From                 Address
	V                    *big.Int
	R                    *big.Int
	S                    *big.Int
	Hash                 Hash

	BlockNumber      *big.Int
	BlockHash        Hash
	TransactionIndex uint64
}

type Event struct {
	Name   string                 `json:"name"`
	Fields map[string]interface{} `json:"fields"`
}

func (t *Transaction) UnmarshalJSON(data []byte) error {
	var r rpcTransaction
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	return r.copyTo(t)
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	var r rpcTransaction
	r.copyFrom(t)
	return json.Marshal(&r)
}
