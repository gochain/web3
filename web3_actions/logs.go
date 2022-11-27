package web3_actions

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/common/hexutil"
	"github.com/gochain/gochain/v4/core/types"
	zlog "github.com/rs/zerolog/log"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

func getInputs(args abi.Arguments, indexed bool) []abi.Argument {
	var out []abi.Argument
	for _, arg := range args {
		if arg.Indexed == indexed {
			out = append(out, arg)
		}
	}
	return out
}

// ParseLogs func ParseReceipt(myabi abi.ABI, receipt *Receipt) (map[string]map[string]interface{}, error) {
func ParseLogs(myabi abi.ABI, logs []*types.Log) ([]web3_types.Event, error) {
	var output []web3_types.Event
	// output := make(map[string]map[string]interface{})
	for _, log := range logs {
		var out []interface{}
		//event id is always in the first topic
		event := FindEventById(myabi, log.Topics[0])
		fields := make(map[string]interface{})
		//TODO use event.Inputs.UnpackIntoMap instead when available
		nonIndexed := getInputs(event.Inputs, false)
		for _, t := range nonIndexed {
			x, err := convertOutputParameter(t)
			if err != nil {
				zlog.Err(err).Msg("ParseLogs: convertOutputParameter")
				return nil, err
			}
			out = append(out, x)
		}
		if len(nonIndexed) > 1 {
			out, err := myabi.Unpack(event.Name, log.Data)
			if err != nil {
				zlog.Err(err).Msg("ParseLogs: Unpack")
				return nil, err
			}
			for i, o := range out {
				fields[nonIndexed[i].Name] = reflect.ValueOf(o).Elem().Interface()
			}
		} else if len(out) > 0 {
			o, err := myabi.Unpack(event.Name, log.Data)
			if err != nil {
				zlog.Err(err).Msg("ParseLogs: Unpack")
				return nil, err
			}
			fields[nonIndexed[0].Name] = o
		}

		for i, input := range getInputs(event.Inputs, true) {
			fields[input.Name] = log.Topics[i+1].String()
		}
		output = append(output, web3_types.Event{Name: event.Name, Fields: fields})
	}
	return output, nil
}

// TODO Deprecated: prefer built-in UnpackValues() func and convertOutputParams.
func convertOutputParameter(t abi.Argument) (interface{}, error) {
	switch t.Type.T {
	case abi.BoolTy:
		return new(bool), nil
	case abi.UintTy:
		switch size := t.Type.Size; {
		case size > 64:
			i := new(big.Int)
			return &i, nil
		case size > 32:
			return new(uint64), nil
		case size > 16:
			return new(uint32), nil
		case size > 8:
			return new(uint16), nil
		default:
			return new(uint8), nil
		}
	case abi.IntTy:
		switch size := t.Type.Size; {
		case size > 64:
			i := new(big.Int)
			return &i, nil
		case size > 32:
			return new(int64), nil
		case size > 16:
			return new(int32), nil
		case size > 8:
			return new(int16), nil
		default:
			return new(int8), nil
		}
	case abi.StringTy:
		return new(string), nil
	case abi.AddressTy:
		return new(common.Address), nil
	case abi.BytesTy:
		return new(hexutil.Bytes), nil
	case abi.FixedBytesTy:
		switch size := t.Type.Size; {
		case size == 32:
			return new(common.Hash), nil
		default:
			return nil, fmt.Errorf("unsupported output byte array size %v", size)
		}
	default:
		return nil, fmt.Errorf("unsupported output type %v", t.Type.T)
	}
}
