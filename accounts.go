package web3

import (
	"crypto/ecdsa"
	"encoding/hex"
)

func CreateAccount() (*Account, error) {
	key, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	return &Account{key: key}, nil
}

func ParsePrivateKey(pkHex string) (*Account, error) {
	key, err := HexToECDSA(pkHex)
	if err != nil {
		return nil, err
	}
	return &Account{key: key}, nil
}

type Account struct {
	key *ecdsa.PrivateKey
}

func NewAccountFromKey(key *ecdsa.PrivateKey) *Account {
	return &Account{key: key}
}

func (a *Account) Key() *ecdsa.PrivateKey {
	return a.key
}

func (a *Account) Address() Address {
	return PubkeyToAddress(a.key.PublicKey)
}

func (a *Account) PublicKey() string {
	return a.Address().Hex()
}

func (a *Account) PrivateKey() string {
	return "0x" + hex.EncodeToString(FromECDSA(a.key))
}
