package web3_types

import (
	"fmt"
	"math"
	"math/big"

	"github.com/shopspring/decimal"
)

var (
	weiPerGO   = big.NewInt(1e18)
	weiPerGwei = big.NewInt(1e9)
)

// Base converts b base units to wei (*1e18).
func Base(b int64) *big.Int {
	i := big.NewInt(b)
	return i.Mul(i, weiPerGO)
}

// Gwei converts g gwei to wei (*1e9).
func Gwei(g int64) *big.Int {
	i := big.NewInt(g)
	return i.Mul(i, weiPerGwei)
}

// WeiAsBase converts w wei in to the base unit, and formats it as a decimal fraction with full precision (up to 18 decimals).
func WeiAsBase(w *big.Int) string {
	return new(big.Rat).SetFrac(w, weiPerGO).FloatString(18)
}

// WeiAsGwei converts w wei in to gwei, and formats it as a decimal fraction with full precision (up to 9 decimals).
func WeiAsGwei(w *big.Int) string {
	return new(big.Rat).SetFrac(w, weiPerGwei).FloatString(9)
}

// IntAsFloat converts a *big.Int (ie: wei), to *big.Float (ie: ETH)
func IntAsFloat(i *big.Int, decimals int) *big.Float {
	f := new(big.Float)
	f.SetPrec(100)
	f.SetInt(i)
	f.Quo(f, big.NewFloat(math.Pow10(decimals)))
	return f
}

// DecToInt converts a decimal to a big int
func DecToInt(d decimal.Decimal, decimals int32) *big.Int {
	// multiply amount by number of decimals
	d1 := decimal.New(1, decimals)
	d = d.Mul(d1)
	return d.BigInt()
}

// IntToDec converts a big int to a decimal
func IntToDec(i *big.Int, decimals int32) decimal.Decimal {
	d := decimal.NewFromBigInt(i, 0)
	d = d.Div(decimal.New(1, decimals))
	return d
}

// FloatAsInt converts a float to a *big.Int based on the decimals passed in
func FloatAsInt(amountF *big.Float, decimals int) *big.Int {
	bigval := new(big.Float)
	bigval.SetPrec(100)
	bigval.SetString(amountF.String()) // have to do this to not lose precision

	coinDecimals := new(big.Float)
	coinDecimals.SetFloat64(math.Pow10(decimals))
	bigval.Mul(bigval, coinDecimals)

	amountI := new(big.Int)
	// todo: could sanity check the accuracy here
	bigval.Int(amountI) // big.NewInt(int64(amountInWeiF)) // amountInGo.Mul(amountInGo, big.NewInt(int64(math.Pow10(18))))
	return amountI
}

// ConvertInt converts a big.Int in to the provided type.
func ConvertInt(signed bool, size int, i *big.Int) (interface{}, error) {
	if signed {
		switch {
		case size > 64:
			return i, nil
		case size > 32:
			if !i.IsInt64() {
				return nil, fmt.Errorf("integer overflows int64: %s", i)
			}
			return i.Int64(), nil
		case size > 16:
			if !i.IsInt64() || i.Int64() > math.MaxInt32 {
				return nil, fmt.Errorf("integer overflows int32: %s", i)
			}
			return int32(i.Int64()), nil
		case size > 8:
			if !i.IsInt64() || i.Int64() > math.MaxInt16 {
				return nil, fmt.Errorf("integer overflows int16: %s", i)
			}
			return int16(i.Int64()), nil
		default:
			if !i.IsInt64() || i.Int64() > math.MaxInt8 {
				return nil, fmt.Errorf("integer overflows int8: %s", i)
			}
			return int8(i.Int64()), nil
		}
	} else {
		switch {
		case size > 64:
			if i.Sign() == -1 {
				return nil, fmt.Errorf("negative value in unsigned field: %s", i)
			}
			return i, nil
		case size > 32:
			if !i.IsUint64() {
				return nil, fmt.Errorf("integer overflows uint64: %s", i)
			}
			return i.Uint64(), nil
		case size > 16:
			if !i.IsUint64() || i.Uint64() > math.MaxUint32 {
				return nil, fmt.Errorf("integer overflows uint32: %s", i)
			}
			return uint32(i.Uint64()), nil
		case size > 8:
			if !i.IsUint64() || i.Uint64() > math.MaxUint16 {
				return nil, fmt.Errorf("integer overflows uint16: %s", i)
			}
			return uint16(i.Uint64()), nil
		default:
			if !i.IsUint64() || i.Uint64() > math.MaxUint8 {
				return nil, fmt.Errorf("integer overflows uint8: %s", i)
			}
			return uint8(i.Uint64()), nil
		}
	}
}
