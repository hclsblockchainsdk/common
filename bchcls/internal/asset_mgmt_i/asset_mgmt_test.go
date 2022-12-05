/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package asset_mgmt_i

import (
	"common/bchcls/asset_mgmt/asset_key_func"
	"common/bchcls/asset_mgmt/asset_manager"
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/datastore"
	"common/bchcls/index"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/datastore_i/datastore_c"
	"common/bchcls/internal/datatype_i"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/simple_rule"
	"common/bchcls/test_utils"
	"common/bchcls/utils"

	"bytes"
	"crypto/rsa"
	"encoding/json"
	"net/url"
	"os"
	"runtime/debug"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func setup(t *testing.T) *test_utils.NewMockStub {
	logger.SetLevel(shim.LogDebug)
	mstub := test_utils.CreateNewMockStub(t)
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	datatype_i.Init(stub, shim.LogDebug)
	datastore_c.Init(stub, shim.LogDebug)
	Init(stub)
	mstub.MockTransactionEnd("t123")
	return mstub
}

func TestPutAssetByKey(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutAssetByKey function called")

	// create a MockStub
	mstub := setup(t)

	owner := test_utils.CreateTestUser("ownerId")
	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()

	// 1. Add New Asset

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	// add key1 to KeyGraph
	err := key_mgmt_i.AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")

	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// add testAsset to ledger
	testAsset := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	testAsset.OwnerIds = []string{"ownerId"}

	err = putAssetByKey(stub, owner, testAsset, "key1", key1, "", nil, nil)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")

	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// check ledger for asset and assetId
	checkAsset(t, stub, testAsset.AssetId, testAsset.PublicData, testAsset.PrivateData, "key1", key1, testAsset.OwnerIds)
	checkAssetId(t, stub, testAsset.AssetId, "key1")
	mstub.MockTransactionEnd("t123")

	// 2. Replace Existing Asset With Correct AssetKey

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// add testAsset to ledger
	testAsset.PublicData = test_utils.CreateTestAssetData("public2")
	err = putAssetByKey(stub, owner, testAsset, "key1", key1, "", nil, nil)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	//check ledger for asset and assetId
	checkAsset(t, stub, testAsset.AssetId, testAsset.PublicData, testAsset.PrivateData, "key1", key1, testAsset.OwnerIds)
	checkAssetId(t, stub, testAsset.AssetId, "key1")
	mstub.MockTransactionEnd("t123")

	// 3. Replace Existing Asset With Incorrect AssetKey
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// the correct key is key1
	err = putAssetByKey(stub, owner, testAsset, "key2", key2, "", nil, nil)
	test_utils.AssertTrue(t, err != nil, "Expected PutAsset to fail")
	mstub.MockTransactionEnd("t123")
}

// assetKey is not in initially in KeyGraph
// add assetKey to KeyGraph with additional symkey
func TestPutAssetByKey_AddKey(t *testing.T) {
	logger.Info("TestPutAssetByKey_AddKey function called")

	// create a MockStub
	mstub := setup(t)

	// generate key without adding it to KeyGraph
	key3 := test_utils.GenerateSymKey()

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)

	// add testAsset to ledger
	testAsset := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	testAsset.OwnerIds = []string{"ownerId"}

	owner := test_utils.CreateTestUser("ownerId")
	myKey := owner.SymKey

	// No assetKey or assetKeyID - this should return an error
	err := putAssetByKey(stub, owner, testAsset, "", nil, "myKeyId", myKey, myKey)
	test_utils.AssertTrue(t, err != nil, "Expected PutAsset to fail")

	// No assetKey - this should return an error because assetKey can't be found in the graph
	err = putAssetByKey(stub, owner, testAsset, "key3", nil, "myKeyId", myKey, myKey)
	test_utils.AssertTrue(t, err != nil, "Expected PutAsset to fail")

	// Valid assetKey & myKey are provided - this should succeed
	err = putAssetByKey(stub, owner, testAsset, "key3", key3, "myKeyId", myKey, myKey)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")

	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	// check ledger for asset and assetId
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, testAsset.AssetId, testAsset.PublicData, testAsset.PrivateData, "key3", key3, testAsset.OwnerIds)
	checkAssetId(t, stub, testAsset.AssetId, "key3")
	mstub.MockTransactionEnd("t123")
}

// assetKey is not in initially in KeyGraph
// add assetKey to KeyGraph with additional symkey
// try to change asset without write access
func TestUpdateAsset_WriteAccess(t *testing.T) {
	logger.Info("TestPutAssetByKey_WriteAccess function called")

	// create a MockStub
	mstub := setup(t)

	// caller's keys
	owner := test_utils.CreateTestUser("ownerId")
	privateKey1 := owner.PrivateKey
	priv1 := crypto.PrivateKeyToBytes(privateKey1)
	ownerKey := data_model.Key{}
	ownerKey.ID = owner.GetPubPrivKeyId()
	ownerKey.KeyBytes = priv1
	ownerKey.Type = global.KEY_TYPE_PRIVATE

	//pub1 := crypto.PublicKeyToBytes(privateKey1.Public().(*rsa.PublicKey))

	// not-owner's keys
	notOwner := test_utils.CreateTestUser("not-ownerId")
	privateKey2 := notOwner.PrivateKey
	priv2 := crypto.PrivateKeyToBytes(privateKey2)
	//pub2 := crypto.PublicKeyToBytes(privateKey2.Public().(*rsa.PublicKey))
	notownerKey := data_model.Key{}
	notownerKey.ID = notOwner.GetPubPrivKeyId()
	notownerKey.KeyBytes = priv2
	notownerKey.Type = global.KEY_TYPE_PRIVATE

	// generate key without adding it to KeyGraph
	key1 := test_utils.GenerateSymKey()
	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.KeyBytes = key1
	assetKey.Type = global.KEY_TYPE_SYM

	testAsset := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	testAsset.OwnerIds = []string{"ownerId"}
	testAsset.AssetKeyId = "key1"
	testAsset.AssetKeyHash = crypto.Hash(key1)

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	err := GetAssetManager(stub, owner).AddAsset(testAsset, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected UpdateAsset to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	// check ledger for asset and assetId
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, testAsset.AssetId, testAsset.PublicData, testAsset.PrivateData, "key1", key1, testAsset.OwnerIds)
	checkAssetId(t, stub, testAsset.AssetId, "key1")

	// now try to overwrite without write access
	testAsset.PublicData = test_utils.CreateTestAssetData("publicNew")
	testAsset.PrivateData = test_utils.CreateTestAssetData("privateNew")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)

	err = GetAssetManager(stub, notOwner).UpdateAsset(testAsset, assetKey)
	test_utils.AssertTrue(t, err != nil, "Expected UpdateAsset to fail")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	// check ledger for asset and assetId shouldn't have changed
	stub = cached_stub.NewCachedStub(mstub)
	checkAssetId(t, stub, testAsset.AssetId, "key1")

	// now give write access to not-ownerId
	am := GetAssetManager(stub, owner)
	accessControl := data_model.AccessControl{}
	accessControl.Access = global.ACCESS_WRITE
	accessControl.UserId = "not-ownwerId"
	accessControl.AssetId = testAsset.AssetId
	accessControl.AssetKey = &assetKey
	accessControl.UserKey = &notownerKey
	err = am.AddAccessToAsset(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccessToAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// now try to overwrite with write access
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// add testAsset to ledger
	err = GetAssetManager(stub, notOwner).UpdateAsset(testAsset, assetKey)
	test_utils.AssertTrue(t, err == nil, "Expected UpdateAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// now try to overwrite with write access
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// add testAsset to ledger
	err = GetAssetManager(stub, notOwner).UpdateAsset(testAsset, assetKey)
	test_utils.AssertTrue(t, err == nil, "Expected UpdateAsset to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	// check ledger for asset and assetId shouldn't have changed
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, testAsset.AssetId, testAsset.PublicData, testAsset.PrivateData, "key1", key1, testAsset.OwnerIds)
	checkAssetId(t, stub, testAsset.AssetId, "key1")
	mstub.MockTransactionEnd("t123")
}

// assetKey is not initially in KeyGraph
// PutAsset fails because additional symkey is not provided
func TestPutAssetByKey_WithoutKey(t *testing.T) {
	logger.Info("TestPutAssetByKey_WithoutKey function called")

	// create a MockStub
	mstub := setup(t)
	owner := test_utils.CreateTestUser("ownerId")

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)

	// add testAsset to ledger
	testAsset := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	testAsset.OwnerIds = []string{"ownerId"}

	err := putAssetByKey(stub, owner, testAsset, "key1", nil, "", nil, nil)
	test_utils.AssertTrue(t, err != nil, "Expected PutAsset to fail")

	mstub.MockTransactionEnd("t123")
}

func TestPutAssetByKey_AssetKeyNotRequired(t *testing.T) {
	logger.Info("TestPutAssetByKey_AssetKeyNotRequired function called")

	// create a MockStub
	mstub := setup(t)

	owner := test_utils.CreateTestUser("ownerId")
	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()
	//key3 := test_utils.GenerateSymKey()

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	// add key1 to KeyGraph
	var err = key_mgmt_i.AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// add testAsset to ledger
	testAsset := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	testAsset.OwnerIds = []string{"ownerId"}

	err = putAssetByKey(stub, owner, testAsset, "key1", key1, "", nil, nil)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// get asset with no asset key - returns encrypted private data
	asset1, _ := getAssetByKey(stub, testAsset.AssetId, nil)
	asset1.PublicData = test_utils.CreateTestAssetData("public2")

	// put existing asset without asset key
	err = putAssetByKey(stub, owner, *asset1, "key1", key1, "", nil, nil)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset without assetKey to succeed")
	mstub.MockTransactionEnd("t123")
}

func TestPutAssetByKey_PrivateData(t *testing.T) {
	logger.Info("TestPutAssetByKey_PrivateData function called")

	// create a MockStub
	mstub := setup(t)

	owner := test_utils.CreateTestUser("ownerId")
	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()
	key3 := test_utils.GenerateSymKey()

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)

	// add key1 to KeyGraph
	err := key_mgmt_i.AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")

	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// add testAsset to ledger (without private data)
	testAsset := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	testAsset.PublicData = test_utils.CreateTestAssetData("public1")
	testAsset.PrivateData = nil
	testAsset.OwnerIds = []string{"ownerId"}

	// create asset without asset key
	err = putAssetByKey(stub, owner, testAsset, "", nil, "", nil, nil)
	test_utils.AssertTrue(t, err != nil, "Expected PutAsset to fail")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// create asset with asset key + without private data
	err = putAssetByKey(stub, owner, testAsset, "key1", key1, "", nil, nil)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	asset1, _ := getAssetByKey(stub, testAsset.AssetId, key1)
	asset1.PublicData = test_utils.CreateTestAssetData("public2")
	// update existing asset without asset key + without private data
	err = putAssetByKey(stub, owner, *asset1, "key1", nil, "", nil, nil)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset without assetKeyByte to succeed")
	mstub.MockTransactionEnd("t123")

	asset1.PrivateData = test_utils.CreateTestAssetData("private1")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// put existing asset without asset key + with private data
	err = putAssetByKey(stub, owner, *asset1, "key1", nil, "", nil, nil)
	test_utils.AssertTrue(t, err != nil, "Expected PutAsset without assetKey to fail")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// put existing asset with wrong asset key + with private data
	err = putAssetByKey(stub, owner, *asset1, "key3", key3, "", nil, nil)
	test_utils.AssertTrue(t, err != nil, "Expected PutAsset with wrong assetKey to fail")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// put existing asset with correct asset key + with private data
	err = putAssetByKey(stub, owner, *asset1, "key1", key1, "", nil, nil)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset with correct assetKey to succeed")
	mstub.MockTransactionEnd("t123")
}

func TestAddAsset(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestAddAsset function called")

	// create a MockStub
	mstub := setup(t)

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)

	// add asset - provide asset key id and asset bytes
	caller := test_utils.CreateTestUser("owner1")
	am := GetAssetManager(stub, caller)

	key1 := test_utils.GenerateSymKey()
	testPublicData := test_utils.CreateTestAssetData("public1")
	testPrivateData := test_utils.CreateTestAssetData("private1")
	assetData := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	assetData.AssetKeyId = "key1"
	assetData.AssetKeyHash = crypto.Hash(key1)
	assetData.PrivateData = testPrivateData
	assetData.PublicData = testPublicData
	assetData.OwnerIds = []string{"owner1"}
	assetKey := data_model.Key{}
	// no assetKey.ID
	assetKey.Type = global.KEY_TYPE_SYM
	assetKey.KeyBytes = key1

	// add asset with empty assetKey.ID
	err := am.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err != nil, "Expected AddAsset to fail")
	mstub.MockTransactionEnd("t123")

	// add asset with valid assetKey
	assetKey.ID = "key1"
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	err = am.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, assetKey.ID, assetKey.KeyBytes, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, "key1")
	mstub.MockTransactionEnd("t123")
}

func TestAddAsset_NoKeyBytes(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestAddAsset_NoKeyBytes function called")

	// create a MockStub
	mstub := setup(t)

	caller := data_model.User{}
	caller.PrivateKey = test_utils.GeneratePrivateKey()
	caller.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller.PrivateKey))
	pub := crypto.PublicKeyToBytes(caller.PrivateKey.Public().(*rsa.PublicKey))
	caller.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller.ID = "owner1"

	key1 := test_utils.GenerateSymKey()
	testPublicData := test_utils.CreateTestAssetData("public1")
	testPrivateData := test_utils.CreateTestAssetData("private1")
	assetData := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	assetData.AssetKeyId = "key1"
	assetData.AssetKeyHash = crypto.Hash(key1)
	assetData.PrivateData = testPrivateData
	assetData.PublicData = testPublicData
	assetData.OwnerIds = []string{"owner1"}
	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = global.KEY_TYPE_SYM
	// not providing key bytes (assetKey.Key)

	// add asset - key does not exist, provide asset key id, do not provide key bytes
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	am := GetAssetManager(stub, caller)
	err := am.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err != nil, "Expected AddAsset to fail")
	mstub.MockTransactionEnd("t123")

}

func TestUpdateAsset(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestUpdateAsset function called")

	// create a MockStub
	mstub := setup(t)

	caller1 := data_model.User{}
	caller1.PrivateKey = test_utils.GeneratePrivateKey()
	caller1.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller1.PrivateKey))
	pub := crypto.PublicKeyToBytes(caller1.PrivateKey.Public().(*rsa.PublicKey))
	caller1.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller1.ID = "owner1"

	caller2 := data_model.User{}
	caller2.PrivateKey = test_utils.GeneratePrivateKey()
	caller2.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller2.PrivateKey))
	pub = crypto.PublicKeyToBytes(caller2.PrivateKey.Public().(*rsa.PublicKey))
	caller2.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller2.ID = "owner2"

	key1 := test_utils.GenerateSymKey()
	testPublicData := test_utils.CreateTestAssetData("public1")
	testPrivateData := test_utils.CreateTestAssetData("private1")
	assetData := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	assetData.AssetKeyId = "key1"
	assetData.AssetKeyHash = crypto.Hash(key1)
	assetData.PrivateData = testPrivateData
	assetData.PublicData = testPublicData
	assetData.OwnerIds = []string{"owner1"}

	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = global.KEY_TYPE_SYM
	assetKey.KeyBytes = key1

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	am1 := GetAssetManager(stub, caller1)
	err := am1.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// check ledger for asset and assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, assetKey.ID, assetKey.KeyBytes, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, assetKey.ID)
	mstub.MockTransactionEnd("t123")

	// attempt to replace existing asset without access (caller2)
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	testPublicData = test_utils.CreateTestAssetData("public2")
	assetData.PublicData = testPublicData
	am2 := GetAssetManager(stub, caller2)
	err = am2.UpdateAsset(assetData, assetKey)
	test_utils.AssertTrue(t, err != nil, "Expected UpdateAsset to fail")
	mstub.MockTransactionEnd("t123")

	// replace existing asset with access (caller1)
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 = GetAssetManager(stub, caller1)
	err = am1.UpdateAsset(assetData, assetKey)
	test_utils.AssertTrue(t, err == nil, "Expected UpdateAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// check ledger for asset and assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, assetKey.ID, assetKey.KeyBytes, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, assetKey.ID)
	mstub.MockTransactionEnd("t123")
}

func TestUpdateAsset_Index(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestUpdateAsset_Index function called")

	// create a MockStub
	mstub := setup(t)

	caller1 := data_model.User{}
	caller1.PrivateKey = test_utils.GeneratePrivateKey()
	caller1.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller1.PrivateKey))
	pub := crypto.PublicKeyToBytes(caller1.PrivateKey.Public().(*rsa.PublicKey))
	caller1.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller1.ID = "owner1"

	key1 := test_utils.GenerateSymKey()
	assetData := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	assetData.AssetKeyId = "key1"
	assetData.AssetKeyHash = crypto.Hash(key1)
	testPublicDataMap := make(map[string]interface{})
	testPublicDataMap["assetId"] = assetData.AssetId
	testPublicData, _ := json.Marshal(testPublicDataMap)
	testPrivateDataMap := make(map[string]interface{})
	testPrivateData, _ := json.Marshal(testPrivateDataMap)
	assetData.PrivateData = testPrivateData
	assetData.PublicData = testPublicData
	assetData.OwnerIds = []string{"owner1"}
	assetData.IndexTableName = "CustomAssetIndex"
	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = global.KEY_TYPE_SYM
	assetKey.KeyBytes = key1

	// create index table
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	table := index.GetTable(stub, "CustomAssetIndex", "assetId")
	err := table.AddIndex([]string{"age", "assetId"}, false)
	err = table.SaveToLedger()
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 := GetAssetManager(stub, caller1)
	err = am1.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err != nil, "Expected AddAsset to fail, missing index key - age")
	mstub.MockTransactionEnd("t123")

	testPrivateDataMap["age"] = "20"
	testPrivateData, _ = json.Marshal(testPrivateDataMap)
	assetData.PrivateData = testPrivateData
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 = GetAssetManager(stub, caller1)
	err = am1.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// check ledger for asset and assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, "key1", key1, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, "key1")

	// check ledger for index values
	rows1 := testGetRow(stub, "CustomAssetIndex", assetData.AssetId, "20")
	test_utils.AssertInLists(t, assetData.AssetId, rows1, "Expected assetId index to be saved successfully")
	mstub.MockTransactionEnd("t123")

	testPrivateDataMap["age"] = "22"
	testPrivateData, _ = json.Marshal(testPrivateDataMap)
	assetData.PrivateData = testPrivateData
	// replace existing asset with access (caller1)
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 = GetAssetManager(stub, caller1)
	err = am1.UpdateAsset(assetData, assetKey)
	test_utils.AssertTrue(t, err == nil, "Expected UpdateAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// check ledger for asset and assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, "key1", key1, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, "key1")

	// check ledger for index values
	rows1 = testGetRow(stub, "CustomAssetIndex", assetData.AssetId, "22")
	test_utils.AssertInLists(t, assetData.AssetId, rows1, "Expected assetId index to be saved successfully")

	mstub.MockTransactionEnd("t123")
}

func TestUpdateAsset_Index_DefaultPrimaryKey(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestUpdateAsset_Index_DefaultPrimaryKey function called")

	// create a MockStub
	mstub := setup(t)

	caller1 := data_model.User{}
	caller1.PrivateKey = test_utils.GeneratePrivateKey()
	caller1.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller1.PrivateKey))
	pub := crypto.PublicKeyToBytes(caller1.PrivateKey.Public().(*rsa.PublicKey))
	caller1.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller1.ID = "owner1"

	key1 := test_utils.GenerateSymKey()
	assetData := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	assetData.AssetKeyId = "key1"
	assetData.AssetKeyHash = crypto.Hash(key1)
	testPublicDataMap := make(map[string]interface{})
	testPublicDataMap["assetId"] = assetData.AssetId
	testPublicData, _ := json.Marshal(testPublicDataMap)
	testPrivateDataMap := make(map[string]interface{})
	testPrivateData, _ := json.Marshal(testPrivateDataMap)
	assetData.PrivateData = testPrivateData
	assetData.PublicData = testPublicData
	assetData.OwnerIds = []string{"owner1"}
	assetData.IndexTableName = "CustomAssetIndex"
	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = global.KEY_TYPE_SYM
	assetKey.KeyBytes = key1

	// create index table
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	table := index.GetTable(stub, "CustomAssetIndex")
	err := table.AddIndex([]string{"age", table.GetPrimaryKeyId()}, false)
	err = table.SaveToLedger()
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 := GetAssetManager(stub, caller1)
	err = am1.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err != nil, "Expected AddAsset to fail, missing index key - age")
	mstub.MockTransactionEnd("t123")

	testPrivateDataMap["age"] = "20"
	testPrivateData, _ = json.Marshal(testPrivateDataMap)
	assetData.PrivateData = testPrivateData
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 = GetAssetManager(stub, caller1)
	err = am1.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// check ledger for asset and assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, "key1", key1, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, "key1")

	// check ledger for index values
	rows1 := testGetRow(stub, "CustomAssetIndex", assetData.AssetId, "20")
	test_utils.AssertInLists(t, assetData.AssetId, rows1, "Expected assetId index to be saved successfully")

	mstub.MockTransactionEnd("t123")

	testPrivateDataMap["age"] = "22"
	testPrivateData, _ = json.Marshal(testPrivateDataMap)
	assetData.PrivateData = testPrivateData
	// replace existing asset with access (caller1)
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 = GetAssetManager(stub, caller1)
	err = am1.UpdateAsset(assetData, assetKey)
	test_utils.AssertTrue(t, err == nil, "Expected UpdateAsset to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	// check ledger for asset and assetId
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, "key1", key1, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, "key1")

	// check ledger for index values
	rows1 = testGetRow(stub, "CustomAssetIndex", assetData.AssetId, "22")
	test_utils.AssertInLists(t, assetData.AssetId, rows1, "Expected assetId index to be saved successfully")

	mstub.MockTransactionEnd("t123")
}

func TestUpdateAsset_SamePrivateData(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestUpdateAsset_SamePrivateData function called")

	// create a MockStub
	mstub := setup(t)

	caller1 := test_utils.CreateTestUser("owner1")

	assetKey := test_utils.CreateSymKey("key1")
	testPublicData := test_utils.CreateTestAssetData("public1")
	testPrivateData := test_utils.CreateTestAssetData("private1")
	assetData := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	assetData.AssetKeyId = assetKey.ID
	assetData.AssetKeyHash = crypto.Hash(assetKey.KeyBytes)
	assetData.PrivateData = testPrivateData
	assetData.PublicData = testPublicData
	assetData.OwnerIds = []string{"owner1"}

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	am1 := GetAssetManager(stub, caller1)
	err := am1.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 = GetAssetManager(stub, caller1)

	// check ledger for asset and assetId
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, assetKey.ID, assetKey.KeyBytes, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, assetKey.ID)

	// get asset without private data
	assetEncPrivateData, err := am1.GetAsset(assetData.AssetId, assetKey)
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")

	// replace existing asset with access (caller1)
	err = am1.UpdateAsset(*assetEncPrivateData, assetKey)
	test_utils.AssertTrue(t, err == nil, "Expected UpdateAsset to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	// check ledger for asset and assetId
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, assetKey.ID, assetKey.KeyBytes, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, assetKey.ID)
	mstub.MockTransactionEnd("t123")
}

func testGetRow(stub cached_stub.CachedStubInterface, tableName string, id string, age string) []string {
	table := index.GetTable(stub, tableName)
	iter, err := table.GetRowsByPartialKey([]string{"age", table.GetPrimaryKeyId()}, []string{age, id})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	defer iter.Close()
	rows := []string{}
	row := make(map[string]string)
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		err = json.Unmarshal(rowBytes, &row)

		rows = append(rows, row[table.GetPrimaryKeyId()])
		logger.Infof("=== row %v", row)
	}

	return rows
}

func TestDeleteAsset_WriteAccess(t *testing.T) {
	logger.Info("TestDeleteAssetByKey_WriteAccess function called")

	// create a MockStub
	mstub := setup(t)

	owner := test_utils.CreateTestUser("ownerId")
	privateKey1 := owner.PrivateKey
	priv1 := crypto.PrivateKeyToBytes(privateKey1)
	pub1 := crypto.PublicKeyToBytes(privateKey1.Public().(*rsa.PublicKey))

	// not-owner's keys
	notOwner := test_utils.CreateTestUser("not-ownerId")
	privateKey2 := notOwner.PrivateKey
	priv2 := crypto.PrivateKeyToBytes(privateKey2)
	//pub2 := crypto.PublicKeyToBytes(privateKey2.Public().(*rsa.PublicKey))
	notownerKey := data_model.Key{}
	notownerKey.ID = notOwner.GetPubPrivKeyId()
	notownerKey.KeyBytes = priv2
	notownerKey.Type = global.KEY_TYPE_PRIVATE

	// generate key without adding it to KeyGraph
	key1 := test_utils.GenerateSymKey()
	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.KeyBytes = key1
	assetKey.Type = global.KEY_TYPE_SYM

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)

	testAsset := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	testAsset.PublicData = test_utils.CreateTestAssetData("public1")
	testAsset.PrivateData = test_utils.CreateTestAssetData("private1")
	testAsset.OwnerIds = []string{"ownerId"}
	testAsset.AssetKeyId = "key1"
	testAsset.AssetKeyHash = crypto.Hash(key1)
	err := putAssetByKey(stub, owner, testAsset, "key1", key1, owner.GetPubPrivKeyId(), priv1, pub1)
	test_utils.AssertTrue(t, err == nil, "Expected putAssetByKey to succeed")

	mstub.MockTransactionEnd("t123")

	// try to delete without write access
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am := GetAssetManager(stub, notOwner)
	err = am.DeleteAsset(testAsset.AssetId, assetKey)
	test_utils.AssertTrue(t, err != nil, "Expected deleteAssetByKey to fail")
	mstub.MockTransactionEnd("t123")

	// give write access to not-ownerId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am = GetAssetManager(stub, owner)
	accessControl := data_model.AccessControl{}
	accessControl.Access = global.ACCESS_WRITE
	accessControl.UserId = "not-ownwerId"
	accessControl.AssetId = testAsset.AssetId
	accessControl.AssetKey = &assetKey
	accessControl.UserKey = &notownerKey
	err = am.AddAccessToAsset(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccessTpAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// not-ownerId attempts to remove ownerId's write access
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am = GetAssetManager(stub, notOwner)
	accessControl = data_model.AccessControl{}
	accessControl.Access = global.ACCESS_READ
	accessControl.UserId = "ownwerId"
	accessControl.AssetId = testAsset.AssetId
	accessControl.AssetKey = &assetKey
	err = am.RemoveAccessFromAsset(accessControl)
	test_utils.AssertTrue(t, err != nil, "Expected RemoveAccessFromAsset to fail")
	mstub.MockTransactionEnd("t123")

	// ownerId attempts to remove their own write access - not allowed to remove original owner
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	testAsset.OwnerIds = []string{"not-ownerId"}
	err = putAssetByKey(stub, owner, testAsset, "key1", key1, owner.GetPubPrivKeyId(), priv1, pub1)
	test_utils.AssertTrue(t, err != nil, "Expected putAssetByKey to fail")
	mstub.MockTransactionEnd("t123")

	// try to delete with write access
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am = GetAssetManager(stub, notOwner)
	err = am.DeleteAsset(testAsset.AssetId, assetKey)
	test_utils.AssertTrue(t, err == nil, "Expected DeleteAsset to succeed")
	mstub.MockTransactionEnd("t123")
}

func TestDeleteAsset(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestDeleteAsset function called")

	mstub := setup(t)

	caller := data_model.User{}
	caller.PrivateKey = test_utils.GeneratePrivateKey()
	caller.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller.PrivateKey))
	pub := crypto.PublicKeyToBytes(caller.PrivateKey.Public().(*rsa.PublicKey))
	caller.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller.ID = "ownerId"

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	var err = key_mgmt_i.AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t123")

	var testPublicData = test_utils.CreateTestAssetData("public1")
	var testPrivateData = test_utils.CreateTestAssetData("private1")

	assetData := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	assetData.AssetKeyId = "key1"
	assetData.AssetKeyHash = crypto.Hash(key1)
	assetData.PrivateData = testPrivateData
	assetData.PublicData = testPublicData
	assetData.OwnerIds = []string{"ownerId"}
	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = global.KEY_TYPE_SYM
	assetKey.KeyBytes = key1

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am := GetAssetManager(stub, caller)
	err = am.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// check ledger for asset and assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, "key1", key1, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, "key1")
	mstub.MockTransactionEnd("t123")

	// delete asset
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am = GetAssetManager(stub, caller)
	err = am.DeleteAsset(assetData.AssetId, assetKey)
	test_utils.AssertTrue(t, err == nil, "Expected DeleteAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// try to get asset from ledger, expected to fail
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	deletedAsset, err := getAssetByKey(stub, assetData.AssetId, key1)
	test_utils.AssertTrue(t, err == nil, "Expected get asset to pass")
	test_utils.AssertTrue(t, len(deletedAsset.AssetId) == 0, "Expected empty asset ID")
	mstub.MockTransactionEnd("t123")
}

func checkAsset(t *testing.T, stub cached_stub.CachedStubInterface, assetId string, publicData []byte, privateData []byte, assetKeyId string, assetKey []byte, ownerIds []string) {
	// check ledger for asset

	assetLedgerKey := assetId

	assetBytes, err := stub.GetState(assetLedgerKey)
	test_utils.AssertTrue(t, err == nil, "Expected asset bytes")
	// retrieved asset
	assetData := data_model.Asset{}
	json.Unmarshal(assetBytes, &assetData)

	// check retrieved asset values
	test_utils.AssertTrue(t, assetData.AssetKeyId == assetKeyId, "incorrect assetKeyId")
	test_utils.AssertTrue(t, bytes.Equal(assetData.AssetKeyHash, crypto.Hash(assetKey)), "incorrect assetKeyHash")
	decryptedPrivateData := []byte{}
	connectionID := assetData.GetDatastoreConnectionID()
	if len(connectionID) == 0 && len(defaultDatastoreConnectionID) != 0 {
		connectionID = defaultDatastoreConnectionID
	}
	if len(connectionID) != 0 {
		decryptedPrivateData = assetData.PrivateData
		myDatastore, _ := datastore_c.GetDatastoreImpl(stub, connectionID)
		encryptedData, _ := crypto.EncryptWithSymKey(assetKey, privateData)
		privateData = []byte(myDatastore.ComputeHash(stub, encryptedData))
	} else {
		decryptedPrivateData, _ = crypto.DecryptWithSymKey(assetKey, assetData.PrivateData)
	}
	test_utils.AssertTrue(t, bytes.Equal(decryptedPrivateData, privateData), "PrivateData could not be decrypted")
	test_utils.AssertTrue(t, bytes.Equal(assetData.PublicData, publicData), "PublicData could not be decrypted")
	test_utils.AssertTrue(t, len(ownerIds) == len(assetData.OwnerIds), "incorrect ownerIds")
	for _, owner := range ownerIds {
		test_utils.AssertTrue(t, utils.InList(assetData.OwnerIds, owner), "incorrect ownerIds: "+owner+" is missing")
	}
}

func checkAssetId(t *testing.T, stub cached_stub.CachedStubInterface, assetId string, assetKeyId string) {
	//get asset key id
	indexAssetKeyId, err := GetAssetKeyId(stub, assetId)
	test_utils.AssertTrue(t, err == nil, "Expected to get asset key id")
	test_utils.AssertTrue(t, indexAssetKeyId == assetKeyId, "incorrect assetId")
}

// Test function for GetAsset
func TestGetAssetByKey(t *testing.T) {
	logger.Info("TestGetAssetByKey function called")

	// create a MockStub
	mstub := setup(t)

	owner := test_utils.CreateTestUser("ownerId")
	privateKey := owner.PrivateKey
	priv := crypto.PrivateKeyToBytes(privateKey)
	pub := crypto.PublicKeyToBytes(privateKey.Public().(*rsa.PublicKey))
	wrongKey := test_utils.GenerateSymKey()

	key1 := test_utils.GenerateSymKey()

	// generate graph of sym keys
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)

	// add testAsset to ledger
	testAsset := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	publicData1 := test_utils.CreateTestAssetData("public1")
	privateData1 := test_utils.CreateTestAssetData("private1")
	testAsset.PublicData = publicData1
	testAsset.PrivateData = privateData1
	testAsset.OwnerIds = []string{"ownerId"}
	putAssetByKey(stub, owner, testAsset, "key1", key1, "myKey", priv, pub)
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// get asset 1 from ledger using start key 1
	result, _ := getAssetByKey(stub, testAsset.AssetId, key1)
	test_utils.AssertTrue(t, bytes.Equal(publicData1, result.PublicData), "Failed to get asset 1 public data")
	test_utils.AssertTrue(t, bytes.Equal(privateData1, result.PrivateData), "Failed to get asset 1 private data")

	// Tested for non happy path
	result, _ = getAssetByKey(stub, testAsset.AssetId, wrongKey)
	// since wrongKey cannot decrypt the private data, GetAsset returns the encrypted private data
	encryptedData := data_model.EncryptedData{}
	err := json.Unmarshal(result.PrivateData, &encryptedData)
	test_utils.AssertTrue(t, err == nil, "result.PrivateData should be encryptedData")
	mstub.MockTransactionEnd("t123")
}

func TestAssetManagerImpl_GetStub(t *testing.T) {
	mstub := setup(t)
	caller := test_utils.CreateTestUser("myUser")
	stub := cached_stub.NewCachedStub(mstub)
	assetManager := GetAssetManager(stub, caller)

	retStub := assetManager.GetStub()
	test_utils.AssertTrue(t, retStub == stub, "Expected stub to be returned.")
}

func TestAssetManagerImpl_GetCaller(t *testing.T) {
	mstub := setup(t)
	caller := test_utils.CreateTestUser("myUser")
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	assetManager := GetAssetManager(stub, caller)

	retCaller := assetManager.GetCaller()
	test_utils.AssertTrue(t, retCaller.ID == caller.ID, "Expected caller to be returned.")
	mstub.MockTransactionEnd("t123")
}

func TestGetAsset(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetAsset function called")

	mstub := setup(t)

	caller1 := test_utils.CreateTestUser("owner1")
	caller2 := test_utils.CreateTestUser("owner2")

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	var err = key_mgmt_i.AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	var testPublicData = test_utils.CreateTestAssetData("public1")
	var testPrivateData = test_utils.CreateTestAssetData("private1")
	assetData := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	assetData.AssetKeyId = "key1"
	assetData.AssetKeyHash = crypto.Hash(key1)
	assetData.PrivateData = testPrivateData
	assetData.PublicData = testPublicData
	assetData.OwnerIds = []string{"owner1"}
	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = global.KEY_TYPE_SYM
	assetKey.KeyBytes = key1

	am1 := GetAssetManager(stub, caller1)
	err = am1.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// check ledger for asset and assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, "key1", key1, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, "key1")
	mstub.MockTransactionEnd("t123")

	// get asset
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 = GetAssetManager(stub, caller1)
	result, err := am1.GetAsset(assetData.AssetId, assetKey)
	mstub.MockTransactionEnd("t123")

	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, bytes.Equal(testPublicData, result.PublicData), "Failed to get asset 1 public data")
	test_utils.AssertTrue(t, bytes.Equal(testPrivateData, result.PrivateData), "Failed to get asset 1 private data")

	// get asset without retrieving private data
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 = GetAssetManager(stub, caller1)
	result, err = am1.GetAsset(assetData.AssetId, data_model.Key{})
	mstub.MockTransactionEnd("t123")

	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, bytes.Equal(testPublicData, result.PublicData), "Failed to get asset 1 public data")
	test_utils.AssertTrue(t, !bytes.Equal(testPrivateData, result.PrivateData), "Expected encrypted PrivateData")
	test_utils.AssertTrue(t, data_model.IsEncryptedData(result.PrivateData), "PrivateData should be encrypted.")

	// get asset without access to private data, wrong key
	wrongKeyByte := test_utils.GenerateSymKey()
	wrongKey := data_model.Key{}
	wrongKey.ID = "wrongkey"
	wrongKey.KeyBytes = wrongKeyByte
	wrongKey.Type = global.KEY_TYPE_SYM
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am2 := GetAssetManager(stub, caller2)
	result, err = am2.GetAsset(assetData.AssetId, wrongKey)
	test_utils.AssertTrue(t, err != nil, "Expected GetAsset to fail")
	mstub.MockTransactionEnd("t123")

	// get asset without retrieving private data and without access to private data
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am2 = GetAssetManager(stub, caller2)
	result, err = am2.GetAsset(assetData.AssetId, data_model.Key{})
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, bytes.Equal(testPublicData, result.PublicData), "Failed to get asset 1 public data")
	test_utils.AssertTrue(t, !bytes.Equal(testPrivateData, result.PrivateData), "Expected encrypted PrivateData")
	test_utils.AssertTrue(t, data_model.IsEncryptedData(result.PrivateData), "PrivateData should be encrypted.")
	mstub.MockTransactionEnd("t123")

	// get asset with invalid assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 = GetAssetManager(stub, caller1)
	result, err = am1.GetAsset(GetAssetId("data_model.Asset", "assetX"), assetKey)
	mstub.MockTransactionEnd("t123")
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, len(result.AssetId) == 0, "Expected GetAsset to return empty asset")
}

func TestGetAsset_Datastore(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetAsset_Datastore function called")

	mstub := setup(t)
	// register connection (ledger)
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	connection := datastore.DatastoreConnection{
		ID:   "ledger2",
		Type: datastore.DATASTORE_TYPE_DEFAULT_LEDGER,
	}
	err := datastore_c.PutDatastoreConnection(stub, connection)
	test_utils.AssertTrue(t, err == nil, "Expected PutDatastoreConnection to succeed")
	mstub.MockTransactionEnd("t123")

	caller1 := test_utils.CreateTestUser("owner1")
	caller2 := test_utils.CreateTestUser("owner2")

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	err = key_mgmt_i.AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)

	var testPublicData = test_utils.CreateTestAssetData("public1")
	var testPrivateData = test_utils.CreateTestAssetData("private1")
	assetData := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	assetData.AssetKeyId = "key1"
	assetData.AssetKeyHash = crypto.Hash(key1)
	assetData.PrivateData = testPrivateData
	assetData.PublicData = testPublicData
	assetData.OwnerIds = []string{"owner1"}
	assetData.SetDatastoreConnectionID("ledger2")

	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = global.KEY_TYPE_SYM
	assetKey.KeyBytes = key1

	am1 := GetAssetManager(stub, caller1)
	err = am1.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// check ledger for asset and assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, "key1", key1, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, "key1")
	mstub.MockTransactionEnd("t123")

	// get asset
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 = GetAssetManager(stub, caller1)
	result, err := am1.GetAsset(assetData.AssetId, assetKey)
	mstub.MockTransactionEnd("t123")

	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, bytes.Equal(testPublicData, result.PublicData), "Failed to get asset 1 public data")
	test_utils.AssertTrue(t, bytes.Equal(testPrivateData, result.PrivateData), "Failed to get asset 1 private data")

	// get asset without retrieving private data
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 = GetAssetManager(stub, caller1)
	result, err = am1.GetAsset(assetData.AssetId, data_model.Key{})
	mstub.MockTransactionEnd("t123")

	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, bytes.Equal(testPublicData, result.PublicData), "Failed to get asset 1 public data")
	test_utils.AssertTrue(t, !bytes.Equal(testPrivateData, result.PrivateData), "Expected encrypted PrivateData")
	test_utils.AssertTrue(t, data_model.IsEncryptedData(result.PrivateData), "PrivateData should be encrypted.")

	// get asset without access to private data, wrong key
	wrongKeyByte := test_utils.GenerateSymKey()
	wrongKey := data_model.Key{}
	wrongKey.ID = "wrongkey"
	wrongKey.KeyBytes = wrongKeyByte
	wrongKey.Type = global.KEY_TYPE_SYM
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am2 := GetAssetManager(stub, caller2)
	result, err = am2.GetAsset(assetData.AssetId, wrongKey)
	test_utils.AssertTrue(t, err != nil, "Expected GetAsset to fail")
	mstub.MockTransactionEnd("t123")

	// get asset without retrieving private data and without access to private data
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am2 = GetAssetManager(stub, caller2)
	result, err = am2.GetAsset(assetData.AssetId, data_model.Key{})
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, bytes.Equal(testPublicData, result.PublicData), "Failed to get asset 1 public data")
	test_utils.AssertTrue(t, !bytes.Equal(testPrivateData, result.PrivateData), "Expected encrypted PrivateData")
	test_utils.AssertTrue(t, data_model.IsEncryptedData(result.PrivateData), "PrivateData should be encrypted.")
	mstub.MockTransactionEnd("t123")

	// get asset with invalid assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am1 = GetAssetManager(stub, caller1)
	result, err = am1.GetAsset(GetAssetId("data_model.Asset", "assetX"), assetKey)
	mstub.MockTransactionEnd("t123")
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, len(result.AssetId) == 0, "Expected GetAsset to return empty asset")
}

// Disable cache in this test
func TestGetAsset_Datastore_DisableCache(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetAsset_Datastore_DisableCache function called")

	mstub := setup(t)
	// register connection (ledger)
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub, false, false)
	connection := datastore.DatastoreConnection{
		ID:   "ledger2",
		Type: datastore.DATASTORE_TYPE_DEFAULT_LEDGER,
	}
	err := datastore_c.PutDatastoreConnection(stub, connection)
	test_utils.AssertTrue(t, err == nil, "Expected PutDatastoreConnection to succeed")
	mstub.MockTransactionEnd("t123")

	caller1 := test_utils.CreateTestUser("owner1")
	caller2 := test_utils.CreateTestUser("owner2")

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)
	err = key_mgmt_i.AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)

	var testPublicData = test_utils.CreateTestAssetData("public1")
	var testPrivateData = test_utils.CreateTestAssetData("private1")
	assetData := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	assetData.AssetKeyId = "key1"
	assetData.AssetKeyHash = crypto.Hash(key1)
	assetData.PrivateData = testPrivateData
	assetData.PublicData = testPublicData
	assetData.OwnerIds = []string{"owner1"}
	assetData.SetDatastoreConnectionID("ledger2")

	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = global.KEY_TYPE_SYM
	assetKey.KeyBytes = key1

	am1 := GetAssetManager(stub, caller1)
	err = am1.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// check ledger for asset and assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, "key1", key1, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, "key1")
	mstub.MockTransactionEnd("t123")

	// get asset
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)
	am1 = GetAssetManager(stub, caller1)
	result, err := am1.GetAsset(assetData.AssetId, assetKey)
	mstub.MockTransactionEnd("t123")

	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, bytes.Equal(testPublicData, result.PublicData), "Failed to get asset 1 public data")
	test_utils.AssertTrue(t, bytes.Equal(testPrivateData, result.PrivateData), "Failed to get asset 1 private data")

	// get asset without retrieving private data
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)
	am1 = GetAssetManager(stub, caller1)
	result, err = am1.GetAsset(assetData.AssetId, data_model.Key{})
	mstub.MockTransactionEnd("t123")

	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, bytes.Equal(testPublicData, result.PublicData), "Failed to get asset 1 public data")
	test_utils.AssertTrue(t, !bytes.Equal(testPrivateData, result.PrivateData), "Expected encrypted PrivateData")
	test_utils.AssertTrue(t, data_model.IsEncryptedData(result.PrivateData), "PrivateData should be encrypted.")

	// get asset without access to private data, wrong key
	wrongKeyByte := test_utils.GenerateSymKey()
	wrongKey := data_model.Key{}
	wrongKey.ID = "wrongkey"
	wrongKey.KeyBytes = wrongKeyByte
	wrongKey.Type = global.KEY_TYPE_SYM
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am2 := GetAssetManager(stub, caller2)
	result, err = am2.GetAsset(assetData.AssetId, wrongKey)
	test_utils.AssertTrue(t, err != nil, "Expected GetAsset to fail")
	mstub.MockTransactionEnd("t123")

	// get asset without retrieving private data and without access to private data
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)
	am2 = GetAssetManager(stub, caller2)
	result, err = am2.GetAsset(assetData.AssetId, data_model.Key{})
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, bytes.Equal(testPublicData, result.PublicData), "Failed to get asset 1 public data")
	test_utils.AssertTrue(t, !bytes.Equal(testPrivateData, result.PrivateData), "Expected encrypted PrivateData")
	test_utils.AssertTrue(t, data_model.IsEncryptedData(result.PrivateData), "PrivateData should be encrypted.")
	mstub.MockTransactionEnd("t123")

	// get asset with invalid assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)
	am1 = GetAssetManager(stub, caller1)
	result, err = am1.GetAsset(GetAssetId("data_model.Asset", "assetX"), assetKey)
	mstub.MockTransactionEnd("t123")
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, len(result.AssetId) == 0, "Expected GetAsset to return empty asset")
}

// if cloudant is not ready, it will skip the test
func TestGetAsset_Datastore_Cloudant_DisableCache(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetAsset_Datastore_DisableCache function called")

	mstub := setup(t)
	utils.SetLogLevel(shim.LogDebug)

	// register connection (ledger)
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub, false, false)

	username := "admin"
	password := "pass"
	database := "test"
	host := "http://127.0.0.1:9080"
	// Get values from environment variables
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_USERNAME")) {
		username = os.Getenv("CLOUDANT_USERNAME")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_PASSWORD")) {
		password = os.Getenv("CLOUDANT_PASSWORD")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_DATABASE")) {
		database = os.Getenv("CLOUDANT_DATABASE")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_HOST")) {
		host = os.Getenv("CLOUDANT_HOST")
	}

	params := url.Values{}
	params.Add("username", username)
	params.Add("password", password)
	params.Add("database", database)
	params.Add("host", host)
	params.Add("create_database", "true")

	connection := datastore.DatastoreConnection{
		ID:         "cloudant1",
		Type:       datastore.DATASTORE_TYPE_DEFAULT_CLOUDANT,
		ConnectStr: params.Encode(),
	}
	err := datastore_c.PutDatastoreConnection(stub, connection)
	test_utils.AssertTrue(t, err == nil, "Expected PutDatastoreConnection to succeed")
	mstub.MockTransactionEnd("t123")

	// check if cloudant is available or not
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	cloudant, err := datastore_c.GetDatastoreImpl(stub, "cloudant1")
	if err != nil {
		logger.Errorf("Error Getting Cloudant datastore; Halt TestCloudantDatastore: %v", err)
		return
	}
	test_utils.AssertTrue(t, cloudant.IsReady(), "cloudant should be isReady true")
	mstub.MockTransactionEnd("t123")

	caller1 := test_utils.CreateTestUser("owner1")
	caller2 := test_utils.CreateTestUser("owner2")

	key1 := test_utils.GenerateSymKey()
	key2 := test_utils.GenerateSymKey()

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)
	err = key_mgmt_i.AddAccessWithKeys(stub, key1, "key1", key2, "key2", key1)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)

	var testPublicData = test_utils.CreateTestAssetData("public1")
	var testPrivateData = test_utils.CreateTestAssetData("private1")
	assetData := test_utils.CreateTestAsset(GetAssetId("data_model.Asset", "asset1"))
	assetData.AssetKeyId = "key1"
	assetData.AssetKeyHash = crypto.Hash(key1)
	assetData.PrivateData = testPrivateData
	assetData.PublicData = testPublicData
	assetData.OwnerIds = []string{"owner1"}
	assetData.SetDatastoreConnectionID("cloudant1")

	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = global.KEY_TYPE_SYM
	assetKey.KeyBytes = key1

	am1 := GetAssetManager(stub, caller1)
	err = am1.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// check ledger for asset and assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)
	checkAsset(t, stub, assetData.AssetId, testPublicData, testPrivateData, "key1", key1, assetData.OwnerIds)
	checkAssetId(t, stub, assetData.AssetId, "key1")
	mstub.MockTransactionEnd("t123")

	// get asset
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)
	am1 = GetAssetManager(stub, caller1)
	result, err := am1.GetAsset(assetData.AssetId, assetKey)
	mstub.MockTransactionEnd("t123")

	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, bytes.Equal(testPublicData, result.PublicData), "Failed to get asset 1 public data")
	test_utils.AssertTrue(t, bytes.Equal(testPrivateData, result.PrivateData), "Failed to get asset 1 private data")

	// get asset without retrieving private data
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)
	am1 = GetAssetManager(stub, caller1)
	result, err = am1.GetAsset(assetData.AssetId, data_model.Key{})
	mstub.MockTransactionEnd("t123")

	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, bytes.Equal(testPublicData, result.PublicData), "Failed to get asset 1 public data")
	test_utils.AssertTrue(t, !bytes.Equal(testPrivateData, result.PrivateData), "Expected encrypted PrivateData")
	test_utils.AssertTrue(t, data_model.IsEncryptedData(result.PrivateData), "PrivateData should be encrypted.")

	// get asset without access to private data, wrong key
	wrongKeyByte := test_utils.GenerateSymKey()
	wrongKey := data_model.Key{}
	wrongKey.ID = "wrongkey"
	wrongKey.KeyBytes = wrongKeyByte
	wrongKey.Type = global.KEY_TYPE_SYM
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am2 := GetAssetManager(stub, caller2)
	result, err = am2.GetAsset(assetData.AssetId, wrongKey)
	test_utils.AssertTrue(t, err != nil, "Expected GetAsset to fail")
	mstub.MockTransactionEnd("t123")

	// get asset without retrieving private data and without access to private data
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)
	am2 = GetAssetManager(stub, caller2)
	result, err = am2.GetAsset(assetData.AssetId, data_model.Key{})
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, bytes.Equal(testPublicData, result.PublicData), "Failed to get asset 1 public data")
	test_utils.AssertTrue(t, !bytes.Equal(testPrivateData, result.PrivateData), "Expected encrypted PrivateData")
	test_utils.AssertTrue(t, data_model.IsEncryptedData(result.PrivateData), "PrivateData should be encrypted.")
	mstub.MockTransactionEnd("t123")

	// get asset with invalid assetId
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub, false, false)
	am1 = GetAssetManager(stub, caller1)
	result, err = am1.GetAsset(GetAssetId("data_model.Asset", "assetX"), assetKey)
	mstub.MockTransactionEnd("t123")
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, len(result.AssetId) == 0, "Expected GetAsset to return empty asset")
}

// Test function for GetAssets key caching
func TestGetAssetIter_KeyCaching(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Infof("testing TestGetAssetIter_KeyCaching")

	// create a MockStub
	mstub := setup(t)
	caller := test_utils.CreateTestUser("ownerId")

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	key1 := test_utils.CreateSymKey("key1")
	key2 := test_utils.CreateSymKey("key2")
	key3 := test_utils.CreateSymKey("key3")
	key_mgmt_i.AddAccessWithKeys(stub, key1.KeyBytes, key1.ID, key2.KeyBytes, key2.ID, key1.KeyBytes)
	mstub.MockTransactionEnd("t123")

	// create index table
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	indexName := "CustomAssetIndex"
	table := index.GetTable(stub, indexName, "objectId")
	err := table.AddIndex([]string{"age", "objectId"}, false)
	test_utils.AssertTrue(t, err == nil, "Expected to succeed")
	err = table.SaveToLedger()
	test_utils.AssertTrue(t, err == nil, "Expected to succeed")
	mstub.MockTransactionEnd("t123")

	assetNamespace := "package.Generic"
	putTestObjectForTest(t, mstub, caller, "uniqueId1", "20", assetNamespace, indexName, []string{caller.ID}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId2", "20", assetNamespace, indexName, []string{caller.ID}, key2, false)
	putTestObjectForTest(t, mstub, caller, "uniqueId3", "20", assetNamespace, indexName, []string{caller.ID}, key3, false)
	putTestObjectForTest(t, mstub, caller, "uniqueId4", "53", assetNamespace, indexName, []string{caller.ID}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId5", "53", assetNamespace, indexName, []string{caller.ID}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId6", "53", assetNamespace, indexName, []string{caller.ID}, key3, false)
	putTestObjectForTest(t, mstub, caller, "uniqueId7", "53", assetNamespace, indexName, []string{caller.ID}, key2, false)
	putTestObjectForTest(t, mstub, caller, "uniqueId8", "53", assetNamespace, indexName, []string{caller.ID}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId9", "53", assetNamespace, indexName, []string{caller.ID}, key2, false)
	putTestObjectForTest(t, mstub, caller, "uniqueId10", "40", assetNamespace, indexName, []string{caller.ID}, key3, false)
	putTestObjectForTest(t, mstub, caller, "uniqueId11", "40", assetNamespace, indexName, []string{caller.ID}, key3, false)
	putTestObjectForTest(t, mstub, caller, "uniqueId12", "40", assetNamespace, indexName, []string{caller.ID}, key3, false)
	putTestObjectForTest(t, mstub, caller, "uniqueId13", "73", assetNamespace, indexName, []string{caller.ID}, key2, false)
	putTestObjectForTest(t, mstub, caller, "uniqueId14", "73", assetNamespace, indexName, []string{caller.ID}, key2, false)

	var keyFunc asset_key_func.AssetKeyPathFunc = func(stub cached_stub.CachedStubInterface, caller data_model.User, asset data_model.Asset) ([]string, error) {
		keyPath := []string{caller.GetPubPrivKeyId()}
		if asset.AssetKeyId == "key1" {
			keyPath = append(keyPath, "key1")
		} else if asset.AssetKeyId == "key2" {
			keyPath = append(keyPath, "key1")
			keyPath = append(keyPath, "key2")
		} else if asset.AssetKeyId == "key3" {
			keyPath = append(keyPath, "key3")
		}
		return keyPath, nil
	}

	var keyByteFunc asset_key_func.AssetKeyByteFunc = func(stub cached_stub.CachedStubInterface, caller data_model.User, asset data_model.Asset) ([]byte, error) {
		keyPath := []string{caller.GetPubPrivKeyId()}
		if asset.AssetKeyId == "key1" {
			keyPath = append(keyPath, "key1")
		} else if asset.AssetKeyId == "key2" {
			keyPath = append(keyPath, "key1")
			keyPath = append(keyPath, "key2")
		} else if asset.AssetKeyId == "key3" {
			keyPath = append(keyPath, "key3")
		}
		return GetAssetKey(stub, asset.AssetId, keyPath, crypto.PrivateKeyToBytes(caller.PrivateKey))
	}

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am := GetAssetManager(stub, caller)
	// GetAssetIter will have to get key1, and fail to get key2 and key3
	assetIter, err := am.GetAssetIter(
		assetNamespace,
		indexName,
		[]string{"age"},
		[]string{"20"},
		[]string{"20"},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	test_utils.AssertTrue(t, getNumUnencryptedAssets(assetIter) == 1, "Expected GetAssetIter to return 1 asset")

	// GetAssetIter will have to get key1, and fail to get key2 and key3
	assetIter, err = am.GetAssetIter(
		assetNamespace,
		indexName,
		[]string{"age"},
		[]string{"20"},
		[]string{"20"},
		true,
		false,
		keyFunc,
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	test_utils.AssertTrue(t, getNumUnencryptedAssets(assetIter) == 2, "Expected GetAssetIter to return 2 asset")

	// GetAssetIter will have to get key1, reuse key1, fail to get key3, get key2, then reuse key1 and key2
	assetIter, err = am.GetAssetIter(
		assetNamespace,
		indexName,
		[]string{"age"},
		[]string{"53"},
		[]string{"53"},
		true,
		false,
		keyByteFunc,
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	test_utils.AssertTrue(t, getNumUnencryptedAssets(assetIter) == 5, "Expected GetAssetIter to return 5 asset")

	// GetAssetIter will fail to decrypt all three of the assets the fit the query and return empty iter instead of err
	assetIter, err = am.GetAssetIter(
		assetNamespace,
		indexName,
		[]string{"age"},
		[]string{"40"},
		[]string{"40"},
		true,
		false,
		keyFunc,
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	test_utils.AssertTrue(t, getNumUnencryptedAssets(assetIter) == 0, "Expected GetAssetIter to return 0 asset")

	// Test that keys are acutally being cached and that Next is not just getting the key from the ledger every time
	// GetAssetIter will get two assets which are encrypted with key2
	assetIter, err = am.GetAssetIter(
		assetNamespace,
		indexName,
		[]string{"age"},
		[]string{"73"},
		[]string{"73"},
		true,
		false,
		keyFunc,
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	test_utils.AssertTrue(t, assetIter.HasNext(), "Expected AssetIter not to be empty")
	// call Next() to get key2 into the key cache
	assetIter.Next()
	test_utils.AssertTrue(t, err == nil, "Expected assetIter.Next() to succeed")
	test_utils.AssertTrue(t, assetIter.HasNext(), "Expected AssetIter to have 2 assets")
	// delete access to key2
	err = key_mgmt_i.RevokeAccess(stub, key1.ID, key2.ID)
	test_utils.AssertTrue(t, err == nil, "Expected RevokeAccess to succeed")
	// if key2 was actually cached properly then the next asset will still be decrypted properly
	asset, err := assetIter.Next()
	test_utils.AssertTrue(t, err == nil, "Expected assetIter.Next() to succeed")
	test_utils.AssertTrue(t, !data_model.IsEncryptedData(asset.PrivateData), "Expected asset private data to be decrypted")

	mstub.MockTransactionEnd("t123")
}

func putTestObjectForTest(t *testing.T, mstub *test_utils.NewMockStub, caller data_model.User, id, age, assetNamespace, indexName string, owners []string, assetKey data_model.Key, giveAccessToCaller bool) data_model.Asset {
	assetData := test_utils.CreateTestAsset(GetAssetId(assetNamespace, id))
	assetData.AssetKeyId = assetKey.ID
	assetData.AssetKeyHash = crypto.Hash(assetKey.KeyBytes)
	testPublicDataMap := make(map[string]string)
	testPublicDataMap["objectId"] = id
	owner2 := ""
	if len(owners) > 1 {
		owner2 = owners[1]
	}
	testPublicDataMap["owner2"] = owner2
	testPublicData, _ := json.Marshal(testPublicDataMap)
	testPrivateDataMap := make(map[string]string)
	testPrivateDataMap["age"] = age
	testPrivateData, _ := json.Marshal(testPrivateDataMap)
	assetData.PrivateData = testPrivateData
	assetData.PublicData = testPublicData
	assetData.OwnerIds = owners
	assetData.IndexTableName = indexName

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	am := GetAssetManager(stub, caller)
	err := am.AddAsset(assetData, assetKey, giveAccessToCaller)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed")
	mstub.MockTransactionEnd("t123")
	return assetData
}

func getNumUnencryptedAssets(assetIter asset_manager.AssetIteratorInterface) int {
	count := 0
	totalCount := 0
	for assetIter.HasNext() {
		asset, err := assetIter.Next()
		if err != nil {
			continue
		} else if !data_model.IsEncryptedData(asset.PrivateData) {
			count++
		}
		totalCount++
	}
	logger.Debugf("total count: %v unecypted count: %v", totalCount, count)
	return count
}

// This vehicle struct has a variety of numeric fields
type vehicle struct {
	ID        string  `json:"id"`
	MfrDate   int64   `json:"mfr_date"`
	NumMiles  int64   `json:"num_miles"`
	NumWheels int     `json:"num_wheels"`
	MPG       float64 `json:"mpg"`
	Cost      float64 `json:"cost"`
	Color     string  `json:"color"`
}

const vehicleTableName = "vehicleTable"
const vehicleNamespace = "vehicle"

var compact, truck, dansMiniVan vehicle
var compactAsset, truckAsset, vanAsset data_model.Asset

// Creates 3 vehicles and stores them as assets with indices
func setupVehicleAssets(mstub *test_utils.NewMockStub) data_model.User {

	// Create 3 vehicles
	compact = vehicle{
		ID:        "compact",
		MfrDate:   time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		NumMiles:  10923,
		NumWheels: 4,
		MPG:       32.89432,
		Cost:      19000.99,
		Color:     "blue",
	}
	truck = vehicle{
		ID:        "truck",
		MfrDate:   time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		NumMiles:  30831,
		NumWheels: 18,
		MPG:       12.493,
		Cost:      39999.99,
		Color:     "blue",
	}
	dansMiniVan = vehicle{
		ID:        "dan's mini van",
		MfrDate:   time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		NumMiles:  225000,
		NumWheels: 3,
		MPG:       20.94,
		Cost:      -1599,
		Color:     "green",
	}

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	// Create indices on the vehicle fields
	vehicleTable := index.GetTable(stub, vehicleTableName, "id")
	vehicleTable.AddIndex([]string{"mfr_date", "id"}, false)
	vehicleTable.AddIndex([]string{"num_miles", "id"}, false)
	vehicleTable.AddIndex([]string{"num_wheels", "id"}, false)
	vehicleTable.AddIndex([]string{"mpg", "id"}, false)
	vehicleTable.AddIndex([]string{"cost", "id"}, false)
	vehicleTable.AddIndex([]string{"color", "id"}, false)
	vehicleTable.AddIndex([]string{"color", "mfr_date", "id"}, false)

	vehicleTable.SaveToLedger()
	mstub.MockTransactionEnd("t123")

	// Store the vehicles as assets
	caller := test_utils.CreateTestUser("caller")

	assetKey := test_utils.CreateSymKey("assetKey")
	assetKeyHash := crypto.Hash(assetKey.KeyBytes)

	// Store the compact
	compactAsset = test_utils.CreateTestAsset("")
	compactAsset.IndexTableName = vehicleTableName
	compactAsset.AssetId = GetAssetId(vehicleNamespace, compact.ID)
	compactAsset.PrivateData, _ = json.Marshal(compact)
	compactAsset.AssetKeyId = assetKey.ID
	compactAsset.AssetKeyHash = assetKeyHash
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am := GetAssetManager(stub, caller)
	am.AddAsset(compactAsset, assetKey, true)
	mstub.MockTransactionEnd("t123")
	// Store the truck
	truckAsset = test_utils.CreateTestAsset("")
	truckAsset.IndexTableName = vehicleTableName
	truckAsset.AssetId = GetAssetId(vehicleNamespace, truck.ID)
	truckAsset.PrivateData, _ = json.Marshal(truck)
	truckAsset.AssetKeyId = assetKey.ID
	truckAsset.AssetKeyHash = assetKeyHash
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am = GetAssetManager(stub, caller)
	am.AddAsset(truckAsset, assetKey, true)
	mstub.MockTransactionEnd("t123")
	// Store the van
	vanAsset = test_utils.CreateTestAsset("")
	vanAsset.IndexTableName = vehicleTableName
	vanAsset.AssetId = GetAssetId(vehicleNamespace, dansMiniVan.ID)
	vanAsset.PrivateData, _ = json.Marshal(dansMiniVan)
	vanAsset.AssetKeyId = assetKey.ID
	vanAsset.AssetKeyHash = assetKeyHash
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am = GetAssetManager(stub, caller)
	am.AddAsset(vanAsset, assetKey, true)
	mstub.MockTransactionEnd("t123")

	return caller
}

func TestGetAssets_NumericIndices(t *testing.T) {

	// create a MockStub
	mstub := setup(t)

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)

	// Set up vehicle assets w/ indices
	caller := setupVehicleAssets(mstub)

	// Get asset manager
	am := GetAssetManager(stub, caller)

	// Now query each index and check the results
	// Query by mfr_date
	resultsIter, err := am.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"mfr_date"},
		[]string{},
		[]string{},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssets to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{vanAsset, compactAsset, truckAsset}, resultsIter)

	// Query by num_miles
	resultsIter, err = am.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"num_miles"},
		[]string{},
		[]string{},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssets to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{compactAsset, truckAsset, vanAsset}, resultsIter)

	// Query by num_wheels
	resultsIter, err = am.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"num_wheels"},
		[]string{},
		[]string{},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssets to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{vanAsset, compactAsset, truckAsset}, resultsIter)

	// Query by mpg
	resultsIter, err = am.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"mpg"},
		[]string{},
		[]string{},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssets to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{truckAsset, vanAsset, compactAsset}, resultsIter)

	// Query by mpg
	resultsIter, err = am.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"mpg"},
		[]string{},
		[]string{},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssets to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{truckAsset, vanAsset, compactAsset}, resultsIter)

	// Query by cost
	resultsIter, err = am.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"cost"},
		[]string{},
		[]string{},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssets to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{vanAsset, compactAsset, truckAsset}, resultsIter)

	mstub.MockTransactionEnd("t123")

	// Let's make sure the sorting is right when we have multiple negative values
	truck.Cost = -1598
	truckAsset.PrivateData, _ = json.Marshal(truck)
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am = GetAssetManager(stub, caller)
	//get asset key
	keyPath := []string{caller.GetPubPrivKeyId(), truckAsset.AssetKeyId}
	assetKey, err := am.GetAssetKey(truckAsset.AssetId, keyPath)
	test_utils.AssertTrue(t, err == nil, "Expected to succeed")

	err = am.UpdateAsset(truckAsset, assetKey)
	test_utils.AssertTrue(t, err == nil, "Expected UpdateAsset to succeed")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am = GetAssetManager(stub, caller)
	resultsIter, err = am.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"cost"},
		[]string{},
		[]string{},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssets to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{vanAsset, truckAsset, compactAsset}, resultsIter)
	mstub.MockTransactionEnd("t123")
}

func TestGetAssetIter(t *testing.T) {

	// create a MockStub
	mstub := setup(t)

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	// Set up vehicle assets w/ indices
	caller := setupVehicleAssets(mstub)
	// Get asset manager
	am := GetAssetManager(stub, caller)

	// ----------------------------------------------------
	// Query by color
	// ----------------------------------------------------

	// Test filtering
	// Filter on "blue" cars -> compact, truck
	assetIter, err := am.GetAssetIter(vehicleNamespace, vehicleTableName,
		[]string{"color"},
		[]string{"blue"},
		[]string{"blue"},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{compactAsset, truckAsset}, assetIter)

	// Test open-ended range (startKey only)
	// Range starting with "blue" cars (inclusive) -> compact, truck, van
	assetIter, err = am.GetAssetIter(vehicleNamespace, vehicleTableName,
		[]string{"color"},
		[]string{"blue"},
		[]string{},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{compactAsset, truckAsset, vanAsset}, assetIter)

	// Test open-ended range (endKey only)
	// Range ending with "green" cars (exclusive) -> compact, truck
	assetIter, err = am.GetAssetIter(vehicleNamespace, vehicleTableName,
		[]string{"color"},
		[]string{},
		[]string{"green"},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{compactAsset, truckAsset}, assetIter)

	// Test double-sided range
	// Range starting with "blue" cars (inclusive) and ending with "green" cars (exclusive) -> compact, truck
	assetIter, err = am.GetAssetIter(vehicleNamespace, vehicleTableName,
		[]string{"color"},
		[]string{"blue"},
		[]string{"green"},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{compactAsset, truckAsset}, assetIter)

	// ----------------------------------------------------
	// Query by mfr_date
	// ----------------------------------------------------

	// Test filtering
	// Filter on 2001 -> van
	mfrDateStr, _ := utils.ConvertToString(time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	assetIter, err = am.GetAssetIter(vehicleNamespace, vehicleTableName,
		[]string{"mfr_date"},
		[]string{mfrDateStr},
		[]string{mfrDateStr},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{vanAsset}, assetIter)

	// Test open-ended range (startKey only)
	// Range(1998, "") -> van, compact, truck
	mfrDateStr, _ = utils.ConvertToString(time.Date(1998, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	assetIter, err = am.GetAssetIter(vehicleNamespace, vehicleTableName,
		[]string{"mfr_date"},
		[]string{mfrDateStr},
		[]string{},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{vanAsset, compactAsset, truckAsset}, assetIter)

	// Test open-ended range (endKey only)
	// Range("", 2018) -> van, compact
	mfrDateStr, _ = utils.ConvertToString(time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	assetIter, err = am.GetAssetIter(vehicleNamespace, vehicleTableName,
		[]string{"mfr_date"},
		[]string{},
		[]string{mfrDateStr},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{vanAsset, compactAsset}, assetIter)

	// Test double-sided range
	// Range(2002, 2018)  -> compact
	mfrDateStr, _ = utils.ConvertToString(time.Date(2002, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	mfrDateStrEnd, _ := utils.ConvertToString(time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	assetIter, err = am.GetAssetIter(vehicleNamespace, vehicleTableName,
		[]string{"mfr_date"},
		[]string{mfrDateStr},
		[]string{mfrDateStrEnd},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{compactAsset}, assetIter)

	// ----------------------------------------------------
	// Query by color + mfr_date
	// ----------------------------------------------------

	// Get blue cars from 2002 onward -> compact, truck
	mfrDateStr, _ = utils.ConvertToString(time.Date(2002, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	assetIter, err = am.GetAssetIter(vehicleNamespace, vehicleTableName,
		[]string{"color", "mfr_date"},
		[]string{"blue", mfrDateStr},
		[]string{"blue"},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{compactAsset, truckAsset}, assetIter)

	// Get blue cars from 2002 to 2018 -> compact
	mfrDateStr, _ = utils.ConvertToString(time.Date(2002, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	mfrDateStrEnd, _ = utils.ConvertToString(time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	assetIter, err = am.GetAssetIter(vehicleNamespace, vehicleTableName,
		[]string{"color", "mfr_date"},
		[]string{"blue", mfrDateStr},
		[]string{"blue", mfrDateStrEnd},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{compactAsset}, assetIter)

	// Get green cars from 2015 onward -> none
	mfrDateStr, _ = utils.ConvertToString(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	assetIter, err = am.GetAssetIter(vehicleNamespace, vehicleTableName,
		[]string{"color", "mfr_date"},
		[]string{"green", mfrDateStr},
		[]string{"green"},
		false,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetIter to succeed")
	assertAssetIterListsEqual(t, []data_model.Asset{}, assetIter)

	mstub.MockTransactionEnd("t123")
}

func TestGetAssetIter_publicData(t *testing.T) {

	// create a MockStub
	mstub := setup(t)

	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	// Set up vehicle assets w/ indices
	caller := setupVehicleAssets(mstub)
	// Get asset manager
	am := GetAssetManager(stub, caller)

	// Test no limit (limit = -1)
	assetIter, err := am.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{},
		[]string{},
		[]string{},
		false,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"", 0, nil)
	assetPage, previousKey, err := assetIter.GetAssetPage()
	logger.Debugf("len asssetPage = %v %v", len(assetPage), err)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(assetPage) == 0, "Expected 0 vehicles in the page")

	assetIter, err = am.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{},
		[]string{},
		[]string{},
		false,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"", -2, nil)
	assetPage, previousKey, err = assetIter.GetAssetPage()
	logger.Debugf("len asssetPage = %v %v", len(assetPage), err)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(assetPage) == 0, "Expected 0 vehicles in the page")

	assetIter, err = am.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{},
		[]string{},
		[]string{},
		false,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"", -1, nil)
	assetPage, previousKey, err = assetIter.GetAssetPage()
	logger.Debugf("len asssetPage = %v %v", len(assetPage), err)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(assetPage) == 3, "Expected 3 vehicles in the page")

	// Test with no startKey
	assetIter, err = am.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"num_wheels"},
		[]string{},
		[]string{},
		false,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"", 2, nil)
	assetPage, previousKey, err = assetIter.GetAssetPage()
	logger.Debugf("len asssetPage = %v %v", len(assetPage), err)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(assetPage) == 2, "Expected 2 vehicles in the page")
	test_utils.AssertTrue(t, assetPage[0].AssetId == GetAssetId(vehicleNamespace, dansMiniVan.ID), "Expected van")
	test_utils.AssertTrue(t, assetPage[1].AssetId == GetAssetId(vehicleNamespace, compact.ID), "Expected compact")

	// Now get the next page, starting from previousKey
	assetIter, err = am.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"num_wheels"},
		[]string{},
		[]string{},
		false,
		false,
		[]string{caller.GetPubPrivKeyId()},
		previousKey, 2, nil)
	assetPage, previousKey, err = assetIter.GetAssetPage()
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(assetPage) == 1, "Expected 1 vehicle in the page")
	test_utils.AssertTrue(t, assetPage[0].AssetId == GetAssetId(vehicleNamespace, truck.ID), "Expected truck")

	// Test with startKey
	numWheelsStr, _ := utils.ConvertToString(4)
	assetIter, err = am.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"num_wheels"},
		[]string{numWheelsStr},
		[]string{},
		false,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"", 10, nil)
	assetPage, previousKey, err = assetIter.GetAssetPage()
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(assetPage) == 2, "Expected 2 vehicles in the page")
	test_utils.AssertTrue(t, assetPage[0].AssetId == GetAssetId(vehicleNamespace, compact.ID), "Expected compact")
	test_utils.AssertTrue(t, assetPage[1].AssetId == GetAssetId(vehicleNamespace, truck.ID), "Expected truck")

	mstub.MockTransactionEnd("t123")
}

func TestGetAssetIter_IncludePrivateData(t *testing.T) {

	// create a MockStub
	mstub := setup(t)

	// set up keys
	key1 := test_utils.CreateSymKey("key1")
	key2 := test_utils.CreateSymKey("key2")

	// create index table
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	indexName := "CustomAssetIndex"
	table := index.GetTable(stub, indexName, "objectId")
	err := table.AddIndex([]string{"age", "objectId"}, false)
	err = table.SaveToLedger()
	mstub.MockTransactionEnd("t123")

	caller := test_utils.CreateTestUser("owenrId")

	// set up assets
	assetNamespace := "package.Generic"
	putTestObjectForTest(t, mstub, caller, "uniqueId0", "20", assetNamespace, indexName, []string{caller.ID}, key2, false)
	putTestObjectForTest(t, mstub, caller, "uniqueId1", "21", assetNamespace, indexName, []string{caller.ID}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId2", "22", assetNamespace, indexName, []string{caller.ID}, key2, false)
	putTestObjectForTest(t, mstub, caller, "uniqueId3", "23", assetNamespace, indexName, []string{caller.ID}, key2, false)
	putTestObjectForTest(t, mstub, caller, "uniqueId4", "24", assetNamespace, indexName, []string{caller.ID}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId5", "25", assetNamespace, indexName, []string{caller.ID}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId6", "26", assetNamespace, indexName, []string{caller.ID}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId7", "27", assetNamespace, indexName, []string{caller.ID}, key2, false)
	putTestObjectForTest(t, mstub, caller, "uniqueId8", "28", assetNamespace, indexName, []string{caller.ID}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId9", "29", assetNamespace, indexName, []string{caller.ID}, key2, false)

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am := GetAssetManager(stub, caller)
	// when includePrivateData is false, previousKey will be not empty and the first three assets will be returned in the first page
	assetIter, err := am.GetAssetIter(assetNamespace, indexName, []string{"age"}, []string{"20"}, []string{"30"}, false, false, []string{caller.GetPubPrivKeyId()}, "", 3, nil)
	assetPage, previousKey, err := assetIter.GetAssetPage()
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(previousKey) > 0, "Expected PreviousKey to be not empty")
	test_utils.AssertTrue(t, len(assetPage) == 3, "Expected GetAssetPage to return 3 assets")
	expectedList := []string{assetPage[0].AssetId, assetPage[1].AssetId, assetPage[2].AssetId}
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId0"), expectedList, "Expected asset to be uniqueId0")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId1"), expectedList, "Expected asset to be uniqueId1")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId2"), expectedList, "Expected asset to be uniqueId2")

	// when includePrivateData is true, the assets which cannot be decrypted by the caller will be skipped and the first three than can be
	// decrypted will be returned in the first page
	assetIter, err = am.GetAssetIter(assetNamespace, indexName, []string{"age"}, []string{"20"}, []string{"30"}, true, true, []string{caller.GetPubPrivKeyId()}, "", 3, nil)
	assetPage, previousKey, err = assetIter.GetAssetPage()
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(assetPage) == 3, "Expected GetAssetPage to return 3 assets")
	expectedList = []string{assetPage[0].AssetId, assetPage[1].AssetId, assetPage[2].AssetId}
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId1"), expectedList, "Expected asset to be uniqueId1")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId4"), expectedList, "Expected asset to be uniqueId4")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId5"), expectedList, "Expected asset to be uniqueId5")

	// the second page of 3 will only have 2 assets because only two more can be decrypted by the caller
	assetIter, err = am.GetAssetIter(assetNamespace, indexName, []string{"age"}, []string{"20"}, []string{"30"}, true, true, []string{caller.GetPubPrivKeyId()}, previousKey, 3, nil)
	assetPage, previousKey, err = assetIter.GetAssetPage()
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(assetPage) == 2, "Expected GetAssetPage to return 2 assets")
	expectedList = []string{assetPage[0].AssetId, assetPage[1].AssetId}
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId6"), expectedList, "Expected asset to be uniqueId6")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId8"), expectedList, "Expected asset to be uniqueId8")

	mstub.MockTransactionEnd("t123")
}

func TestGetAssetIter_FilterRule(t *testing.T) {

	// create a MockStub
	mstub := setup(t)
	caller := test_utils.CreateTestUser("owenrId")

	// set up keys
	key1 := test_utils.CreateSymKey("key1")

	// create index table
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	indexName := "CustomAssetIndex"
	table := index.GetTable(stub, indexName, "objectId")
	err := table.AddIndex([]string{"age", "objectId"}, false)
	err = table.SaveToLedger()
	mstub.MockTransactionEnd("t123")

	// set up assets
	assetNamespace := "package.Generic"
	putTestObjectForTest(t, mstub, caller, "uniqueId0", "20", assetNamespace, indexName, []string{caller.ID}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId1", "21", assetNamespace, indexName, []string{caller.ID, "owner2"}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId2", "22", assetNamespace, indexName, []string{caller.ID, "owner2"}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId3", "23", assetNamespace, indexName, []string{caller.ID}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId4", "24", assetNamespace, indexName, []string{caller.ID, "owner2"}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId5", "25", assetNamespace, indexName, []string{caller.ID}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId6", "26", assetNamespace, indexName, []string{caller.ID, "owner2"}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId7", "27", assetNamespace, indexName, []string{caller.ID, "owner2"}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId8", "28", assetNamespace, indexName, []string{caller.ID}, key1, true)
	putTestObjectForTest(t, mstub, caller, "uniqueId9", "29", assetNamespace, indexName, []string{caller.ID}, key1, true)

	// filter against asset level data
	//rule := simple_rule.NewRule(`{"!=" : [ { "var" : "asset_id" }, "AssetIdPrefix-package.Generic-uniqueId1" ]}`)
	rule := simple_rule.NewRule(simple_rule.R("!=",
		simple_rule.R("var", "asset_id"),
		GetAssetId(assetNamespace, "uniqueId1")),
	)

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am := GetAssetManager(stub, caller)
	assetIter, err := am.GetAssetIter(assetNamespace, indexName, []string{"age"}, []string{"20"}, []string{"30"}, true, true, []string{caller.GetPubPrivKeyId()}, "", 3, &rule)
	assetPage, _, err := assetIter.GetAssetPage()
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(assetPage) == 3, "Expected GetAssetPage to return 3 assets")
	expectedList := []string{assetPage[0].AssetId, assetPage[1].AssetId, assetPage[2].AssetId}
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId0"), expectedList, "Expected asset to be uniqueId0")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId2"), expectedList, "Expected asset to be uniqueId2")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId3"), expectedList, "Expected asset to be uniqueId3")

	// filter against solution level public data
	rule = simple_rule.NewRule(simple_rule.R("!=",
		simple_rule.R("var", "public_data.objectId"),
		"uniqueId0",
	))
	assetIter, err = am.GetAssetIter(assetNamespace, indexName, []string{"age"}, []string{"20"}, []string{"30"}, true, true, []string{caller.GetPubPrivKeyId()}, "", 3, &rule)
	assetPage, _, err = assetIter.GetAssetPage()
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(assetPage) == 3, "Expected GetAssetPage to return 3 assets")
	expectedList = []string{assetPage[0].AssetId, assetPage[1].AssetId, assetPage[2].AssetId}
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId1"), expectedList, "Expected asset to be uniqueId1")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId2"), expectedList, "Expected asset to be uniqueId2")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId3"), expectedList, "Expected asset to be uniqueId3")

	// filter against solution level private data
	rule = simple_rule.NewRule(simple_rule.R(">=",
		simple_rule.R("var", "private_data.age"),
		"24",
	))
	assetIter, err = am.GetAssetIter(assetNamespace, indexName, []string{"age"}, []string{"20"}, []string{"30"}, true, true, []string{caller.GetPubPrivKeyId()}, "", 5, &rule)
	assetPage, _, err = assetIter.GetAssetPage()
	logger.Debugf("%v", assetPage)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(assetPage) == 5, "Expected GetAssetPage to return 5 assets")
	expectedList = []string{assetPage[0].AssetId, assetPage[1].AssetId, assetPage[2].AssetId, assetPage[3].AssetId, assetPage[4].AssetId}
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId4"), expectedList, "Expected asset to be uniqueId4")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId5"), expectedList, "Expected asset to be uniqueId5")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId6"), expectedList, "Expected asset to be uniqueId6")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId7"), expectedList, "Expected asset to be uniqueId7")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId8"), expectedList, "Expected asset to be uniqueId8")

	// complex rule involving numeric filters, solution level data, and asset level data
	rule = simple_rule.NewRule(simple_rule.R("and",
		simple_rule.R(">=", simple_rule.R("var", "private_data.age"), "22"),
		simple_rule.R("<", simple_rule.R("var", "private_data.age"), "26"),
		simple_rule.R("==", "owner2", simple_rule.R("var", "public_data.owner2")),
	))

	assetIter, err = am.GetAssetIter(assetNamespace, indexName, []string{"age"}, []string{"20"}, []string{"30"}, true, true, []string{caller.GetPubPrivKeyId()}, "", 5, &rule)
	assetPage, _, err = assetIter.GetAssetPage()
	logger.Debugf("len:%v, err:%v, %v", len(assetPage), err, assetPage)
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(assetPage) == 2, "Expected GetAssetPage to return 2 assets")
	expectedList = []string{assetPage[0].AssetId, assetPage[1].AssetId}
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId2"), expectedList, "Expected asset to be uniqueId2")
	test_utils.AssertInLists(t, GetAssetId(assetNamespace, "uniqueId4"), expectedList, "Expected asset to be uniqueId4")

	// filter that does not fit any assets should return 0 assets
	rule = simple_rule.NewRule(simple_rule.R("in", "owner3", simple_rule.R("var", "owner_ids")))
	assetIter, err = am.GetAssetIter(assetNamespace, indexName, []string{"age"}, []string{"20"}, []string{"30"}, true, true, []string{caller.GetPubPrivKeyId()}, "", 5, &rule)
	assetPage, _, err = assetIter.GetAssetPage()
	test_utils.AssertTrue(t, err == nil, "Expected GetAssetPage to succeed")
	test_utils.AssertTrue(t, len(assetPage) == 0, "Expected GetAssetPage to return 0 assets")
	mstub.MockTransactionEnd("t123")
}

func TestConvertToString(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestConvertToString")

	// bool
	var b bool
	b = false
	output, err := utils.ConvertToString(b)
	test_utils.AssertTrue(t, err == nil, "Expected convertToString to succeed")
	test_utils.AssertTrue(t, output == "false", "Expected convertToString to return a different string")

	// string
	var s string
	s = "hello world"
	output, err = utils.ConvertToString(s)
	test_utils.AssertTrue(t, err == nil, "Expected convertToString to succeed")
	test_utils.AssertTrue(t, output == s, "Expected convertToString to return a different string")

	// int
	var i int
	i = 491034234
	output, err = utils.ConvertToString(i)
	test_utils.AssertTrue(t, err == nil, "Expected convertToString to succeed")
	test_utils.AssertTrue(t, output == "1000491034234.0000", "Expected convertToString to return a different string")

	// int64
	var i64 int64
	i64 = -2347289341
	output, err = utils.ConvertToString(i64)
	test_utils.AssertTrue(t, err == nil, "Expected convertToString to succeed")
	test_utils.AssertTrue(t, output == "0997652710659.0000", "Expected convertToString to return a different string")

	// float64
	var f64 float64
	f64 = 1.2341
	output, err = utils.ConvertToString(f64)
	test_utils.AssertTrue(t, err == nil, "Expected convertToString to succeed")
	test_utils.AssertTrue(t, output == "1000000000001.2341", "Expected convertToString to return a different string")

	// nil
	var n interface{}
	output, err = utils.ConvertToString(n)
	test_utils.AssertTrue(t, err == nil, "Expected convertToString to succeed")
	test_utils.AssertTrue(t, len(output) == 0, "Expected convertToString to return an empty string")
}

// Asserts that two lists of assets are equal
func assertAssetIterListsEqual(t *testing.T, expectedAssetList []data_model.Asset, actualListIter asset_manager.AssetIteratorInterface) {

	idx := 0
	defer actualListIter.Close()
	for actualListIter.HasNext() {
		if len(expectedAssetList) < idx+1 {
			// expectedAssetList is too short
			debug.PrintStack()
			t.Fatalf("Expected asset list was shorter than actual asset list. Index: %v", idx)
		}
		actualAsset, err := actualListIter.Next()
		if err != nil {
			debug.PrintStack()
			t.Fatalf("Error getting actualListIter.Next(): %v", err)
		}
		if actualAsset.AssetId != expectedAssetList[idx].AssetId {
			debug.PrintStack()
			t.Fatalf("Assets not equal. Expected %v, got %v", expectedAssetList[idx].AssetId, actualAsset.AssetId)
		}
		idx++
	}
	if idx != len(expectedAssetList) {
		// expectedAssetList is too long
		debug.PrintStack()
		t.Fatalf("Expected asset list was longer than actual asset list. Expected %v, got %v", len(expectedAssetList), idx)
	}
}

// Tests the ability to add an asset with an owner other than the caller
// Gives the caller no access to the asset.
func TestAddAsset_OwnerOtherThanCaller_NoAccess(t *testing.T) {

	mstub := setup(t)

	caller := test_utils.CreateTestUser("myCaller")
	privateKey1 := caller.PrivateKey
	priv1 := crypto.PrivateKeyToBytes(privateKey1)
	callerKey := data_model.Key{}
	callerKey.ID = caller.GetPubPrivKeyId()
	callerKey.KeyBytes = priv1
	callerKey.Type = global.KEY_TYPE_PRIVATE

	owner := test_utils.CreateTestUser("myOwner")
	privateKey2 := owner.PrivateKey
	priv2 := crypto.PrivateKeyToBytes(privateKey2)
	ownerKey := data_model.Key{}
	ownerKey.ID = owner.GetPubPrivKeyId()
	ownerKey.KeyBytes = priv2
	ownerKey.Type = global.KEY_TYPE_PRIVATE

	assetId := GetAssetId("test", "myAssetId")
	testPublicData := test_utils.CreateTestAssetData("public1")
	testPrivateData := test_utils.CreateTestAssetData("private1")
	asset := test_utils.CreateTestAsset(assetId)
	asset.PrivateData = testPrivateData
	asset.PublicData = testPublicData
	asset.OwnerIds = []string{owner.ID}
	assetKey := test_utils.CreateSymKey("myAssetKey")

	// now give write_only access to caller
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	am := GetAssetManager(stub, owner)
	accessControl := data_model.AccessControl{}
	accessControl.Access = global.ACCESS_WRITE_ONLY
	accessControl.UserId = "myCaller"
	accessControl.AssetId = asset.AssetId
	accessControl.AssetKey = &assetKey
	accessControl.UserKey = &callerKey
	err := am.AddAccessToAsset(accessControl, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccessToAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// Add the asset with an owner other than caller, and don't give access to the caller
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am = GetAssetManager(stub, caller)
	err = am.AddAsset(asset, assetKey, false)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed.")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am = GetAssetManager(stub, caller)
	checkAsset(t, stub, asset.AssetId, testPublicData, testPrivateData, assetKey.ID, assetKey.KeyBytes, asset.OwnerIds)
	checkAssetId(t, stub, asset.AssetId, assetKey.ID)

	// Confirm that caller doesn't have access
	keyPath := []string{caller.GetPubPrivKeyId(), assetKey.ID}
	assetKey2, _ := am.GetAssetKey(assetId, keyPath)
	foundAsset, err := am.GetAsset(assetId, assetKey2)
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed.")
	test_utils.AssertTrue(t, data_model.IsEncryptedData(foundAsset.PrivateData), "PrivateData should be encrypted.")
	mstub.MockTransactionEnd("t123")
}

// Tests the ability to add an asset with an owner other than the caller.
// Gives the caller read-access to the asset.
func TestAddAsset_OwnerOtherThanCaller_WithAccess(t *testing.T) {

	mstub := setup(t)

	caller := test_utils.CreateTestUser("myCaller")
	privateKey1 := caller.PrivateKey
	priv1 := crypto.PrivateKeyToBytes(privateKey1)
	callerKey := data_model.Key{}
	callerKey.ID = caller.GetPubPrivKeyId()
	callerKey.KeyBytes = priv1
	callerKey.Type = global.KEY_TYPE_PRIVATE

	owner := test_utils.CreateTestUser("myOwner")
	privateKey2 := owner.PrivateKey
	priv2 := crypto.PrivateKeyToBytes(privateKey2)
	ownerKey := data_model.Key{}
	ownerKey.ID = owner.GetPubPrivKeyId()
	ownerKey.KeyBytes = priv2
	ownerKey.Type = global.KEY_TYPE_PRIVATE

	assetId := GetAssetId("test", "myAssetId")
	testPublicData := test_utils.CreateTestAssetData("public1")
	testPrivateData := test_utils.CreateTestAssetData("private1")
	asset := test_utils.CreateTestAsset(assetId)
	asset.PrivateData = testPrivateData
	asset.PublicData = testPublicData
	asset.OwnerIds = []string{owner.ID}
	assetKey := test_utils.CreateSymKey("myAssetKey")

	// now give write_only access to caller
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	am := GetAssetManager(stub, owner)
	accessControl := data_model.AccessControl{}
	accessControl.Access = global.ACCESS_WRITE_ONLY
	accessControl.UserId = "myCaller"
	accessControl.AssetId = asset.AssetId
	accessControl.AssetKey = &assetKey
	accessControl.UserKey = &callerKey
	err := am.AddAccessToAsset(accessControl, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccessToAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// Add the asset with an owner other than caller, and give access to the caller
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am = GetAssetManager(stub, caller)
	err = am.AddAsset(asset, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed.")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	checkAsset(t, stub, asset.AssetId, testPublicData, testPrivateData, assetKey.ID, assetKey.KeyBytes, asset.OwnerIds)
	checkAssetId(t, stub, asset.AssetId, assetKey.ID)

	// Confirm that caller has access
	keyPath := []string{caller.GetPubPrivKeyId(), assetKey.ID}
	assetKey2, _ := am.GetAssetKey(assetId, keyPath)
	foundAsset, err := am.GetAsset(assetId, assetKey2)
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed.")
	test_utils.AssertFalse(t, data_model.IsEncryptedData(foundAsset.PrivateData), "PrivateData should NOT be encrypted.")
	mstub.MockTransactionEnd("t123")
}
