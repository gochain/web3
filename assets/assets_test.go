package assets

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
)

func TestBundledABIsValidJSON(t *testing.T) {
	for name, raw := range map[string]string{
		"ERC20ABI":        ERC20ABI,
		"ERC721ABI":       ERC721ABI,
		"ERC1155ABI":      ERC1155ABI,
		"EIP1967ProxyABI": EIP1967ProxyABI,
	} {
		var parsed []map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			t.Fatalf("%s invalid JSON: %v", name, err)
		}
		if len(parsed) == 0 {
			t.Fatalf("%s is empty", name)
		}
	}
}

func TestOpenZeppelinV5Generators(t *testing.T) {
	erc20Src, err := GenERC20(context.Background(), "5.3.0", &Erc20Params{
		Symbol:        "TEST",
		TokenName:     "Test Token",
		Decimals:      6,
		InitialSupply: big.NewInt(1000000),
		Mintable:      true,
		Burnable:      true,
		Pausable:      true,
	})
	if err != nil {
		t.Fatalf("GenERC20 failed: %v", err)
	}
	if !strings.Contains(erc20Src, "Ownable(msg.sender)") || !strings.Contains(erc20Src, "return 6;") || !strings.Contains(erc20Src, "1000000 * 10 ** decimals()") {
		t.Fatalf("unexpected ERC20 output:\n%s", erc20Src)
	}

	erc20ZeroDec, err := GenERC20(context.Background(), "5.3.0", &Erc20Params{
		Symbol:      "ZERO",
		TokenName:   "Zero Decimals",
		Decimals:    0,
		DecimalsSet: true,
	})
	if err != nil || !strings.Contains(erc20ZeroDec, "return 0;") {
		t.Fatalf("expected 0 decimals override, got err=%v:\n%s", err, erc20ZeroDec)
	}

	// ERC721 without --base-uri (matching README quickstart)
	erc721Src, err := GenERC721("5.3.0", &Erc721Params{
		Symbol:    "CAT",
		TokenName: "Kitties",
		Mintable:  true,
		Burnable:  true,
	})
	if err != nil {
		t.Fatalf("GenERC721 failed: %v", err)
	}
	if !strings.Contains(erc721Src, "contract CAT is ERC721, ERC721Burnable, Ownable") || strings.Contains(erc721Src, "_baseURI") {
		t.Fatalf("unexpected ERC721 output:\n%s", erc721Src)
	}

	erc1155Src, err := GenERC1155("5.3.0", &Erc1155Params{
		Symbol:   "GAME",
		URI:      "https://api.example.com/item/{id}.json",
		Mintable: true,
	})
	if err != nil {
		t.Fatalf("GenERC1155 failed: %v", err)
	}
	if !strings.Contains(erc1155Src, "contract GAME is ERC1155, Ownable") {
		t.Fatalf("unexpected ERC1155 output:\n%s", erc1155Src)
	}
}
