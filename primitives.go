package web3

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"golang.org/x/crypto/sha3"
)

const (
	AddressLength = 20
	HashLength    = 32
	BloomLength   = 256
)

var (
	ZeroAddress Address
	ZeroHash    Hash
)

// Address represents a 20-byte EVM account or contract address.
type Address [AddressLength]byte

// BytesToAddress right-aligns b into a 20-byte Address.
func BytesToAddress(b []byte) Address {
	var a Address
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
	return a
}

// HexToAddress parses a hex string (with or without 0x prefix) into an Address.
func HexToAddress(s string) Address {
	return BytesToAddress(FromHex(s))
}

// IsHexAddress reports whether s is a valid 40-hex-character EVM address (with optional 0x prefix).
func IsHexAddress(s string) bool {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if len(s) != 2*AddressLength {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func (a Address) Bytes() []byte { return a[:] }
func (a Address) IsZero() bool  { return a == ZeroAddress }

// Hex returns the EIP-55 checksummed hex representation of the address.
func (a Address) Hex() string {
	unchecksummed := hex.EncodeToString(a[:])
	hw := sha3.NewLegacyKeccak256()
	hw.Write([]byte(unchecksummed))
	hash := hw.Sum(nil)

	result := []byte("0x" + unchecksummed)
	for i := 2; i < len(result); i++ {
		hashByte := hash[(i-2)/2]
		if i%2 == 0 {
			hashByte >>= 4
		} else {
			hashByte &= 0x0f
		}
		if result[i] > '9' && hashByte > 7 {
			result[i] -= 32
		}
	}
	return string(result)
}

func (a Address) String() string { return a.Hex() }

func (a Address) MarshalText() ([]byte, error) {
	return []byte(a.Hex()), nil
}

func (a *Address) UnmarshalText(input []byte) error {
	b, err := decodeFixedHex("address", input, AddressLength)
	if err != nil {
		return err
	}
	copy(a[:], b)
	return nil
}

func (a Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Hex())
}

func (a *Address) UnmarshalJSON(input []byte) error {
	if bytes.Equal(input, []byte("null")) {
		return nil
	}
	var s string
	if err := json.Unmarshal(input, &s); err != nil {
		return err
	}
	return a.UnmarshalText([]byte(s))
}

// Hash represents a 32-byte Keccak-256 hash.
type Hash [HashLength]byte

func BytesToHash(b []byte) Hash {
	var h Hash
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}
	copy(h[HashLength-len(b):], b)
	return h
}

func HexToHash(s string) Hash {
	return BytesToHash(FromHex(s))
}

func (h Hash) Bytes() []byte  { return h[:] }
func (h Hash) IsZero() bool   { return h == ZeroHash }
func (h Hash) Hex() string    { return EncodeHex(h[:]) }
func (h Hash) String() string { return h.Hex() }

func (h Hash) MarshalText() ([]byte, error) {
	return []byte(h.Hex()), nil
}

func (h *Hash) UnmarshalText(input []byte) error {
	b, err := decodeFixedHex("hash", input, HashLength)
	if err != nil {
		return err
	}
	copy(h[:], b)
	return nil
}

func (h Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.Hex())
}

func (h *Hash) UnmarshalJSON(input []byte) error {
	if bytes.Equal(input, []byte("null")) {
		return nil
	}
	var s string
	if err := json.Unmarshal(input, &s); err != nil {
		return err
	}
	return h.UnmarshalText([]byte(s))
}

// Bloom represents a 2048-bit (256-byte) logs bloom filter.
type Bloom [BloomLength]byte

func (b Bloom) Bytes() []byte { return b[:] }
func (b Bloom) Hex() string   { return EncodeHex(b[:]) }

func (b Bloom) MarshalText() ([]byte, error) {
	return []byte(b.Hex()), nil
}

func (b *Bloom) UnmarshalText(input []byte) error {
	raw, err := decodeFixedHex("bloom", input, BloomLength)
	if err != nil {
		return err
	}
	copy(b[:], raw)
	return nil
}

// BlockNonce represents an 8-byte block header nonce.
type BlockNonce [8]byte

func (n BlockNonce) Uint64() uint64 {
	var v uint64
	for i := 0; i < 8; i++ {
		v = (v << 8) | uint64(n[i])
	}
	return v
}

func (n BlockNonce) MarshalText() ([]byte, error) {
	return []byte(EncodeHex(n[:])), nil
}

func (n *BlockNonce) UnmarshalText(input []byte) error {
	raw, err := decodeFixedHex("nonce", input, 8)
	if err != nil {
		return err
	}
	copy(n[:], raw)
	return nil
}

// HexBytes marshals/unmarshals byte slices as 0x-prefixed hex strings in JSON.
type HexBytes []byte

func (b HexBytes) String() string { return EncodeHex(b) }

func (b HexBytes) MarshalText() ([]byte, error) {
	return []byte(EncodeHex(b)), nil
}

func (b *HexBytes) UnmarshalText(input []byte) error {
	raw, err := DecodeHex(string(input))
	if err != nil {
		return err
	}
	*b = raw
	return nil
}

// HexUint64 marshals/unmarshals uint64 as 0x-prefixed quantity hex in JSON.
type HexUint64 uint64

func (u HexUint64) MarshalText() ([]byte, error) {
	return []byte(EncodeUint64(uint64(u))), nil
}

func (u *HexUint64) UnmarshalText(input []byte) error {
	val, err := DecodeUint64(string(input))
	if err != nil {
		return err
	}
	*u = HexUint64(val)
	return nil
}

// HexBig marshals/unmarshals big.Int as 0x-prefixed quantity hex in JSON.
type HexBig big.Int

func (b *HexBig) ToInt() *big.Int {
	if b == nil {
		return nil
	}
	return (*big.Int)(b)
}

func (b *HexBig) MarshalText() ([]byte, error) {
	if b == nil {
		return []byte("0x0"), nil
	}
	return []byte(EncodeBig((*big.Int)(b))), nil
}

func (b *HexBig) UnmarshalText(input []byte) error {
	val, err := DecodeBig(string(input))
	if err != nil {
		return err
	}
	*b = HexBig(*val)
	return nil
}

// EncodeHex encodes b as a hex string with 0x prefix.
func EncodeHex(b []byte) string {
	out := make([]byte, len(b)*2+2)
	copy(out, "0x")
	hex.Encode(out[2:], b)
	return string(out)
}

// DecodeHex decodes a 0x-prefixed (or unprefixed) even-length hex string.
func DecodeHex(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return nil, errors.New("empty hex string")
	}
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if len(s) == 0 {
		return []byte{}, nil
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}

// FromHex decodes hex string s (ignoring errors, returning empty slice on invalid hex).
func FromHex(s string) []byte {
	s = strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(s), "0x"), "0X")
	if len(s)%2 == 1 {
		s = "0" + s
	}
	b, _ := hex.DecodeString(s)
	return b
}

// EncodeUint64 encodes i as a 0x-prefixed quantity hex string (0x0 for zero).
func EncodeUint64(i uint64) string {
	return "0x" + strconv.FormatUint(i, 16)
}

// DecodeUint64 parses a 0x-prefixed quantity hex string into uint64.
func DecodeUint64(s string) (uint64, error) {
	raw, err := trimQuantityHex(s)
	if err != nil {
		return 0, err
	}
	if len(raw) > 16 {
		return 0, fmt.Errorf("hex uint64 %q overflows 64 bits", s)
	}
	return strconv.ParseUint(raw, 16, 64)
}

// EncodeBig encodes bigint as a 0x-prefixed quantity hex string.
func EncodeBig(bigint *big.Int) string {
	if bigint == nil || bigint.Sign() == 0 {
		return "0x0"
	}
	if bigint.Sign() < 0 {
		return "-0x" + bigint.Text(16)[1:]
	}
	return "0x" + bigint.Text(16)
}

// DecodeBig parses a 0x-prefixed quantity hex string into *big.Int.
func DecodeBig(s string) (*big.Int, error) {
	raw, err := trimQuantityHex(s)
	if err != nil {
		return nil, err
	}
	i, ok := new(big.Int).SetString(raw, 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex big.Int %q", s)
	}
	return i, nil
}

func trimQuantityHex(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return "", fmt.Errorf("hex string %q missing 0x prefix", s)
	}
	raw := s[2:]
	if len(raw) == 0 {
		return "0", nil
	}
	return raw, nil
}

func decodeFixedHex(typeName string, input []byte, size int) ([]byte, error) {
	s := strings.TrimSpace(string(input))
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if len(s) != size*2 {
		return nil, fmt.Errorf("invalid %s hex length %d: want %d", typeName, len(s), size*2)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid %s hex: %w", typeName, err)
	}
	return b, nil
}
