package accounts

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/crypto"
	"github.com/rs/zerolog/log"
)

type Account struct {
	key *ecdsa.PrivateKey
}

func CreateAccount() (*Account, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		log.Err(err).Msg("CreateAccount: crypto.GenerateKey()")
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
		log.Err(err).Msg("ParsePrivateKey: HexToECDSA")
		return nil, err
	}
	return &Account{
		key: key,
	}, nil
}

func (a *Account) EcdsaPrivateKey() *ecdsa.PrivateKey {
	return a.key
}

func (a *Account) Address() common.Address {
	return crypto.PubkeyToAddress(a.key.PublicKey)
}

func (a *Account) PublicKey() string {
	return crypto.PubkeyToAddress(a.key.PublicKey).Hex()
}

func (a *Account) PrivateKey() string {
	return "0x" + hex.EncodeToString(crypto.FromECDSA(a.key))
}

func (a *Account) EcdsaPublicKey() *ecdsa.PublicKey {
	privateKey := a.EcdsaPrivateKey()
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		err := errors.New("error casting public key to ECDSA")
		log.Panic().Err(err).Msg("EcdsaPublicKey")
		panic(err)
	}
	return publicKeyECDSA
}
