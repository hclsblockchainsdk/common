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
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/internal/asset_mgmt_i"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/datastore_i/datastore_c"
	"common/bchcls/internal/datatype_i"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/internal/user_mgmt_i"
	"common/bchcls/test_utils"

	"crypto/rsa"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("asset_mgmt_i_test")

func setup(t *testing.T) *test_utils.NewMockStub {
	logger.SetLevel(shim.LogDebug)
	mstub := test_utils.CreateNewMockStub(t)
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	datastore_c.Init(stub, shim.LogDebug)
	datatype_i.Init(stub, shim.LogDebug)
	asset_mgmt_i.Init(stub, shim.LogDebug)
	user_mgmt_i.Init(stub, shim.LogDebug)
	mstub.MockTransactionEnd("t123")
	return mstub
}

func TestPutAssetByKey_Datatypes(t *testing.T) {
	logger.Info("TestPutAssetByKey_Datatypes function called")

	// create a MockStub
	mstub := setup(t)

	sysadmin := data_model.User{}
	sysadmin.PrivateKey = test_utils.GeneratePrivateKey()
	sysadmin.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(sysadmin.PrivateKey))
	sysadminPub := crypto.PublicKeyToBytes(sysadmin.PrivateKey.Public().(*rsa.PublicKey))
	sysadmin.PublicKeyB64 = crypto.EncodeToB64String(sysadminPub)
	sysadmin.ID = "sysadmin"
	sysadmin.Role = "system"

	// owner's keys
	owner := test_utils.CreateTestUser("ownerId")
	privateKey := owner.PrivateKey
	priv := crypto.PrivateKeyToBytes(privateKey)
	pub := crypto.PublicKeyToBytes(privateKey.Public().(*rsa.PublicKey))

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	// Create caller and add to ledger
	err := user_mgmt_i.RegisterUserWithParams(stub, owner, owner, false)
	test_utils.AssertTrue(t, err == nil, "Register caller user should not have returned an error")
	mstub.MockTransactionEnd("t1")

	// asset key
	assetKey := test_utils.GenerateSymKey()

	// register datatypes
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype2", "datatype2", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t123")

	//add datatype sym keys
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, owner, "datatype1", owner.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	_, err = datatype_i.AddDatatypeSymKey(stub, owner, "datatype2", owner.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	testAsset := data_model.Asset{}
	testAsset.AssetId = "asset1"
	testAsset.Datatypes = []string{"datatype1"}
	testAsset.PublicData = test_utils.CreateTestAssetData("public1")
	testAsset.OwnerIds = []string{"ownerId"}
	testAsset.Metadata = make(map[string]string)

	// create asset without keys
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	err = asset_mgmt_i.PutAssetByKey(mstub, owner, testAsset, "", nil, "", nil, nil)
	test_utils.AssertTrue(t, err != nil, "Expected PutAsset to fail")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	err = asset_mgmt_i.PutAssetByKey(mstub, owner, testAsset, "", nil, "myKey", priv, pub)
	test_utils.AssertTrue(t, err != nil, "Expected PutAsset without asset key to fail")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	err = asset_mgmt_i.PutAssetByKey(mstub, owner, testAsset, "assetKey", assetKey, "", nil, nil)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset without caller's keys to succeed")
	mstub.MockTransactionEnd("t123")

	// create asset with keys
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	err = asset_mgmt_i.PutAssetByKey(mstub, owner, testAsset, "assetKey", assetKey, "myKey", priv, pub)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")
	mstub.MockTransactionEnd("t123")

	testAsset.PublicData = test_utils.CreateTestAssetData("public2")
	// update asset without keys and unchanged datatypes
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	err = asset_mgmt_i.PutAssetByKey(mstub, owner, testAsset, "assetKey", nil, "", nil, nil)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")
	mstub.MockTransactionEnd("t123")

	testAsset.Datatypes = []string{"datatype1", "datatype2"}
	// update asset without keys and changed datatypes
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	err = asset_mgmt_i.PutAssetByKey(mstub, owner, testAsset, "assetKey", nil, "", nil, nil)
	test_utils.AssertTrue(t, err != nil, "Expected PutAsset to fail")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	err = asset_mgmt_i.PutAssetByKey(mstub, owner, testAsset, "", nil, "myKey", priv, pub)
	test_utils.AssertTrue(t, err != nil, "Expected PutAsset to fail")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	err = asset_mgmt_i.PutAssetByKey(mstub, owner, testAsset, "assetKey", assetKey, "", nil, nil)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// update asset with keys
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	testAsset.Datatypes = []string{"datatype1"}
	err = asset_mgmt_i.PutAssetByKey(mstub, owner, testAsset, "assetKey", assetKey, "myKey", priv, pub)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")
	mstub.MockTransactionEnd("t123")
}

func TestPutAssetByKey_InactiveDatatypes(t *testing.T) {
	logger.Info("TestPutAssetByKey_InactiveDatatypes function called")

	// create a MockStub
	mstub := setup(t)

	sysadmin := data_model.User{}
	sysadmin.PrivateKey = test_utils.GeneratePrivateKey()
	sysadmin.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(sysadmin.PrivateKey))
	sysadminPub := crypto.PublicKeyToBytes(sysadmin.PrivateKey.Public().(*rsa.PublicKey))
	sysadmin.PublicKeyB64 = crypto.EncodeToB64String(sysadminPub)
	sysadmin.ID = "sysadmin"
	sysadmin.Role = "system"

	// owner's keys
	owner := test_utils.CreateTestUser("ownerId")
	privateKey := owner.PrivateKey
	priv := crypto.PrivateKeyToBytes(privateKey)
	pub := crypto.PublicKeyToBytes(privateKey.Public().(*rsa.PublicKey))

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	// Create caller and add to ledger
	err := user_mgmt_i.RegisterUserWithParams(stub, owner, owner, false)
	test_utils.AssertTrue(t, err == nil, "Register caller user should not have returned an error")
	mstub.MockTransactionEnd("t1")

	// asset key
	assetKey := test_utils.GenerateSymKey()

	// register datatypes
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t123")

	// register inactive datatypes
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype2", "datatype2", false, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t123")

	//add datatype sym keys
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, owner, "datatype1", owner.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	_, err = datatype_i.AddDatatypeSymKey(stub, owner, "datatype2", owner.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	testAsset := data_model.Asset{}
	testAsset.AssetId = "asset1"
	testAsset.Datatypes = []string{"datatype1"}
	testAsset.PublicData = test_utils.CreateTestAssetData("public1")
	testAsset.OwnerIds = []string{"ownerId"}
	testAsset.Metadata = make(map[string]string)

	// create asset with keys
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	err = asset_mgmt_i.PutAssetByKey(mstub, owner, testAsset, "assetKey", assetKey, "myKey", priv, pub)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// add inactive datatype
	testAsset.Datatypes = []string{"datatype1", "datatype2"}

	// update asset with keys should fail because datatype2 is inactive
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	err = asset_mgmt_i.PutAssetByKey(mstub, owner, testAsset, "assetKey", assetKey, "myKey", priv, pub)
	test_utils.AssertTrue(t, err != nil, "Expected PutAsset to fail")
	mstub.MockTransactionEnd("t123")
}

func TestAddAssetToDatatype(t *testing.T) {
	logger.Info("TestAddAssetToDatatype function called")

	mstub := setup(t)

	caller := test_utils.CreateTestUser("sysadmin")
	caller.Role = "system"

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	// Create caller and add to ledger
	err := user_mgmt_i.RegisterUserWithParams(stub, caller, caller, false)
	test_utils.AssertTrue(t, err == nil, "Register caller user should not have returned an error")
	mstub.MockTransactionEnd("t1")

	assetKey := data_model.Key{ID: "key1", KeyBytes: test_utils.GenerateSymKey(), Type: global.KEY_TYPE_SYM}

	assetData := test_utils.CreateTestAsset(asset_mgmt_i.GetAssetId("data_model.Asset", "asset1"))
	assetData.AssetKeyId = "key1"
	assetData.AssetKeyHash = crypto.Hash(assetKey.KeyBytes)
	assetData.Datatypes = []string{"datatype1", "datatype2"}
	assetData.PublicData = test_utils.CreateTestAssetData("public2")
	assetData.PrivateData = test_utils.CreateTestAssetData("private2")
	assetData.OwnerIds = []string{"sysadmin"}

	// attempt to add asset before registering datatype
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am := asset_mgmt_i.GetAssetManager(stub, caller)
	err = am.AddAsset(assetData, assetKey, true)
	logger.Debugf("Result add asset: %v", err)
	test_utils.AssertTrue(t, err != nil, "Expected AddAsset to fail")
	// Don't commit transaction
	//mstub.MockTransactionEnd("t123")

	// register datatype1
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t123")

	// register datatype2
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype2", "datatype2", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t123")

	// verify no access from datatype key to asset key
	keyPath, err := key_mgmt_i.SlowVerifyAccess(stub, datatype_i.GetDatatypeKeyID("datatype1", caller.ID), assetKey.ID)
	test_utils.AssertTrue(t, err == nil, "VerifyAccess should be successful")
	test_utils.AssertTrue(t, len(keyPath) == 0, "Access from datatype key to asset key should not exist")

	// verify no access from datatype key to asset key
	keyPath, err = key_mgmt_i.SlowVerifyAccess(stub, datatype_i.GetDatatypeKeyID("datatype2", caller.ID), assetKey.ID)
	test_utils.AssertTrue(t, err == nil, "VerifyAccess should be successful")
	test_utils.AssertTrue(t, len(keyPath) == 0, "Access from datatype key to asset key should not exist")

	//add datatype sym keys
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype1", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype2", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// add asset (calls AddAssetToDatatype)
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am = asset_mgmt_i.GetAssetManager(stub, caller)
	err = am.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// verify access from datatype key to asset key
	keyPath, err = key_mgmt_i.SlowVerifyAccess(stub, datatype_i.GetDatatypeKeyID("datatype1", caller.ID), assetKey.ID)
	test_utils.AssertTrue(t, err == nil, "VerifyAccess should be successful")
	test_utils.AssertTrue(t, len(keyPath) > 0, "Access from datatype key to asset key should exist")

	// verify access from datatype key to asset key
	keyPath, err = key_mgmt_i.SlowVerifyAccess(stub, datatype_i.GetDatatypeKeyID("datatype2", caller.ID), assetKey.ID)
	test_utils.AssertTrue(t, err == nil, "VerifyAccess should be successful")
	test_utils.AssertTrue(t, len(keyPath) > 0, "Access from datatype key to asset key should exist")
}

func TestNormalizeAssetDatatypes(t *testing.T) {
	mstub := setup(t)

	caller := test_utils.CreateTestUser("sysadmin")
	caller.Role = "system"
	//pub := crypto.PublicKeyToBytes(caller.PrivateKey.Public().(*rsa.PublicKey))

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	// Create caller and add to ledger
	err := user_mgmt_i.RegisterUserWithParams(stub, caller, caller, false)
	test_utils.AssertTrue(t, err == nil, "Register caller user should not have returned an error")
	mstub.MockTransactionEnd("t1")

	// register datatype1
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t123")

	// register datatype2
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype2", "datatype2", true, "datatype1")
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t123")

	// register datatype3
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype3", "datatype3", true, "datatype2")
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t123")

	//add datatype sym keys; adding datatype symkey for datatype3 should create and link symkeys for its parents
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype3", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	assetKey := data_model.Key{ID: "key1", KeyBytes: test_utils.GenerateSymKey(), Type: global.KEY_TYPE_SYM}
	assetData := data_model.Asset{
		AssetId:      asset_mgmt_i.GetAssetId("data_model.Asset", "asset1"),
		AssetKeyId:   "key1",
		AssetKeyHash: crypto.Hash(assetKey.KeyBytes),
		Datatypes:    []string{"datatype1", "datatype2", "datatype3"},
		PrivateData:  test_utils.CreateTestAssetData("public1"),
		PublicData:   test_utils.CreateTestAssetData("private1"),
		OwnerIds:     []string{"sysadmin"},
		Metadata:     make(map[string]string)}

	// add asset
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am := asset_mgmt_i.GetAssetManager(stub, caller)
	err = am.AddAsset(assetData, assetKey, true)
	test_utils.AssertTrue(t, err == nil, "Expected AddAsset to succeed")
	mstub.MockTransactionEnd("t123")

	// check asset's datatypes
	// adding asset should normalize datatypes.
	// Should only set datatype3 for the asset since datatype1 & datatype2 are datatype3's ancestor
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	am = asset_mgmt_i.GetAssetManager(stub, caller)
	expectedDatatypes1 := []string{"datatype3"}
	asset1, err := am.GetAsset(asset_mgmt_i.GetAssetId("data_model.Asset", "asset1"), assetKey)
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	logger.Debugf("datatypes: expect %v, result %v", expectedDatatypes1, asset1.Datatypes)
	test_utils.AssertSetsEqual(t, expectedDatatypes1, asset1.Datatypes)
	mstub.MockTransactionEnd("t123")
}
