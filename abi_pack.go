package web3

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
)

var (
	twoTo256 = new(big.Int).Lsh(big.NewInt(1), 256)
	maxU256  = new(big.Int).Sub(twoTo256, big.NewInt(1))
	maxI256  = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1))
	minI256  = new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 255))
)

// Pack ABI-encodes a call to name (or constructor when name == "") with args.
func (a ABI) Pack(name string, args ...interface{}) ([]byte, error) {
	if name == "" {
		return a.Constructor.Inputs.Pack(args...)
	}
	method, ok := a.ResolveMethod(name, len(args))
	if !ok {
		return nil, fmt.Errorf("method %q not found in ABI", name)
	}
	payload, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 4+len(payload))
	copy(out[:4], method.ID)
	copy(out[4:], payload)
	return out, nil
}

// Pack ABI-encodes values matching the ABIArguments specification.
func (args ABIArguments) Pack(values ...interface{}) ([]byte, error) {
	converted, err := ConvertArguments(args, values)
	if err != nil {
		return nil, err
	}
	types := make([]ABIType, len(args))
	for i, a := range args {
		types[i] = a.Type
	}
	return encodeTupleValues(types, converted)
}

func encodeTupleValues(types []ABIType, values []interface{}) ([]byte, error) {
	if len(types) != len(values) {
		return nil, fmt.Errorf("mismatched argument (%d) and parameter (%d) counts", len(types), len(values))
	}
	headSize := 0
	for _, t := range types {
		if t.isDynamic() {
			headSize += 32
		} else {
			headSize += t.staticByteSize()
		}
	}

	var headBuf, tailBuf bytes.Buffer
	headBuf.Grow(headSize)

	for i, t := range types {
		enc, err := encodeSingleValue(t, values[i])
		if err != nil {
			return nil, fmt.Errorf("arg %d (%s): %w", i, t.CanonicalString(), err)
		}
		if t.isDynamic() {
			offset := big.NewInt(int64(headSize + tailBuf.Len()))
			headBuf.Write(padLeft32(offset.Bytes()))
			tailBuf.Write(enc)
		} else {
			headBuf.Write(enc)
		}
	}
	headBuf.Write(tailBuf.Bytes())
	return headBuf.Bytes(), nil
}

func encodeSingleValue(t ABIType, val interface{}) ([]byte, error) {
	switch t.T {
	case UintTy:
		bi, err := toBigInt(val)
		if err != nil {
			return nil, err
		}
		if bi.Sign() < 0 || bi.Cmp(maxU256) > 0 {
			return nil, fmt.Errorf("uint value %s out of 256-bit range", bi.String())
		}
		return padLeft32(bi.Bytes()), nil

	case IntTy:
		bi, err := toBigInt(val)
		if err != nil {
			return nil, err
		}
		if bi.Cmp(minI256) < 0 || bi.Cmp(maxI256) > 0 {
			return nil, fmt.Errorf("int value %s out of 256-bit range", bi.String())
		}
		if bi.Sign() < 0 {
			twos := new(big.Int).Add(twoTo256, bi)
			return padLeft32(twos.Bytes()), nil
		}
		return padLeft32(bi.Bytes()), nil

	case BoolTy:
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool, got %T", val)
		}
		var word [32]byte
		if b {
			word[31] = 1
		}
		return word[:], nil

	case AddressTy:
		var addr Address
		switch a := val.(type) {
		case Address:
			addr = a
		case *Address:
			if a != nil {
				addr = *a
			}
		default:
			return nil, fmt.Errorf("expected Address, got %T", val)
		}
		return padLeft32(addr[:]), nil

	case HashTy:
		h, ok := val.(Hash)
		if !ok {
			return nil, fmt.Errorf("expected Hash, got %T", val)
		}
		return h[:], nil

	case FixedBytesTy:
		raw, err := extractBytes(val)
		if err != nil {
			return nil, err
		}
		if len(raw) != t.Size {
			return nil, fmt.Errorf("invalid fixed bytes length %d: expected %d", len(raw), t.Size)
		}
		return padRight32(raw), nil

	case BytesTy:
		raw, err := extractBytes(val)
		if err != nil {
			return nil, err
		}
		return encodeDynamicBytes(raw), nil

	case StringTy:
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", val)
		}
		return encodeDynamicBytes([]byte(s)), nil

	case SliceTy:
		elems, err := extractSliceElements(val)
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		out.Write(padLeft32(big.NewInt(int64(len(elems))).Bytes()))
		elemTypes := make([]ABIType, len(elems))
		for i := range elemTypes {
			elemTypes[i] = *t.Elem
		}
		body, err := encodeTupleValues(elemTypes, elems)
		if err != nil {
			return nil, err
		}
		out.Write(body)
		return out.Bytes(), nil

	case ArrayTy:
		elems, err := extractSliceElements(val)
		if err != nil {
			return nil, err
		}
		if len(elems) != t.Size {
			return nil, fmt.Errorf("array length mismatch: got %d, expected %d", len(elems), t.Size)
		}
		elemTypes := make([]ABIType, t.Size)
		for i := range elemTypes {
			elemTypes[i] = *t.Elem
		}
		return encodeTupleValues(elemTypes, elems)

	case TupleTy:
		fields, ok := val.([]interface{})
		if !ok {
			return nil, fmt.Errorf("expected normalized tuple fields, got %T", val)
		}
		fieldTypes := make([]ABIType, len(t.TupleElems))
		for i, c := range t.TupleElems {
			fieldTypes[i] = c.Type
		}
		return encodeTupleValues(fieldTypes, fields)

	default:
		return nil, fmt.Errorf("unsupported ABI type %v", t)
	}
}

func encodeDynamicBytes(b []byte) []byte {
	paddedLen := ((len(b) + 31) / 32) * 32
	out := make([]byte, 32+paddedLen)
	copy(out[:32], padLeft32(big.NewInt(int64(len(b))).Bytes()))
	copy(out[32:], b)
	return out
}

func padLeft32(b []byte) []byte {
	var word [32]byte
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	copy(word[32-len(b):], b)
	return word[:]
}

func padRight32(b []byte) []byte {
	var word [32]byte
	if len(b) > 32 {
		b = b[:32]
	}
	copy(word[:len(b)], b)
	return word[:]
}

// ConvertArguments converts CLI strings, JSON numbers/arrays/objects, or Go values to ABI-ready Go types.
func ConvertArguments(args ABIArguments, params []interface{}) ([]interface{}, error) {
	if len(args) != len(params) {
		return nil, fmt.Errorf("mismatched argument (%d) and parameter (%d) counts", len(args), len(params))
	}
	converted := make([]interface{}, len(args))
	for i, input := range args {
		val, err := ConvertArgument(input.Type, params[i])
		if err != nil {
			return nil, err
		}
		converted[i] = val
	}
	return converted, nil
}

// ConvertArgument converts param to the Go representation expected by abiType.
func ConvertArgument(abiType ABIType, param interface{}) (interface{}, error) {
	size := abiType.Size
	switch abiType.T {
	case StringTy:
		if s, ok := param.(string); ok {
			return s, nil
		}
		return fmt.Sprint(param), nil

	case BoolTy:
		switch b := param.(type) {
		case bool:
			return b, nil
		case string:
			val, err := strconv.ParseBool(strings.TrimSpace(b))
			if err != nil {
				return nil, fmt.Errorf("failed to parse bool %q: %w", b, err)
			}
			return val, nil
		default:
			return nil, fmt.Errorf("expected bool, got %T", param)
		}

	case UintTy, IntTy:
		signed := abiType.T == IntTy
		if size == 0 {
			size = 256
		}
		if j, ok := param.(json.Number); ok {
			param = string(j)
		}
		switch p := param.(type) {
		case string:
			s := strings.TrimSpace(p)
			val, ok := new(big.Int).SetString(s, 0)
			if !ok {
				return nil, fmt.Errorf("failed to parse integer %q", p)
			}
			return ConvertInt(signed, size, val)
		case *big.Int:
			if p == nil {
				return ConvertInt(signed, size, big.NewInt(0))
			}
			return ConvertInt(signed, size, p)
		case big.Int:
			return ConvertInt(signed, size, &p)
		}
		v := reflect.ValueOf(param)
		if !v.IsValid() {
			return nil, errors.New("nil integer argument")
		}
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return ConvertInt(signed, size, new(big.Int).SetInt64(v.Int()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return ConvertInt(signed, size, new(big.Int).SetUint64(v.Uint()))
		case reflect.Float64, reflect.Float32:
			return nil, errors.New("floating point numbers are not valid in web3 - please use an integer or string instead (including big.Int and json.Number)")
		default:
			return nil, fmt.Errorf("unsupported integer input %T", param)
		}

	case AddressTy:
		switch a := param.(type) {
		case Address:
			return a, nil
		case *Address:
			if a == nil {
				return ZeroAddress, nil
			}
			return *a, nil
		case string:
			s := strings.TrimSpace(a)
			if !IsHexAddress(s) {
				return nil, fmt.Errorf("invalid hex address: %s", s)
			}
			return HexToAddress(s), nil
		default:
			return nil, fmt.Errorf("expected address, got %T", param)
		}

	case BytesTy:
		switch b := param.(type) {
		case []byte:
			return b, nil
		case HexBytes:
			return []byte(b), nil
		case string:
			s := strings.TrimSpace(b)
			if s == "" || s == "0x" || s == "0X" {
				return []byte{}, nil
			}
			val, err := DecodeHex(s)
			if err != nil {
				return nil, fmt.Errorf("failed to parse bytes %q: %w", s, err)
			}
			return val, nil
		default:
			return nil, fmt.Errorf("expected bytes, got %T", param)
		}

	case HashTy:
		switch h := param.(type) {
		case Hash:
			return h, nil
		case string:
			val, err := DecodeHex(h)
			if err != nil {
				return nil, fmt.Errorf("failed to parse hash %q: %w", h, err)
			}
			if len(val) != HashLength {
				return nil, fmt.Errorf("invalid hash length %d: hash must be 32 bytes", len(val))
			}
			return BytesToHash(val), nil
		default:
			return nil, fmt.Errorf("expected hash, got %T", param)
		}

	case FixedBytesTy:
		if size == 32 {
			if h, ok := param.(Hash); ok {
				return h, nil
			}
			if s, ok := param.(string); ok {
				val, err := DecodeHex(s)
				if err != nil {
					return nil, fmt.Errorf("failed to parse bytes32 %q: %w", s, err)
				}
				if len(val) != HashLength {
					return nil, fmt.Errorf("invalid hash length %d: hash must be 32 bytes", len(val))
				}
				return BytesToHash(val), nil
			}
		}
		if s, ok := param.(string); ok {
			val, err := DecodeHex(s)
			if err != nil {
				return nil, fmt.Errorf("failed to parse bytes%d %q: %w", size, s, err)
			}
			if len(val) != size {
				return nil, fmt.Errorf("invalid byte array length %d: size is %d bytes", len(val), size)
			}
			arrayT := reflect.ArrayOf(size, reflect.TypeOf(byte(0)))
			array := reflect.New(arrayT).Elem()
			reflect.Copy(array, reflect.ValueOf(val))
			return array.Interface(), nil
		}
		raw, err := extractBytes(param)
		if err != nil {
			return nil, err
		}
		if len(raw) != size {
			return nil, fmt.Errorf("invalid byte array length %d: size is %d bytes", len(raw), size)
		}
		if size == 32 {
			return BytesToHash(raw), nil
		}
		arrayT := reflect.ArrayOf(size, reflect.TypeOf(byte(0)))
		array := reflect.New(arrayT).Elem()
		reflect.Copy(array, reflect.ValueOf(raw))
		return array.Interface(), nil

	case SliceTy, ArrayTy:
		rawItems, err := parseArrayOrSliceInput(param, abiType.Elem.T == StringTy)
		if err != nil {
			return nil, err
		}
		if abiType.T == ArrayTy && len(rawItems) != size {
			return nil, fmt.Errorf("fixed array expects %d elements, got %d", size, len(rawItems))
		}
		converted := make([]interface{}, len(rawItems))
		for i, elem := range rawItems {
			val, err := ConvertArgument(*abiType.Elem, elem)
			if err != nil {
				return nil, fmt.Errorf("element %d: %w", i, err)
			}
			converted[i] = val
		}
		return converted, nil

	case TupleTy:
		return convertTupleInput(abiType.TupleElems, param)

	default:
		return nil, fmt.Errorf("unsupported input type %v", abiType)
	}
}

func parseArrayOrSliceInput(param interface{}, elemIsString bool) ([]interface{}, error) {
	if s, ok := param.(string); ok {
		trimmed := strings.TrimSpace(s)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			// Try strict JSON array with UseNumber first (handles nested tuples/arrays/quoted strings).
			dec := json.NewDecoder(strings.NewReader(trimmed))
			dec.UseNumber()
			var jsonArr []interface{}
			if err := dec.Decode(&jsonArr); err == nil {
				return jsonArr, nil
			}
			// Fallback to simple CSV bracket syntax: [a,b,c]
			inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			if inner == "" {
				return []interface{}{}, nil
			}
			parts, err := splitTopLevelCSV(inner)
			if err != nil {
				return nil, err
			}
			out := make([]interface{}, len(parts))
			for i, p := range parts {
				if elemIsString {
					p = strings.Trim(p, `"'`)
				}
				out[i] = p
			}
			return out, nil
		}
		if trimmed == "" {
			return []interface{}{}, nil
		}
		parts, err := splitTopLevelCSV(trimmed)
		if err != nil {
			return nil, err
		}
		out := make([]interface{}, len(parts))
		for i, p := range parts {
			out[i] = p
		}
		return out, nil
	}

	rv := reflect.ValueOf(param)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return nil, fmt.Errorf("invalid array/slice argument %T", param)
	}
	out := make([]interface{}, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = rv.Index(i).Interface()
	}
	return out, nil
}

func convertTupleInput(fields []ABIArgument, param interface{}) ([]interface{}, error) {
	if s, ok := param.(string); ok {
		trimmed := strings.TrimSpace(s)
		dec := json.NewDecoder(strings.NewReader(trimmed))
		dec.UseNumber()
		var parsed interface{}
		if err := dec.Decode(&parsed); err != nil {
			// Fallback to "(a,b)" or "a,b" CSV
			trimmed = strings.TrimPrefix(strings.TrimSuffix(trimmed, ")"), "(")
			parts, splitErr := splitTopLevelCSV(trimmed)
			if splitErr != nil {
				return nil, fmt.Errorf("invalid tuple %q: %w", s, err)
			}
			raw := make([]interface{}, len(parts))
			for i, p := range parts {
				raw[i] = strings.Trim(p, `"'`)
			}
			return ConvertArguments(fields, raw)
		}
		param = parsed
	}

	switch v := param.(type) {
	case []interface{}:
		return ConvertArguments(fields, v)
	case map[string]interface{}:
		raw := make([]interface{}, len(fields))
		for i, f := range fields {
			val, exists := v[f.Name]
			if !exists {
				return nil, fmt.Errorf("missing tuple field %q", f.Name)
			}
			raw[i] = val
		}
		return ConvertArguments(fields, raw)
	}

	rv := reflect.ValueOf(param)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		raw := make([]interface{}, len(fields))
		for i, f := range fields {
			matched := findStructField(rv, f.Name, i)
			if !matched.IsValid() {
				return nil, fmt.Errorf("struct %T missing field for %q", param, f.Name)
			}
			raw[i] = matched.Interface()
		}
		return ConvertArguments(fields, raw)
	}
	return nil, fmt.Errorf("unsupported tuple value %T", param)
}

func findStructField(rv reflect.Value, name string, index int) reflect.Value {
	rt := rv.Type()
	if name != "" {
		for i := 0; i < rt.NumField(); i++ {
			sf := rt.Field(i)
			if strings.EqualFold(sf.Name, name) || strings.Split(sf.Tag.Get("json"), ",")[0] == name {
				return rv.Field(i)
			}
		}
	}
	if index < rt.NumField() {
		return rv.Field(index)
	}
	return reflect.Value{}
}

// ConvertInt converts i into the exact Go integer type corresponding to signedness and ABI bit size,
// validating both overflow AND underflow bounds.
func ConvertInt(signed bool, size int, i *big.Int) (interface{}, error) {
	if size <= 0 || size > 256 {
		return nil, fmt.Errorf("invalid integer bit size %d", size)
	}
	if signed {
		// Exact bit-width check [-(2^(size-1)), 2^(size-1)-1]
		maxBound := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(size-1)), big.NewInt(1))
		minBound := new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), uint(size-1)))
		if i.Cmp(maxBound) > 0 || i.Cmp(minBound) < 0 {
			return nil, fmt.Errorf("integer out of int%d range: %s", size, i)
		}
		switch {
		case size > 64:
			return new(big.Int).Set(i), nil
		case size > 32:
			if !i.IsInt64() {
				return nil, fmt.Errorf("integer out of int64 range: %s", i)
			}
			return i.Int64(), nil
		case size > 16:
			v := i.Int64()
			if v > math.MaxInt32 || v < math.MinInt32 {
				return nil, fmt.Errorf("integer out of int32 range: %s", i)
			}
			return int32(v), nil
		case size > 8:
			v := i.Int64()
			if v > math.MaxInt16 || v < math.MinInt16 {
				return nil, fmt.Errorf("integer out of int16 range: %s", i)
			}
			return int16(v), nil
		default:
			v := i.Int64()
			if v > math.MaxInt8 || v < math.MinInt8 {
				return nil, fmt.Errorf("integer out of int8 range: %s", i)
			}
			return int8(v), nil
		}
	}

	if i.Sign() < 0 {
		return nil, fmt.Errorf("negative value in unsigned field: %s", i)
	}
	maxUBound := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(size)), big.NewInt(1))
	if i.Cmp(maxUBound) > 0 {
		return nil, fmt.Errorf("integer overflows uint%d: %s", size, i)
	}
	switch {
	case size > 64:
		return new(big.Int).Set(i), nil
	case size > 32:
		return i.Uint64(), nil
	case size > 16:
		return uint32(i.Uint64()), nil
	case size > 8:
		return uint16(i.Uint64()), nil
	default:
		return uint8(i.Uint64()), nil
	}
}

func toBigInt(val interface{}) (*big.Int, error) {
	switch v := val.(type) {
	case *big.Int:
		if v == nil {
			return big.NewInt(0), nil
		}
		return v, nil
	case big.Int:
		return &v, nil
	}
	rv := reflect.ValueOf(val)
	if !rv.IsValid() {
		return nil, errors.New("nil integer")
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return big.NewInt(rv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return new(big.Int).SetUint64(rv.Uint()), nil
	default:
		return nil, fmt.Errorf("expected integer, got %T", val)
	}
}

func extractBytes(val interface{}) ([]byte, error) {
	switch v := val.(type) {
	case []byte:
		return v, nil
	case HexBytes:
		return []byte(v), nil
	case Hash:
		return v[:], nil
	}
	rv := reflect.ValueOf(val)
	if rv.IsValid() && (rv.Kind() == reflect.Array || rv.Kind() == reflect.Slice) && rv.Type().Elem().Kind() == reflect.Uint8 {
		out := make([]byte, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = byte(rv.Index(i).Uint())
		}
		return out, nil
	}
	return nil, fmt.Errorf("expected byte slice/array, got %T", val)
}

func extractSliceElements(val interface{}) ([]interface{}, error) {
	if s, ok := val.([]interface{}); ok {
		return s, nil
	}
	rv := reflect.ValueOf(val)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return nil, fmt.Errorf("expected slice or array, got %T", val)
	}
	out := make([]interface{}, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = rv.Index(i).Interface()
	}
	return out, nil
}
