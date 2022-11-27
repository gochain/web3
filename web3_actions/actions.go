package web3_actions

import (
	"github.com/zeus-fyi/gochain/web3/accounts"
	web3_client "github.com/zeus-fyi/gochain/web3/client"
)

type Web3Actions struct {
	web3_client.Client
	NodeURL string
	accounts.Account
}

func (w *Web3Actions) Dial() {
	c, err := web3_client.Dial(w.NodeURL)
	if err != nil {
		panic(err)
	}
	w.Client = c
}

func NewWeb3ActionsClient(nodeUrl string) Web3Actions {
	return Web3Actions{
		NodeURL: nodeUrl,
	}
}
