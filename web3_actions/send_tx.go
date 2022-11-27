package web3_actions

import (
	"context"
	"fmt"
	"math/big"

	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/gochain/gochain/v4/crypto"
	"github.com/gochain/gochain/v4/rlp"
	"github.com/rs/zerolog/log"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

// Send performs a regular native coin transaction (not a contract)
func (w *Web3Actions) Send(ctx context.Context, address common.Address, amount *big.Int, gasPrice *big.Int, gasLimit uint64) (*web3_types.Transaction, error) {
	w.Dial()
	defer w.Close()
	var err error
	if gasPrice == nil || gasPrice.Int64() == 0 {
		gasPrice, err = w.GetGasPrice(ctx)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("Send: GetGasPrice")
			return nil, fmt.Errorf("cannot get gas price: %v", err)
		}
	}
	chainID, err := w.GetChainID(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Send: GetChainID")
		return nil, fmt.Errorf("couldn't get chain ID: %v", err)
	}

	if gasLimit == 0 {
		gasLimit = 21000
	}
	publicKeyECDSA := w.EcdsaPublicKey()
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := w.GetPendingTransactionCount(ctx, fromAddress)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Send: GetPendingTransactionCount")
		return nil, fmt.Errorf("cannot get nonce: %v", err)
	}
	tx := types.NewTransaction(nonce, address, amount, gasLimit, gasPrice, nil)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), w.EcdsaPrivateKey())
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Send: SignTx")
		return nil, fmt.Errorf("cannot sign transaction: %v", err)
	}
	err = w.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Send: SendTransaction")
		return nil, fmt.Errorf("failed to send transaction: %v", err)
	}
	return ConvertTx(signedTx, fromAddress), nil
}

// SendTransaction sends the Transaction
func (w *Web3Actions) SendTransaction(ctx context.Context, signedTx *types.Transaction) error {
	w.Dial()
	defer w.Close()
	raw, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("SendTransaction: rlp.EncodeToBytes")
		return err
	}
	return w.SendRawTransaction(ctx, raw)
}

func ConvertTx(tx *types.Transaction, from common.Address) *web3_types.Transaction {
	rtx := &web3_types.Transaction{}
	rtx.Nonce = tx.Nonce()
	rtx.GasPrice = tx.GasPrice()
	rtx.GasLimit = tx.Gas()
	rtx.To = tx.To()
	rtx.Value = tx.Value()
	rtx.Input = tx.Data()
	rtx.Hash = tx.Hash()
	rtx.From = from
	rtx.V, rtx.R, rtx.S = tx.RawSignatureValues()
	return rtx
}
