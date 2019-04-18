package did

import (
	"time"
)

// ContextV1 is the required context for all DID documents.
const ContextV1 = "https://w3id.org/did/v1"

// Document represents a DID document.
type Document struct {
	// MUST be set to ContextV1.
	Context string `json:"@context,omitempty"`

	// The identifier that the DID Document is about, i.e. the DID.
	ID string `json:"id,omitempty"`

	// Public keys are used for digital signatures, encryption and other
	// cryptographic operations, which in turn are the basis for purposes such
	// as authentication, or establishing secure communication with service
	// endpoints. In addition, public keys may play a role in authorization
	// mechanisms of DID CRUD operations
	PublicKeys []PublicKey `json:"publicKey,omitempty"`

	// Specifies zero or more embedded or referenced public keys by which a
	// DID subject can cryptographically prove that they are associated with a DID.
	//
	// Each element MUST be a PublicKey (embedded) or string (referenced).
	Authentications []interface{} `json:"authentication,omitempty"`

	// Represent any type of service the subject wishes to advertise, including
	// decentralized identity management services for further discovery,
	// authentication, authorization, or interaction.
	Services []Service `json:"services,omitempty"`

	// Timestamp when document was first created, normalized to UTC. Optional.
	Created *time.Time `json:"created,omitempty"`

	// Timestamp when document was last updated, normalized to UTC. Optional.
	Updated *time.Time `json:"updated,omitempty"`

	// Cryptographic proof of the integrity of the DID Document.
	// This proof is NOT proof of the binding between a DID and a DID Document.
	Proof *Proof `json:"proof,omitempty"`
}

// NewDocument returns a new Document with the appropriate context.
func NewDocument() *Document {
	return &Document{Context: ContextV1}
}

// PublicKey represents a specification of public key on the document.
type PublicKey struct {
	// Unique identifier of the key within the document.
	ID string `json:"id,omitempty"`

	// Type of encryption, as specified in Linked Data Cryptographic Suite Registry.
	// https://w3c-ccg.github.io/ld-cryptosuite-registry/
	Type string `json:"type,omitempty"`

	// DID identifying the controller of the corresponding private key.
	Controller string `json:"controller,omitempty"`

	// Only one of these can be specified based on type.
	PublicKeyPEM       string `json:"publicKeyPem,omitempty"`
	PublicKeyJWK       string `json:"publicKeyJwk,omitempty"`
	PublicKeyHex       string `json:"publicKeyHex,omitempty"`
	PublicKeyBase64    string `json:"publicKeyBase64,omitempty"`
	PublicKeyBase58    string `json:"publicKeyBase58,omitempty"`
	PublicKeyMultibase string `json:"publicKeyMultibase,omitempty"`
}

// Service represents a service endpoint specification.
type Service struct {
	// Unique identifier of the service within the document.
	ID              string `json:"id,omitempty"`
	Type            string `json:"type,omitempty"`
	ServiceEndpoint string `json:"serviceEndpoint,omitempty"`
}

// Proof represents a JSON-LD proof of the integrity of a DID document.
type Proof struct {
	Type           string `json:"type,omitempty"`
	Creator        string `json:"creator,omitempty"`
	Created        string `json:"created,omitempty"`
	Domain         string `json:"domain,omitempty"`
	Nonce          string `json:"nonce,omitempty"`
	SignatureValue string `json:"signatureValue,omitempty"`
}
