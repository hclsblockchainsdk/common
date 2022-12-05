/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package key_mgmt_c

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/common/graph"
	"common/bchcls/test_utils"

	"bytes"
	"crypto/x509"
	"encoding/json"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// Adds an edge to the graph using a symmetrical encryption key, and confirms that the edge can be found & decrypted
func TestAddAccess_symKey(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestAddAccess_symKey function called")

	// Create the mock stub
	mstub := test_utils.CreateNewMockStub(t)

	// Generate a random sym key
	symKey := test_utils.GenerateSymKey()

	// Generate a random targetKey
	targetKey := test_utils.GenerateSymKey()

	startKeyId := "myStartKeyId"
	targetKeyId := "myTargetKeyId"

	// Start the transaction
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	err := AddAccessWithKeys(stub, symKey, startKeyId, targetKey, targetKeyId, symKey)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")

	// End the transaction
	mstub.MockTransactionEnd("t1")

	// Get the edge
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	keyGraphEdgeBytes, _, _ := graph.GetEdge(stub, global.KEY_GRAPH_PREFIX, startKeyId, targetKeyId)
	edge := keyGraphEdge{}
	json.Unmarshal([]byte(keyGraphEdgeBytes), &edge)

	// Check that the edge was added properly
	test_utils.AssertTrue(t, edge.StartKeyId == startKeyId, "incorrect startKeyId")
	test_utils.AssertTrue(t, edge.TargetKeyId == targetKeyId, "incorrect targetKeyId")
	startKeyNode, _ := getKeyGraphNode(stub, startKeyId)
	targetKeyNode, _ := getKeyGraphNode(stub, targetKeyId)
	test_utils.AssertTrue(t, bytes.Equal(startKeyNode.SymKeyHash, crypto.Hash(symKey)), "incorrect startKeyHash")
	test_utils.AssertTrue(t, bytes.Equal(targetKeyNode.SymKeyHash, crypto.Hash(targetKey)), "incorrect targetKeyHash")
	// Decrypt EncryptedTargetKey and make sure it equals targetKey
	decryptedTargetKey, _ := crypto.DecryptWithSymKey(symKey, edge.EncryptedTargetKey)
	test_utils.AssertTrue(t, bytes.Equal(targetKey, decryptedTargetKey), "EncryptedTargetKey could not be decrypted")
	mstub.MockTransactionEnd("t1")
}

// Adds an edge to the graph using a nil encryption key, and confirms that the edge can be found & decrypted
func TestAddAccess_encKey_nil(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestAddAccess_encKey_nil function called")

	// Create the mock stub
	mstub := test_utils.CreateNewMockStub(t)

	// Generate a random sym key
	symKey := test_utils.GenerateSymKey()

	// Generate a random targetKey
	targetKey := test_utils.GenerateSymKey()

	startKeyId := "myStartKeyId"
	targetKeyId := "myTargetKeyId"

	// Start the transaction
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	err := AddAccessWithKeys(stub, symKey, startKeyId, targetKey, targetKeyId, nil)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")

	// End the transaction
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Get the edge
	keyGraphEdgeBytes, _, _ := graph.GetEdge(stub, global.KEY_GRAPH_PREFIX, startKeyId, targetKeyId)
	edge := keyGraphEdge{}
	json.Unmarshal([]byte(keyGraphEdgeBytes), &edge)

	// Check that the edge was added properly
	test_utils.AssertTrue(t, edge.StartKeyId == startKeyId, "incorrect startKeyId")
	test_utils.AssertTrue(t, edge.TargetKeyId == targetKeyId, "incorrect targetKeyId")
	startKeyNode, _ := getKeyGraphNode(stub, startKeyId)
	targetKeyNode, _ := getKeyGraphNode(stub, targetKeyId)
	test_utils.AssertTrue(t, bytes.Equal(startKeyNode.SymKeyHash, crypto.Hash(symKey)), "incorrect startKeyHash")
	test_utils.AssertTrue(t, bytes.Equal(targetKeyNode.SymKeyHash, crypto.Hash(targetKey)), "incorrect targetKeyHash")
	// Decrypt EncryptedTargetKey and make sure it equals targetKey
	decryptedTargetKey, _ := crypto.DecryptWithSymKey(symKey, edge.EncryptedTargetKey)
	test_utils.AssertTrue(t, bytes.Equal(targetKey, decryptedTargetKey), "EncryptedTargetKey could not be decrypted")
	mstub.MockTransactionEnd("t1")
}

// Adds an edge to the graph using a public encryption key, and confirms that the edge can be found & decrypted
func TestAddAccess_pubKey(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestAddAccess_pubKey function called")

	// Create the mock stub
	mstub := test_utils.CreateNewMockStub(t)

	// Generate a random private key
	privateStartKey := test_utils.GeneratePrivateKey()
	// Get public key for encryption
	publicEncKey := privateStartKey.Public()
	// Marshal privateStartKey into byte slice
	privateStartKeyBytes := x509.MarshalPKCS1PrivateKey(privateStartKey)
	// Marshal publicEncKey into byte slice
	publicEncKeyBytes, _ := x509.MarshalPKIXPublicKey(publicEncKey)

	// Generate a random targetKey
	targetKey := test_utils.GenerateSymKey()

	startKeyId := "myStartKeyId"
	targetKeyId := "myTargetKeyId"

	// Start the transaction
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	err := AddAccessWithKeys(stub, privateStartKeyBytes, startKeyId, targetKey, targetKeyId, publicEncKeyBytes)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")

	// End the transaction
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Get the edge
	//edgeKey, _ := stub.CreateCompositeKey(KEY_GRAPH_PREFIX, []string{startKeyId, targetKeyId})
	//keyGraphEdgeBytes, _ := stub.GetState(edgeKey)
	keyGraphEdgeBytes, _, _ := graph.GetEdge(stub, global.KEY_GRAPH_PREFIX, startKeyId, targetKeyId)

	edge := keyGraphEdge{}
	json.Unmarshal([]byte(keyGraphEdgeBytes), &edge)

	// Check that the edge was added properly
	test_utils.AssertTrue(t, edge.StartKeyId == startKeyId, "incorrect startKeyId")
	test_utils.AssertTrue(t, edge.TargetKeyId == targetKeyId, "incorrect targetKeyId")
	startKeyNode, _ := getKeyGraphNode(stub, startKeyId)
	targetKeyNode, _ := getKeyGraphNode(stub, targetKeyId)
	test_utils.AssertTrue(t, bytes.Equal(startKeyNode.PublicKey, publicEncKeyBytes), "incorrect startKeyHash")
	test_utils.AssertTrue(t, bytes.Equal(targetKeyNode.SymKeyHash, crypto.Hash(targetKey)), "incorrect targetKeyHash")
	// Decrypt EncryptedTargetKey and make sure it equals targetKey
	decryptedTargetKey, _ := crypto.DecryptWithPrivateKey(privateStartKey, edge.EncryptedTargetKey)
	test_utils.AssertTrue(t, bytes.Equal(targetKey, decryptedTargetKey), "EncryptedTargetKey could not be decrypted")
	mstub.MockTransactionEnd("t1")
}

// Tests the errors in AddAccess
func TestAddAccess_error(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)
	badStub := test_utils.CreateMisbehavingMockStub(t)

	// Generate a random private key
	privateStartKey := test_utils.GeneratePrivateKey()
	// Get public key for encryption
	publicEncKey := privateStartKey.Public()
	// Marshal privateStartKey into byte slice
	privateStartKeyBytes := x509.MarshalPKCS1PrivateKey(privateStartKey)
	// Marshal publicEncKey into byte slice
	publicEncKeyBytes, _ := x509.MarshalPKIXPublicKey(publicEncKey)

	// Generate a random targetKey
	targetKey := test_utils.GenerateSymKey()

	startKeyId := "myStartKeyId"
	targetKeyId := "myTargetKeyId"

	// Misbehaving stub
	badStub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(badStub)
	err1 := AddAccessWithKeys(stub, privateStartKeyBytes, startKeyId, targetKey, targetKeyId, publicEncKeyBytes)
	logger.Info("err1 = ", err1)
	test_utils.AssertTrue(t, err1 != nil, "Expected to get an error")
	badStub.MockTransactionEnd("t123")

	// Invalid start key
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err2 := AddAccessWithKeys(stub, nil, startKeyId, targetKey, targetKeyId, publicEncKeyBytes)
	test_utils.AssertFalse(t, err2 != nil, "Expected to get an error")
	mstub.MockTransactionEnd("t1")
}

// Adds an edge to the KeyGraph, then deletes it and confirms that access is lost
func TestRevokeAccess(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestRevokeAccess function called")

	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)

	// Generate keys
	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()

	// Add edge to graph: key1 -> key2
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	mstub.MockTransactionEnd("t1")

	// Verify access to key2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	retrievedKey, _ := SlowVerifyAccessAndGetKey(stub, "key1", key1, "key2")
	test_utils.AssertTrue(t, bytes.Equal(retrievedKey, key2), "Could not access key2")
	mstub.MockTransactionEnd("t1")

	// Revoke access
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err := RevokeAccess(stub, "key1", "key2")
	test_utils.AssertTrue(t, err == nil, "RevokeAccess should not have returned an error")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Verify that access is lost
	retrievedKey2, err := SlowVerifyAccessAndGetKey(stub, "key1", key1, "key2")
	test_utils.AssertTrue(t, retrievedKey2 == nil, "Should not be able to access key2")
	test_utils.AssertTrue(t, err == nil, "VerifyAccessAndGetKey should not return an error")

	// Verify that reverse path is deleted
	ownerKeys, _ := GetOwnerKeys(stub, "key2")
	test_utils.AssertTrue(t, len(ownerKeys) == 0, "key2 should have no owner keys")
	mstub.MockTransactionEnd("t1")
}

// Tests errors in revokeAccess
func TestRevokeAccess_error(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestRevokeAccess_error function called")

	// create a MockStub
	badStub := test_utils.CreateMisbehavingMockStub(t)

	// Generate keys
	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()

	// Add edge to graph: key1 -> key2
	badStub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(badStub)
	AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	badStub.MockTransactionEnd("t123")

	// Revoke access
	badStub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(badStub)
	err := RevokeAccess(stub, "key1", "key2")
	test_utils.AssertTrue(t, err != nil, "RevokeAccess should have returned an error")
	badStub.MockTransactionEnd("t123")
}

// Test function for ValidateKeyId
func TestValidateKeyIdByGraph(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	// create MockStub
	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	// Generate keys
	key1 := test_utils.GenerateSymKey()
	keyId := "key1"

	//Case where mustExist is true
	isValid, err := ValidateKey(stub, data_model.Key{ID: keyId, KeyBytes: key1, Type: global.KEY_TYPE_SYM}, true)
	test_utils.AssertTrue(t, isValid == false && err == nil, "ValidateKey should return false, no error")
}

// Test function for VerifyAccess
// VerifyAccess returns a list of KeyIds or empty [] from start KeyId to target KeyId
func TestVerifyAccess(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestVerifyAccess function called")

	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()
	key3 := test_utils.GenerateSymKey()
	key4 := test_utils.GenerateSymKey()
	key5 := test_utils.GenerateSymKey()
	key6 := test_utils.GenerateSymKey()
	key7 := test_utils.GenerateSymKey()
	key8 := test_utils.GenerateSymKey()
	key9 := test_utils.GenerateSymKey()

	// generate graph of sym keys
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)

	AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	AddAccessWithKeys(stub, key1, "key1", key3, "key3", key1)
	AddAccessWithKeys(stub, key2, "key2", key4, "key4", key2)
	AddAccessWithKeys(stub, key1, "key1", key8, "key8", key1)
	AddAccessWithKeys(stub, key2, "key2", key5, "key5", key2)
	AddAccessWithKeys(stub, key5, "key5", key6, "key6", key5)
	AddAccessWithKeys(stub, key5, "key5", key1, "key1", key5)
	AddAccessWithKeys(stub, key3, "key3", key7, "key7", key3)
	AddAccessWithKeys(stub, key1, "key1", key9, "key9", key1)
	AddAccessWithKeys(stub, key9, "key9", key1, "key1", key9)

	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	expectedKeys1 := []string{"key1", "key2", "key4"}
	listKeys, err1 := SlowVerifyAccess(stub, "key1", "key4")
	if err1 != nil {
		t.Errorf("Could not verfiy access from key1 to key4: %v", err1)
	}
	test_utils.AssertListsEqual(t, expectedKeys1, listKeys)

	expectedKeys2 := []string{"key1", "key8"}
	listKeys2, err2 := SlowVerifyAccess(stub, "key1", "key8")
	if err2 != nil {
		t.Errorf("Could not verfiy access from key1 to key8: %v", err2)
	}
	test_utils.AssertListsEqual(t, expectedKeys2, listKeys2)

	expectedKeys3 := []string{"key1", "key2", "key5", "key6"}
	listKeys3, err3 := SlowVerifyAccess(stub, "key1", "key6")
	if err3 != nil {
		t.Errorf("Could not verfiy access from key1 to key6: %v", err3)
	}
	test_utils.AssertListsEqual(t, expectedKeys3, listKeys3)

	expectedKeys4 := []string{"key2", "key5", "key6"}
	listKeys4, err4 := SlowVerifyAccess(stub, "key2", "key6")
	if err4 != nil {
		t.Errorf("Could not verfiy access from key2 to key6: %v", err4)
	}
	test_utils.AssertListsEqual(t, expectedKeys4, listKeys4)

	expectedKeys5 := []string{"key1", "key9"}
	listKeys5, err5 := SlowVerifyAccess(stub, "key1", "key9")
	if err5 != nil {
		t.Errorf("Could not verfiy access from key1 to key9: %v", err5)
	}
	test_utils.AssertListsEqual(t, expectedKeys5, listKeys5)

	expectedKeys6 := []string{"key9", "key1"}
	listKeys6, err6 := SlowVerifyAccess(stub, "key9", "key1")
	if err6 != nil {
		t.Errorf("Could not verfiy access from key9 to key1: %v", err6)
	}
	test_utils.AssertListsEqual(t, expectedKeys6, listKeys6)

	// Tested for non happy path
	expectedKeys7 := []string{}
	listKeys7, err7 := SlowVerifyAccess(stub, "key6", "key7")
	if err7 != nil {
		t.Errorf("Could not verfiy access from key6 to key7: %v", err7)
	}
	test_utils.AssertListsEqual(t, expectedKeys7, listKeys7)

	expectedKeys8 := []string{"key1"}
	listKeys8, err8 := SlowVerifyAccess(stub, "key1", "key1")
	if err8 != nil {
		t.Errorf("Could not verfiy access from key1 to key1: %v", err8)
	}
	test_utils.AssertListsEqual(t, expectedKeys8, listKeys8)
	mstub.MockTransactionEnd("t1")
}

// Test function for VerifyAccessPath
// VerifyAccess returns a list of KeyIds or empty [] from start KeyId to target KeyId
func TestVerifyAccessPath(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestVerifyAccessPath function called")

	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()
	key3 := test_utils.GenerateSymKey()
	key4 := test_utils.GenerateSymKey()
	key5 := test_utils.GenerateSymKey()
	key6 := test_utils.GenerateSymKey()
	key7 := test_utils.GenerateSymKey()
	key8 := test_utils.GenerateSymKey()
	key9 := test_utils.GenerateSymKey()

	// generate graph of sym keys
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)

	AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	AddAccessWithKeys(stub, key1, "key1", key3, "key3", key1)
	AddAccessWithKeys(stub, key2, "key2", key4, "key4", key2)
	AddAccessWithKeys(stub, key1, "key1", key8, "key8", key1)
	AddAccessWithKeys(stub, key2, "key2", key5, "key5", key2)
	AddAccessWithKeys(stub, key5, "key5", key6, "key6", key5)
	AddAccessWithKeys(stub, key5, "key5", key1, "key1", key5)
	AddAccessWithKeys(stub, key3, "key3", key7, "key7", key3)
	AddAccessWithKeys(stub, key1, "key1", key9, "key9", key1)
	AddAccessWithKeys(stub, key9, "key9", key1, "key1", key9)

	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	expectedKeys1 := []string{"key1", "key2", "key4"}
	verified1, err1 := VerifyAccessPath(stub, expectedKeys1)
	if err1 != nil {
		t.Errorf("Could not verfiy access from key1 to key4: %v", err1)
	}
	test_utils.AssertTrue(t, verified1, "should be true")

	expectedKeys2 := []string{"key1", "key8"}
	verified2, err2 := VerifyAccessPath(stub, expectedKeys2)
	if err2 != nil {
		t.Errorf("Could not verfiy access from key1 to key8: %v", err2)
	}
	test_utils.AssertTrue(t, verified2, "should be true")

	expectedKeys3 := []string{"key1", "key2", "key5", "key6"}
	verified3, err3 := VerifyAccessPath(stub, expectedKeys3)
	if err3 != nil {
		t.Errorf("Could not verfiy access from key1 to key6: %v", err3)
	}
	test_utils.AssertTrue(t, verified3, "should be true")

	expectedKeys4 := []string{"key2", "key5", "key6"}
	verified4, err4 := VerifyAccessPath(stub, expectedKeys4)
	if err4 != nil {
		t.Errorf("Could not verfiy access from key2 to key6: %v", err4)
	}
	test_utils.AssertTrue(t, verified4, "should be true")

	expectedKeys5 := []string{"key1", "key9"}
	verified5, err5 := VerifyAccessPath(stub, expectedKeys5)
	if err5 != nil {
		t.Errorf("Could not verfiy access from key1 to key9: %v", err5)
	}
	test_utils.AssertTrue(t, verified5, "should be true")

	expectedKeys6 := []string{"key9", "key1"}
	verified6, err6 := VerifyAccessPath(stub, expectedKeys6)
	if err6 != nil {
		t.Errorf("Could not verfiy access from key9 to key1: %v", err6)
	}
	test_utils.AssertTrue(t, verified6, "should be true")

	expectedKeys8 := []string{"key1", "key1"}
	verified8, err8 := VerifyAccessPath(stub, expectedKeys8)
	if err8 != nil {
		t.Errorf("Could not verfiy access from key1 to key1: %v", err8)
	}
	test_utils.AssertTrue(t, verified8 == true, "should be true")
	mstub.MockTransactionEnd("t1")

	// Tested for non happy path
	expectedKeys7 := []string{"key6", "key7"}
	verified7, err7 := VerifyAccessPath(stub, expectedKeys7)
	if err7 != nil {
		t.Errorf("Could not verfiy access from key6 to key7: %v", err7)
	}
	test_utils.AssertTrue(t, verified7 == false, "should be false")
}

// Test function for errors in VerifyAccess
func TestVerifyAccess_error(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestVerifyAccess function called")

	// create a MockStub
	badStub := test_utils.CreateMisbehavingMockStub(t)
	stub := cached_stub.NewCachedStub(badStub)

	_, err1 := SlowVerifyAccess(stub, "key1", "key2")
	test_utils.AssertTrue(t, err1 != nil, "Expected error in VerifyAccess")
}

// Test function for GetKey
// keys are encrypted using a combination of sym and pub keys during addAccess
// Returns the decrypted target key of the last item on a list of key Ids
func TestGetKey(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetKey function called")

	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)

	// Generate a random private key
	privateStartKey := test_utils.GeneratePrivateKey()
	// Marshal privateStartKey into byte slice
	key1_priv := x509.MarshalPKCS1PrivateKey(privateStartKey)
	// Get public key for encryption
	publicEncKey := privateStartKey.Public()
	// Marshal publicEncKey into byte slice
	key1_pub, _ := x509.MarshalPKIXPublicKey(publicEncKey)

	key2 := test_utils.GenerateSymKey()
	key3 := test_utils.GenerateSymKey()
	key4 := test_utils.GenerateSymKey()
	key5 := test_utils.GenerateSymKey()
	key6 := test_utils.GenerateSymKey()
	key7 := test_utils.GenerateSymKey()
	key8 := test_utils.GenerateSymKey()

	// generate graph of sym keys
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)

	AddAccessWithKeys(stub, key1_priv, "key1", key2, "key2", key1_pub)
	AddAccessWithKeys(stub, key1_priv, "key1", key3, "key3", key1_pub)
	AddAccessWithKeys(stub, key2, "key2", key4, "key4", key2)
	AddAccessWithKeys(stub, key1_priv, "key1", key8, "key8", key1_pub)
	AddAccessWithKeys(stub, key2, "key2", key5, "key5", key2)
	AddAccessWithKeys(stub, key5, "key5", key6, "key6", key5)
	AddAccessWithKeys(stub, key3, "key3", key7, "key7", key3)

	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	expectedKey1 := key6
	listKeys := []string{"key1", "key2", "key5", "key6"}
	result1, err1 := GetKey(stub, listKeys, key1_priv)
	if err1 != nil {
		t.Errorf("Could not get key6: %+v", err1)
	}
	test_utils.AssertTrue(t, bytes.Equal(expectedKey1, result1), "Expected to get key6")

	expectedKey2 := key5
	listKeys = []string{"key2", "key5"}
	result2, err2 := GetKey(stub, listKeys, key2)
	if err2 != nil {
		t.Errorf("Could not get key6: %v", err2)
	}
	test_utils.AssertTrue(t, bytes.Equal(expectedKey2, result2), "Expected to get key5")

	// Tested for non happy path
	listKeys = []string{"key5", "key6"}
	_, err3 := GetKey(stub, listKeys, key2)
	test_utils.AssertTrue(t, err3 != nil, "Expected to fail")
	mstub.MockTransactionEnd("t1")
}

func TestGetKeyFromCache(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetKeyFromCache function called")

	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)

	// Generate a random private key
	privateStartKey := test_utils.GeneratePrivateKey()
	// Marshal privateStartKey into byte slice
	key1_priv := x509.MarshalPKCS1PrivateKey(privateStartKey)
	// Get public key for encryption
	publicEncKey := privateStartKey.Public()
	// Marshal publicEncKey into byte slice
	key1_pub, _ := x509.MarshalPKIXPublicKey(publicEncKey)

	key2 := test_utils.GenerateSymKey()
	key3 := test_utils.GenerateSymKey()
	key4 := test_utils.GenerateSymKey()
	key5 := test_utils.GenerateSymKey()
	key6 := test_utils.GenerateSymKey()
	key7 := test_utils.GenerateSymKey()
	key8 := test_utils.GenerateSymKey()

	// generate graph of sym keys
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)

	AddAccessWithKeys(stub, key1_priv, "key1", key2, "key2", key1_pub)
	AddAccessWithKeys(stub, key2, "key2", key3, "key3", key2)
	AddAccessWithKeys(stub, key3, "key3", key4, "key4", key3)
	AddAccessWithKeys(stub, key4, "key4", key5, "key5", key4)
	AddAccessWithKeys(stub, key5, "key5", key6, "key6", key5)
	AddAccessWithKeys(stub, key6, "key6", key7, "key7", key6)
	AddAccessWithKeys(stub, key7, "key7", key8, "key8", key7)

	mstub.MockTransactionEnd("t1")

	// first time getting keys
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	expectedKey1 := key8
	listKeys := []string{"key1", "key2", "key3", "key4", "key5", "key6", "key7", "key8"}

	t1 := time.Now().UnixNano()
	result1, err1 := GetKey(stub, listKeys, key1_priv)
	t2 := time.Now().UnixNano()
	test_utils.AssertTrue(t, err1 == nil, "Expected no error to get key8")
	logger.Debugf("Time to get key: %v", t2-t1)
	test_utils.AssertTrue(t, bytes.Equal(expectedKey1, result1), "Expected to get key6")

	// second time getting keys
	// should get it from cache
	t3 := time.Now().UnixNano()
	result1, err1 = GetKey(stub, listKeys, key1_priv)
	t4 := time.Now().UnixNano()
	test_utils.AssertTrue(t, err1 == nil, "Expected no error to get key8")
	logger.Debugf("Time to get key with cache: %v", t4-t3)
	test_utils.AssertTrue(t, bytes.Equal(expectedKey1, result1), "Expected to get key6")

	// getting from cache should be much faster
	test_utils.AssertTrue(t, (t4-t3) < (t2-t1), "Cache should be faster")
	mstub.MockTransactionEnd("t1")
}

// Tests function for GetMyKeys
func TestGetMyKeys(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetMyKeys function called")

	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()
	key3 := test_utils.GenerateSymKey()
	key4 := test_utils.GenerateSymKey()

	// generate graph of sym keys
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	AddAccessWithKeys(stub, key1, "key1", key3, "key3", key1)
	AddAccessWithKeys(stub, key2, "key2", key4, "key4", key2)
	AddAccessWithKeys(stub, key4, "key4", key1, "key1", key4) // cycle
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// find child nodes of key1
	expectedKeys_key1 := []string{"key2", "key3", "key4"}
	var listKeys, err = SlowGetMyKeys(stub, "key1")
	if err != nil {
		t.Errorf("Could not retrieve keys: %v", err)
	}
	test_utils.AssertSetsEqual(t, expectedKeys_key1, listKeys)

	// find child nodes of key2
	expectedKeys_key2 := []string{"key4", "key1", "key3"}
	listKeys, err = SlowGetMyKeys(stub, "key2")
	if err != nil {
		t.Errorf("Could not retrieve keys: %v", err)
	}
	test_utils.AssertSetsEqual(t, expectedKeys_key2, listKeys)

	// find child nodes of key3
	expectedKeys_key3 := []string{}
	listKeys, err = SlowGetMyKeys(stub, "key3")
	if err != nil {
		t.Errorf("Could not retrieve keys: %v", err)
	}
	test_utils.AssertSetsEqual(t, expectedKeys_key3, listKeys)

	// find child nodes of key4
	expectedKeys_key4 := []string{"key1", "key2", "key3"}
	listKeys, err = SlowGetMyKeys(stub, "key4")
	if err != nil {
		t.Errorf("Could not retrieve keys: %v", err)
	}
	test_utils.AssertSetsEqual(t, expectedKeys_key4, listKeys)
	mstub.MockTransactionEnd("t1")
}

//Tests errors in GetMyKeys
func TestGetMyKeys_error(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetMyKeys_error function called")

	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)
	badStub := test_utils.CreateMisbehavingMockStub(t)

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()
	key3 := test_utils.GenerateSymKey()
	key4 := test_utils.GenerateSymKey()

	// generate graph of sym keys
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	AddAccessWithKeys(stub, key1, "key1", key3, "key3", key1)
	AddAccessWithKeys(stub, key2, "key2", key4, "key4", key2)
	AddAccessWithKeys(stub, key4, "key4", key1, "key1", key4) // cycle
	mstub.MockTransactionEnd("t1")

	badStub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(badStub)
	_, err := SlowGetMyKeys(stub, "key1")
	test_utils.AssertTrue(t, err != nil, "GetMyKeys returned error")
	badStub.MockTransactionEnd("t1")
}

// Tests function for GetUserKeys
func TestGetUserKeys(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetUserKeys function called")

	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)

	//Create user object?
	user := test_utils.CreateTestUser("user1")
	userPrivKey := x509.MarshalPKCS1PrivateKey(user.PrivateKey)
	userPubKey, _ := x509.MarshalPKIXPublicKey(user.PublicKey)
	userPubPrivKeyId := user.GetPubPrivKeyId()
	logger.Info("userPrivKeyId = " + userPubPrivKeyId)
	//create some other keys
	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()
	key3 := test_utils.GenerateSymKey()

	//connect user's private key to those other keys
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	AddAccessWithKeys(stub, userPrivKey, userPubPrivKeyId, key1, "key1", userPubKey)
	AddAccessWithKeys(stub, userPrivKey, userPubPrivKeyId, key2, "key2", userPubKey)
	AddAccessWithKeys(stub, key1, "key1", key3, "key3", key1)
	AddAccessWithKeys(stub, key3, "key3", userPrivKey, userPubPrivKeyId, key3) // cycle
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	//Find user keys
	expectedKeys := []string{"key1", "key2", "key3"}
	var listKeys, err = GetUserKeys(stub, user)
	if err != nil {
		t.Errorf("Could not retrieve keys: %v", err)
	}
	test_utils.AssertSetsEqual(t, expectedKeys, listKeys)
	mstub.MockTransactionEnd("t1")
}

// Adds several keys to the graph, then confirms that key4 can be accessed with key1
func TestVerifyAccessAndGetKey(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestVerifyAccessAndGetKey function called")

	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()
	key3 := test_utils.GenerateSymKey()
	key4 := test_utils.GenerateSymKey()

	// generate graph of sym keys
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1) // Encrypt key2 w/ key1
	AddAccessWithKeys(stub, key2, "key2", key3, "key3", key2) // Encrypt key3 w/ key2
	AddAccessWithKeys(stub, key3, "key3", key4, "key4", key3) // Encrypt key4 w/ key3
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Call the function we're testing
	retrievedKey, _ := SlowVerifyAccessAndGetKey(stub, "key1", key1, "key4")

	// Check the results
	test_utils.AssertTrue(t, bytes.Equal(retrievedKey, key4), "Retrieved key4 was not equal to actual key4")
	mstub.MockTransactionEnd("t1")
}

// Adds several keys to the graph, then confirms that key4 can be accessed with key1
func TestVerifyAccessAndGetKey_error(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestVerifyAccessAndGetKey_error function called")

	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()
	key3 := test_utils.GenerateSymKey()
	key4 := test_utils.GenerateSymKey()

	// generate graph of sym keys
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1) // Encrypt key2 w/ key1
	AddAccessWithKeys(stub, key2, "key2", key3, "key3", key2) // Encrypt key3 w/ key2
	AddAccessWithKeys(stub, key3, "key3", key4, "key4", key3) // Encrypt key4 w/ key3
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Call the function we're testing
	_, err := SlowVerifyAccessAndGetKey(stub, "key1", nil, "key4")

	// Check the results
	test_utils.AssertTrue(t, err != nil, "Received error verifying and getting key")
	mstub.MockTransactionEnd("t1")
}

// Adds several keys to the graph, then confirms that key4 can be accessed with key1
func TestVerifyAccessPathAndGetKey(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestVerifyAccessAndGetKey function called")

	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()
	key3 := test_utils.GenerateSymKey()
	key4 := test_utils.GenerateSymKey()

	// generate graph of sym keys
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1) // Encrypt key2 w/ key1
	AddAccessWithKeys(stub, key2, "key2", key3, "key3", key2) // Encrypt key3 w/ key2
	AddAccessWithKeys(stub, key3, "key3", key4, "key4", key3) // Encrypt key4 w/ key3
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Call the function we're testing
	retrievedKey, _ := VerifyAccessPathAndGetKey(stub, "key1", key1, []string{"key1", "key2", "key3", "key4"})

	// Check the results
	test_utils.AssertTrue(t, bytes.Equal(retrievedKey, key4), "Retrieved key4 was not equal to actual key4")
	mstub.MockTransactionEnd("t1")
}

// Adds several keys to the graph, then confirms that key4 can be accessed with key1
func TestVerifyAccessPathAndGetKey_error(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestVerifyAccessAndGetKey_error function called")

	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()
	key3 := test_utils.GenerateSymKey()
	key4 := test_utils.GenerateSymKey()

	// generate graph of sym keys
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1) // Encrypt key2 w/ key1
	AddAccessWithKeys(stub, key2, "key2", key3, "key3", key2) // Encrypt key3 w/ key2
	AddAccessWithKeys(stub, key3, "key3", key4, "key4", key3) // Encrypt key4 w/ key3
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Call the function we're testing
	_, err := VerifyAccessPathAndGetKey(stub, "key1", nil, []string{"key1", "key2", "key3", "key4"})

	// Check the results
	test_utils.AssertTrue(t, err != nil, "Received error verifying and getting key")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Call the function we're testing
	_, err = VerifyAccessPathAndGetKey(stub, "key1", key2, []string{"key1", "key2", "key3", "key4"})

	// Check the results
	test_utils.AssertTrue(t, err != nil, "Received error verifying and getting key")

	mstub.MockTransactionEnd("t1")
}

//Tests function GetOwnerKeys
func TestGetOwnerKeys(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetOwnerKeys function called")

	// create a MockStub
	mstub := test_utils.CreateNewMockStub(t)

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()
	key3 := test_utils.GenerateSymKey()
	key4 := test_utils.GenerateSymKey()

	// generate graph of sym keys
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	AddAccessWithKeys(stub, key1, "key1", key3, "key3", key1)
	AddAccessWithKeys(stub, key2, "key2", key4, "key4", key2)
	AddAccessWithKeys(stub, key4, "key4", key1, "key1", key4) // cycle
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// find parent nodes of key1
	expectedKeys_key1 := []string{"key4", "key2"}
	var listKeys, err = GetOwnerKeys(stub, "key1")
	if err != nil {
		t.Errorf("Could not retrieve keys: %v", err)
	}
	test_utils.AssertListsEqual(t, expectedKeys_key1, listKeys)

	// find parent nodes of key2
	expectedKeys_key2 := []string{"key1", "key4"}
	listKeys, err = GetOwnerKeys(stub, "key2")
	if err != nil {
		t.Errorf("Could not retrieve keys: %v", err)
	}
	test_utils.AssertListsEqual(t, expectedKeys_key2, listKeys)

	// find parent nodes of key3
	expectedKeys_key3 := []string{"key1", "key4", "key2"}
	listKeys, err = GetOwnerKeys(stub, "key3")
	if err != nil {
		t.Errorf("Could not retrieve keys: %v", err)
	}
	test_utils.AssertListsEqual(t, expectedKeys_key3, listKeys)

	// find parent nodes of key4
	expectedKeys_key4 := []string{"key2", "key1"}
	listKeys, err = GetOwnerKeys(stub, "key4")
	if err != nil {
		t.Errorf("Could not retrieve keys: %v", err)
	}
	test_utils.AssertListsEqual(t, expectedKeys_key4, listKeys)
	mstub.MockTransactionEnd("t1")
}

//Tests errors in GetOwnerKeys
func TestGetOwnerKeys_error(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetOwnerKeys_error function called")

	// create a MockStub
	mbadStub := test_utils.CreateMisbehavingMockStub(t)
	badStub := cached_stub.NewCachedStub(mbadStub)

	_, err := GetOwnerKeys(badStub, "key1")
	test_utils.AssertTrue(t, err != nil, "Expected error in GetOwnerKeys")
}
