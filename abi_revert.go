package web3

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

var (
	revertErrorSelector = [4]byte{0x08, 0xc3, 0x79, 0xa0} // Error(string)
	panicErrorSelector  = [4]byte{0x4e, 0x48, 0x7b, 0x71} // Panic(uint256)
)

var solidityPanicReasons = map[uint64]string{
	0x00: "generic compiler panic",
	0x01: "assertion failed (assert)",
	0x11: "arithmetic overflow or underflow",
	0x12: "division or modulo by zero",
	0x21: "enum conversion out of bounds",
	0x22: "incorrectly encoded storage byte array",
	0x31: "pop() on empty array",
	0x32: "array index out of bounds",
	0x41: "out of memory",
	0x51: "called zero-initialized internal function",
}

// DecodeRevert decodes EVM revert return data into a human-readable message,
// handling standard Error(string), Panic(uint256), and custom ABI errors when myabi is provided.
func DecodeRevert(data []byte, myabi *ABI) (string, bool) {
	if len(data) < 4 {
		return "", false
	}
	var sel [4]byte
	copy(sel[:], data[:4])

	if sel == revertErrorSelector {
		strArgs := ABIArguments{{Type: ABIType{T: StringTy}}}
		if vals, err := strArgs.UnpackValues(data[4:]); err == nil && len(vals) == 1 {
			if s, ok := vals[0].(string); ok {
				return fmt.Sprintf("execution reverted: %s", s), true
			}
		}
	}

	if sel == panicErrorSelector {
		u256Args := ABIArguments{{Type: ABIType{T: UintTy, Size: 256}}}
		if vals, err := u256Args.UnpackValues(data[4:]); err == nil && len(vals) == 1 {
			if code, ok := vals[0].(*big.Int); ok && code.IsUint64() {
				u := code.Uint64()
				if desc, exists := solidityPanicReasons[u]; exists {
					return fmt.Sprintf("execution panicked (0x%02x): %s", u, desc), true
				}
				return fmt.Sprintf("execution panicked (0x%x)", u), true
			}
		}
	}

	if myabi != nil {
		for _, abiErr := range myabi.Errors {
			if abiErr.ID == sel {
				vals, err := abiErr.Inputs.UnpackValues(data[4:])
				if err != nil && len(abiErr.Inputs) > 0 {
					return fmt.Sprintf("execution reverted with custom error %s (raw: %s)", abiErr.Sig(), EncodeHex(data[4:])), true
				}
				parts := make([]string, len(vals))
				for i, v := range vals {
					if abiErr.Inputs[i].Name != "" {
						parts[i] = fmt.Sprintf("%s=%v", abiErr.Inputs[i].Name, v)
					} else {
						parts[i] = fmt.Sprintf("%v", v)
					}
				}
				return fmt.Sprintf("execution reverted: %s(%s)", abiErr.Name, strings.Join(parts, ", ")), true
			}
		}
	}

	return fmt.Sprintf("execution reverted (selector %s, data %s)", EncodeHex(sel[:]), EncodeHex(data)), true
}

// FormatRPCError enriches an RPCError with decoded Solidity revert details if revert data is present.
func FormatRPCError(err error, myabi *ABI) error {
	if err == nil {
		return nil
	}
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) {
		if revData := rpcErr.RevertData(); len(revData) > 0 {
			if msg, ok := DecodeRevert(revData, myabi); ok {
				return fmt.Errorf("%s (%w)", msg, err)
			}
		}
	}
	// Check if raw hex selector appears inline
	if bytes.Contains([]byte(err.Error()), []byte("0x08c379a0")) || bytes.Contains([]byte(err.Error()), []byte("0x4e487b71")) {
		s := err.Error()
		if idx := strings.Index(s, "0x"); idx >= 0 {
			hexPart := strings.Fields(s[idx:])[0]
			if raw, decErr := DecodeHex(strings.Trim(hexPart, `",)`)); decErr == nil {
				if msg, ok := DecodeRevert(raw, myabi); ok {
					return fmt.Errorf("%s (%w)", msg, err)
				}
			}
		}
	}
	return err
}
