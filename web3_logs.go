package web3

import (
	"reflect"

	zlog "github.com/rs/zerolog/log"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/core/types"
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
func ParseLogs(myabi abi.ABI, logs []*types.Log) ([]Event, error) {
	var output []Event
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
		output = append(output, Event{Name: event.Name, Fields: fields})
	}
	return output, nil
}
