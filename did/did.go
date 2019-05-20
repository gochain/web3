package did

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// DID represents a decentralized identifier.
// https://w3c-ccg.github.io/did-spec/#decentralized-identifiers-dids
type DID struct {
	Method   string
	ID       string
	Path     string
	Fragment string
}

// String returns the string representation of the DID.
func (d *DID) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "did:%s:%s%s", d.Method, d.ID, escape(d.Path, encodePath))
	if d.Fragment != "" {
		fmt.Fprintf(&buf, "#%s", escape(d.Fragment, encodeFragment))
	}
	return buf.String()
}

func Parse(rawdid string) (*DID, error) {
	var err error
	var did DID

	// Ensure scheme is for a DID.
	if !strings.HasPrefix(rawdid, "did:") {
		return nil, errors.New("did.Parse(): invalid scheme")
	}
	rawdid = rawdid[4:]

	// Separate fragment & path.
	if idx := strings.Index(rawdid, "#"); idx >= 0 {
		rawdid, did.Fragment = rawdid[:idx], rawdid[idx+1:]
	}
	if idx := strings.Index(rawdid, "/"); idx >= 0 {
		rawdid, did.Path = rawdid[:idx], rawdid[idx:]
	}

	// Parse method & idstring
	if did.Method, rawdid, err = parseMethod(rawdid); err != nil {
		return nil, err
	} else if did.ID, err = parseIDString(rawdid); err != nil {
		return nil, err
	}

	// Escape path & fragment.
	if did.Path, err = unescape(did.Path, encodePath); err != nil {
		return nil, err
	} else if did.Fragment, err = unescape(did.Fragment, encodeFragment); err != nil {
		return nil, err
	}
	return &did, nil
}

func parseMethod(s string) (method, rest string, err error) {
	if len(s) == 0 {
		return "", "", errors.New("did.Parse(): missing method")
	} else if s[0] == ':' {
		return "", "", errors.New("did.Parse(): empty method not allowed")
	}

	var buf bytes.Buffer
	for i, ch := range s {
		if ch == ':' {
			return buf.String(), s[i+1:], nil
		} else if !isMethodChar(ch) {
			return "", "", fmt.Errorf("did.Parse(): invalid method character: %q", ch)
		}
		buf.WriteRune(ch)
	}
	return "", "", errors.New("did.Parse(): missing id separator")
}

func parseIDString(s string) (id string, err error) {
	if len(s) == 0 {
		return "", errors.New("did.Parse(): missing id")
	}

	var buf bytes.Buffer
	for _, ch := range s {
		if !isIDChar(ch) {
			return "", fmt.Errorf("did.Parse(): invalid id character: %q", ch)
		}
		buf.WriteRune(ch)
	}
	return buf.String(), nil
}

// IsValidIDString returns true if s is non-blank and contains only valid ID characters.
func IsValidIDString(s string) bool {
	for _, ch := range s {
		if !isIDChar(ch) {
			return false
		}
	}
	return len(s) > 0
}

// isMethodChar returns true if ch is a valid character for the method.
func isMethodChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}

// isIDChar returns true if ch is a valid character for the idstring.
func isIDChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '.' || ch == '-'
}
