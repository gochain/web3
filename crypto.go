package web3

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	secp_ecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"golang.org/x/crypto/sha3"
)

var secp256k1HalfN = new(big.Int).Rsh(secp256k1.S256().N, 1)

// Keccak256 computes the Ethereum Keccak-256 hash of the concatenated input slices.
func Keccak256(data ...[]byte) []byte {
	hw := sha3.NewLegacyKeccak256()
	for _, b := range data {
		hw.Write(b)
	}
	return hw.Sum(nil)
}

// Keccak256Hash computes the Ethereum Keccak-256 hash and returns it as a Hash.
func Keccak256Hash(data ...[]byte) Hash {
	return BytesToHash(Keccak256(data...))
}

// GenerateKey generates a new secp256k1 ECDSA private key.
func GenerateKey() (*ecdsa.PrivateKey, error) {
	priv, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	return priv.ToECDSA(), nil
}

// HexToECDSA parses a 32-byte hex-encoded secp256k1 private key (with or without 0x prefix).
func HexToECDSA(hexKey string) (*ecdsa.PrivateKey, error) {
	hexKey = strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(hexKey), "0x"), "0X")
	b, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex private key: %w", err)
	}
	return ToECDSA(b)
}

// ToECDSA converts raw 32-byte secret scalar bytes into an *ecdsa.PrivateKey on secp256k1.
func ToECDSA(d []byte) (*ecdsa.PrivateKey, error) {
	if len(d) != 32 {
		return nil, fmt.Errorf("invalid private key length %d: must be 32 bytes", len(d))
	}
	var scalar secp256k1.ModNScalar
	if overflow := scalar.SetByteSlice(d); overflow || scalar.IsZero() {
		scalar.Zero()
		return nil, errors.New("private key out of secp256k1 scalar range")
	}
	priv := secp256k1.NewPrivateKey(&scalar)
	scalar.Zero()
	return priv.ToECDSA(), nil
}

// FromECDSA exports a private key as a 32-byte big-endian slice.
func FromECDSA(priv *ecdsa.PrivateKey) []byte {
	if priv == nil {
		return nil
	}
	var out [32]byte
	b := priv.D.Bytes()
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	copy(out[32-len(b):], b)
	return out[:]
}

// FromECDSAPub serializes a secp256k1 public key in 65-byte uncompressed format (0x04 || X || Y).
func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	var fx, fy secp256k1.FieldVal
	fx.SetByteSlice(pub.X.Bytes())
	fy.SetByteSlice(pub.Y.Bytes())
	sp := secp256k1.NewPublicKey(&fx, &fy)
	return sp.SerializeUncompressed()
}

// PubkeyToAddress derives the 20-byte EVM address from a secp256k1 public key.
func PubkeyToAddress(p ecdsa.PublicKey) Address {
	pubBytes := FromECDSAPub(&p)
	if len(pubBytes) != 65 {
		return ZeroAddress
	}
	return BytesToAddress(Keccak256(pubBytes[1:])[12:])
}

// SignHash produces a 65-byte recoverable secp256k1 signature in [R(32) || S(32) || V(1)] format,
// where V is 0 or 1 (recovery ID) and S is canonical low-S (EIP-2).
func SignHash(digestHash []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	if len(digestHash) != 32 {
		return nil, fmt.Errorf("digest hash must be 32 bytes (got %d)", len(digestHash))
	}
	if prv == nil {
		return nil, errors.New("nil private key")
	}
	var scalar secp256k1.ModNScalar
	if overflow := scalar.SetByteSlice(FromECDSA(prv)); overflow || scalar.IsZero() {
		scalar.Zero()
		return nil, errors.New("invalid private key scalar")
	}
	secpPriv := secp256k1.NewPrivateKey(&scalar)
	scalar.Zero()

	// SignCompact returns [header(1) || R(32) || S(32)] where header = 27 + recID (+ 4 if isCompressedKey).
	compact := secp_ecdsa.SignCompact(secpPriv, digestHash, false)
	secpPriv.Zero()

	sig := make([]byte, 65)
	copy(sig[0:64], compact[1:65])
	sig[64] = (compact[0] - 27) & 0x03
	return sig, nil
}

// RecoverPubkey recovers the secp256k1 public key from a 32-byte digest and 65-byte [R || S || V] signature
// (accepting V in {0, 1} or {27, 28}).
func RecoverPubkey(digestHash, sig []byte) (*ecdsa.PublicKey, error) {
	if len(digestHash) != 32 {
		return nil, fmt.Errorf("digest hash must be 32 bytes (got %d)", len(digestHash))
	}
	if len(sig) != 65 {
		return nil, fmt.Errorf("signature must be 65 bytes (got %d)", len(sig))
	}
	v := sig[64]
	if v >= 27 {
		v -= 27
	}
	if v > 1 {
		return nil, fmt.Errorf("invalid signature recovery id %d", sig[64])
	}
	sVal := new(big.Int).SetBytes(sig[32:64])
	if sVal.Sign() <= 0 || sVal.Cmp(secp256k1HalfN) > 0 {
		return nil, errors.New("non-canonical high-S or zero signature")
	}

	var compact [65]byte
	compact[0] = 27 + v
	copy(compact[1:65], sig[0:64])
	pub, _, err := secp_ecdsa.RecoverCompact(compact[:], digestHash)
	if err != nil {
		return nil, fmt.Errorf("failed to recover public key: %w", err)
	}
	return pub.ToECDSA(), nil
}

// RecoverAddress recovers the signer's EVM Address from a 32-byte digest and 65-byte signature.
func RecoverAddress(digestHash, sig []byte) (Address, error) {
	pub, err := RecoverPubkey(digestHash, sig)
	if err != nil {
		return ZeroAddress, err
	}
	return PubkeyToAddress(*pub), nil
}

// PersonalMessageHash computes the EIP-191 personal_sign digest ("\x19Ethereum Signed Message:\n" + len + msg).
func PersonalMessageHash(message []byte) Hash {
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%s", strconv.Itoa(len(message)))
	return Keccak256Hash([]byte(prefix), message)
}

// PersonalSign signs message using EIP-191 personal_sign, returning a 65-byte signature with V in {27, 28}.
func PersonalSign(message []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	digest := PersonalMessageHash(message)
	sig, err := SignHash(digest[:], prv)
	if err != nil {
		return nil, err
	}
	sig[64] += 27
	return sig, nil
}

// PersonalRecover recovers the EVM address that produced an EIP-191 personal_sign signature.
func PersonalRecover(message, sig []byte) (Address, error) {
	digest := PersonalMessageHash(message)
	return RecoverAddress(digest[:], sig)
}

// CreateAddress computes the deterministic contract address deployed by caller at nonce (RLP([caller, nonce])).
func CreateAddress(caller Address, nonce uint64) Address {
	encoded, err := RLPEncode([]interface{}{caller, nonce})
	if err != nil {
		return ZeroAddress
	}
	return BytesToAddress(Keccak256(encoded)[12:])
}
