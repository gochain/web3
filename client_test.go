package web3

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMockRPCClientAndEIP1559Flow(t *testing.T) {
	var sentRawTx string
	var gotUserAgent string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")

		var raw json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&raw)
		if len(raw) > 0 && raw[0] == '[' {
			// Batch call (GetID)
			var reqs []jsonrpcRequest
			_ = json.Unmarshal(raw, &reqs)
			resps := make([]map[string]interface{}, len(reqs))
			for i, req := range reqs {
				var res interface{}
				switch req.Method {
				case "eth_getBlockByNumber":
					res = map[string]interface{}{
						"parentHash":   ZeroHash.Hex(),
						"number":       "0x0",
						"gasLimit":     "0x1c9c380",
						"gasUsed":      "0x0",
						"timestamp":    "0x60000000",
						"hash":         "0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3",
					}
				case "net_version":
					res = "1"
				case "eth_chainId":
					res = "0x1"
				}
				resps[i] = map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": res}
			}
			_ = json.NewEncoder(w).Encode(resps)
			return
		}

		var req jsonrpcRequest
		_ = json.Unmarshal(raw, &req)
		reply := func(res interface{}) {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  res,
			})
		}

		switch req.Method {
		case "eth_chainId":
			reply("0x1")
		case "eth_getTransactionCount":
			reply("0x5")
		case "eth_getBlockByNumber":
			reply(map[string]interface{}{
				"parentHash":    ZeroHash.Hex(),
				"number":        "0x100",
				"gasLimit":      "0x1c9c380",
				"gasUsed":       "0x5208",
				"baseFeePerGas": "0x3b9aca00", // 10 gwei = 0x2540be400, here 1 gwei
				"timestamp":     "0x65000000",
				"hash":          "0x1111111111111111111111111111111111111111111111111111111111111111",
				"transactions":  []string{},
			})
		case "eth_maxPriorityFeePerGas":
			reply("0x59682f00") // 1.5 gwei
		case "eth_estimateGas":
			reply("0xc350") // 50,000
		case "eth_getCode":
			reply("0x60806040")
		case "eth_sendRawTransaction":
			sentRawTx, _ = req.Params[0].(string)
			reply(Keccak256Hash(FromHex(sentRawTx)).Hex())
		case "eth_call":
			// Return ABI-encoded uint8(18) for decimals()
			var word [32]byte
			word[31] = 18
			reply(EncodeHex(word[:]))
		case "eth_getTransactionReceipt":
			reply(map[string]interface{}{
				"type":              "0x2",
				"status":            "0x1",
				"cumulativeGasUsed": "0xc350",
				"effectiveGasPrice": "0x9502f900",
				"gasUsed":           "0xc350",
				"transactionHash":   req.Params[0],
				"transactionIndex":  "0x0",
				"blockHash":         "0x1111111111111111111111111111111111111111111111111111111111111111",
				"blockNumber":       "0x101",
				"from":              "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
				"to":                "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
				"logs":              []interface{}{},
			})
		}
	}))
	defer srv.Close()

	c, err := Dial(srv.URL)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	id, err := c.GetID(ctx)
	if err != nil {
		t.Fatalf("GetID failed: %v", err)
	}
	if id.ChainID.Int64() != 1 || id.NetworkID.Int64() != 1 {
		t.Fatalf("unexpected ID: %+v", id)
	}
	if !strings.HasPrefix(gotUserAgent, "gochain-web3") {
		t.Fatalf("expected User-Agent header, got %q", gotUserAgent)
	}

	erc20ABI, err := GetABI("erc20")
	if err != nil {
		t.Fatal(err)
	}
	decRes, err := CallConstantFunction(ctx, c, *erc20ABI, "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", "decimals")
	if err != nil || decRes[0].(uint8) != 18 {
		t.Fatalf("CallConstantFunction decimals = %v, err=%v", decRes, err)
	}

	// CallFunctionWithArgs with gasLimit=0 -> auto-estimates 50,000 * 1.2 = 60,000 & auto-uses EIP-1559 Type 2!
	const pk = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	tx, err := CallFunctionWithArgs(ctx, c, pk, "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
		big.NewInt(0), nil, 0, *erc20ABI, "transfer", "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC", big.NewInt(1000))
	if err != nil {
		t.Fatalf("CallFunctionWithArgs failed: %v", err)
	}
	if tx.Type != TxTypeDynamicFee {
		t.Fatalf("expected EIP-1559 Type 2 tx, got %d", tx.Type)
	}
	if tx.GasLimit != 60000 {
		t.Fatalf("expected auto-estimated gasLimit 60000 (50000 + 20%%), got %d", tx.GasLimit)
	}
	if !strings.HasPrefix(sentRawTx, "0x02") {
		t.Fatalf("expected 0x02 EIP-1559 raw tx prefix, got %s", sentRawTx)
	}

	rc, err := WaitForReceipt(ctx, c, tx.Hash)
	if err != nil || rc.Status != ReceiptStatusSuccessful {
		t.Fatalf("WaitForReceipt = %+v, err=%v", rc, err)
	}
}

func TestParseLogsSafety(t *testing.T) {
	erc20ABI, err := GetABI("erc20")
	if err != nil {
		t.Fatal(err)
	}
	transferEv := erc20ABI.Events["Transfer"]
	from := HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	to := HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	var valWord [32]byte
	valWord[31] = 42

	logs := []*Log{
		// Anonymous log (0 topics) — must not panic
		{Topics: nil, Data: []byte{1, 2, 3}},
		// Unknown external event topic — must not panic
		{Topics: []Hash{Keccak256Hash([]byte("UnknownEvent(uint256)"))}, Data: valWord[:]},
		// Valid ERC20 Transfer event
		{
			Topics: []Hash{
				transferEv.ID,
				BytesToHash(from[:]),
				BytesToHash(to[:]),
			},
			Data: valWord[:],
		},
	}

	events, err := ParseLogs(*erc20ABI, logs)
	if err != nil {
		t.Fatalf("ParseLogs failed: %v", err)
	}
	if len(events) != 1 || events[0].Name != "Transfer" {
		t.Fatalf("unexpected parsed events: %+v", events)
	}
	if events[0].Fields["from"].(Address) != from || events[0].Fields["to"].(Address) != to {
		t.Fatalf("unexpected indexed fields: %+v", events[0].Fields)
	}
	if events[0].Fields["value"].(*big.Int).Int64() != 42 {
		t.Fatalf("unexpected value field: %+v", events[0].Fields["value"])
	}
}
