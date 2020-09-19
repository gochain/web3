package web3

import (
	"crypto/ecdsa"
	"encoding/hex"
	"strings"

	"github.com/gochain/gochain/v3/crypto"
)

func CreateAccount() (*Account, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	return &Account{
		key: key,
	}, nil
}

func ParsePrivateKey(pkHex string) (*Account, error) {
	fromPK := strings.TrimPrefix(pkHex, "0x")
	key, err := crypto.HexToECDSA(fromPK)
	if err != nil {
		return nil, err
	}
	return &Account{
		key: key,
	}, nil
}

type Account struct {
	key *ecdsa.PrivateKey
}

func (a *Account) Key() *ecdsa.PrivateKey {
	return a.key
}

func (a *Account) PublicKey() string {
	return crypto.PubkeyToAddress(a.key.PublicKey).Hex()
}

func (a *Account) PrivateKey() string {
	return "0x" + hex.EncodeToString(crypto.FromECDSA(a.key))
}
