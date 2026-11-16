package web3

import (
	"errors"
	"fmt"
	"math/big"
)

// UnpackValues decodes ABI-encoded output bytes according to args.
func (args ABIArguments) UnpackValues(data []byte) ([]interface{}, error) {
	if len(args) == 0 {
		return nil, nil
	}
	if len(data) == 0 {
		return nil, errors.New("abi: attempting to unmarshal empty output")
	}
	return decodeTupleValues(args, data)
}

// Unpack decodes output bytes for a named method or event.
func (a ABI) Unpack(name string, data []byte) ([]interface{}, error) {
	if m, ok := a.Methods[name]; ok {
		return m.Outputs.UnpackValues(data)
	}
	if ev, ok := a.Events[name]; ok {
		var nonIndexed ABIArguments
		for _, in := range ev.Inputs {
			if !in.Indexed {
				nonIndexed = append(nonIndexed, in)
			}
		}
		return nonIndexed.UnpackValues(data)
	}
	return nil, fmt.Errorf("abi: %q not found in methods or events", name)
}

func decodeTupleValues(args ABIArguments, data []byte) ([]interface{}, error) {
	out := make([]interface{}, len(args))
	cursor := 0

	for i, arg := range args {
		t := arg.Type
		if t.isDynamic() {
			if cursor > len(data)-32 {
				return nil, fmt.Errorf("abi: truncated offset at arg %d (%s)", i, t.CanonicalString())
			}
			offset, err := readOffset(data[cursor : cursor+32])
			if err != nil {
				return nil, err
			}
			if offset > len(data) {
				return nil, fmt.Errorf("abi: offset %d out of bounds (len %d)", offset, len(data))
			}
			val, err := decodeDynamicValue(t, data[offset:])
			if err != nil {
				return nil, fmt.Errorf("arg %d (%s): %w", i, t.CanonicalString(), err)
			}
			out[i] = val
			cursor += 32
		} else {
			sz := t.staticByteSize()
			if sz > len(data)-cursor {
				return nil, fmt.Errorf("abi: truncated static value at arg %d (%s)", i, t.CanonicalString())
			}
			val, err := decodeStaticValue(t, data[cursor:cursor+sz])
			if err != nil {
				return nil, fmt.Errorf("arg %d (%s): %w", i, t.CanonicalString(), err)
			}
			out[i] = val
			cursor += sz
		}
	}
	return out, nil
}

func decodeStaticValue(t ABIType, chunk []byte) (interface{}, error) {
	switch t.T {
	case UintTy:
		bi := new(big.Int).SetBytes(chunk[:32])
		size := t.Size
		if size == 0 {
			size = 256
		}
		return ConvertInt(false, size, bi)

	case IntTy:
		bi := new(big.Int).SetBytes(chunk[:32])
		if chunk[0]&0x80 != 0 {
			bi.Sub(bi, twoTo256)
		}
		size := t.Size
		if size == 0 {
			size = 256
		}
		return ConvertInt(true, size, bi)

	case BoolTy:
		for i := 0; i < 31; i++ {
			if chunk[i] != 0 {
				return nil, errors.New("abi: invalid padded bool bytes")
			}
		}
		switch chunk[31] {
		case 0:
			return false, nil
		case 1:
			return true, nil
		default:
			return nil, fmt.Errorf("abi: invalid bool byte %d", chunk[31])
		}

	case AddressTy:
		return BytesToAddress(chunk[12:32]), nil

	case HashTy:
		return BytesToHash(chunk[:32]), nil

	case FixedBytesTy:
		if t.Size == 32 {
			return BytesToHash(chunk[:32]), nil
		}
		b := make([]byte, t.Size)
		copy(b, chunk[:t.Size])
		return HexBytes(b), nil

	case ArrayTy:
		if t.Size == 0 {
			return []interface{}{}, nil
		}
		synthetic := make(ABIArguments, t.Size)
		for i := 0; i < t.Size; i++ {
			synthetic[i] = ABIArgument{Type: *t.Elem}
		}
		return decodeTupleValues(synthetic, chunk)

	case TupleTy:
		valMap, sliceVals, err := decodeTupleStruct(t.TupleElems, chunk)
		if err != nil {
			return nil, err
		}
		if valMap != nil {
			return valMap, nil
		}
		return sliceVals, nil

	default:
		return nil, fmt.Errorf("abi: unexpected static type %s", t.CanonicalString())
	}
}

func decodeDynamicValue(t ABIType, tail []byte) (interface{}, error) {
	switch t.T {
	case StringTy:
		raw, err := readDynamicBytes(tail)
		if err != nil {
			return nil, err
		}
		return string(raw), nil

	case BytesTy:
		raw, err := readDynamicBytes(tail)
		if err != nil {
			return nil, err
		}
		return HexBytes(raw), nil

	case SliceTy:
		if len(tail) < 32 {
			return nil, errors.New("abi: truncated slice length")
		}
		length, err := readOffset(tail[:32])
		if err != nil {
			return nil, err
		}
		if length == 0 {
			return []interface{}{}, nil
		}
		if length > len(tail)/32 {
			return nil, fmt.Errorf("abi: slice length %d exceeds buffer", length)
		}
		synthetic := make(ABIArguments, length)
		for i := 0; i < length; i++ {
			synthetic[i] = ABIArgument{Type: *t.Elem}
		}
		return decodeTupleValues(synthetic, tail[32:])

	case ArrayTy:
		synthetic := make(ABIArguments, t.Size)
		for i := 0; i < t.Size; i++ {
			synthetic[i] = ABIArgument{Type: *t.Elem}
		}
		return decodeTupleValues(synthetic, tail)

	case TupleTy:
		valMap, sliceVals, err := decodeTupleStruct(t.TupleElems, tail)
		if err != nil {
			return nil, err
		}
		if valMap != nil {
			return valMap, nil
		}
		return sliceVals, nil

	default:
		return nil, fmt.Errorf("abi: unexpected dynamic type %s", t.CanonicalString())
	}
}

func decodeTupleStruct(elems []ABIArgument, buf []byte) (map[string]interface{}, []interface{}, error) {
	vals, err := decodeTupleValues(elems, buf)
	if err != nil {
		return nil, nil, err
	}
	allNamed := len(elems) > 0
	for _, e := range elems {
		if e.Name == "" {
			allNamed = false
			break
		}
	}
	if allNamed {
		m := make(map[string]interface{}, len(elems))
		for i, e := range elems {
			m[e.Name] = vals[i]
		}
		return m, vals, nil
	}
	return nil, vals, nil
}

func readDynamicBytes(tail []byte) ([]byte, error) {
	if len(tail) < 32 {
		return nil, errors.New("abi: truncated bytes length")
	}
	length, err := readOffset(tail[:32])
	if err != nil {
		return nil, err
	}
	if length > len(tail)-32 {
		return nil, fmt.Errorf("abi: bytes payload length %d exceeds buffer %d", length, len(tail)-32)
	}
	out := make([]byte, length)
	copy(out, tail[32:32+length])
	return out, nil
}

func readOffset(word []byte) (int, error) {
	bi := new(big.Int).SetBytes(word)
	if !bi.IsInt64() {
		return 0, fmt.Errorf("abi: offset %s overflows int64", bi.String())
	}
	v := bi.Int64()
	if v < 0 || v > int64(^uint(0)>>1) {
		return 0, fmt.Errorf("abi: negative or overflowing offset %d", v)
	}
	return int(v), nil
}

// FindEventById finds an event in myabi matching topic0 id, or nil if unknown.
func FindEventById(myabi ABI, id Hash) *ABIEvent {
	for _, ev := range myabi.Events {
		if ev.ID == id {
			e := ev
			return &e
		}
	}
	return nil
}

// ParseLogs decodes matching events from logs, safely skipping anonymous or unrecognized external events.
func ParseLogs(myabi ABI, logs []*Log) ([]Event, error) {
	var output []Event
	for _, l := range logs {
		if l == nil || len(l.Topics) == 0 {
			continue
		}
		event := FindEventById(myabi, l.Topics[0])
		if event == nil {
			continue
		}

		fields := make(map[string]interface{})
		var nonIndexed ABIArguments
		var indexed []ABIArgument
		for _, arg := range event.Inputs {
			if arg.Indexed {
				indexed = append(indexed, arg)
			} else {
				nonIndexed = append(nonIndexed, arg)
			}
		}

		if len(nonIndexed) > 0 && len(l.Data) > 0 {
			vals, err := nonIndexed.UnpackValues(l.Data)
			if err != nil {
				return nil, fmt.Errorf("unpacking event %s data: %w", event.RawName, err)
			}
			for i, v := range vals {
				name := nonIndexed[i].Name
				if name == "" {
					name = fmt.Sprintf("arg%d", i)
				}
				fields[name] = v
			}
		}

		for i, input := range indexed {
			if i+1 >= len(l.Topics) {
				break
			}
			name := input.Name
			if name == "" {
				name = fmt.Sprintf("topic%d", i+1)
			}
			fields[name] = decodeIndexedTopic(input.Type, l.Topics[i+1])
		}

		output = append(output, Event{Name: event.RawName, Fields: fields})
	}
	return output, nil
}

func decodeIndexedTopic(t ABIType, topic Hash) interface{} {
	if t.isDynamic() || t.T == ArrayTy || t.T == TupleTy {
		// Indexed reference types store keccak256(value) in the topic.
		return topic.Hex()
	}
	if v, err := decodeStaticValue(t, topic[:]); err == nil {
		return v
	}
	return topic.Hex()
}
