package web3_actions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"

	"github.com/gochain/gochain/v4/common"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

func constructSendEtherPayload(amount *big.Int, address common.Address, gasPrice *big.Int, gasLimit uint64) SendEtherPayload {
	params := SendEtherPayload{
		TransferArgs: TransferArgs{
			Amount:    amount,
			ToAddress: address,
		},
		GasPriceLimits: GasPriceLimits{
			GasPrice: gasPrice,
			GasLimit: gasLimit,
		},
	}
	return params
}

func ValidateToAddress(ctx context.Context, toAddress string) error {
	if toAddress == "" {
		err := errors.New("the recipient address cannot be empty")
		log.Ctx(ctx).Err(err).Msg("Transfer: toAddress")
		return err
	}
	if !common.IsHexAddress(toAddress) {
		err := fmt.Errorf("invalid to 'address': %s", toAddress)
		log.Ctx(ctx).Err(err).Msg("Transfer: IsHexAddress")
		return err
	}
	return nil
}

func ConvertTailForTransfer(ctx context.Context, tail []string) (TransferArgs, error) {
	if len(tail) < 3 {
		err := errors.New("invalid arguments. format is: `transfer X to ADDRESS`")
		log.Ctx(ctx).Err(err).Msg("Web3Actions: Transfer")
		return TransferArgs{}, err
	}
	amountS := tail[0]
	amountD, err := decimal.NewFromString(amountS)
	if err != nil {
		err = fmt.Errorf("invalid amount %v", amountS)
		log.Ctx(ctx).Err(err).Msg("Transfer: decimal.NewFromString")
		return TransferArgs{}, err
	}
	toAddress := tail[2]

	err = ValidateToAddress(ctx, toAddress)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Transfer: ValidateToAddress")
		return TransferArgs{}, err
	}
	address := common.HexToAddress(toAddress)
	argsIn := TransferArgs{
		Amount:    amountD.BigInt(),
		ToAddress: address,
	}
	return argsIn, err
}

func marshalJSON(ctx context.Context, data interface{}) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("marshalJSON")
		return "", err
	}
	return string(b), err
}

func isValidUrl(toTest string) bool {
	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	return true
}
func downloadFile(ctx context.Context, url string) ([]byte, error) {
	var dst bytes.Buffer
	response, err := http.Get(url)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("downloadFile: Get")
		return nil, err
	}
	defer response.Body.Close()
	_, err = io.Copy(&dst, response.Body)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("downloadFile: Copy")
		return nil, err
	}
	return dst.Bytes(), nil
}
