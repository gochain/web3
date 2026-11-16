package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gochain/web3"
	"github.com/gochain/web3/assets"
)

func runCLI(t *testing.T, args ...string) string {
	t.Helper()
	format = ""
	verbose = false

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	app := buildApp(context.Background())
	fullArgs := append([]string{"web3"}, args...)
	normArgs := normalizeCLIArgs(app, fullArgs)
	runErr := app.RunContext(context.Background(), normArgs)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if runErr != nil {
		t.Fatalf("CLI %v failed: %v\nOutput: %s", args, runErr, buf.String())
	}
	return strings.TrimSpace(buf.String())
}

func TestNormalizeCLIArgsFlagsAfterPositional(t *testing.T) {
	app := buildApp(context.Background())
	got := normalizeCLIArgs(app, []string{"web3", "contract", "build", "hello.sol", "--solc-version", "0.8.24", "-o", "out"})
	want := []string{"web3", "contract", "build", "--solc-version", "0.8.24", "-o", "out", "hello.sol"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v, want %v", got, want)
	}

	// Ensure negative numbers in contract call stay positional
	gotNeg := normalizeCLIArgs(app, []string{"web3", "contract", "call", "--abi", "erc20", "--function", "foo", "-42", "--wait"})
	wantNeg := []string{"web3", "contract", "call", "--abi", "erc20", "--function", "foo", "--wait", "-42"}
	if strings.Join(gotNeg, "|") != strings.Join(wantNeg, "|") {
		t.Fatalf("got %v, want %v", gotNeg, wantNeg)
	}
}

func TestOfflineCLICommands(t *testing.T) {
	const pk = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	const addr = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

	// myaddress
	if out := runCLI(t, "myaddress", "--pk", pk); out != addr {
		t.Fatalf("myaddress = %q, want %q", out, addr)
	}

	// keccak
	if out := runCLI(t, "keccak", "transfer(address,uint256)"); !strings.HasPrefix(out, "0xa9059cbb") {
		t.Fatalf("keccak = %q", out)
	}

	// units
	unitsOut := runCLI(t, "--format", "json", "units", "1.5eth")
	if !strings.Contains(unitsOut, `"wei": "1500000000000000000"`) {
		t.Fatalf("unexpected units json: %s", unitsOut)
	}

	// sign + verify-sig
	sig := runCLI(t, "sign", "Ship it!", "--pk", pk)
	verified := runCLI(t, "verify-sig", "--message", "Ship it!", "--signature", sig, "--address", addr)
	if verified != addr {
		t.Fatalf("verify-sig got %q, want %q", verified, addr)
	}

	// encode + decode
	encoded := runCLI(t, "encode", "transfer(address,uint256)", addr, "1000")
	decoded := runCLI(t, "decode", "transfer(address,uint256)", encoded)
	if !strings.Contains(decoded, addr) || !strings.Contains(decoded, "1000") {
		t.Fatalf("unexpected decoded output: %s", decoded)
	}

	// contract list (erc721 & erc1155)
	erc721List := runCLI(t, "contract", "list", "--abi", "erc721")
	if !strings.Contains(erc721List, "tokenURI") || !strings.Contains(erc721List, "safeTransferFrom") {
		t.Fatalf("erc721 list missing standard EIP-721 methods:\n%s", erc721List)
	}
	erc1155List := runCLI(t, "contract", "list", "--abi", "erc1155")
	if !strings.Contains(erc1155List, "balanceOfBatch") {
		t.Fatalf("erc1155 list missing balanceOfBatch:\n%s", erc1155List)
	}

	// flatten + generate code in temp dir
	tmp := t.TempDir()
	baseSol := filepath.Join(tmp, "Base.sol")
	childSol := filepath.Join(tmp, "Child.sol")
	_ = os.WriteFile(baseSol, []byte("// SPDX-License-Identifier: MIT\npragma solidity ^0.8.20;\nerror CustomErr();\nabstract contract Base { uint256 public x; }\n"), 0644)
	_ = os.WriteFile(childSol, []byte("// SPDX-License-Identifier: MIT\npragma solidity ^0.8.20;\nimport {Base, CustomErr} from \"./Base.sol\";\ncontract Child is Base {}\n"), 0644)

	flatOut := filepath.Join(tmp, "Child_flat.sol")
	runCLI(t, "contract", "flatten", childSol, "-o", flatOut)
	flatBytes, err := os.ReadFile(flatOut)
	if err != nil {
		t.Fatal(err)
	}
	flatStr := string(flatBytes)
	if !strings.Contains(flatStr, "error CustomErr();") || !strings.Contains(flatStr, "contract Child is Base") {
		t.Fatalf("flattened output missing declarations:\n%s", flatStr)
	}
	if strings.Count(flatStr, "SPDX-License-Identifier") != 1 || strings.Count(flatStr, "pragma solidity") != 1 {
		t.Fatalf("expected single SPDX and pragma in flattened output:\n%s", flatStr)
	}

	// generate code
	bindOut := filepath.Join(tmp, "erc20_bind.go")
	runCLI(t, "generate", "code", "--abi", "erc20", "--pkg", "bindings", "--type", "ERC20Token", "--out", bindOut)
	if _, err := os.Stat(bindOut); err != nil {
		t.Fatalf("expected generated binding at %s: %v", bindOut, err)
	}
}

func TestOnlineCLICommandsWithMockRPC(t *testing.T) {
	const pk = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	implAddr := web3.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		method, _ := req["method"].(string)
		id := req["id"]
		reply := func(res interface{}) {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": id, "result": res})
		}

		switch method {
		case "eth_chainId":
			reply("0x3c")
		case "eth_getBalance":
			reply("0xde0b6b3a7640000") // 1.0
		case "eth_getStorageAt":
			params := req["params"].([]interface{})
			if strings.EqualFold(params[1].(string), assets.EIP1967ImplementationSlot) {
				reply(web3.BytesToHash(implAddr[:]).Hex())
				return
			}
			reply(web3.ZeroHash.Hex())
		case "eth_getTransactionByHash":
			// Pending tx for increasegas & cancel tests
			reply(map[string]interface{}{
				"type":     "0x2",
				"chainId":  "0x3c",
				"nonce":    "0x9",
				"gas":      "0x5208",
				"maxFeePerGas":         "0x77359400",
				"maxPriorityFeePerGas": "0x3b9aca00",
				"to":       implAddr.Hex(),
				"value":    "0x0",
				"input":    "0x",
				"from":     "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
				"v":        "0x0",
				"r":        "0x1",
				"s":        "0x1",
				"hash":     "0x2222222222222222222222222222222222222222222222222222222222222222",
			})
		case "eth_sendRawTransaction":
			rawHex := req["params"].([]interface{})[0].(string)
			reply(web3.Keccak256Hash(web3.FromHex(rawHex)).Hex())
		}
	}))
	defer srv.Close()

	balOut := runCLI(t, "--rpc-url", srv.URL, "balance", implAddr.Hex())
	if balOut != "1" {
		t.Fatalf("balance = %q, want 1", balOut)
	}

	targetOut := runCLI(t, "--rpc-url", srv.URL, "contract", "target", "--address", "0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	if targetOut != implAddr.Hex() {
		t.Fatalf("contract target = %q, want %q", targetOut, implAddr.Hex())
	}

	// increasegas with decimal GWEI (previously broken!)
	incOut := runCLI(t, "--rpc-url", srv.URL, "increasegas", "--pk", pk, "--tx", "0x2222222222222222222222222222222222222222222222222222222222222222", "--amount", "1.5")
	if !strings.Contains(incOut, "Replaced pending transaction (nonce 9)") {
		t.Fatalf("unexpected increasegas output: %s", incOut)
	}

	// cancel pending tx
	cancelOut := runCLI(t, "--rpc-url", srv.URL, "cancel", "--pk", pk, "--tx", "0x2222222222222222222222222222222222222222222222222222222222222222")
	if !strings.Contains(cancelOut, "Submitted cancellation transaction (nonce 9)") {
		t.Fatalf("unexpected cancel output: %s", cancelOut)
	}
}
