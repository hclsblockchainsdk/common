/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
)

func ExampleEncryptWithPublicKey() {
	data := []byte("data")
	privateKey := GeneratePrivateKey()
	publicKey := privateKey.Public().(*rsa.PublicKey)

	EncryptWithPublicKey(publicKey, data)
}

func ExampleDecryptWithPrivateKey() {
	data := []byte("data")
	privateKey := GeneratePrivateKey()
	publicKey := privateKey.Public().(*rsa.PublicKey)

	encryptedData, _ := EncryptWithPublicKey(publicKey, data)

	DecryptWithPrivateKey(privateKey, encryptedData)
}

func ExampleEncryptWithSymKey() {
	data := []byte("data")
	symKey := GenerateSymKey()

	EncryptWithSymKey(symKey, data)
}

func ExampleDecryptWithSymKey() {
	data := []byte("data")
	symKey := GenerateSymKey()

	encryptedData, _ := EncryptWithSymKey(symKey, data)

	DecryptWithSymKey(symKey, encryptedData)
}

func ExampleHash() {
	data := []byte("data")

	Hash(data)
}

func ExampleValidateSymKey() {
	symKey := GenerateSymKey()

	validSymKey := ValidateSymKey(symKey)

	fmt.Println(validSymKey)
	// Output: true
}

func ExampleParseSymKeyB64() {
	symKeyB64 := base64.StdEncoding.EncodeToString(GenerateSymKey())

	ParseSymKeyB64(symKeyB64)
}

func ExampleGetSymKeyFromHash() {
	data := []byte("data")

	GetSymKeyFromHash(data)
}

func ExampleMarshalPrivateKey() {
	privateKey := GeneratePrivateKey()

	MarshalPrivateKey(privateKey)
}

func ExampleParsePrivateKey() {
	privateKeyBytes := MarshalPrivateKey(GeneratePrivateKey())

	ParsePrivateKey(privateKeyBytes)
}

func ExampleValidatePrivateKey() {
	privateKey := GeneratePrivateKey()
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)

	validPrivateKey := ValidatePrivateKey(privateKeyBytes)

	fmt.Println(validPrivateKey)
	// Output: true
}

func ExampleParsePublicKey() {
	privateKey := GeneratePrivateKey()
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(privateKey.Public())

	ParsePublicKey(publicKeyBytes)
}

func ExampleValidatePublicKey() {
	privateKey := GeneratePrivateKey()
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(privateKey.Public())

	validPublicKey := ValidatePublicKey(publicKeyBytes)

	fmt.Println(validPublicKey)
	// Output: true
}

func ExampleParsePrivateKeyB64() {
	privateKey := GeneratePrivateKey()
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyB64 := base64.StdEncoding.EncodeToString(privateKeyBytes)

	ParsePrivateKeyB64(privateKeyB64)
}

func ExampleParsePublicKeyB64() {
	privateKey := GeneratePrivateKey()
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(privateKey.Public())
	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKeyBytes)

	ParsePublicKeyB64(publicKeyB64)
}

func ExamplePrivateKeyToBytes() {
	privateKey := GeneratePrivateKey()

	PrivateKeyToBytes(privateKey)
}

func ExamplePublicKeyToBytes() {
	privateKey := GeneratePrivateKey()
	publicKey := privateKey.Public().(*rsa.PublicKey)

	PublicKeyToBytes(publicKey)
}

func ExampleDecodeStringB64() {
	b64String := base64.StdEncoding.EncodeToString([]byte("data"))

	DecodeStringB64(b64String)
}

func ExampleEncodeToB64String() {
	data := []byte("data")

	EncodeToB64String(data)
}
