package web3_actions

import (
	"context"

	"github.com/zeus-fyi/gochain/web3/accounts"
	web3_client "github.com/zeus-fyi/gochain/web3/client"
)

type Web3Actions struct {
	web3_client.Client
	*accounts.Account
	Headers map[string]string
	NodeURL string
	Network string
}

func (w *Web3Actions) Dial() {
	c, err := web3_client.Dial(w.NodeURL)
	if err != nil {
		panic(err)
	}
	w.Client = c
	for k, h := range w.Headers {
		w.Client.SetHeader(context.Background(), k, h)
	}
}

func NewWeb3ActionsClient(nodeUrl string) Web3Actions {
	return Web3Actions{
		NodeURL: nodeUrl,
	}
}

func NewWeb3ActionsClientWithAccount(nodeUrl string, account *accounts.Account) Web3Actions {
	return Web3Actions{
		NodeURL: nodeUrl,
		Account: account,
	}
}
