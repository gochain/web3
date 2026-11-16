package web3

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ABIMethod represents a function, constructor, fallback, or receive entry in an ABI.
type ABIMethod struct {
	Name            string       `json:"name"`
	RawName         string       `json:"-"`
	Type            string       `json:"type"`
	StateMutability string       `json:"stateMutability,omitempty"`
	Constant        bool         `json:"constant,omitempty"`
	Payable         bool         `json:"payable,omitempty"`
	Inputs          ABIArguments `json:"inputs"`
	Outputs         ABIArguments `json:"outputs"`
	ID              []byte       `json:"-"` // 4-byte function selector
}

// Sig returns the canonical signature string, e.g. "transfer(address,uint256)".
func (m ABIMethod) Sig() string {
	types := make([]string, len(m.Inputs))
	for i, in := range m.Inputs {
		types[i] = in.Type.CanonicalString()
	}
	return fmt.Sprintf("%s(%s)", m.RawName, strings.Join(types, ","))
}

// IsConstant reports whether the method is read-only (view, pure, or legacy constant: true).
func (m ABIMethod) IsConstant() bool {
	return m.StateMutability == "view" || m.StateMutability == "pure" || m.Constant
}

func (m ABIMethod) String() string {
	inParts := make([]string, len(m.Inputs))
	for i, in := range m.Inputs {
		if in.Name != "" {
			inParts[i] = in.Type.CanonicalString() + " " + in.Name
		} else {
			inParts[i] = in.Type.CanonicalString()
		}
	}
	outParts := make([]string, len(m.Outputs))
	for i, out := range m.Outputs {
		if out.Name != "" {
			outParts[i] = out.Type.CanonicalString() + " " + out.Name
		} else {
			outParts[i] = out.Type.CanonicalString()
		}
	}
	mut := m.StateMutability
	if mut == "" && m.Constant {
		mut = "view"
	}
	s := fmt.Sprintf("function %s(%s)", m.RawName, strings.Join(inParts, ", "))
	if mut != "" && mut != "nonpayable" {
		s += " " + mut
	}
	if len(outParts) > 0 {
		s += fmt.Sprintf(" returns (%s)", strings.Join(outParts, ", "))
	}
	return s
}

// ABIEvent represents an event definition in an ABI.
type ABIEvent struct {
	Name      string       `json:"name"`
	RawName   string       `json:"-"`
	Anonymous bool         `json:"anonymous"`
	Inputs    ABIArguments `json:"inputs"`
	ID        Hash         `json:"-"` // 32-byte topic0 hash
}

func (e ABIEvent) Sig() string {
	types := make([]string, len(e.Inputs))
	for i, in := range e.Inputs {
		types[i] = in.Type.CanonicalString()
	}
	return fmt.Sprintf("%s(%s)", e.RawName, strings.Join(types, ","))
}

// ABIError represents a Solidity custom error definition in an ABI.
type ABIError struct {
	Name   string       `json:"name"`
	Inputs ABIArguments `json:"inputs"`
	ID     [4]byte      `json:"-"` // 4-byte error selector
}

func (e ABIError) Sig() string {
	types := make([]string, len(e.Inputs))
	for i, in := range e.Inputs {
		types[i] = in.Type.CanonicalString()
	}
	return fmt.Sprintf("%s(%s)", e.Name, strings.Join(types, ","))
}

// ABI holds parsed methods, constructor, events, and custom errors of a smart contract.
type ABI struct {
	Constructor ABIMethod
	Methods     map[string]ABIMethod
	Overloads   map[string][]ABIMethod
	Events      map[string]ABIEvent
	Errors      map[string]ABIError
}

type rawABIParam struct {
	Name         string        `json:"name"`
	Type         string        `json:"type"`
	InternalType string        `json:"internalType"`
	Indexed      bool          `json:"indexed"`
	Components   []rawABIParam `json:"components"`
}

type rawABIElement struct {
	Type            string        `json:"type"`
	Name            string        `json:"name"`
	Inputs          []rawABIParam `json:"inputs"`
	Outputs         []rawABIParam `json:"outputs"`
	StateMutability string        `json:"stateMutability"`
	Constant        bool          `json:"constant"`
	Payable         bool          `json:"payable"`
	Anonymous       bool          `json:"anonymous"`
}

// ParseABIJSON parses a JSON ABI from reader.
func ParseABIJSON(r io.Reader) (ABI, error) {
	var raw []rawABIElement
	dec := json.NewDecoder(r)
	if err := dec.Decode(&raw); err != nil {
		return ABI{}, err
	}
	return buildABI(raw)
}

// ParseABI parses a JSON ABI string.
func ParseABI(abiJSON string) (ABI, error) {
	return ParseABIJSON(strings.NewReader(abiJSON))
}

func buildABI(elements []rawABIElement) (ABI, error) {
	parsed := ABI{
		Methods:   make(map[string]ABIMethod),
		Overloads: make(map[string][]ABIMethod),
		Events:    make(map[string]ABIEvent),
		Errors:    make(map[string]ABIError),
	}

	for _, el := range elements {
		typ := el.Type
		if typ == "" {
			typ = "function"
		}
		inputs, err := convertRawParams(el.Inputs)
		if err != nil {
			return ABI{}, fmt.Errorf("in %s %q inputs: %w", typ, el.Name, err)
		}
		outputs, err := convertRawParams(el.Outputs)
		if err != nil {
			return ABI{}, fmt.Errorf("in %s %q outputs: %w", typ, el.Name, err)
		}

		switch typ {
		case "constructor":
			parsed.Constructor = ABIMethod{
				Name:            "constructor",
				RawName:         "constructor",
				Type:            "constructor",
				StateMutability: el.StateMutability,
				Payable:         el.Payable || el.StateMutability == "payable",
				Inputs:          inputs,
			}
		case "function":
			m := ABIMethod{
				Name:            el.Name,
				RawName:         el.Name,
				Type:            "function",
				StateMutability: el.StateMutability,
				Constant:        el.Constant,
				Payable:         el.Payable || el.StateMutability == "payable",
				Inputs:          inputs,
				Outputs:         outputs,
			}
			m.ID = Keccak256([]byte(m.Sig()))[:4]
			parsed.Overloads[el.Name] = append(parsed.Overloads[el.Name], m)
			key := el.Name
			if _, exists := parsed.Methods[key]; exists {
				idx := len(parsed.Overloads[el.Name]) - 1
				key = fmt.Sprintf("%s%d", el.Name, idx-1)
				m.Name = key
			}
			parsed.Methods[key] = m
			// Also register by canonical signature for unambiguous lookup
			parsed.Methods[m.Sig()] = m

		case "event":
			ev := ABIEvent{
				Name:      el.Name,
				RawName:   el.Name,
				Anonymous: el.Anonymous,
				Inputs:    inputs,
			}
			ev.ID = Keccak256Hash([]byte(ev.Sig()))
			key := el.Name
			for i := 0; ; i++ {
				if _, exists := parsed.Events[key]; !exists {
					ev.Name = key
					parsed.Events[key] = ev
					break
				}
				key = fmt.Sprintf("%s%d", el.Name, i)
			}

		case "error":
			er := ABIError{
				Name:   el.Name,
				Inputs: inputs,
			}
			copy(er.ID[:], Keccak256([]byte(er.Sig()))[:4])
			parsed.Errors[el.Name] = er

		case "fallback", "receive":
			// Valid ABI entries; no method call packing required
		}
	}
	return parsed, nil
}

func convertRawParams(raw []rawABIParam) (ABIArguments, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	args := make(ABIArguments, len(raw))
	for i, r := range raw {
		comps, err := convertRawParams(r.Components)
		if err != nil {
			return nil, err
		}
		t, err := ParseABIType(r.Type, comps)
		if err != nil {
			return nil, err
		}
		t.TupleRaw = r.InternalType
		args[i] = ABIArgument{
			Name:         r.Name,
			Type:         t,
			RawType:      t.CanonicalString(),
			InternalType: r.InternalType,
			Indexed:      r.Indexed,
		}
	}
	return args, nil
}

// ResolveMethod looks up a method by exact name, canonical signature ("transfer(address,uint256)"),
// or overload matching argCount when multiple methods share the same name.
func (a ABI) ResolveMethod(name string, argCount int) (ABIMethod, bool) {
	if overloads := a.Overloads[name]; len(overloads) > 1 && argCount >= 0 {
		for _, m := range overloads {
			if len(m.Inputs) == argCount {
				return m, true
			}
		}
	}
	if m, ok := a.Methods[name]; ok {
		return m, true
	}
	return ABIMethod{}, false
}

// ParseHumanSignature parses a human-readable Solidity function signature such as
// "transfer(address to, uint256 amount) returns (bool)" or "balanceOf(address):(uint256)".
func ParseHumanSignature(sig string) (ABIMethod, error) {
	s := strings.TrimSpace(sig)
	s = strings.TrimPrefix(s, "function ")
	s = strings.TrimSpace(s)

	openIdx := strings.IndexByte(s, '(')
	if openIdx <= 0 {
		return ABIMethod{}, fmt.Errorf("invalid function signature %q: missing '('", sig)
	}
	name := strings.TrimSpace(s[:openIdx])
	depth := 0
	closeIdx := -1
	for i := openIdx; i < len(s); i++ {
		if s[i] == '(' {
			depth++
		} else if s[i] == ')' {
			depth--
			if depth == 0 {
				closeIdx = i
				break
			}
		}
	}
	if closeIdx < 0 {
		return ABIMethod{}, fmt.Errorf("invalid function signature %q: unclosed '('", sig)
	}
	inStr := s[openIdx+1 : closeIdx]
	inParts, err := splitTopLevelCSV(inStr)
	if err != nil {
		return ABIMethod{}, err
	}
	var inputs ABIArguments
	for _, p := range inParts {
		arg, err := parseParamDecl(p)
		if err != nil {
			return ABIMethod{}, err
		}
		inputs = append(inputs, arg)
	}

	rest := strings.TrimSpace(s[closeIdx+1:])
	mut := "nonpayable"
	if strings.Contains(rest, "view") {
		mut = "view"
	} else if strings.Contains(rest, "pure") {
		mut = "pure"
	} else if strings.Contains(rest, "payable") {
		mut = "payable"
	}

	var outputs ABIArguments
	retIdx := strings.Index(rest, "returns")
	colonIdx := strings.IndexByte(rest, ':')
	var retPart string
	if retIdx >= 0 {
		retPart = strings.TrimSpace(rest[retIdx+len("returns"):])
	} else if colonIdx >= 0 {
		retPart = strings.TrimSpace(rest[colonIdx+1:])
	}
	if retPart != "" {
		if strings.HasPrefix(retPart, "(") && strings.HasSuffix(retPart, ")") {
			retPart = retPart[1 : len(retPart)-1]
		}
		outParts, err := splitTopLevelCSV(retPart)
		if err != nil {
			return ABIMethod{}, err
		}
		for _, p := range outParts {
			arg, err := parseParamDecl(p)
			if err != nil {
				return ABIMethod{}, err
			}
			outputs = append(outputs, arg)
		}
	}

	m := ABIMethod{
		Name:            name,
		RawName:         name,
		Type:            "function",
		StateMutability: mut,
		Constant:        mut == "view" || mut == "pure",
		Payable:         mut == "payable",
		Inputs:          inputs,
		Outputs:         outputs,
	}
	m.ID = Keccak256([]byte(m.Sig()))[:4]
	return m, nil
}
