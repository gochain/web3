package web3

import (
	"strconv"
	"strings"
)

func HexToInt64(hex string) (int64, error) {
	hex = strings.TrimPrefix(hex, "0x")
	return strconv.ParseInt(hex, 16, 64)
}
