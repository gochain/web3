package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gochain/gochain/v4/accounts/abi"
	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/core/types"
	"github.com/gochain/gochain/v4/crypto"
	"github.com/zeus-fyi/gochain/web3/accounts"
	"github.com/zeus-fyi/gochain/web3/assets"
	"github.com/zeus-fyi/gochain/web3/did"
	"github.com/zeus-fyi/gochain/web3/vc"
	"github.com/zeus-fyi/gochain/web3/web3_actions"
	"golang.org/x/crypto/sha3"
)

// MaxDIDLength is the maximum size of the idstring of the GoChain DID.
const MaxDIDLength = 32

func CreateDID(ctx context.Context, rpcURL string, chainID *big.Int, privateKey, id, registryAddress string, timeoutInSeconds uint64) {
	if registryAddress == "" {
		log.Fatalf("Registry contract address required")
	} else if id == "" {
		log.Fatalf("DID required")
	}

	d, err := did.Parse(id)
	if err != nil {
		log.Fatalf("Invalid DID: %s", err)
	} else if d.Method != "go" {
		log.Fatalf("Only 'go' DID methods can be registered.")
	} else if len(id) > MaxDIDLength {
		log.Fatalf("ID must be less than 32 characters")
	}

	// Parse key.
	acc, err := accounts.ParsePrivateKey(privateKey)
	if err != nil {
		log.Fatalf("Cannot parse private key: %s", err)
	}
	publicKey := acc.EcdsaPrivateKey().PublicKey

	// Build DID identifier.
	publicKeyID := *d
	publicKeyID.Fragment = "owner"

	// Build DID document.
	now := time.Now()
	doc := did.NewDocument()
	doc.ID = d.String()
	doc.Created = &now
	doc.Updated = &now
	doc.PublicKeys = []did.PublicKey{{
		ID:           publicKeyID.String(),
		Type:         "Secp256k1VerificationKey2018",
		Controller:   d.String(),
		PublicKeyHex: common.ToHex(crypto.FromECDSAPub(&publicKey)),
	}}
	doc.Authentications = []interface{}{publicKeyID.String()}

	// Pretty print document.
	data, err := json.MarshalIndent(doc, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	// Upload to IPFS.
	hash, err := IPFSUpload(ctx, "did.json", data)
	if err != nil {
		log.Fatal(err)
	}

	ac := web3_actions.NewWeb3ActionsClient(rpcURL)
	ac.Dial()
	defer ac.Close()
	ac.SetChainID(chainID)

	myabi, err := abi.JSON(strings.NewReader(assets.DIDRegistryABI))
	if err != nil {
		log.Fatalf("Cannot initialize DIDRegistry ABI: %v", err)
	}

	var idBytes32 [32]byte
	copy(idBytes32[:], d.ID)

	ac.Account = acc
	gp := web3_actions.GasPriceLimits{
		GasPrice: nil,
		GasLimit: 70000,
	}

	scp := &web3_actions.SendContractTxPayload{
		SmartContractAddr: registryAddress,
		SendEtherPayload: web3_actions.SendEtherPayload{
			TransferArgs:   web3_actions.TransferArgs{},
			GasPriceLimits: gp,
		},
		ContractFile: "",
		ContractABI:  &myabi,
		MethodName:   "register",
		Params:       []interface{}{&big.Int{}, gp, idBytes32, hash},
	}
	tx, err := ac.CallTransactFunction(ctx, scp)
	if err != nil {
		log.Fatalf("Cannot register DID identifier: %v", err)
	}

	ctx, cancelFn := context.WithTimeout(ctx, time.Duration(timeoutInSeconds)*time.Second)
	defer cancelFn()
	receipt, err := ac.WaitForReceipt(ctx, tx.Hash)
	if err != nil {
		log.Fatalf("Cannot get the receipt for transaction with hash '%v': %v", tx.Hash.Hex(), err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		fatalExit(fmt.Errorf("DID contract call failed: %s", tx.Hash.Hex()))
	}

	fmt.Println("Successfully registered DID:", d.String())
	fmt.Println("DID Document IPFS Hash:", hash)
	fmt.Println("Transaction address:", receipt.TxHash.Hex())
}

func DIDOwner(ctx context.Context, rpcURL, privateKey, id, registryAddress string) {
	if registryAddress == "" {
		log.Fatalf("Registry contract address required")
	}

	d, err := did.Parse(id)
	if err != nil {
		log.Fatalf("Invalid DID: %s", err)
	}

	ac := web3_actions.NewWeb3ActionsClient(rpcURL)
	ac.Dial()
	defer ac.Close()

	myabi, err := abi.JSON(strings.NewReader(assets.DIDRegistryABI))
	if err != nil {
		log.Fatalf("Cannot initialize DIDRegistry ABI: %v", err)
	}

	var idBytes32 [32]byte
	copy(idBytes32[:], d.ID)

	scp := &web3_actions.SendContractTxPayload{
		SmartContractAddr: "",
		SendEtherPayload:  web3_actions.SendEtherPayload{},
		ContractFile:      "",
		ContractABI:       &myabi,
		MethodName:        "owner",
		Params:            []interface{}{idBytes32},
	}
	result, err := ac.CallConstantFunction(ctx, scp)
	if err != nil {
		log.Fatalf("Cannot call the contract: %v", err)
	}
	if len(result) != 1 {
		log.Fatalf("Expected single result but got: %v", result)
	}
	address := result[0].(common.Address)
	fmt.Println(address.Hex())
}

func DIDHash(ctx context.Context, rpcURL, privateKey, id, registryAddress string) {
	if registryAddress == "" {
		log.Fatalf("Registry contract address required")
	}

	d, err := did.Parse(id)
	if err != nil {
		log.Fatalf("Invalid DID: %s", id)
	}

	ac := web3_actions.NewWeb3ActionsClient(rpcURL)
	ac.Dial()
	defer ac.Close()

	myabi, err := abi.JSON(strings.NewReader(assets.DIDRegistryABI))
	if err != nil {
		log.Fatalf("Cannot initialize DIDRegistry ABI: %v", err)
	}

	var idBytes32 [32]byte
	copy(idBytes32[:], d.ID)

	scp := &web3_actions.SendContractTxPayload{
		SmartContractAddr: registryAddress,
		SendEtherPayload:  web3_actions.SendEtherPayload{},
		ContractFile:      "",
		ContractABI:       &myabi,
		MethodName:        "hash",
		Params:            []interface{}{idBytes32},
	}
	result, err := ac.CallConstantFunction(ctx, scp)
	if err != nil {
		log.Fatalf("Cannot call the contract: %v", err)
	}
	if len(result) != 1 {
		log.Fatalf("Expected single result but got: %v", result)
	}
	hash := result[0].(string)
	fmt.Println(hash)
}

func ShowDID(ctx context.Context, rpcURL, privateKey, id, registryAddress string) {
	// Read current DID document for ID from IPFS.
	doc, err := readDIDDocument(ctx, rpcURL, registryAddress, id)
	if err != nil {
		log.Fatal(err)
	}

	// Pretty print document.
	data, err := json.MarshalIndent(doc, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func SignClaim(ctx context.Context, rpcURL, privateKey, id, typ, issuerID, subjectID, subjectJSON string) {
	if id == "" {
		log.Fatalf("Credential ID required")
	} else if typ == "" {
		log.Fatalf("Credential type required")
	}
	if issuerID == "" {
		log.Fatalf("Credential issuer DID required")
	} else if _, err := did.Parse(issuerID); err != nil {
		log.Fatalf("Invalid credential issuer DID: %s", err)
	}
	if subjectID == "" {
		log.Fatalf("Credential subject DID required")
	} else if _, err := did.Parse(subjectID); err != nil {
		log.Fatalf("Invalid credential subject DID: %s", err)
	}

	// Parse key.
	acc, err := accounts.ParsePrivateKey(privateKey)
	if err != nil {
		log.Fatalf("Cannot parse private key: %s", err)
	}

	// Parse subject object.
	subject := make(map[string]interface{})
	if subjectJSON != "" {
		if err := json.Unmarshal([]byte(subjectJSON), &subject); err != nil {
			log.Fatalf("Cannot parse subject JSON data: %s", err)
		}
	}
	subject["id"] = subjectID

	// Store current time to the second.
	now := time.Now().UTC().Truncate(1 * time.Second)

	// Build verifiable credential.
	cred := vc.NewVerifiableCredential()
	cred.ID = id
	cred.Type = append(cred.Type, typ)
	cred.Issuer = issuerID
	cred.IssuanceDate = &now
	cred.CredentialSubject = subject

	// Marshal data without proof.
	hw := sha3.NewLegacyKeccak256()
	if err := json.NewEncoder(hw).Encode(cred); err != nil {
		log.Fatalf("Cannot marshal credential to JSON: %s", err)
	}

	// Sign hash of credential document.
	var h common.Hash
	hw.Sum(h[:0])
	proofValue, err := crypto.Sign(h[:], acc.EcdsaPrivateKey())
	if err != nil {
		log.Fatalf("Cannot sign credential: %s", err)
	}

	// Trim "V" off end of proof value.
	proofValue = proofValue[:len(proofValue)-1]

	// Add proof to credential.
	cred.Proof = &vc.Proof{
		Type:       "Secp256k1VerificationKey2018",
		Created:    &now,
		ProofValue: common.Bytes2Hex(proofValue),
	}

	// Pretty print credential.
	output, err := json.MarshalIndent(cred, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(output))
}

func VerifyClaim(ctx context.Context, rpcURL, privateKey, registryAddress, filename string) {
	// Decode file into VerifiableCredential.
	var cred vc.VerifiableCredential
	if buf, err := ioutil.ReadFile(filename); err != nil {
		log.Fatalf("Cannot read file: %s", err)
	} else if err := json.Unmarshal(buf, &cred); err != nil {
		log.Fatalf("Cannot decode credential: %s", err)
	}

	// Read issuer DID document.
	doc, err := readDIDDocument(ctx, rpcURL, registryAddress, cred.Issuer)
	if err != nil {
		log.Fatalf("Cannot read issuer DID document: %s", err)
	}

	// Encode credential to JSON without proof to generate hash.
	other := cred // shallow copy
	other.Proof = nil
	hw := sha3.NewLegacyKeccak256()
	if err := json.NewEncoder(hw).Encode(other); err != nil {
		log.Fatalf("Cannot hash claim: %s", err)
	}
	var h common.Hash
	hw.Sum(h[:0])

	// Attempt verification against each of issuer's public keys.
	// Only Secp256k1 is currently supported.
	var verified bool
	for _, pub := range doc.PublicKeys {
		if pub.Type != "Secp256k1VerificationKey2018" {
			continue
		}

		pubkey := common.Hex2Bytes(strings.TrimPrefix(pub.PublicKeyHex, "0x"))
		if crypto.VerifySignature(pubkey, h[:], common.Hex2Bytes(cred.Proof.ProofValue)) {
			verified = true
			break
		}
	}

	// Display error if no keys can verify the signature.
	if !verified {
		fmt.Println("Status: NOT VERIFIED")
		os.Exit(1)
	}

	// Extract subject & extract ID.
	subject := cred.CredentialSubject
	if subject == nil {
		subject = make(map[string]interface{})
	}
	subjectID := subject["id"]
	delete(subject, "id")

	// Sort subject keys.
	keys := make([]string, 0, len(subject))
	for k := range subject {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Display credential info on success.

	fmt.Printf("ID:      %s\n", cred.ID)
	fmt.Printf("Type:    %s\n", strings.Join(cred.Type, ", "))
	fmt.Println("Status:  VERIFIED")
	fmt.Println("")

	fmt.Printf("Subject:   %s\n", subjectID)
	fmt.Printf("Issuer:    %s\n", cred.Issuer)
	fmt.Printf("Issued On: %s\n", cred.IssuanceDate)
	fmt.Println("")

	if len(keys) != 0 {
		fmt.Println("CLAIMS:")
		for _, k := range keys {
			fmt.Printf("%s: %v\n", k, subject[k])
		}
		fmt.Println("")
	}
}

func readDIDDocument(ctx context.Context, rpcURL, registryAddress, id string) (*did.Document, error) {
	if registryAddress == "" {
		return nil, fmt.Errorf("Registry contract address required")
	}
	d, err := did.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("Invalid DID: %s", id)
	}
	ac := web3_actions.NewWeb3ActionsClient(rpcURL)
	ac.Dial()
	defer ac.Close()
	myabi, err := abi.JSON(strings.NewReader(assets.DIDRegistryABI))
	if err != nil {
		return nil, fmt.Errorf("Cannot initialize DIDRegistry ABI: %v", err)
	}

	var idBytes32 [32]byte
	copy(idBytes32[:], d.ID)

	scp := &web3_actions.SendContractTxPayload{
		SmartContractAddr: registryAddress,
		SendEtherPayload:  web3_actions.SendEtherPayload{},
		ContractFile:      "",
		ContractABI:       &myabi,
		MethodName:        "hash",
		Params:            []interface{}{idBytes32},
	}
	result, err := ac.CallConstantFunction(ctx, scp)
	if err != nil {
		return nil, fmt.Errorf("Cannot call the contract: %v", err)
	}
	if len(result) != 1 {
		log.Fatalf("Expected single result but got: %v", result)
	}

	hash := result[0].(string)
	resp, err := http.Get(fmt.Sprintf("https://ipfs.infura.io:5001/api/v0/cat?arg=%s", hash))
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch DID document from IPFS: %s", err)
	}
	defer resp.Body.Close()

	var doc did.Document
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("Unable to decode DID document: %s", err)
	}
	return &doc, nil
}
