/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package crypto handles all encryption, decryption, hashing, and parsing of keys.
package crypto

import (
	"common/bchcls/custom_errors"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"io"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("crypto")

// EncryptWithPublicKey encrypts data using the provided RSA public key.
func EncryptWithPublicKey(publicKey *rsa.PublicKey, data []byte) ([]byte, error) {
	b := make([]byte, 2048)
	r := strings.NewReader(string(b[:]))
	return rsa.EncryptPKCS1v15(r, publicKey, data)
}

// DecryptWithPrivateKey decrypts data using the provided RSA private key.
func DecryptWithPrivateKey(privateKey *rsa.PrivateKey, data []byte) ([]byte, error) {
	b := make([]byte, 2048)
	r := strings.NewReader(string(b[:]))
	return rsa.DecryptPKCS1v15(r, privateKey, data)
}

// EncryptWithSymKey encrypts data using the provided AES sym key.
func EncryptWithSymKey(symKey []byte, data []byte) ([]byte, error) {
	// Check that symKey is a valid AES sym key
	if !ValidateSymKey(symKey) {
		err := errors.WithStack(&custom_errors.InvalidSymKeyError{})
		logger.Errorf("%v", err)
		return nil, err
	}

	// CBC mode works on blocks so plaintexts may need to be padded to the
	// next whole block.
	data = pad(data)

	// Create a new cipher using the key you want to use.
	block, err := aes.NewCipher(symKey)
	if err != nil {
		logger.Errorf("Failed aes.NewCipher: %v", err)
		return nil, errors.Wrap(err, "Failed aes.NewCipher")
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	copy(iv, data)

	// Set up the encrypter
	mode := cipher.NewCBCEncrypter(block, iv)
	// Encrypt the string
	mode.CryptBlocks(ciphertext[aes.BlockSize:], data)
	if ciphertext == nil {
		err = errors.WithStack(&custom_errors.CiphertextEmptyError{})
		logger.Errorf("%v", err)
		return nil, err
	}
	return ciphertext, nil
}

// DecryptWithSymKey decrypts data with the provided AES sym key.
func DecryptWithSymKey(symKey []byte, encryptedData []byte) ([]byte, error) {
	// Check that symKey is a valid AES sym key
	if !ValidateSymKey(symKey) {
		err := errors.WithStack(&custom_errors.InvalidSymKeyError{})
		logger.Errorf("%v", err)
		return nil, err
	}

	// Create a new cipher using the key you want to use.
	block, err := aes.NewCipher(symKey)
	if err != nil {
		logger.Errorf("Failed aes.NewCipher: %v", err)
		return nil, errors.Wrap(err, "Failed aes.NewCipher")
	}

	if encryptedData == nil {
		err = errors.WithStack(&custom_errors.CiphertextEmptyError{})
		logger.Errorf("%v", err)
		return nil, err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(encryptedData) < aes.BlockSize {
		err = errors.WithStack(&custom_errors.CiphertextLengthError{})
		logger.Errorf("%v", err)
		return nil, err
	}
	iv := encryptedData[:aes.BlockSize]
	//logger.Debugf("iv: %x", iv)
	encryptedData = encryptedData[aes.BlockSize:]
	//logger.Debugf("text: %x", ciphertext)

	// CBC mode always works in whole blocks.
	if len(encryptedData)%aes.BlockSize != 0 {
		err = errors.WithStack(&custom_errors.CiphertextBlockSizeError{})
		logger.Errorf("%v", err)
		return nil, err
	}

	// Set up the decrypter
	mode := cipher.NewCBCDecrypter(block, iv)

	// CryptBlocks can work in-place if the two arguments are the same.
	mode.CryptBlocks(encryptedData, encryptedData)

	// If the original plaintext lengths are not a multiple of the block
	// size, padding would have to be added when encrypting, which would be
	// removed at this point.
	encryptedData = unpad(encryptedData)

	// If encryptedData is nil after going through the unpad function,
	// then the plaintext was malformed, signifying that the cipher
	// text was not decrypted properly
	if encryptedData == nil {
		err = errors.New("Decryption Failure")
		logger.Errorf("%v", err)
		return nil, err
	}

	return encryptedData, nil
}

func pad(in []byte) []byte {
	//aes.BlockSize = 16
	padding := aes.BlockSize - (len(in) % aes.BlockSize)

	if padding == 0 {
		padding = aes.BlockSize
	}

	for i := 0; i < padding; i++ {
		in = append(in, byte(padding))
	}
	return in
}

// Remove the characters that are present after decrypting an AES string
func unpad(in []byte) []byte {
	if len(in) == 0 {
		return nil
	}

	padding := in[len(in)-1]

	if int(padding) > len(in) || padding > aes.BlockSize {
		return nil
	} else if padding == 0 {
		return nil
	}

	for i := len(in) - 1; i > len(in)-int(padding)-1; i-- {
		if in[i] != padding {
			return nil
		}
	}

	return in[:len(in)-int(padding)]
}

// HashShort returns the hash of the given key using SHA-1.
func HashShort(key []byte) []byte {
	h := sha1.New()
	io.WriteString(h, string(key[:]))
	return h.Sum(nil)
}

// HashShortB64 returns the base64 encoding of the SHA-1 hash of the given key.
func HashShortB64(key []byte) string {
	return EncodeToB64String(HashShort(key))
}

// Hash returns the hash of the given key using SHA-256.
func Hash(key []byte) []byte {
	h := sha256.New()
	io.WriteString(h, string(key[:]))
	return h.Sum(nil)
}

// HashB64 returns the base64 encoding of the SHA-256 hash of the given key.
func HashB64(key []byte) string {
	return EncodeToB64String(Hash(key))
}

// HashLong returns the SHA-512 of the given key. len=32
func HashLong(key []byte) []byte {
	h := sha512.New()
	io.WriteString(h, string(key[:]))
	return h.Sum(nil)

}

// HashLongB64 returns the base64 encoding of the SHA-512 hash of the given key.
func HashLongB64(key []byte) string {
	return EncodeToB64String(HashLong(key))
}

// ValidateSymKey validates a key is a sym key by checking its length.
func ValidateSymKey(key []byte) bool {
	return len(key) == 32
}

// ParseSymKeyB64 decodes a B64 sym key, confirms the result is a valid sym key, and returns.
// If not, return error.
func ParseSymKeyB64(symKeyB64 string) ([]byte, error) {
	symkey, err := base64.StdEncoding.DecodeString(symKeyB64)
	if err != nil || !ValidateSymKey(symkey) {
		custom_err := errors.WithStack(&custom_errors.InvalidSymKeyError{})
		logger.Errorf("%v", custom_err)
		return nil, custom_err
	}
	return symkey, nil
}

// GetSymKeyFromHash returns a sym key derived from the hash of the input seed.
func GetSymKeyFromHash(seed []byte) []byte {
	seedHash := Hash(seed)
	return seedHash[:32]
}

// MarshalPrivateKey marshals a *rsa.PrivateKey into a []byte.
func MarshalPrivateKey(key *rsa.PrivateKey) []byte {
	return x509.MarshalPKCS1PrivateKey(key)
}

// ParsePrivateKey parses key bytes into a *rsa.PrivateKey.
func ParsePrivateKey(key []byte) (*rsa.PrivateKey, error) {
	var privkey *rsa.PrivateKey
	if cs1key, err1 := x509.ParsePKCS1PrivateKey(key); err1 == nil {
		privkey = cs1key
	} else if cs8key, err2 := x509.ParsePKCS8PrivateKey(key); err2 == nil {
		privkey = cs8key.(*rsa.PrivateKey)
	} else {
		logger.Errorf("Unable to parse private key using x509.ParsePKCS1PrivateKey: %v", err1)
		logger.Errorf("Unable to parse private key using x509.ParsePKCS8PrivateKey: %v", err2)
		return nil, errors.New("Unable to parse private key")
	}
	return privkey, nil
}

// ValidatePrivateKey returns true if the key is a valid RSA private key.
func ValidatePrivateKey(key []byte) bool {
	_, err1 := x509.ParsePKCS1PrivateKey(key)
	if err1 == nil {
		return true
	}
	_, err2 := x509.ParsePKCS8PrivateKey(key)
	if err2 == nil {
		return true
	}
	return false
}

// ParsePublicKey parses key bytes into a *rsa.PublicKey.
func ParsePublicKey(key []byte) (*rsa.PublicKey, error) {
	puk, err := x509.ParsePKIXPublicKey(key)
	if err != nil || puk == nil {
		logger.Errorf("Unable to parse public key: %v", err)
		return nil, errors.New("Unable to parse public key")
	}

	publicKey, ok := puk.(*rsa.PublicKey)
	if !ok || publicKey == nil {
		logger.Error("Unable to cast public key to *rsa.PublicKey")
		return nil, errors.New("Unable to cast public key to *rsa.PublicKey")
	}

	return publicKey, nil
}

// ValidatePublicKey returns true if the key is a valid RSA public key.
func ValidatePublicKey(key []byte) bool {
	puk, err := x509.ParsePKIXPublicKey(key)
	if err != nil || puk == nil {
		return false
	}
	publicKey, ok := puk.(*rsa.PublicKey)
	if !ok || publicKey == nil {
		return false
	}
	return true
}

// ParsePrivateKeyB64 parses a B64 key string into a *rsa.PrivateKey.
func ParsePrivateKeyB64(privateKeyB64 string) (*rsa.PrivateKey, error) {
	decodedPrivateKey, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil || decodedPrivateKey == nil {
		logger.Errorf("invalid privateKeyB64 - %v: %v", privateKeyB64, err)
		return nil, errors.New("invalid privateKeyB64")
	}
	return ParsePrivateKey(decodedPrivateKey)
}

// ParsePublicKeyB64 parses a B64 key string into a *rsa.PublicKey.
func ParsePublicKeyB64(publicKeyB64 string) (*rsa.PublicKey, error) {
	decodedPublicKey, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil || decodedPublicKey == nil {
		logger.Errorf("invalid publicKeyB64 - %v: %v", publicKeyB64, err)
		return nil, errors.New("invalid publicKeyB64")
	}
	return ParsePublicKey(decodedPublicKey)
}

// PrivateKeyToBytes marshals a *rsa.PrivateKey into a []byte.
func PrivateKeyToBytes(privateKey *rsa.PrivateKey) []byte {
	return x509.MarshalPKCS1PrivateKey(privateKey)
}

// PublicKeyToBytes marshals a *rsa.PublicKey into a []byte.
func PublicKeyToBytes(publicKey *rsa.PublicKey) []byte {
	pub, _ := x509.MarshalPKIXPublicKey(publicKey)
	return pub
}

// DecodeStringB64 parses a B64 string into a []byte.
func DecodeStringB64(stringB64 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(stringB64)
}

// EncodeToB64String encodes data to a b64 string.
func EncodeToB64String(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// GeneratePrivateKey generates a random 2048-bit RSA Private Key.
func GeneratePrivateKey() *rsa.PrivateKey {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	return privateKey
}

// GenerateSymKey generates a random 32-byte AES symmetric key.
func GenerateSymKey() []byte {
	symKey := make([]byte, 32)
	rand.Read(symKey)
	return symKey
}
