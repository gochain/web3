package did_test

import (
	"testing"

	"github.com/zeus-fyi/gochain/web3/did"
)

func TestParse(t *testing.T) {
	t.Run("Minimal", func(t *testing.T) {
		if d, err := did.Parse("did:x:y"); err != nil {
			t.Fatalf("unexpected error: %s", err)
		} else if *d != (did.DID{Method: "x", ID: "y"}) {
			t.Fatalf("unexpected did: %#v", d)
		}
	})
	t.Run("AllChars", func(t *testing.T) {
		if d, err := did.Parse("did:abcxyz123:abcxyzABCXYZ123.-"); err != nil {
			t.Fatalf("unexpected error: %s", err)
		} else if *d != (did.DID{Method: "abcxyz123", ID: "abcxyzABCXYZ123.-"}) {
			t.Fatalf("unexpected did: %#v", d)
		}
	})
	t.Run("WithPath", func(t *testing.T) {
		if d, err := did.Parse("did:x:y/foo%20"); err != nil {
			t.Fatalf("unexpected error: %s", err)
		} else if *d != (did.DID{Method: "x", ID: "y", Path: "/foo "}) {
			t.Fatalf("unexpected did: %#v", d)
		}
	})
	t.Run("WithFragment", func(t *testing.T) {
		if d, err := did.Parse("did:x:y#foo%20bar"); err != nil {
			t.Fatalf("unexpected error: %s", err)
		} else if *d != (did.DID{Method: "x", ID: "y", Fragment: "foo bar"}) {
			t.Fatalf("unexpected did: %#v", d)
		}
	})
	t.Run("WithPathAndFragment", func(t *testing.T) {
		if d, err := did.Parse("did:x:y/foo#bar"); err != nil {
			t.Fatalf("unexpected error: %s", err)
		} else if *d != (did.DID{Method: "x", ID: "y", Path: "/foo", Fragment: "bar"}) {
			t.Fatalf("unexpected did: %#v", d)
		}
	})

	t.Run("ErrInvalidScheme", func(t *testing.T) {
		if _, err := did.Parse(""); err == nil || err.Error() != `did.Parse(): invalid scheme` {
			t.Fatalf("unexpected error: %s", err)
		}
	})
	t.Run("ErrMissingMethod", func(t *testing.T) {
		if _, err := did.Parse("did:"); err == nil || err.Error() != `did.Parse(): missing method` {
			t.Fatalf("unexpected error: %s", err)
		}
	})
	t.Run("ErrEmptyMethod", func(t *testing.T) {
		if _, err := did.Parse("did::foo"); err == nil || err.Error() != `did.Parse(): empty method not allowed` {
			t.Fatalf("unexpected error: %s", err)
		}
	})
	t.Run("ErrInvalidMethod", func(t *testing.T) {
		if _, err := did.Parse("did:foo!bar:baz"); err == nil || err.Error() != `did.Parse(): invalid method character: '!'` {
			t.Fatalf("unexpected error: %s", err)
		}
	})
	t.Run("ErrMissingIDSeparator", func(t *testing.T) {
		if _, err := did.Parse("did:x"); err == nil || err.Error() != `did.Parse(): missing id separator` {
			t.Fatalf("unexpected error: %s", err)
		}
	})
	t.Run("ErrMissingID", func(t *testing.T) {
		if _, err := did.Parse("did:x:"); err == nil || err.Error() != `did.Parse(): missing id` {
			t.Fatalf("unexpected error: %s", err)
		}
	})
	t.Run("ErrInvalidID", func(t *testing.T) {
		if _, err := did.Parse("did:foo:bar!baz"); err == nil || err.Error() != `did.Parse(): invalid id character: '!'` {
			t.Fatalf("unexpected error: %s", err)
		}
	})
}
