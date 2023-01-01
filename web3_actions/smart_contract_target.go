package web3_actions

import (
	"context"
	"strings"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/zeus-fyi/gochain/web3/assets"
)

func (w *Web3Actions) GetTargetContract(ctx context.Context, rpcURL, contractAddress string) (string, error) {
	w.Dial()
	defer w.Close()
	ac := NewWeb3ActionsClient(rpcURL)
	ac.Dial()
	defer ac.Close()
	myabi, err := abi.JSON(strings.NewReader(assets.UpgradeableProxyABI))
	if err != nil {
		return "", err
	}
	payload := SendContractTxPayload{
		SmartContractAddr: contractAddress,
		ContractABI:       &myabi,
		SendEtherPayload:  SendEtherPayload{},
		MethodName:        "target",
		Params:            nil,
	}
	res, err := ac.CallConstantFunction(ctx, &payload)
	if err != nil {
		return "", err
	}
	if len(res) != 1 {
		return "", err

	}
	return res[0].(string), err

}
