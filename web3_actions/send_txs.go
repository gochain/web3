package web3_actions

import (
	"context"
	"fmt"

	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/gochain/gochain/v4/crypto"
	"github.com/gochain/gochain/v4/rlp"
	"github.com/rs/zerolog/log"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

// Send performs a regular native coin transaction (not a contract)
func (w *Web3Actions) Send(ctx context.Context, params SendEtherPayload) (*web3_types.Transaction, error) {
	w.Dial()
	defer w.Close()
	signedTx, err := w.GetSignedSendTx(ctx, params)
	err = w.SendSignedTransaction(ctx, signedTx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Send: SendTransaction")
		return nil, fmt.Errorf("failed to send transaction: %v", err)
	}
	publicKeyECDSA := w.EcdsaPublicKey()
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	return ConvertTx(signedTx, fromAddress), nil
}

// SendSignedTransaction sends the Transaction
func (w *Web3Actions) SendSignedTransaction(ctx context.Context, signedTx *types.Transaction) error {
	w.Dial()
	defer w.Close()
	raw, err := EncodeSignedTx(ctx, signedTx)
	if err != nil {
		return err
	}
	return w.SendRawTransaction(ctx, raw)
}

func (w *Web3Actions) SubmitSignedTxAndReturnTxData(ctx context.Context, signedTx *types.Transaction) (*web3_types.Transaction, error) {
	w.Dial()
	defer w.Close()
	err := w.SendSignedTransaction(ctx, signedTx)
	if err != nil {
		return nil, err
	}
	publicKeyECDSA := w.EcdsaPublicKey()
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	return ConvertTx(signedTx, fromAddress), nil
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

func EncodeSignedTx(ctx context.Context, signedTx *types.Transaction) ([]byte, error) {
	raw, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("EncodeSignedTx: EncodeToBytes")
		return nil, err
	}
	return raw, err
}
