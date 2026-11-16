package web3

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"
)

// RLPEncode encodes val using Ethereum Recursive Length Prefix (RLP) encoding.
func RLPEncode(val interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := rlpEncodeValue(&buf, val); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func rlpEncodeValue(buf *bytes.Buffer, val interface{}) error {
	if val == nil {
		buf.WriteByte(0x80)
		return nil
	}
	switch v := val.(type) {
	case []byte:
		rlpWriteBytes(buf, v)
		return nil
	case HexBytes:
		rlpWriteBytes(buf, v)
		return nil
	case Address:
		rlpWriteBytes(buf, v[:])
		return nil
	case *Address:
		if v == nil {
			buf.WriteByte(0x80)
		} else {
			rlpWriteBytes(buf, v[:])
		}
		return nil
	case Hash:
		rlpWriteBytes(buf, v[:])
		return nil
	case string:
		rlpWriteBytes(buf, []byte(v))
		return nil
	case bool:
		if v {
			buf.WriteByte(0x01)
		} else {
			buf.WriteByte(0x80)
		}
		return nil
	case uint64:
		rlpWriteUint(buf, v)
		return nil
	case uint:
		rlpWriteUint(buf, uint64(v))
		return nil
	case uint8:
		rlpWriteUint(buf, uint64(v))
		return nil
	case uint16:
		rlpWriteUint(buf, uint64(v))
		return nil
	case uint32:
		rlpWriteUint(buf, uint64(v))
		return nil
	case *big.Int:
		if v == nil || v.Sign() == 0 {
			buf.WriteByte(0x80)
			return nil
		}
		if v.Sign() < 0 {
			return errors.New("rlp: cannot encode negative big.Int")
		}
		rlpWriteBytes(buf, v.Bytes())
		return nil
	case []interface{}:
		var listBuf bytes.Buffer
		for _, item := range v {
			if err := rlpEncodeValue(&listBuf, item); err != nil {
				return err
			}
		}
		rlpWriteListHeader(buf, listBuf.Len())
		buf.Write(listBuf.Bytes())
		return nil
	}

	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			buf.WriteByte(0x80)
			return nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			b := make([]byte, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				b[i] = byte(rv.Index(i).Uint())
			}
			rlpWriteBytes(buf, b)
			return nil
		}
		var listBuf bytes.Buffer
		for i := 0; i < rv.Len(); i++ {
			if err := rlpEncodeValue(&listBuf, rv.Index(i).Interface()); err != nil {
				return err
			}
		}
		rlpWriteListHeader(buf, listBuf.Len())
		buf.Write(listBuf.Bytes())
		return nil
	case reflect.Struct:
		var listBuf bytes.Buffer
		for i := 0; i < rv.NumField(); i++ {
			f := rv.Type().Field(i)
			if f.PkgPath != "" {
				continue
			}
			if err := rlpEncodeValue(&listBuf, rv.Field(i).Interface()); err != nil {
				return err
			}
		}
		rlpWriteListHeader(buf, listBuf.Len())
		buf.Write(listBuf.Bytes())
		return nil
	default:
		return fmt.Errorf("rlp: unsupported type %T", val)
	}
}

func rlpWriteUint(buf *bytes.Buffer, i uint64) {
	if i == 0 {
		buf.WriteByte(0x80)
		return
	}
	var tmp [8]byte
	binary.BigEndian.PutUint64(tmp[:], i)
	start := 0
	for start < 8 && tmp[start] == 0 {
		start++
	}
	rlpWriteBytes(buf, tmp[start:])
}

func rlpWriteBytes(buf *bytes.Buffer, b []byte) {
	if len(b) == 1 && b[0] < 0x80 {
		buf.WriteByte(b[0])
		return
	}
	if len(b) <= 55 {
		buf.WriteByte(byte(0x80 + len(b)))
		buf.Write(b)
		return
	}
	lenBytes := encodeBigEndianLen(len(b))
	buf.WriteByte(byte(0xb7 + len(lenBytes)))
	buf.Write(lenBytes)
	buf.Write(b)
}

func rlpWriteListHeader(buf *bytes.Buffer, payloadLen int) {
	if payloadLen <= 55 {
		buf.WriteByte(byte(0xc0 + payloadLen))
		return
	}
	lenBytes := encodeBigEndianLen(payloadLen)
	buf.WriteByte(byte(0xf7 + len(lenBytes)))
	buf.Write(lenBytes)
}

func encodeBigEndianLen(n int) []byte {
	var tmp [8]byte
	binary.BigEndian.PutUint64(tmp[:], uint64(n))
	start := 0
	for start < 8 && tmp[start] == 0 {
		start++
	}
	out := make([]byte, 8-start)
	copy(out, tmp[start:])
	return out
}

// RLPItem represents a decoded RLP element (either Bytes or a List of RLPItems).
type RLPItem struct {
	IsList bool
	Bytes  []byte
	List   []RLPItem
}

func (it RLPItem) Uint64() (uint64, error) {
	if it.IsList {
		return 0, errors.New("rlp: expected bytes for uint64, got list")
	}
	if len(it.Bytes) > 8 {
		return 0, errors.New("rlp: uint64 overflow")
	}
	if len(it.Bytes) > 0 && it.Bytes[0] == 0 {
		return 0, errors.New("rlp: non-canonical leading zero in uint")
	}
	var v uint64
	for _, b := range it.Bytes {
		v = (v << 8) | uint64(b)
	}
	return v, nil
}

func (it RLPItem) BigInt() (*big.Int, error) {
	if it.IsList {
		return nil, errors.New("rlp: expected bytes for big.Int, got list")
	}
	if len(it.Bytes) > 0 && it.Bytes[0] == 0 {
		return nil, errors.New("rlp: non-canonical leading zero in big.Int")
	}
	return new(big.Int).SetBytes(it.Bytes), nil
}

// RLPDecode decodes a full RLP byte slice into an RLPItem.
func RLPDecode(data []byte) (RLPItem, error) {
	item, rest, err := rlpDecodeOne(data)
	if err != nil {
		return RLPItem{}, err
	}
	if len(rest) != 0 {
		return RLPItem{}, fmt.Errorf("rlp: %d trailing bytes after value", len(rest))
	}
	return item, nil
}

func rlpDecodeOne(data []byte) (RLPItem, []byte, error) {
	if len(data) == 0 {
		return RLPItem{}, nil, errors.New("rlp: unexpected EOF")
	}
	b := data[0]
	switch {
	case b < 0x80:
		return RLPItem{Bytes: []byte{b}}, data[1:], nil
	case b <= 0xb7:
		strLen := int(b - 0x80)
		if len(data) < 1+strLen {
			return RLPItem{}, nil, errors.New("rlp: string truncated")
		}
		if strLen == 1 && data[1] < 0x80 {
			return RLPItem{}, nil, errors.New("rlp: non-canonical single byte")
		}
		out := make([]byte, strLen)
		copy(out, data[1:1+strLen])
		return RLPItem{Bytes: out}, data[1+strLen:], nil
	case b <= 0xbf:
		lenOfLen := int(b - 0xb7)
		if len(data) < 1+lenOfLen {
			return RLPItem{}, nil, errors.New("rlp: string length truncated")
		}
		strLen, err := decodeLength(data[1 : 1+lenOfLen])
		if err != nil || strLen <= 55 {
			return RLPItem{}, nil, errors.New("rlp: invalid long string length")
		}
		if len(data) < 1+lenOfLen+strLen {
			return RLPItem{}, nil, errors.New("rlp: long string truncated")
		}
		out := make([]byte, strLen)
		copy(out, data[1+lenOfLen:1+lenOfLen+strLen])
		return RLPItem{Bytes: out}, data[1+lenOfLen+strLen:], nil
	case b <= 0xf7:
		listLen := int(b - 0xc0)
		if len(data) < 1+listLen {
			return RLPItem{}, nil, errors.New("rlp: list truncated")
		}
		items, err := rlpDecodeListPayload(data[1 : 1+listLen])
		if err != nil {
			return RLPItem{}, nil, err
		}
		return RLPItem{IsList: true, List: items}, data[1+listLen:], nil
	default:
		lenOfLen := int(b - 0xf7)
		if len(data) < 1+lenOfLen {
			return RLPItem{}, nil, errors.New("rlp: list length truncated")
		}
		listLen, err := decodeLength(data[1 : 1+lenOfLen])
		if err != nil || listLen <= 55 {
			return RLPItem{}, nil, errors.New("rlp: invalid long list length")
		}
		if len(data) < 1+lenOfLen+listLen {
			return RLPItem{}, nil, errors.New("rlp: long list truncated")
		}
		items, err := rlpDecodeListPayload(data[1+lenOfLen : 1+lenOfLen+listLen])
		if err != nil {
			return RLPItem{}, nil, err
		}
		return RLPItem{IsList: true, List: items}, data[1+lenOfLen+listLen:], nil
	}
}

func rlpDecodeListPayload(payload []byte) ([]RLPItem, error) {
	var items []RLPItem
	rest := payload
	for len(rest) > 0 {
		var it RLPItem
		var err error
		it, rest, err = rlpDecodeOne(rest)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func decodeLength(b []byte) (int, error) {
	if len(b) == 0 || len(b) > 8 || b[0] == 0 {
		return 0, errors.New("invalid rlp length prefix")
	}
	var n uint64
	for _, x := range b {
		n = (n << 8) | uint64(x)
	}
	if n > uint64(^uint(0)>>1) {
		return 0, errors.New("rlp length overflow")
	}
	return int(n), nil
}
