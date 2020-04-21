package web3

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/gochain/gochain/v3/accounts/abi"
	"github.com/gochain/gochain/v3/common"
)

func Test_parseParam(t *testing.T) {
	const addr = "0xa25b5e2d2d63dad7fa940e239925f29320f5103d"
	const hash = "0x0123456789012345678901234567890101234567890123456789012345678901"
	tests := []struct {
		name  string
		t     byte
		s     int
		param interface{}

		want    interface{}
		wantErr bool
	}{
		{"int256<-int", abi.IntTy, 256, 1, big.NewInt(1), false},
		{"int256<-big.Int", abi.IntTy, 256, big.NewInt(1), big.NewInt(1), false},

		{"uint256<-int", abi.UintTy, 256, 1, big.NewInt(1), false},
		{"uint256<-big.Int", abi.UintTy, 256, big.NewInt(1), big.NewInt(1), false},

		{"int8<-int", abi.IntTy, 8, 1, int8(1), false},
		{"int8<-big.Int", abi.IntTy, 8, big.NewInt(1), int8(1), false},

		{"uint8<-int", abi.UintTy, 8, 1, uint8(1), false},
		{"uint8<-big.Int", abi.UintTy, 8, big.NewInt(1), uint8(1), false},

		{"int256<-hex", abi.IntTy, 256, "0x1", big.NewInt(1), false},
		{"int256<-string", abi.IntTy, 256, "1", big.NewInt(1), false},

		{"uint256<-zero", abi.UintTy, 256, "0", big.NewInt(0), false},
		{"uint256<-json", abi.UintTy, 64, json.Number("10000000000000001"), uint64(10000000000000001), false},

		{"address<-address", abi.AddressTy, 0, common.HexToAddress(addr), common.HexToAddress(addr), false},
		{"address<-hex", abi.AddressTy, 0, addr, common.HexToAddress(addr), false},

		{"hash<-hash", abi.FixedBytesTy, 32, common.HexToHash(hash), common.HexToHash(hash), false},
		{"hash<-hex", abi.FixedBytesTy, 32, hash, common.HexToHash(hash), false},

		{"bytes<-bytes", abi.BytesTy, 0, common.Hex2Bytes("1234"), common.Hex2Bytes("1234"), false},
		{"bytes<-hex", abi.BytesTy, 0, "0x1234", common.Hex2Bytes("1234"), false},

		// Error cases:
		{"uint256<-negative", abi.UintTy, 256, -1, nil, true},
		{"uint8<-negative", abi.UintTy, 8, -1, nil, true},
		{"int256<-float64", abi.IntTy, 256, float64(1), nil, true},
		{"uint8<-float", abi.UintTy, 8, 1.1, nil, true},
		{"uint8<-negative-float", abi.UintTy, 8, -1.1, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertArgument(tt.t, tt.s, tt.param)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr %v; error = %v", tt.wantErr, err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got (%T): %v; want (%T): %v", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestParseGwei(t *testing.T) {
	for _, tt := range []struct {
		val    string
		exp    *big.Int
		expErr bool
	}{
		{val: "1", exp: weiPerGwei},
		{val: "10", exp: Gwei(10)},
		{val: "1.1", exp: new(big.Int).Add(Gwei(1), big.NewInt(100000000))},
		{val: "100000", exp: Gwei(100000)},
		{val: "1.000000001", exp: new(big.Int).Add(Gwei(1), big.NewInt(1))},
		{val: "1.0000000001", expErr: true},
	} {
		t.Run(tt.val, func(t *testing.T) {
			got, err := ParseGwei(tt.val)
			if err != nil {
				if !tt.expErr {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if tt.expErr {
				t.Errorf("expected error, but got: %s", got)
				return
			}
			if got.Cmp(tt.exp) != 0 {
				t.Errorf("expected %s but got %s", tt.exp, got)
			}
		})
	}
}
