package web3

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/shopspring/decimal"
)

var (
	weiPerGO   = big.NewInt(1e18)
	weiPerGwei = big.NewInt(1e9)
)

// Base converts b base units (ETH/GO) to wei (*1e18).
func Base(b int64) *big.Int {
	i := big.NewInt(b)
	return i.Mul(i, weiPerGO)
}

// Gwei converts g gwei to wei (*1e9).
func Gwei(g int64) *big.Int {
	i := big.NewInt(g)
	return i.Mul(i, weiPerGwei)
}

// WeiAsBase formats w wei in base units (ETH/GO) with 18 decimals.
func WeiAsBase(w *big.Int) string {
	if w == nil {
		return "0.000000000000000000"
	}
	return new(big.Rat).SetFrac(w, weiPerGO).FloatString(18)
}

// WeiAsGwei formats w wei in gwei with 9 decimals.
func WeiAsGwei(w *big.Int) string {
	if w == nil {
		return "0.000000000"
	}
	return new(big.Rat).SetFrac(w, weiPerGwei).FloatString(9)
}

// IntAsFloat converts a *big.Int (wei) to *big.Float given decimals.
func IntAsFloat(i *big.Int, decimals int) *big.Float {
	f := new(big.Float).SetPrec(100).SetInt(i)
	return f.Quo(f, big.NewFloat(math.Pow10(decimals)))
}

// DecToInt scales decimal d by 10^decimals and returns the truncated *big.Int.
func DecToInt(d decimal.Decimal, decimals int32) *big.Int {
	return d.Mul(decimal.New(1, decimals)).BigInt()
}

// IntToDec divides *big.Int i by 10^decimals into a decimal.Decimal.
func IntToDec(i *big.Int, decimals int32) decimal.Decimal {
	if i == nil {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(i, -decimals)
}

// FloatAsInt scales *big.Float amountF by 10^decimals into *big.Int.
func FloatAsInt(amountF *big.Float, decimals int) *big.Int {
	bigval := new(big.Float).SetPrec(100)
	bigval.SetString(amountF.String())

	coinDecimals := new(big.Float).SetFloat64(math.Pow10(decimals))
	bigval.Mul(bigval, coinDecimals)

	amountI := new(big.Int)
	bigval.Int(amountI)
	return amountI
}

// ParseAmount parses a human-readable amount with optional unit suffix (wei, attogo, gwei, nanogo, eth, go) into wei.
func ParseAmount(amount string) (*big.Int, error) {
	s := strings.ToLower(strings.TrimSpace(amount))
	if s == "" {
		return nil, errors.New("cannot parse empty amount")
	}
	switch {
	case strings.HasSuffix(s, "nanogo"):
		return ParseGwei(strings.TrimSuffix(s, "nanogo"))
	case strings.HasSuffix(s, "gwei"):
		return ParseGwei(strings.TrimSuffix(s, "gwei"))
	case strings.HasSuffix(s, "attogo"):
		return ParseBigInt(strings.TrimSpace(strings.TrimSuffix(s, "attogo")))
	case strings.HasSuffix(s, "wei"):
		return ParseBigInt(strings.TrimSpace(strings.TrimSuffix(s, "wei")))
	case strings.HasSuffix(s, "ether"):
		return ParseBase(strings.TrimSuffix(s, "ether"))
	case strings.HasSuffix(s, "eth"):
		return ParseBase(strings.TrimSuffix(s, "eth"))
	case strings.HasSuffix(s, "go"):
		return ParseBase(strings.TrimSuffix(s, "go"))
	default:
		return ParseBase(s)
	}
}

// ParseBigInt parses a base-10 integer string into *big.Int.
func ParseBigInt(value string) (*big.Int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("cannot parse empty string")
	}
	i, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse integer %q", value)
	}
	return i, nil
}

// ParseGwei parses a decimal gwei string (up to 9 fractional digits) into wei.
func ParseGwei(g string) (*big.Int, error) {
	return parseUnit(g, weiPerGwei, 9)
}

// ParseBase parses a decimal ETH/GO string (up to 18 fractional digits) into wei.
func ParseBase(b string) (*big.Int, error) {
	return parseUnit(b, weiPerGO, 18)
}

func parseUnit(g string, mult *big.Int, digits int) (*big.Int, error) {
	g = strings.TrimSpace(g)
	if len(g) == 0 {
		return nil, errors.New("empty value")
	}
	if strings.HasPrefix(g, "-") {
		return nil, fmt.Errorf("negative unit value not allowed: %s", g)
	}
	parts := strings.Split(g, ".")
	if len(parts) > 2 {
		return nil, errors.New("invalid value: more than one decimal point")
	}
	wholeStr := parts[0]
	if wholeStr == "" {
		wholeStr = "0"
	}
	whole, ok := new(big.Int).SetString(wholeStr, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse integer part: %s", parts[0])
	}
	whole.Mul(whole, mult)
	if len(parts) == 1 {
		return whole, nil
	}
	decStr := parts[1]
	if len(decStr) == 0 {
		return whole, nil
	}
	if len(decStr) > digits {
		return nil, fmt.Errorf("too many decimal digits %d: limit %d", len(decStr), digits)
	}
	dec, ok := new(big.Int).SetString(decStr+strings.Repeat("0", digits-len(decStr)), 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse decimal part: %s", decStr)
	}
	return whole.Add(whole, dec), nil
}
