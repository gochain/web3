package web3_types

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/common/hexutil"
	"github.com/rs/zerolog/log"
)

// ConvertArguments attempts to convert each param to the matching args type.
// Unrecognized param types are passed through unmodified.
//
// Note: The encoding/json package uses float64 for numbers by default, which is inaccurate
// for many web3 types, and unsupported here. The json.Decoder method UseNumber() will
// switch to using json.Number instead, which is accurate (full precision, backed by the
// original string) and supported here.
func ConvertArguments(args abi.Arguments, params []interface{}) ([]interface{}, error) {
	if len(args) != len(params) {
		err := fmt.Errorf("mismatched argument (%d) and parameter (%d) counts", len(args), len(params))
		log.Err(err).Msg("ConvertArguments")
		return nil, err
	}
	var convertedParams []interface{}
	for i, input := range args {
		param, err := ConvertArgument(input.Type, params[i])
		if err != nil {
			log.Err(err).Msg("ConvertArguments: ConvertArgument")
			return nil, err
		}
		convertedParams = append(convertedParams, param)
	}
	return convertedParams, nil
}

// ConvertArgument attempts to convert argument to the provided ABI type and size.
// Unrecognized types are passed through unmodified.
func ConvertArgument(abiType abi.Type, param interface{}) (interface{}, error) {
	size := abiType.Size
	// fmt.Println("INPUT TYPE:", abiType, "SIZE:", size, "Param", param)
	switch abiType.T {
	case abi.StringTy:
	case abi.BoolTy:
		if s, ok := param.(string); ok {
			val, err := strconv.ParseBool(s)
			if err != nil {
				return nil, fmt.Errorf("failed to parse bool %q: %v", s, err)
			}
			return val, nil
		}
	case abi.UintTy, abi.IntTy:
		if j, ok := param.(json.Number); ok {
			param = string(j)
		}
		if s, ok := param.(string); ok {
			val, ok := new(big.Int).SetString(s, 0)
			if !ok {
				return nil, fmt.Errorf("failed to parse big.Int: %s", s)
			}
			return ConvertInt(abiType.T == abi.IntTy, size, val)
		} else if i, ok := param.(*big.Int); ok {
			return ConvertInt(abiType.T == abi.IntTy, size, i)
		}
		v := reflect.ValueOf(param)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i := new(big.Int).SetInt64(v.Int())
			return ConvertInt(abiType.T == abi.IntTy, size, i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			i := new(big.Int).SetUint64(v.Uint())
			return ConvertInt(abiType.T == abi.IntTy, size, i)
		case reflect.Float64, reflect.Float32:
			return nil, fmt.Errorf("floating point numbers are not valid in web3 - please use an integer or string instead (including big.Int and json.Number)")
		}
	case abi.AddressTy:
		if s, ok := param.(string); ok {
			if !common.IsHexAddress(s) {
				return nil, fmt.Errorf("invalid hex address: %s", s)
			}
			return common.HexToAddress(s), nil
		}
	case abi.SliceTy, abi.ArrayTy:
		s, ok := param.(string)
		if !ok {
			return nil, fmt.Errorf("invalid array: %s", s)
		}
		s = strings.TrimPrefix(s, "[")
		s = strings.TrimSuffix(s, "]")
		inputArray := strings.Split(s, ",")
		switch abiType.Elem.T {

		case abi.AddressTy:
			arrayParams := make([]common.Address, len(inputArray))
			for i, elem := range inputArray {
				converted, err := ConvertArgument(*abiType.Elem, elem)
				if err != nil {
					return nil, err
				}
				arrayParams[i] = converted.(common.Address)
			}
			return arrayParams, nil

		case abi.StringTy:
			arrayParams := make([]string, len(inputArray))
			for i, elem := range inputArray {
				converted, err := ConvertArgument(*abiType.Elem, elem)
				if err != nil {
					return nil, err
				}
				arrayParams[i] = converted.(string)
			}
			return arrayParams, nil

		case abi.BoolTy:
			arrayParams := make([]bool, len(inputArray))
			for i, elem := range inputArray {
				converted, err := ConvertArgument(*abiType.Elem, elem)
				if err != nil {
					return nil, err
				}
				arrayParams[i] = converted.(bool)
			}
			return arrayParams, nil

		default:
			arrayParams := make([]int, len(inputArray))
			for i, elem := range inputArray {
				converted, err := ConvertArgument(*abiType.Elem, elem)
				if err != nil {
					return nil, err
				}
				arrayParams[i] = converted.(int)
			}
			return arrayParams, nil
		}

	case abi.BytesTy:
		if s, ok := param.(string); ok {
			val, err := hexutil.Decode(s)
			if err != nil {
				return nil, fmt.Errorf("failed to parse bytes %q: %v", s, err)
			}
			return val, nil
		}
	case abi.HashTy:
		if s, ok := param.(string); ok {
			val, err := hexutil.Decode(s)
			if err != nil {
				return nil, fmt.Errorf("failed to parse hash %q: %v", s, err)
			}
			if len(val) != common.HashLength {
				return nil, fmt.Errorf("invalid hash length %d:hash must be 32 bytes", len(val))
			}
			return common.BytesToHash(val), nil
		}
	case abi.FixedBytesTy:
		switch {
		case size == 32:
			if s, ok := param.(string); ok {
				val, err := hexutil.Decode(s)
				if err != nil {
					return nil, fmt.Errorf("failed to parse hash %q: %v", s, err)
				}
				if len(val) != common.HashLength {
					return nil, fmt.Errorf("invalid hash length %d:hash must be 32 bytes", len(val))
				}
				return common.BytesToHash(val), nil
			}
		default:
			if s, ok := param.(string); ok {
				fmt.Println(s)
				val, err := hexutil.Decode(s)
				if err != nil {
					return nil, fmt.Errorf("failed to parse hash %q: %v", s, err)
				}
				if len(val) != size {
					return nil, fmt.Errorf("invalid byte array length %d: size is %d bytes", len(val), size)
				}
				arrayT := reflect.ArrayOf(size, reflect.TypeOf(byte(0)))
				array := reflect.New(arrayT).Elem()
				reflect.Copy(array, reflect.ValueOf(val))
				return array.Interface(), nil
			}
		}
	default:
		return nil, fmt.Errorf("unsupported input type %v", abiType)
	}
	return param, nil
}
