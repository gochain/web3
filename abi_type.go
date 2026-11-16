package web3

import (
	"fmt"
	"strconv"
	"strings"
)

// ABI Type kinds (kept compatible with classic Kind constants).
const (
	IntTy byte = iota
	UintTy
	BoolTy
	StringTy
	SliceTy
	ArrayTy
	TupleTy
	AddressTy
	FixedBytesTy
	BytesTy
	HashTy
)

// ABIType represents a parsed Solidity ABI parameter type.
type ABIType struct {
	T          byte
	Size       int           // Bit size for int/uint (8..256), byte length for FixedBytesTy (1..32), array length for ArrayTy
	Elem       *ABIType      // Element type for SliceTy / ArrayTy
	TupleElems []ABIArgument // Fields for TupleTy
	TupleRaw   string        // Optional internal struct name
}

// ABIArgument describes a named ABI input/output/event/error parameter.
type ABIArgument struct {
	Name         string  `json:"name"`
	Type         ABIType `json:"-"`
	RawType      string  `json:"type"`
	InternalType string  `json:"internalType,omitempty"`
	Indexed      bool    `json:"indexed,omitempty"`
}

type ABIArguments []ABIArgument

// CanonicalString returns the canonical Solidity ABI signature representation of t (used in selectors).
func (t ABIType) CanonicalString() string {
	switch t.T {
	case IntTy:
		size := t.Size
		if size == 0 {
			size = 256
		}
		return fmt.Sprintf("int%d", size)
	case UintTy:
		size := t.Size
		if size == 0 {
			size = 256
		}
		return fmt.Sprintf("uint%d", size)
	case BoolTy:
		return "bool"
	case StringTy:
		return "string"
	case AddressTy:
		return "address"
	case BytesTy:
		return "bytes"
	case HashTy:
		return "bytes32"
	case FixedBytesTy:
		return fmt.Sprintf("bytes%d", t.Size)
	case SliceTy:
		return t.Elem.CanonicalString() + "[]"
	case ArrayTy:
		return fmt.Sprintf("%s[%d]", t.Elem.CanonicalString(), t.Size)
	case TupleTy:
		parts := make([]string, len(t.TupleElems))
		for i, c := range t.TupleElems {
			parts[i] = c.Type.CanonicalString()
		}
		return "(" + strings.Join(parts, ",") + ")"
	default:
		return "unknown"
	}
}

func (t ABIType) String() string {
	return t.CanonicalString()
}

func (t ABIType) isDynamic() bool {
	switch t.T {
	case StringTy, BytesTy, SliceTy:
		return true
	case ArrayTy:
		return t.Size > 0 && t.Elem != nil && t.Elem.isDynamic()
	case TupleTy:
		for _, c := range t.TupleElems {
			if c.Type.isDynamic() {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (t ABIType) staticByteSize() int {
	if t.isDynamic() {
		return 32
	}
	switch t.T {
	case ArrayTy:
		if t.Size == 0 || t.Elem == nil {
			return 0
		}
		return t.Size * t.Elem.staticByteSize()
	case TupleTy:
		total := 0
		for _, c := range t.TupleElems {
			total += c.Type.staticByteSize()
		}
		return total
	default:
		return 32
	}
}

// ParseABIType parses a Solidity ABI type string (including arrays and inline tuples like "(string,uint64)[]")
// with optional JSON components for "tuple" types.
func ParseABIType(typeStr string, components []ABIArgument) (ABIType, error) {
	typeStr = strings.TrimSpace(typeStr)
	if typeStr == "" {
		return ABIType{}, fmt.Errorf("empty ABI type")
	}

	// Check trailing array suffix: ...[] or ...[N] (outside parentheses)
	if strings.HasSuffix(typeStr, "]") {
		openIdx := findMatchingBracket(typeStr)
		if openIdx < 0 {
			return ABIType{}, fmt.Errorf("unbalanced brackets in ABI type %q", typeStr)
		}
		baseStr := typeStr[:openIdx]
		lenStr := typeStr[openIdx+1 : len(typeStr)-1]
		elemType, err := ParseABIType(baseStr, components)
		if err != nil {
			return ABIType{}, err
		}
		if lenStr == "" {
			return ABIType{T: SliceTy, Elem: &elemType}, nil
		}
		arrLen, err := strconv.Atoi(lenStr)
		if err != nil || arrLen < 0 {
			return ABIType{}, fmt.Errorf("invalid array length %q in %q", lenStr, typeStr)
		}
		return ABIType{T: ArrayTy, Size: arrLen, Elem: &elemType}, nil
	}

	// Inline tuple syntax: "(address,uint256)" or "(string memo, uint64 timestamp)"
	if strings.HasPrefix(typeStr, "(") && strings.HasSuffix(typeStr, ")") {
		inner := typeStr[1 : len(typeStr)-1]
		parts, err := splitTopLevelCSV(inner)
		if err != nil {
			return ABIType{}, err
		}
		var tupleArgs []ABIArgument
		if len(parts) > 0 {
			tupleArgs = make([]ABIArgument, len(parts))
			for i, p := range parts {
				arg, err := parseParamDecl(p)
				if err != nil {
					return ABIType{}, err
				}
				tupleArgs[i] = arg
			}
		}
		return ABIType{T: TupleTy, TupleElems: tupleArgs}, nil
	}

	if typeStr == "tuple" {
		return ABIType{T: TupleTy, TupleElems: components}, nil
	}

	switch typeStr {
	case "address":
		return ABIType{T: AddressTy, Size: 20}, nil
	case "bool":
		return ABIType{T: BoolTy}, nil
	case "string":
		return ABIType{T: StringTy}, nil
	case "bytes":
		return ABIType{T: BytesTy}, nil
	case "byte":
		return ABIType{T: FixedBytesTy, Size: 1}, nil
	case "uint":
		return ABIType{T: UintTy, Size: 256}, nil
	case "int":
		return ABIType{T: IntTy, Size: 256}, nil
	case "function":
		return ABIType{T: FixedBytesTy, Size: 24}, nil
	}

	if strings.HasPrefix(typeStr, "uint") {
		bits, err := strconv.Atoi(typeStr[4:])
		if err != nil || bits < 8 || bits > 256 || bits%8 != 0 {
			return ABIType{}, fmt.Errorf("invalid uint bit size in %q", typeStr)
		}
		return ABIType{T: UintTy, Size: bits}, nil
	}
	if strings.HasPrefix(typeStr, "int") {
		bits, err := strconv.Atoi(typeStr[3:])
		if err != nil || bits < 8 || bits > 256 || bits%8 != 0 {
			return ABIType{}, fmt.Errorf("invalid int bit size in %q", typeStr)
		}
		return ABIType{T: IntTy, Size: bits}, nil
	}
	if strings.HasPrefix(typeStr, "bytes") {
		size, err := strconv.Atoi(typeStr[5:])
		if err != nil || size < 1 || size > 32 {
			return ABIType{}, fmt.Errorf("invalid fixed bytes size in %q", typeStr)
		}
		return ABIType{T: FixedBytesTy, Size: size}, nil
	}

	return ABIType{}, fmt.Errorf("unsupported ABI type %q", typeStr)
}

func findMatchingBracket(s string) int {
	depth := 0
	for i := len(s) - 1; i >= 0; i-- {
		switch s[i] {
		case ']':
			depth++
		case '[':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func splitTopLevelCSV(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var out []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unbalanced parentheses in %q", s)
			}
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unbalanced parentheses in %q", s)
	}
	out = append(out, strings.TrimSpace(s[start:]))
	return out, nil
}

func parseParamDecl(decl string) (ABIArgument, error) {
	decl = strings.TrimSpace(decl)
	if decl == "" {
		return ABIArgument{}, fmt.Errorf("empty parameter declaration")
	}
	// If tuple "(...) [indexed] [memory|calldata|storage] [name]"
	var typePart, rest string
	if strings.HasPrefix(decl, "(") {
		depth := 0
		end := -1
		for i := 0; i < len(decl); i++ {
			if decl[i] == '(' {
				depth++
			} else if decl[i] == ')' {
				depth--
				if depth == 0 {
					end = i + 1
					break
				}
			}
		}
		if end < 0 {
			return ABIArgument{}, fmt.Errorf("unclosed tuple in %q", decl)
		}
		// Include any array suffixes right after ')'
		for end < len(decl) && (decl[end] == '[' || decl[end] == ']' || (decl[end] >= '0' && decl[end] <= '9')) {
			end++
		}
		typePart = decl[:end]
		rest = strings.TrimSpace(decl[end:])
	} else {
		fields := strings.Fields(decl)
		typePart = fields[0]
		rest = strings.Join(fields[1:], " ")
	}

	var indexed bool
	var name string
	for _, tok := range strings.Fields(rest) {
		switch tok {
		case "indexed":
			indexed = true
		case "memory", "calldata", "storage":
			// ignore data location keywords
		default:
			name = tok
		}
	}

	parsedType, err := ParseABIType(typePart, nil)
	if err != nil {
		return ABIArgument{}, err
	}
	return ABIArgument{
		Name:    name,
		Type:    parsedType,
		RawType: parsedType.CanonicalString(),
		Indexed: indexed,
	}, nil
}
