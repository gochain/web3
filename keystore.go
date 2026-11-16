package web3

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

type encryptedKeyJSONV3 struct {
	Address string     `json:"address"`
	Crypto  cryptoJSON `json:"crypto"`
	CryptoU cryptoJSON `json:"Crypto"`
	ID      string     `json:"id"`
	Version int        `json:"version"`
}

type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

// DecryptKeystore decrypts a Web3 Secret Storage V3 keystore JSON payload into an *Account.
func DecryptKeystore(keyJSON []byte, auth string) (*Account, error) {
	var k encryptedKeyJSONV3
	if err := json.Unmarshal(keyJSON, &k); err != nil {
		return nil, fmt.Errorf("invalid keystore json: %w", err)
	}
	if k.Version != 3 {
		return nil, fmt.Errorf("unsupported keystore version %d (expected 3)", k.Version)
	}
	cj := k.Crypto
	if cj.Cipher == "" {
		cj = k.CryptoU
	}
	if !strings.EqualFold(cj.Cipher, "aes-128-ctr") {
		return nil, fmt.Errorf("unsupported keystore cipher %q", cj.Cipher)
	}

	cipherText, err := hex.DecodeString(cj.CipherText)
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext hex: %w", err)
	}
	iv, err := hex.DecodeString(cj.CipherParams.IV)
	if err != nil {
		return nil, fmt.Errorf("invalid iv hex: %w", err)
	}
	if len(iv) != aes.BlockSize {
		return nil, fmt.Errorf("invalid iv length %d (expected %d)", len(iv), aes.BlockSize)
	}
	mac, err := hex.DecodeString(cj.MAC)
	if err != nil {
		return nil, fmt.Errorf("invalid mac hex: %w", err)
	}

	derivedKey, err := deriveKeystoreKey(cj, auth)
	if err != nil {
		return nil, err
	}
	if len(derivedKey) < 32 {
		return nil, fmt.Errorf("derived key too short (%d bytes, need at least 32)", len(derivedKey))
	}

	calculatedMAC := Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, errors.New("could not decrypt key with given password")
	}

	block, err := aes.NewCipher(derivedKey[:16])
	if err != nil {
		return nil, err
	}
	plainText := make([]byte, len(cipherText))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(plainText, cipherText)

	privKey, err := ToECDSA(plainText)
	for i := range plainText {
		plainText[i] = 0
	}
	if err != nil {
		return nil, err
	}
	return NewAccountFromKey(privKey), nil
}

// EncryptKeystore encrypts an Account into Web3 Secret Storage V3 JSON (scrypt N=4096, r=8, p=6 for fast CLI usage or standard N).
func EncryptKeystore(acc *Account, auth string, scryptN, scryptP int) ([]byte, error) {
	if acc == nil || acc.key == nil {
		return nil, errors.New("nil account")
	}
	if scryptN <= 0 {
		scryptN = 1 << 18 // Standard N=262144
	}
	if scryptP <= 0 {
		scryptP = 1
	}
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	uuidBytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, uuidBytes); err != nil {
		return nil, err
	}
	uuidBytes[6] = (uuidBytes[6] & 0x0f) | 0x40
	uuidBytes[8] = (uuidBytes[8] & 0x3f) | 0x80
	id := fmt.Sprintf("%x-%x-%x-%x-%x", uuidBytes[0:4], uuidBytes[4:6], uuidBytes[6:8], uuidBytes[8:10], uuidBytes[10:16])

	derivedKey, err := scrypt.Key([]byte(auth), salt, scryptN, 8, scryptP, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(derivedKey[:16])
	if err != nil {
		return nil, err
	}
	keyBytes := FromECDSA(acc.key)
	cipherText := make([]byte, len(keyBytes))
	cipher.NewCTR(block, iv).XORKeyStream(cipherText, keyBytes)
	for i := range keyBytes {
		keyBytes[i] = 0
	}
	mac := Keccak256(derivedKey[16:32], cipherText)

	payload := map[string]interface{}{
		"address": strings.ToLower(strings.TrimPrefix(acc.Address().Hex(), "0x")),
		"id":      id,
		"version": 3,
		"crypto": map[string]interface{}{
			"cipher":     "aes-128-ctr",
			"ciphertext": hex.EncodeToString(cipherText),
			"cipherparams": map[string]string{
				"iv": hex.EncodeToString(iv),
			},
			"kdf": "scrypt",
			"kdfparams": map[string]interface{}{
				"dklen": 32,
				"n":     scryptN,
				"r":     8,
				"p":     scryptP,
				"salt":  hex.EncodeToString(salt),
			},
			"mac": hex.EncodeToString(mac),
		},
	}
	return json.MarshalIndent(payload, "", "  ")
}

func deriveKeystoreKey(cj cryptoJSON, auth string) ([]byte, error) {
	saltHex, _ := cj.KDFParams["salt"].(string)
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return nil, fmt.Errorf("invalid kdf salt: %w", err)
	}
	dkLen := jsonInt(cj.KDFParams["dklen"])
	if dkLen < 32 {
		return nil, fmt.Errorf("invalid kdf dklen %d (must be at least 32)", dkLen)
	}

	switch strings.ToLower(cj.KDF) {
	case "scrypt":
		n := jsonInt(cj.KDFParams["n"])
		r := jsonInt(cj.KDFParams["r"])
		p := jsonInt(cj.KDFParams["p"])
		if n <= 1 || r <= 0 || p <= 0 {
			return nil, fmt.Errorf("invalid scrypt parameters (n=%d, r=%d, p=%d)", n, r, p)
		}
		return scrypt.Key([]byte(auth), salt, n, r, p, dkLen)
	case "pbkdf2":
		prf, _ := cj.KDFParams["prf"].(string)
		if prf != "hmac-sha256" {
			return nil, fmt.Errorf("unsupported pbkdf2 prf %q", prf)
		}
		c := jsonInt(cj.KDFParams["c"])
		if c <= 0 {
			return nil, fmt.Errorf("invalid pbkdf2 iteration count %d", c)
		}
		return pbkdf2.Key([]byte(auth), salt, c, dkLen, sha256.New), nil
	default:
		return nil, fmt.Errorf("unsupported keystore kdf %q", cj.KDF)
	}
}

func jsonInt(v interface{}) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case json.Number:
		i, _ := x.Int64()
		return int(i)
	default:
		return 0
	}
}
