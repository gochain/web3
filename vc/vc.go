// package vc contains an implementation of the W3 Verifiable Credentials data model.
// https://www.w3.org/TR/verifiable-claims-data-model
package vc

import (
	"time"
)

// VerifiableCredential represents one or more claims made by the same entity.
//
// https://www.w3.org/TR/verifiable-claims-data-model/#credentials
type VerifiableCredential struct {
	Context           []string               `json:"@context,omitempty"`
	ID                string                 `json:"id,omitempty"`
	Type              []string               `json:"type,omitempty"`
	Issuer            string                 `json:"issuer,omitempty"`
	IssuanceDate      *time.Time             `json:"issuanceDate,omitempty"`
	CredentialSubject map[string]interface{} `json:"credentialSubject,omitempty"`
	Proof             *Proof                 `json:"proof,omitempty"`
}

// NewVerifiableCredential returns a new instance of VerifiableCredential
// with the default context and type assigned.
func NewVerifiableCredential() *VerifiableCredential {
	return &VerifiableCredential{
		Context: []string{"https://www.w3.org/2018/credentials/v1"},
		Type:    []string{"VerifiableCredential"},
	}
}

// VerifiablePresentation combines one or more credentials.
//
// https://www.w3.org/TR/verifiable-claims-data-model/#presentations
type VerifiablePresentation struct {
	Context              []string                `json:"@context,omitempty"`
	ID                   string                  `json:"id,omitempty"`
	Type                 []string                `json:"type,omitempty"`
	VerifiableCredential []*VerifiableCredential `json:"verifiableCredential,omitempty"`
	Proof                *Proof                  `json:"proof,omitempty"`
}

// NewVerifiablePresentation returns a new instance of VerifiablePresentation
// with the default context and type assigned.
func NewVerifiablePresentation() *VerifiablePresentation {
	return &VerifiablePresentation{
		Context: []string{"https://www.w3.org/2018/credentials/v1"},
		Type:    []string{"VerifiablePresentation"},
	}
}

// Proof represents a proof for a verifiable credential or presentation.
//
// https://www.w3.org/TR/verifiable-claims-data-model/#proofs-signatures
type Proof struct {
	Type       string     `json:"type,omitempty"`
	Created    *time.Time `json:"created,omitempty"`
	Creator    string     `json:"creator,omitempty"`
	ProofValue string     `json:"proofValue,omitempty"`
}
