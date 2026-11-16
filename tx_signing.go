package web3

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
)

// SignTransaction signs tx with prv for the given chainID, sets From/ChainID/V/R/S/Hash on tx,
// and returns the raw EIP-2718 or RLP transaction wire bytes ready for eth_sendRawTransaction.
func SignTransaction(tx *Transaction, chainID *big.Int, prv *ecdsa.PrivateKey) ([]byte, error) {
	if tx == nil {
		return nil, errors.New("nil transaction")
	}
	if prv == nil {
		return nil, errors.New("nil private key")
	}
	if chainID == nil || chainID.Sign() <= 0 {
		return nil, errors.New("positive chainID is required for transaction signing")
	}

	tx.ChainID = new(big.Int).Set(chainID)
	tx.From = PubkeyToAddress(prv.PublicKey)
	if tx.Value == nil {
		tx.Value = big.NewInt(0)
	}

	sigHash, err := TxSigningHash(tx, chainID)
	if err != nil {
		return nil, err
	}
	sig, err := SignHash(sigHash[:], prv)
	if err != nil {
		return nil, err
	}

	r := new(big.Int).SetBytes(sig[0:32])
	s := new(big.Int).SetBytes(sig[32:64])
	recID := int64(sig[64])

	tx.R = r
	tx.S = s
	switch tx.Type {
	case TxTypeLegacy:
		// EIP-155: v = recID + 35 + 2 * chainID
		v := new(big.Int).Mul(chainID, big.NewInt(2))
		v.Add(v, big.NewInt(35+recID))
		tx.V = v
	case TxTypeAccessList, TxTypeDynamicFee:
		tx.V = big.NewInt(recID)
	default:
		return nil, fmt.Errorf("unsupported transaction type %d", tx.Type)
	}

	raw, err := EncodeRawTransaction(tx)
	if err != nil {
		return nil, err
	}
	tx.Hash = Keccak256Hash(raw)
	return raw, nil
}

// TxSigningHash computes the digest hash that is signed for tx.
func TxSigningHash(tx *Transaction, chainID *big.Int) (Hash, error) {
	val := tx.Value
	if val == nil {
		val = big.NewInt(0)
	}
	switch tx.Type {
	case TxTypeLegacy:
		if tx.GasPrice == nil {
			return ZeroHash, errors.New("legacy transaction missing gasPrice")
		}
		var payload []interface{}
		if chainID != nil && chainID.Sign() > 0 {
			payload = []interface{}{
				tx.Nonce,
				tx.GasPrice,
				tx.GasLimit,
				tx.To,
				val,
				tx.Input,
				chainID,
				uint64(0),
				uint64(0),
			}
		} else {
			payload = []interface{}{
				tx.Nonce,
				tx.GasPrice,
				tx.GasLimit,
				tx.To,
				val,
				tx.Input,
			}
		}
		enc, err := RLPEncode(payload)
		if err != nil {
			return ZeroHash, err
		}
		return Keccak256Hash(enc), nil

	case TxTypeAccessList:
		if tx.GasPrice == nil {
			return ZeroHash, errors.New("eip-2930 transaction missing gasPrice")
		}
		enc, err := RLPEncode([]interface{}{
			chainID,
			tx.Nonce,
			tx.GasPrice,
			tx.GasLimit,
			tx.To,
			val,
			tx.Input,
			encodeAccessListRLP(tx.AccessList),
		})
		if err != nil {
			return ZeroHash, err
		}
		return Keccak256Hash([]byte{TxTypeAccessList}, enc), nil

	case TxTypeDynamicFee:
		if tx.MaxFeePerGas == nil || tx.MaxPriorityFeePerGas == nil {
			return ZeroHash, errors.New("eip-1559 transaction missing maxFeePerGas or maxPriorityFeePerGas")
		}
		enc, err := RLPEncode([]interface{}{
			chainID,
			tx.Nonce,
			tx.MaxPriorityFeePerGas,
			tx.MaxFeePerGas,
			tx.GasLimit,
			tx.To,
			val,
			tx.Input,
			encodeAccessListRLP(tx.AccessList),
		})
		if err != nil {
			return ZeroHash, err
		}
		return Keccak256Hash([]byte{TxTypeDynamicFee}, enc), nil

	default:
		return ZeroHash, fmt.Errorf("unsupported transaction type %d", tx.Type)
	}
}

// EncodeRawTransaction encodes a signed Transaction into raw wire bytes.
func EncodeRawTransaction(tx *Transaction) ([]byte, error) {
	if tx == nil {
		return nil, errors.New("nil transaction")
	}
	if tx.V == nil || tx.R == nil || tx.S == nil {
		return nil, errors.New("transaction is not signed")
	}
	val := tx.Value
	if val == nil {
		val = big.NewInt(0)
	}

	switch tx.Type {
	case TxTypeLegacy:
		return RLPEncode([]interface{}{
			tx.Nonce,
			tx.GasPrice,
			tx.GasLimit,
			tx.To,
			val,
			tx.Input,
			tx.V,
			tx.R,
			tx.S,
		})

	case TxTypeAccessList:
		enc, err := RLPEncode([]interface{}{
			tx.ChainID,
			tx.Nonce,
			tx.GasPrice,
			tx.GasLimit,
			tx.To,
			val,
			tx.Input,
			encodeAccessListRLP(tx.AccessList),
			tx.V,
			tx.R,
			tx.S,
		})
		if err != nil {
			return nil, err
		}
		return append([]byte{TxTypeAccessList}, enc...), nil

	case TxTypeDynamicFee:
		enc, err := RLPEncode([]interface{}{
			tx.ChainID,
			tx.Nonce,
			tx.MaxPriorityFeePerGas,
			tx.MaxFeePerGas,
			tx.GasLimit,
			tx.To,
			val,
			tx.Input,
			encodeAccessListRLP(tx.AccessList),
			tx.V,
			tx.R,
			tx.S,
		})
		if err != nil {
			return nil, err
		}
		return append([]byte{TxTypeDynamicFee}, enc...), nil

	default:
		return nil, fmt.Errorf("unsupported transaction type %d", tx.Type)
	}
}

// RecoverTransactionSender recovers the sender Address from a signed Transaction.
func RecoverTransactionSender(tx *Transaction) (Address, error) {
	if tx == nil || tx.R == nil || tx.S == nil || tx.V == nil {
		return ZeroAddress, errors.New("unsigned transaction")
	}
	var chainID *big.Int
	var recID byte

	switch tx.Type {
	case TxTypeLegacy:
		v := tx.V
		if v.Cmp(big.NewInt(27)) == 0 || v.Cmp(big.NewInt(28)) == 0 {
			recID = byte(v.Int64() - 27)
		} else if v.Cmp(big.NewInt(35)) >= 0 {
			// v = 2*chainId + 35 + recId
			tmp := new(big.Int).Sub(v, big.NewInt(35))
			recID = byte(new(big.Int).And(tmp, big.NewInt(1)).Int64())
			chainID = new(big.Int).Rsh(tmp, 1)
		} else {
			return ZeroAddress, fmt.Errorf("invalid legacy signature V %s", v.String())
		}
	case TxTypeAccessList, TxTypeDynamicFee:
		chainID = tx.ChainID
		if !tx.V.IsUint64() || tx.V.Uint64() > 1 {
			return ZeroAddress, fmt.Errorf("invalid typed signature yParity %s", tx.V.String())
		}
		recID = byte(tx.V.Uint64())
	default:
		return ZeroAddress, fmt.Errorf("unsupported transaction type %d", tx.Type)
	}

	sigHash, err := TxSigningHash(tx, chainID)
	if err != nil {
		return ZeroAddress, err
	}
	var sig [65]byte
	rBytes := tx.R.Bytes()
	sBytes := tx.S.Bytes()
	if len(rBytes) > 32 || len(sBytes) > 32 {
		return ZeroAddress, errors.New("signature R/S exceeds 32 bytes")
	}
	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):64], sBytes)
	sig[64] = recID
	return RecoverAddress(sigHash[:], sig[:])
}

func encodeAccessListRLP(list []AccessTuple) []interface{} {
	if len(list) == 0 {
		return []interface{}{}
	}
	out := make([]interface{}, len(list))
	for i, item := range list {
		keys := make([]interface{}, len(item.StorageKeys))
		for j, k := range item.StorageKeys {
			keys[j] = k
		}
		out[i] = []interface{}{item.Address, keys}
	}
	return out
}
