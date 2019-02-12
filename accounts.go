package web3

import (
	"crypto/ecdsa"
	"encoding/hex"

	"github.com/gochain-io/gochain/crypto"
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
func ParsePrivateKey(pk string) (*Account, error) {

	key, err := crypto.GenerateKey()
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

func (a *Account) PublicKey() string {
	return crypto.PubkeyToAddress(a.key.PublicKey).Hex()
}
func (a *Account) PrivateKey() string {
	return hex.EncodeToString(a.key.D.Bytes())
}
