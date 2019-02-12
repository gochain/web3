package web3

import (
	"crypto/ecdsa"
	"encoding/hex"
	"strings"

	"github.com/gochain-io/gochain/v3/crypto"
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
	fromPK := Strip0x(pkHex)
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

func (a *Account) PublicKey() string {
	return crypto.PubkeyToAddress(a.key.PublicKey).Hex()
}
func (a *Account) PrivateKey() string {
	return "0x" + hex.EncodeToString(a.key.D.Bytes())
}

func Strip0x(pk string) string {
	if strings.HasPrefix(pk, "0x") {
		return pk[2:len(pk)]
	}
	return pk
}
