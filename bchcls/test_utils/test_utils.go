/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package test_utils contains test utility functions for creating users, printing
// the ledger, and etc.
// These functions should only be used in unit tests.
package test_utils

import (
	"common/bchcls/data_model"
	"common/bchcls/internal/common/global"

	"github.com/hyperledger/fabric/core/chaincode/shim"

	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	mrand "math/rand"
	"reflect"
	"runtime/debug"
	"strconv"
	"testing"
)

var logger = shim.NewLogger("test_utils")

// TestAssetData can be used as a test asset's public or private data.
type TestAssetData struct {
	Data string
}

// AssertTrue asserts that the given boolean is true.
func AssertTrue(t *testing.T, assertion bool, message string) {
	if !assertion {
		debug.PrintStack()
		t.Fatalf(message)
	}
}

// AssertFalse asserts that the given boolean is false.
func AssertFalse(t *testing.T, assertion bool, message string) {
	if assertion {
		debug.PrintStack()
		t.Fatalf(message)
	}
}

// AssertNilError if myError is not nil, prints error details/stack and fails the test
func AssertNilError(t *testing.T, myError error, message string) {
	if myError != nil {
		debug.PrintStack()
		logger.Errorf("%v || ErrorDetails: %v", message, myError)
		t.Fatalf(message)
	}
}

// AssertInLists asserts that expectedValue is in expectedList.
func AssertInLists(t *testing.T, expectedValue string, expectedList []string, message string) {
	assertion := false
	for _, key := range expectedList {
		if key == expectedValue {
			assertion = true
			break
		}
	}
	if !assertion {
		debug.PrintStack()
		t.Log(message)
		t.Fatalf("Key %v is not in the list %v.", expectedValue, expectedList)
	}
}

// AssertListsEqual asserts that two lists are equal.
func AssertListsEqual(t *testing.T, expectedList []string, actualList []string) {
	if len(expectedList) != len(actualList) {
		debug.PrintStack()
		t.Fatalf("List of keys was incorrect, got: %v, want: %v.", actualList, expectedList)
	}
	for i, key := range actualList {
		if key != expectedList[i] {
			debug.PrintStack()
			t.Fatalf("List of keys was incorrect, got: %v, want: %v.", actualList, expectedList)
		}
	}
}

// AssertSetsEqual assets that two sets are equal.
func AssertSetsEqual(t *testing.T, expectedList []string, actualList []string) {
	if len(expectedList) != len(actualList) {
		debug.PrintStack()
		t.Fatalf("List of keys was incorrect, got: %v, want: %v.", actualList, expectedList)
	}
	// use a map to simulate a set
	actualMap := make(map[string]bool)
	// for each element in actualList, store in map
	for _, key := range actualList {
		actualMap[key] = true
	}
	// for each element in expectedList, check if it exists in map
	for _, key := range expectedList {
		if !actualMap[key] {
			debug.PrintStack()
			t.Fatalf("List of keys was incorrect, got: %v, want: %v.", actualList, expectedList)
		}
	}
}

// AssertMapsEqual assets that two maps are equal.
func AssertMapsEqual(t *testing.T, expectedMap interface{}, actualMap interface{}, message string) {
	if !reflect.DeepEqual(expectedMap, actualMap) {
		debug.PrintStack()
		t.Fatalf(message)
	}
}

// AssertStringInArray assets that a given string is in a given array.
func AssertStringInArray(t *testing.T, item string, array []string) {
	inArray := false
	for _, arrayItem := range array {
		if item == arrayItem {
			inArray = true
		}
	}
	if !inArray {
		debug.PrintStack()
		t.Fatalf("Expected item %v to be in array.", item)
	}
}

// AssertNil asserts that the provided value is nil.
// Useful for checking there are no errors: AssertNil(t, err).
func AssertNil(t *testing.T, actual interface{}) {
	if actual != nil {
		debug.PrintStack()
		t.Fatalf("Object %v should be null", actual)
	}
}

// GenerateSymKey generates a random 32-byte AES symmetric key.
// Key generation functions are only for testing.
// In production, keys will never be generated from the chaincode.
func GenerateSymKey() []byte {
	symKey := make([]byte, 32)
	rand.Read(symKey)
	return symKey
}

// CreateSymKey generates a sym key and returns it as part of a Key object.
// Key generation functions are only for testing.
// In production, keys will never be generated from the chaincode.
func CreateSymKey(keyId string) data_model.Key {
	return data_model.Key{ID: keyId, Type: global.KEY_TYPE_SYM, KeyBytes: GenerateSymKey()}
}

// GeneratePrivateKey generates a random 2048-bit RSA Private Key.
// Key generation functions are only for testing.
// In production, keys will never be generated from the chaincode.
func GeneratePrivateKey() *rsa.PrivateKey {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	return privateKey
}

// GenerateRandomTxID is a convenience function that generates a random transaction ID.
// Used only for testing during development when there is 1 peer.
// In production, TxID should never be generated from the chaincode even during testing.
func GenerateRandomTxID() string {
	return strconv.FormatInt(int64(mrand.Int()), 10)
}

// CreateTestAssetData returns data that can be used as a test asset's public or private data.
func CreateTestAssetData(data string) []byte {
	assetData := TestAssetData{Data: data}
	assetDataBytes, _ := json.Marshal(assetData)
	return assetDataBytes
}

// CreateTestAsset returns an asset for tests.
func CreateTestAsset(assetId string) data_model.Asset {
	return data_model.Asset{
		AssetId:     assetId,
		Datatypes:   []string{},
		PrivateData: CreateTestAssetData("private data"),
		PublicData:  CreateTestAssetData("public data"),
		Metadata:    make(map[string]string),
	}
}

// CreateTestUser creates a test user with random keys.
func CreateTestUser(userID string) data_model.User {
	testUser := data_model.User{}
	testUser.ID = userID
	testUser.PrivateKey = GeneratePrivateKey()
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(testUser.PrivateKey)
	testUser.PrivateKeyB64 = base64.StdEncoding.EncodeToString(privateKeyBytes)
	testUser.PublicKey = testUser.PrivateKey.Public().(*rsa.PublicKey)
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(testUser.PublicKey)
	testUser.PublicKeyB64 = base64.StdEncoding.EncodeToString(publicKeyBytes)
	testUser.SymKey = GenerateSymKey()
	testUser.SymKeyB64 = base64.StdEncoding.EncodeToString(testUser.SymKey)
	testUser.KmsPublicKeyId = "kmspubkeyid"
	testUser.KmsPrivateKeyId = "kmsprivkeyid"
	testUser.KmsSymKeyId = "kmssymkeyid"
	testUser.Email = "email@mail.com"
	testUser.Name = getDefaultUserNameFromID(userID)
	testUser.IsGroup = false
	testUser.Status = "active"
	testUser.Role = global.ROLE_USER
	testUser.Secret = "pass0"
	testUser.SolutionPublicData = make(map[string]interface{})
	testUser.SolutionPrivateData = make(map[string]interface{})
	return testUser
}

// CreateTestGroup creates a test group with random keys.
func CreateTestGroup(groupID string) data_model.User {
	testGroup := data_model.User{}
	testGroup.ID = groupID
	testGroup.PrivateKey = GeneratePrivateKey()
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(testGroup.PrivateKey)
	testGroup.PrivateKeyB64 = base64.StdEncoding.EncodeToString(privateKeyBytes)
	testGroup.PublicKey = testGroup.PrivateKey.Public().(*rsa.PublicKey)
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(testGroup.PublicKey)
	testGroup.PublicKeyB64 = base64.StdEncoding.EncodeToString(publicKeyBytes)
	testGroup.SymKey = GenerateSymKey()
	testGroup.SymKeyB64 = base64.StdEncoding.EncodeToString(testGroup.SymKey)
	testGroup.KmsPublicKeyId = "kmspubkeyid"
	testGroup.KmsPrivateKeyId = "kmsprivkeyid"
	testGroup.KmsSymKeyId = "kmssymkeyid"
	testGroup.Email = "none"
	testGroup.Name = getDefaultUserNameFromID(groupID)
	testGroup.IsGroup = true
	testGroup.Status = "active"
	testGroup.Role = global.ROLE_ORG
	testGroup.Secret = "pass0"
	solutionPrivateData := make(map[string]interface{})
	testGroup.SolutionPrivateData = solutionPrivateData
	testGroup.SolutionPublicData = make(map[string]interface{})
	return testGroup
}

// GetTransientMapFromUser returns a transienmap with user information to be used in Invoke
func GetTransientMapFromUser(user data_model.User) map[string][]byte {
	tmap := make(map[string][]byte)
	tmap["id"] = []byte(user.ID)
	tmap["prvkey"], _ = base64.StdEncoding.DecodeString(user.PrivateKeyB64)
	tmap["pubkey"], _ = base64.StdEncoding.DecodeString(user.PublicKeyB64)
	tmap["symkey"], _ = base64.StdEncoding.DecodeString(user.SymKeyB64)
	return tmap
}

// AddPHIArgsToTransientMap adds phi args to the transient map to be used in Invoke
func AddPHIArgsToTransientMap(tmap map[string][]byte, params [][]byte, args ...[]byte) (map[string][]byte, [][]byte) {
	for i, arg := range args {
		tmap["arg"+strconv.Itoa(i)] = arg
		h := sha256.New()
		io.WriteString(h, string(arg))
		hash := h.Sum(nil)
		hexHash := []byte(hex.EncodeToString(hash))
		params = append(params, hexHash)
	}
	if len(args) > 0 {
		tmap["num_args"] = []byte(strconv.Itoa(len(args)))
	}
	return tmap, params
}

func getDefaultUserNameFromID(userId string) string {
	return userId
}
