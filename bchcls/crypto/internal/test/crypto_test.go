/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package test

import (
	"common/bchcls/crypto"
	"common/bchcls/custom_errors"
	"common/bchcls/test_utils"

	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/pkg/errors"
)

func TestPublicPrivateKeyEncryption(t *testing.T) {
	fmt.Println("TestPublicPrivateKeyEncryption function called")
	fmt.Println("-- Tests EncryptWithPublicKey")
	fmt.Println("-- Tests DecryptWithPrivateKey")

	originalData := []byte("mydata")
	privateKey := test_utils.GeneratePrivateKey()

	// Encrypt with public key
	encryptedData, err1 := crypto.EncryptWithPublicKey(privateKey.Public().(*rsa.PublicKey), originalData)
	// Decrypted key
	decryptedData, err2 := crypto.DecryptWithPrivateKey(privateKey, encryptedData)

	// After encryption and decryption, data should be the same
	test_utils.AssertTrue(t, bytes.Equal(originalData, decryptedData), "Expected to get originalData")
	test_utils.AssertTrue(t, err1 == nil, "No error returned from EncryptWithPublicKey function")
	test_utils.AssertTrue(t, err2 == nil, "No error returned from DecryptWithPrivateKey function")

	// Negative test - mismatching private and public keys, DecryptWithPrivateKey should not work
	privateKey2 := test_utils.GeneratePrivateKey()
	encryptedData2, err3 := crypto.EncryptWithPublicKey(privateKey.Public().(*rsa.PublicKey), originalData)
	_, err4 := crypto.DecryptWithPrivateKey(privateKey2, encryptedData2)
	test_utils.AssertTrue(t, err3 == nil, "No error returned from EncryptWithPublicKey function")
	test_utils.AssertFalse(t, err4 == nil, "Should be error because mismatching private and public keys")
}

func TestParsePrivateKey_PKCS1(t *testing.T) {
	fmt.Println("TestParsePrivateKey_PKCS1 function called")
	fmt.Println("-- Tests ParsePrivateKey_PKCS1")

	priv := test_utils.GeneratePrivateKey()
	privM := x509.MarshalPKCS1PrivateKey(priv)
	privateKey, err1 := crypto.ParsePrivateKey(privM)

	test_utils.AssertTrue(t, err1 == nil, "No error returned from ParsePrivateKey function")
	test_utils.AssertTrue(t, privateKey != nil, "privateKey is not nil")
}

func TestParsePrivateKey_PKCS8(t *testing.T) {
	fmt.Println("TestParsePrivateKey_PKCS8 function called")
	fmt.Println("-- Tests ParsePrivateKey_PKCS8")

	priv := test_utils.GeneratePrivateKey()
	privM, _ := convertPrivateKeyToPKCS8(priv)
	privateKey, err1 := crypto.ParsePrivateKey(privM)

	test_utils.AssertTrue(t, err1 == nil, "No error returned from ParsePrivateKey function")
	test_utils.AssertTrue(t, privateKey != nil, "privateKey is not nil")
}

func TestParsePrivateKey_invalidKey(t *testing.T) {
	fmt.Println("TestParsePrivateKey_invalidKey function called")
	fmt.Println("-- Tests ParsePrivateKey with invalid key")

	// Negative test - ParsePrivateKey got passed invalid key
	invalidKey := []byte("key")
	_, err1 := crypto.ParsePrivateKey(invalidKey)
	test_utils.AssertFalse(t, err1 == nil, "Should be error because ParsePrivateKey was passed an invalid private key")
}

func TestParsePublicKey(t *testing.T) {
	fmt.Println("TestParsePublicKey function called")
	fmt.Println("-- Tests ParsePublicKey")

	// generates *rsa.PrivateKey
	privateKey := test_utils.GeneratePrivateKey()
	// serialises a public key to DER-encoded PKIX format
	pub, err1 := x509.MarshalPKIXPublicKey(privateKey.Public())
	// parses public key
	publicKey, err2 := crypto.ParsePublicKey(pub)

	test_utils.AssertTrue(t, err1 == nil, "No error returned from MarshalPKIXPublicKey function")
	test_utils.AssertTrue(t, err2 == nil, "No error returned from ParsePublicKey function")
	test_utils.AssertTrue(t, publicKey != nil, "publicKey is not nil")
}

func TestParsePublicKey_invalidKey(t *testing.T) {
	fmt.Println("TestParsePublicKey_invalidKey function called")
	fmt.Println("-- Tests ParsePublicKey with invalid key")

	// Negative test - ParsePublicKey is passed invalid publicKey
	invalidKey := []byte("key")
	_, err1 := crypto.ParsePublicKey(invalidKey)
	test_utils.AssertFalse(t, err1 == nil, "Should be error because ParsePublicKey was passed an invalid public key")
}

func TestSymKeyEncryption(t *testing.T) {
	fmt.Println("TestSymKeyEncryption function called")
	fmt.Println("-- Tests EncryptWithSymKey")
	fmt.Println("-- Tests DecryptWithSymKey")

	originalData := []byte("mydata")
	// generates sym key byte
	symKey := test_utils.GenerateSymKey()
	// Encrypt with public key
	encryptedData, err1 := crypto.EncryptWithSymKey(symKey, originalData)
	// Decrypted key
	decryptedData, err2 := crypto.DecryptWithSymKey(symKey, encryptedData)

	// After encryption and decryption, data should be the same
	test_utils.AssertTrue(t, bytes.Equal(originalData, decryptedData), "Expected to get originalData")
	test_utils.AssertTrue(t, err1 == nil, "No error returned from EncryptWithSymKey function")
	test_utils.AssertTrue(t, err2 == nil, "No error returned from DecryptWithSymKey function")
}

func TestSymKeyEncryption_invalidKey(t *testing.T) {
	fmt.Println("TestSymKeyEncryption_invalidKey function called")
	fmt.Println("-- Tests EncryptWithSymKey with invalid key")
	fmt.Println("-- Tests DecryptWithSymKey with invalid key")

	originalData := []byte("mydata")

	// Negative test - invalid symkey was passed to EncryptWithSymKey
	symKey := make([]byte, 64)
	_, err1 := crypto.EncryptWithSymKey(symKey, originalData)
	test_utils.AssertFalse(t, err1 == nil, "Should be error because invalid symkey was passed to EncryptWithSymKey")

	// Negative test - invalid symkey was passed to DecryptWithSymKey
	_, err2 := crypto.DecryptWithSymKey(symKey, originalData)
	test_utils.AssertFalse(t, err2 == nil, "Should be error because invalid symkey was passed to DecryptWithSymKey")
}

func TestDecryptWithSymKey_WrongKey(t *testing.T) {
	fmt.Println("TestDecryptWithSymKeyWrongKey function called")

	// Encrypt some data
	originalData := []byte("mydata")
	symKey := test_utils.GenerateSymKey()
	encryptedData, err1 := crypto.EncryptWithSymKey(symKey, originalData)
	test_utils.AssertTrue(t, err1 == nil, "No error returned from EncryptWithSymKey")

	// Decrypt the data with the wrong key
	wrongSymKey := test_utils.GenerateSymKey()
	_, err2 := crypto.DecryptWithSymKey(wrongSymKey, encryptedData)
	test_utils.AssertTrue(t, err2 != nil, "Should be error because wrong symkey was passed to EncryptWithSymKey")
	_, ok := errors.Cause(err2).(*custom_errors.CiphertextEmptyError)
	test_utils.AssertFalse(t, ok, "Should not be ciphertext is empty error")
}

func TestDecryptWithSymKey_EmptyCipherText(t *testing.T) {
	fmt.Println("TestDecryptWithSymKeyEmptyCipherText function called")

	// Decrypt nil ciphertext with valid symkey
	symKey := test_utils.GenerateSymKey()
	_, err := crypto.DecryptWithSymKey(symKey, nil)
	test_utils.AssertTrue(t, err != nil, "Should be error because empty ciphertext was passed to EncryptWithSymKey")
	_, ok := errors.Cause(err).(*custom_errors.CiphertextEmptyError)
	test_utils.AssertTrue(t, ok, "Should be ciphertext is empty error")
}

func TestDecryptWithSymKey_ShortCipherText(t *testing.T) {
	fmt.Println("TestDecryptWithSymKeyEmptyCipherText function called")

	symKey := test_utils.GenerateSymKey()

	// Decrypt len 0 ciphertext with valid symkey
	_, err1 := crypto.DecryptWithSymKey(symKey, make([]byte, 0))
	test_utils.AssertTrue(t, err1 != nil, "Should be error because short ciphertext was passed to EncryptWithSymKey")
	_, ok := errors.Cause(err1).(*custom_errors.CiphertextLengthError)
	test_utils.AssertTrue(t, ok, "Should be ciphertext too short error")

	// Decrypt short ciphertext with valid symkey
	_, err2 := crypto.DecryptWithSymKey(symKey, make([]byte, 1))
	test_utils.AssertTrue(t, err2 != nil, "Should be error because short ciphertext was passed to EncryptWithSymKey")
	_, ok = errors.Cause(err2).(*custom_errors.CiphertextLengthError)
	test_utils.AssertTrue(t, ok, "Should be ciphertext too short error")
}

func TestValidateSymKey(t *testing.T) {
	fmt.Println("TestValidateSymKey function called")
	fmt.Println("-- Tests ValidateSymKey")

	// generates sym key byte
	symKey := test_utils.GenerateSymKey()
	// asserts valid sym key with valid length
	test_utils.AssertTrue(t, crypto.ValidateSymKey(symKey), "sym key validated")
}

func TestValidateSymKey_invalidKey(t *testing.T) {
	fmt.Println("TestValidateSymKey_invalidKey function called")
	fmt.Println("-- Tests ValidateSymKey with invalid key")

	// Negative test - length of symKey is 64
	symKey := make([]byte, 64)
	test_utils.AssertFalse(t, crypto.ValidateSymKey(symKey), "sym key len is 64, should fail")
}

func TestJavascriptB64(t *testing.T) {
	fmt.Println("TestJavascriptB64 function called")
	fmt.Println("-- Tests ParsePrivateKeyB64")
	fmt.Println("-- Tests ParsePublicKeyB64")
	fmt.Println("-- Tests EncryptWithPublicKey")
	fmt.Println("-- Tests DecryptWithPrivateKey")
	fmt.Println("-- Tests ParseSymKeyB64")

	originalData := []byte("mydata")

	privB64 := "MIIEogIBAAKCAQEAiuSU5hEEni5DENF46UDWYym2f9viE+ns8f4jRNFBbjAgGkKai8U2BbGZLyV8MtzoWePkEuo1HnOvdWCqMf6MmQdWmGFHPMJYNMtYG6Ku+ekm7NRNWS3mtgva/DF3JToYhvM/3o/K5Bg+cR9JP2YyKX7wksawSmqArTt+K9CqTNbguvDWeNP90mAADVZRDt1LHFkfp75MSM3gO3KQtR2qUZgYIJQ2/s+B63Vs9ACenaDg2pxG+w9L0IkIsTtLfc55XBSk5o3OEF/8RIJIyN86j53Tsvm0CF9ZlW6B8pXbSXKWvd3irVNk+k+UNfkt8k7S2UVCjCkDfV9Ly0eidZo+UQIDAQABAoIBABFFLPKSiSV2ESbFNSijxESeSjAJ0kmxm6HXfOEwt9cQqt05DOh2RCpfE/IV0iSs7UNIH/LuJl67+cQ5mdAPm8HndLAL4ITAkaE266S8DM/MWue12kxNddOLE9ap++uoFqapFncBIDROg20je8MjXPdl7loB1KfcKFXiAOVH0/Urz8mSIod83nR+7xGiRLbJ1yzBsmEgEP0FG1DTj1GLVp4ml6SgPkUO6opE+KnUbKyqKWTby0dc/9sLSMDzxiO/02J57JjBGuBLGPzM2/zYYrkCzlCkBx1gdqOA3BvZSSAmxj3vfblpYWgvMEIG8uXE2HZHO29twM3DooUSmqu2zLECgYEAzA8+9SCM7a8PvLqCV3gK9jYEx5PLWAlRCPEHACfLDk4Mn1islKkKnxGkZjfxbzYxYbOxa/SaALb1Jdw/su7CCrN/x/S8YvcudCSWkLWjEkyV2HHgfBs8zst7O5gt9a378dcErNbCqJlwz7FscOMm8rBqH6O+nW5A+VkxFOsXvpUCgYEArj8Ach3VV8irWTOvbbJi74FNnfNDd4c0c1SiSOw85YpvDDxM81Ar4HUdnjEr2ePP8EbEJenUmWp3dkEwx0mndefne2FJl4m489l+TsARakiTM7loI+8Id6B3izE9bL2NQCHCgg3CEBzeoMrRghIl87oo1ZtBNNjZ5DhP7K/Q3c0CgYBvdAhL9Gpky5AJ4cidI6jBD4IOy69ttzD2dEcBk7p5ZrHIOrOQQr/VX9puJjINLwlLtsy6DRAsQcGl2yVDgtqi46VwPkDCwQUzVGSUR1D5BrF1VcVpo6rTvBnj09uOa2fHkEwyZt5NHfmaxupWwgqc1TQxccsDy5tbVZbUOQ9v/QKBgFgqpsYXVGETt6fuICEId0krCyQV+BheAGsu8uKlLncTIfd195XSBjGP8QmfZcndnNS+afepJpruJT5f5BeirCpbymRCqOMVm9E/RssAIo+940Xz1b8A7y7gsjxrEOKZ0wQVUP9PiNdwVDHWDkabejqmAo16/naIF1CEMeTYXX4tAoGAPlP+JCyZ3UhwJbCN8v+ftCyD1KM8viZ2q1gjPl7RcUjYrXBsWXq18IQL4xeWQ6W3/J9xinHKoKGcFfXkbCNtjsT85fMGQjxr/UXrpe4AM+uY5a7NdlaE1x4r1CKzMjJRjCeG6RfG/ly7MCBw7hPE2u2tMYSAuHggX+IIo8jwuyE="

	pubB64 := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAiuSU5hEEni5DENF46UDWYym2f9viE+ns8f4jRNFBbjAgGkKai8U2BbGZLyV8MtzoWePkEuo1HnOvdWCqMf6MmQdWmGFHPMJYNMtYG6Ku+ekm7NRNWS3mtgva/DF3JToYhvM/3o/K5Bg+cR9JP2YyKX7wksawSmqArTt+K9CqTNbguvDWeNP90mAADVZRDt1LHFkfp75MSM3gO3KQtR2qUZgYIJQ2/s+B63Vs9ACenaDg2pxG+w9L0IkIsTtLfc55XBSk5o3OEF/8RIJIyN86j53Tsvm0CF9ZlW6B8pXbSXKWvd3irVNk+k+UNfkt8k7S2UVCjCkDfV9Ly0eidZo+UQIDAQAB"

	privateKey, err1 := crypto.ParsePrivateKeyB64(privB64)
	publicKey, err2 := crypto.ParsePublicKeyB64(pubB64)

	// Encrypt with public key
	encryptedData, err3 := crypto.EncryptWithPublicKey(publicKey, originalData)
	// Decrypted key
	decryptedData, err4 := crypto.DecryptWithPrivateKey(privateKey, encryptedData)

	// After encryption and decryption, data should be the same
	test_utils.AssertTrue(t, bytes.Equal(originalData, decryptedData), "Expected to get originalData")
	test_utils.AssertTrue(t, err1 == nil, "No error returned from ParsePrivateKeyB64 function")
	test_utils.AssertTrue(t, err2 == nil, "No error returned from ParsePublicKeyB64 function")
	test_utils.AssertTrue(t, err3 == nil, "No error returned from EncryptWithPublicKey function")
	test_utils.AssertTrue(t, err4 == nil, "No error returned from DecryptWithPrivateKeys function")

	// Test ParseSymKeyB64
	symKeyB64 := "REU+HfdVfh3dbyqw6SEaVK3hR7v931mmWi3IUR+69K4="
	symKey, err5 := crypto.ParseSymKeyB64(symKeyB64)
	test_utils.AssertTrue(t, err5 == nil, "No error returned from ParseSymKeyB64 function")
	test_utils.AssertTrue(t, symKey != nil, "Symkey is not nil")

	// Generate priv, pub, and sym and send to JS
	priv2 := test_utils.GeneratePrivateKey()
	priv2MB64 := base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(priv2))
	pub2M, err5 := x509.MarshalPKIXPublicKey(priv2.Public())
	pub2MB64 := base64.StdEncoding.EncodeToString(pub2M)

	test_utils.AssertTrue(t, err5 == nil, "No error returned from MarshalPKIXPublicKey function")

	fmt.Println()
	fmt.Println("privateKeyB64 (to send to JS): ")
	fmt.Println(priv2MB64)
	fmt.Println()
	fmt.Println("publicKeyB64 (to send to JS): ")
	fmt.Println(pub2MB64)

	symKey2 := test_utils.GenerateSymKey()
	symKey2B64 := base64.StdEncoding.EncodeToString(symKey2)
	fmt.Println()
	fmt.Println("symKeyB64 (to send to JS): ")
	fmt.Println(symKey2B64)
	fmt.Println()
}

func TestParsePrivateKeyB64_invalidKey(t *testing.T) {
	fmt.Println("TestParsePrivateKeyB64_invalidKey function called")
	fmt.Println("-- Tests ParsePrivateKeyB64 with invalid key")

	invalidKey := "key"
	_, err1 := crypto.ParsePrivateKeyB64(invalidKey)
	test_utils.AssertFalse(t, err1 == nil, "Should be error because key passed in was not B64")

	symKeyB64 := "REU+HfdVfh3dbyqw6SEaVK3hR7v931mmWi3IUR+69K4="
	_, err2 := crypto.ParsePrivateKeyB64(symKeyB64)
	test_utils.AssertFalse(t, err2 == nil, "Should be error because key passed in was sym key not private key")
}

func TestParsePublicKeyB64_invalidKey(t *testing.T) {
	fmt.Println("TestParsePublicKeyB64_invalidKey function called")
	fmt.Println("-- Tests ParsePublicKeyB64 with invalid key")

	invalidKey := "key"
	_, err1 := crypto.ParsePublicKeyB64(invalidKey)
	test_utils.AssertFalse(t, err1 == nil, "Should be error because key passed in was not B64")

	symKeyB64 := "REU+HfdVfh3dbyqw6SEaVK3hR7v931mmWi3IUR+69K4="
	_, err2 := crypto.ParsePublicKeyB64(symKeyB64)
	test_utils.AssertFalse(t, err2 == nil, "Should be error because key passed in was sym key not public key")
}

func TestParseSymKeyB64_invalidKey(t *testing.T) {
	fmt.Println("TestParseSymKeyB64_invalidKey function called")
	fmt.Println("-- Tests ParseSymKeyB64 with invalid key")

	invalidKey := "key"
	_, err1 := crypto.ParseSymKeyB64(invalidKey)
	test_utils.AssertFalse(t, err1 == nil, "Should be error because key passed in was not B64")

	pubB64 := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAiuSU5hEEni5DENF46UDWYym2f9viE+ns8f4jRNFBbjAgGkKai8U2BbGZLyV8MtzoWePkEuo1HnOvdWCqMf6MmQdWmGFHPMJYNMtYG6Ku+ekm7NRNWS3mtgva/DF3JToYhvM/3o/K5Bg+cR9JP2YyKX7wksawSmqArTt+K9CqTNbguvDWeNP90mAADVZRDt1LHFkfp75MSM3gO3KQtR2qUZgYIJQ2/s+B63Vs9ACenaDg2pxG+w9L0IkIsTtLfc55XBSk5o3OEF/8RIJIyN86j53Tsvm0CF9ZlW6B8pXbSXKWvd3irVNk+k+UNfkt8k7S2UVCjCkDfV9Ly0eidZo+UQIDAQAB"
	_, err2 := crypto.ParseSymKeyB64(pubB64)
	test_utils.AssertFalse(t, err2 == nil, "Should be error because key passed in was pub key not sym key")
}

func TestGetSymKeyFromHash(t *testing.T) {
	fmt.Println("TestGetSymKeyFromHash function called")
	testBytes1 := []byte("1")
	testBytes2 := []byte("01234567890123456789012345678901")
	testBytes3 := []byte("0123456789012345678901234567890101234567890123456789012345678901012345678901234567890123456789010123456789012345678901234567890101234567890123456789012345678901")

	hash1 := crypto.GetSymKeyFromHash(testBytes1)
	test_utils.AssertTrue(t, len(hash1) == 32, "Expected hash to be 32 bytes long")
	hash2 := crypto.GetSymKeyFromHash(testBytes2)
	test_utils.AssertTrue(t, len(hash2) == 32, "Expected hash to be 32 bytes long")
	hash3 := crypto.GetSymKeyFromHash(testBytes3)
	test_utils.AssertTrue(t, len(hash3) == 32, "Expected hash to be 32 bytes long")
}

func convertPrivateKeyToPKCS8(priv *rsa.PrivateKey) ([]byte, error) {
	// pkcs8 reflects an ASN.1, PKCS#8 PrivateKey. See RFC 5208.
	type pkcs8 struct {
		Version    int
		Algo       []asn1.ObjectIdentifier
		PrivateKey []byte
	}

	var privPKCS8 pkcs8
	privPKCS8.Version = 0
	privPKCS8.Algo = make([]asn1.ObjectIdentifier, 1)
	privPKCS8.Algo[0] = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	privPKCS8.PrivateKey = x509.MarshalPKCS1PrivateKey(priv)

	return asn1.Marshal(privPKCS8)
}
