package web3_types

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// ParseAmount parses a string (human readable amount with units ie 1go, 1nanogo...) and returns big.Int value of this string in wei/atto
func ParseAmount(amount string) (*big.Int, error) {
	var ret = new(big.Int)
	var mul = big.NewInt(1)
	amount = strings.ToLower(amount)
	switch {
	case strings.HasSuffix(amount, "nanogo"):
		amount = strings.TrimSuffix(amount, "nanogo")
		mul = weiPerGwei
	case strings.HasSuffix(amount, "gwei"):
		amount = strings.TrimSuffix(amount, "gwei")
		mul = weiPerGwei
	case strings.HasSuffix(amount, "attogo"):
		amount = strings.TrimSuffix(amount, "attogo")
	case strings.HasSuffix(amount, "wei"):
		amount = strings.TrimSuffix(amount, "wei")
	case strings.HasSuffix(amount, "eth"):
		amount = strings.TrimSuffix(amount, "eth")
		mul = weiPerGO
	default:
		amount = strings.TrimSuffix(amount, "go")
		mul = weiPerGO
	}
	val, err := ParseBigInt(amount)
	if err != nil {
		return nil, err
	}
	return ret.Mul(val, mul), nil
}

// ParseBigInt parses a string (base 10 only) and returns big.Int value of this string in wei/atto
func ParseBigInt(value string) (*big.Int, error) {
	if value == "" {
		return nil, errors.New("Cannot parse empty string")
	}
	i, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("Failed to parse integer %q", value)
	}
	return i, nil
}

func ParseGwei(g string) (*big.Int, error) {
	return parseUnit(g, weiPerGwei, 9)
}

func ParseBase(b string) (*big.Int, error) {
	return parseUnit(b, weiPerGO, 18)
}

func parseUnit(g string, mult *big.Int, digits int) (*big.Int, error) {
	g = strings.TrimSpace(g)
	if len(g) == 0 {
		return nil, errors.New("empty value")
	}
	parts := strings.Split(g, ".")
	whole, ok := new(big.Int).SetString(parts[0], 10)
	if !ok {
		return nil, fmt.Errorf("failed to integer part: %s", parts[0])
	}
	whole = whole.Mul(whole, mult)
	if len(parts) == 1 {
		return whole, nil
	}
	if len(parts) > 2 {
		return nil, errors.New("invalid value: more than one decimal point")
	}
	decStr := parts[1]
	if len(decStr) > digits {
		return nil, fmt.Errorf("too many decimal digits %d: limit %d", len(decStr), digits)
	}
	// Parse right padded with 0s, so we get wei.
	dec, ok := new(big.Int).SetString(decStr+strings.Repeat("0", digits-len(decStr)), 10)
	if !ok {
		return nil, fmt.Errorf("failed to decimal part: %s", decStr)
	}
	return whole.Add(whole, dec), nil
}
