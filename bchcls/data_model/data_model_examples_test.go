/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package data_model contains structs used across packages to prevent circular imports.
// For example, the User struct is needed by both asset_mgmt and user_mgmt, but user_mgmt
// depends on functions in asset_mgmt.
// They can't import each other, so the shared structs live here.
package data_model

import (
	"common/bchcls/internal/key_mgmt_i/key_mgmt_c/key_mgmt_g"
	"fmt"
)

func ExampleGetEncryptedDataBytes() {
	// assume data is encrypted
	dataBytes := []byte{}

	GetEncryptedDataBytes(dataBytes)
}

func ExampleIsEncryptedData() {
	// assume data is encrypted
	encryptedDataBytes := []byte{}
	wrappedEncryptedDataBytes := GetEncryptedDataBytes(encryptedDataBytes)

	isEncryptedData := IsEncryptedData(wrappedEncryptedDataBytes)

	fmt.Println(isEncryptedData)
	// Output: true
}

func ExampleUser_Equal() {
	user := User{
		ID:      "user1",
		Name:    "name1",
		Role:    "user",
		IsGroup: false,
	}
	person := User{
		ID:      "user1",
		Name:    "name1",
		Role:    "user",
		IsGroup: false,
	}

	isEqual := user.Equal(person)

	fmt.Println(isEqual)
	// Output: true
}

func ExampleGetPubPrivKeyId() {
	pubPrivKeyId := key_mgmt_g.GetPubPrivKeyId("id1")

	fmt.Println(pubPrivKeyId)
	// Output: pub-priv-id1
}

func ExampleGetSymKeyId() {
	symKeyId := key_mgmt_g.GetSymKeyId("id1")

	fmt.Println(symKeyId)
	// Output: sym-id1
}

func ExampleGetLogSymKeyId() {
	logSymKeyId := key_mgmt_g.GetLogSymKeyId("id1")

	fmt.Println(logSymKeyId)
	// Output: log-sym-id1
}

func ExampleUser_GetPubPrivKeyId() {
	user := User{ID: "user1"}

	pubPrivKeyId := user.GetPubPrivKeyId()

	fmt.Println(pubPrivKeyId)
	// Output: pub-priv-user1
}

func ExampleUser_GetSymKeyId() {
	user := User{ID: "user1"}

	symKeyId := user.GetSymKeyId()

	fmt.Println(symKeyId)
	// Output: sym-user1
}

func ExampleUser_GetLogSymKeyId() {
	user := User{ID: "user1"}

	logSymKeyId := user.GetLogSymKeyId()

	fmt.Println(logSymKeyId)
	// Output: log-sym-user1
}

func ExampleUser_GetPrivateKeyHashSymKeyId() {
	user := User{ID: "user1"}

	privateKeyHashSymKeyId := user.GetPrivateKeyHashSymKeyId()

	fmt.Println(privateKeyHashSymKeyId)
	// Output: private-hash-user1
}

func ExampleKey_GetLogSymKeyId() {
	key := Key{ID: "key1"}

	logSymKeyId := key.GetLogSymKeyId()

	fmt.Println(logSymKeyId)
	// Output: log-sym-key1
}
