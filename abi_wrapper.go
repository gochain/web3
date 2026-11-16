package web3

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gochain/web3/assets"
)

var bundledContracts = map[string]string{
	"erc20":   assets.ERC20ABI,
	"erc721":  assets.ERC721ABI,
	"erc1155": assets.ERC1155ABI,
	"proxy":   assets.EIP1967ProxyABI,
	"eip1967": assets.EIP1967ProxyABI,
}

// GetABI resolves a built-in ABI name ("erc20", "erc721", "erc1155", "proxy"),
// an inline JSON ABI array, a human-readable signature, a local file path, or an HTTP(S) URL.
func GetABI(abiSpec string) (*ABI, error) {
	spec := strings.TrimSpace(abiSpec)
	if spec == "" {
		return nil, fmt.Errorf("empty ABI specification")
	}

	if builtin, err := ABIBuiltIn(strings.ToLower(spec)); err != nil {
		return nil, fmt.Errorf("cannot get bundled ABI %q: %w", spec, err)
	} else if builtin != nil {
		return builtin, nil
	}

	if strings.HasPrefix(spec, "[") {
		parsed, err := ParseABI(spec)
		if err != nil {
			return nil, err
		}
		return &parsed, nil
	}

	if isValidURL(spec) {
		return ABIOpenURL(spec)
	}

	if info, statErr := os.Stat(spec); statErr == nil && !info.IsDir() {
		return ABIOpenFile(spec)
	} else if os.IsPermission(statErr) {
		return nil, statErr
	}

	// Support inline human-readable Solidity function signature, e.g. "balanceOf(address):(uint256)"
	if strings.Contains(spec, "(") && strings.Contains(spec, ")") {
		if m, err := ParseHumanSignature(spec); err == nil {
			parsed := &ABI{
				Methods:   map[string]ABIMethod{m.Name: m, m.Sig(): m},
				Overloads: map[string][]ABIMethod{m.Name: {m}},
				Events:    map[string]ABIEvent{},
				Errors:    map[string]ABIError{},
			}
			return parsed, nil
		}
	}

	return nil, fmt.Errorf("ABI %q not found as bundled name, file, URL, or function signature", abiSpec)
}

func ABIBuiltIn(name string) (*ABI, error) {
	if val, ok := bundledContracts[name]; ok {
		return readABI(strings.NewReader(val))
	}
	return nil, nil
}

func ABIOpenFile(abiFile string) (*ABI, error) {
	f, err := os.Open(abiFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readABI(f)
}

func ABIOpenURL(abiURL string) (*ABI, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, abiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching ABI from URL: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error fetching ABI (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	return readABI(resp.Body)
}

func readABI(reader io.Reader) (*ABI, error) {
	parsed, err := ParseABIJSON(reader)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
