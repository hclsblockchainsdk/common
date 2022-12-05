/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package user_access_ctrl_i

import (
	"common/bchcls/asset_mgmt/asset_manager"
	"common/bchcls/cached_stub"
	"common/bchcls/consent_mgmt"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/internal/asset_mgmt_i"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/datastore_i/datastore_c"
	"common/bchcls/internal/datatype_i"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/internal/user_mgmt_i"
	"common/bchcls/test_utils"
	"reflect"
	"time"

	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// Define vars that are needed by all tests

const userId = "myUserId"
const callerId = "callerId"

var assetId = asset_mgmt_i.GetAssetId("data_model.Asset", "assetId")
var caller = data_model.User{}
var user = data_model.User{}
var asset data_model.Asset
var assetManager asset_manager.AssetManager

var callerKey data_model.Key
var key1 data_model.Key

// Call this before each test for stub setup
func setup(t *testing.T) *test_utils.NewMockStub {
	mstub := test_utils.CreateNewMockStub(t)
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user_mgmt_i.Init(stub)
	asset_mgmt_i.Init(stub)
	datatype_i.Init(stub)
	consent_mgmt.Init(stub)
	datastore_c.Init(stub)
	mstub.MockTransactionEnd("t1")
	logger.SetLevel(shim.LogDebug)
	return mstub
}

// TODO: This should really be broken into multiple tests...
func TestAccess(t *testing.T) {
	logger.SetLevel(shim.LogDebug)

	t.Log("Running TestAccess")
	mstub := setup(t)

	//1. create caller
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)

	//create caller
	caller := test_utils.CreateTestUser(callerId)
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt_i.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")

	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	newuser, err := user_mgmt_i.GetUserData(stub, caller, caller.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected getUserData to succeed")
	test_utils.AssertTrue(t, newuser.ID == caller.ID, "Expected getUserData to succeed")
	mstub.MockTransactionEnd("t1")

	//2. create user
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	//create user
	user := test_utils.CreateTestUser(userId)
	userBytes, _ := json.Marshal(&user)
	_, err = user_mgmt_i.RegisterUser(stub, caller, []string{string(userBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")

	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	newuser, err = user_mgmt_i.GetUserData(stub, user, user.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected getUserData to succeed")
	test_utils.AssertTrue(t, newuser.ID == user.ID, "Expected getUserData to succeed")
	mstub.MockTransactionEnd("t1")

	// 3. Add key1 to KeyGraph
	// caller has access to key1
	logger.Debug("3. Add key1 to KeyGraph")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	key1 = data_model.Key{}
	key1.ID = "key1"
	key1.Type = global.KEY_TYPE_SYM
	key1.KeyBytes = test_utils.GenerateSymKey()

	callerKey = data_model.Key{}
	callerKey.ID = caller.GetPubPrivKeyId()
	callerKey.Type = global.KEY_TYPE_PUBLIC
	callerKey.KeyBytes = crypto.PublicKeyToBytes(caller.PublicKey)

	err = key_mgmt_i.AddAccess(stub, callerKey, key1)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t1")

	// 4. add asset to ledger (caller is owner)
	logger.Debug("4. add asset to ledger (caller is owner)")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	datatypes := []string{}
	var testPublicData = test_utils.CreateTestAssetData("public1")
	var testPrivateData = test_utils.CreateTestAssetData("private1")
	metadata := make(map[string]string)

	asset = data_model.Asset{
		AssetId:      assetId,
		AssetKeyId:   key1.ID,
		AssetKeyHash: crypto.Hash(key1.KeyBytes),
		Datatypes:    datatypes,
		PrivateData:  testPrivateData,
		PublicData:   testPublicData,
		OwnerIds:     []string{callerId},
		Metadata:     metadata}

	assetManager = asset_mgmt_i.GetAssetManager(stub, caller)
	err = assetManager.AddAsset(asset, key1, true)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")
	mstub.MockTransactionEnd("t1")

	// 5. check caller has write access
	logger.Debug("5. check caller has write access")
	ac := data_model.AccessControl{}
	ac.UserId = caller.ID
	ac.AssetId = asset.AssetId
	ac.Access = global.ACCESS_WRITE
	ac.AssetKey = &key1

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam := GetUserAccessManager(stub, caller)
	ok, err := uam.CheckAccess(ac)
	logger.Debugf("ok: %v, err: %v", ok, err)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, ok == true, "caller should hav write access")

	// 6. check user don't have read access
	ac = data_model.AccessControl{}
	ac.UserId = user.ID
	ac.AssetId = asset.AssetId
	ac.Access = global.ACCESS_READ
	ac.AssetKey = &key1

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller)
	ok, err = uam.CheckAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, ok == false, "user should not have read access")

	// 7. add read access to user and check read access
	ac = data_model.AccessControl{}
	ac.UserId = user.ID
	ac.AssetId = asset.AssetId
	ac.Access = global.ACCESS_READ
	ac.AssetKey = &key1

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller)
	err = uam.AddAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller)
	ok, err = uam.CheckAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, ok == true, "user should have read access")

	// 8. add write access to user and check  access
	ac = data_model.AccessControl{}
	ac.UserId = user.ID
	ac.AssetId = asset.AssetId
	ac.Access = global.ACCESS_WRITE
	ac.AssetKey = &key1

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller)
	err = uam.AddAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller)
	ok, err = uam.CheckAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	test_utils.AssertTrue(t, ok == true, "user should have write access")

	// read access should succeed
	ac2 := data_model.AccessControl{}
	ac2.UserId = user.ID
	ac2.AssetId = asset.AssetId
	ac2.AssetKey = &key1
	ac2.Access = global.ACCESS_READ

	ok, err = uam.CheckAccess(ac2)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	test_utils.AssertTrue(t, ok == true, "user should have read access")
	mstub.MockTransactionEnd("t1")

	// 9. remove write access to user and check  access
	ac = data_model.AccessControl{}
	ac.UserId = user.ID
	ac.AssetId = asset.AssetId
	ac.Access = global.ACCESS_WRITE
	ac.AssetKey = &key1

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller)
	err = uam.RemoveAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected RemoveAccess to succeed")
	mstub.MockTransactionEnd("t1")

	// write access shoudl fail
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller)
	ok, err = uam.CheckAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	test_utils.AssertTrue(t, ok == false, "user should not have write access")

	// read access stil should succeed
	ac2 = data_model.AccessControl{}
	ac2.UserId = user.ID
	ac2.AssetId = asset.AssetId
	ac2.AssetKey = &key1
	ac2.Access = global.ACCESS_READ

	ok, err = uam.CheckAccess(ac2)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	test_utils.AssertTrue(t, ok == true, "user should have read access")
	mstub.MockTransactionEnd("t1")

	// 10. add write access again to user and check  access
	ac = data_model.AccessControl{}
	ac.UserId = user.ID
	ac.AssetId = asset.AssetId
	ac.Access = global.ACCESS_WRITE
	ac.AssetKey = &key1

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller)
	err = uam.AddAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller)
	ok, err = uam.CheckAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, ok == true, "user should have write access")

	// 11. remove read access to user and check  access
	ac = data_model.AccessControl{}
	ac.UserId = user.ID
	ac.AssetId = asset.AssetId
	ac.Access = global.ACCESS_READ
	ac.AssetKey = &key1

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller)
	err = uam.RemoveAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t1")

	// read access shoudl fail
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller)
	ok, err = uam.CheckAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, ok == false, "user should not have read access")

	// write access  shoud succeed
	ac.Access = global.ACCESS_WRITE
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller)
	ok, err = uam.CheckAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, ok == false, "user should not have write access")

}

func TestCheckAccess_DatatypeConsent(t *testing.T) {
	mstub := setup(t)

	// 0. Setup datatype
	mstub.MockTransactionStart("t1")
	// enable putCache so that multiple transactions can be submitted
	stub := cached_stub.NewCachedStub(mstub, true, true)
	datatype1, err := datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "", true, datatype_i.ROOT_DATATYPE_ID)
	datatype2, err2 := datatype_i.RegisterDatatypeWithParams(stub, "datatype2", "", true, datatype1.GetDatatypeID())
	datatype3, err3 := datatype_i.RegisterDatatypeWithParams(stub, "datatype3", "", true, datatype1.GetDatatypeID())
	datatype4, err4 := datatype_i.RegisterDatatypeWithParams(stub, "datatype4", "", true, datatype2.GetDatatypeID())
	mstub.MockTransactionEnd("t1")

	test_utils.AssertTrue(t, err == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err2 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err3 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err4 == nil, "RegisterDatatypeWithParams should not have returned an error")

	//1. create caller
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	//create caller
	caller := test_utils.CreateTestUser(callerId)
	callerBytes, _ := json.Marshal(&caller)
	_, err = user_mgmt_i.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")

	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	newuser, err := user_mgmt_i.GetUserData(stub, caller, caller.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected getUserData to succeed")
	test_utils.AssertTrue(t, newuser.ID == caller.ID, "Expected getUserData to succeed")
	mstub.MockTransactionEnd("t1")

	// Add datatype symkeys for caller
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype1", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype2", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype3", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype4", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	//2. create user
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	//create user
	user := test_utils.CreateTestUser(userId)
	userBytes, _ := json.Marshal(&user)
	_, err = user_mgmt_i.RegisterUser(stub, caller, []string{string(userBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")

	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	newuser, err = user_mgmt_i.GetUserData(stub, user, user.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected getUserData to succeed")
	test_utils.AssertTrue(t, newuser.ID == user.ID, "Expected getUserData to succeed")
	mstub.MockTransactionEnd("t1")

	// 3. add asset1
	logger.Debug("3. add asset1 to ledger (datatype3, datatype4)")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	var testPublicData = test_utils.CreateTestAssetData("public1")
	var testPrivateData = test_utils.CreateTestAssetData("private1")
	metadata := make(map[string]string)

	key1 := data_model.Key{}
	key1.ID = "key1"
	key1.Type = global.KEY_TYPE_SYM
	key1.KeyBytes = test_utils.GenerateSymKey()

	asset1 := data_model.Asset{
		AssetId:      asset_mgmt_i.GetAssetId("test", "asset1"),
		AssetKeyId:   key1.ID,
		AssetKeyHash: crypto.Hash(key1.KeyBytes),
		Datatypes:    []string{datatype4.GetDatatypeID(), datatype3.GetDatatypeID()},
		PrivateData:  testPrivateData,
		PublicData:   testPublicData,
		OwnerIds:     []string{callerId},
		Metadata:     metadata}

	assetManager = asset_mgmt_i.GetAssetManager(stub, caller)
	err = assetManager.AddAsset(asset1, key1, true)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")
	mstub.MockTransactionEnd("t1")

	// 4. add asset2
	logger.Debug("4. add asset2 to ledger (datatype4)")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	key2 := data_model.Key{}
	key2.ID = "key2"
	key2.Type = global.KEY_TYPE_SYM
	key2.KeyBytes = test_utils.GenerateSymKey()

	asset2 := data_model.Asset{
		AssetId:      asset_mgmt_i.GetAssetId("test", "asset2"),
		AssetKeyId:   key2.ID,
		AssetKeyHash: crypto.Hash(key2.KeyBytes),
		Datatypes:    []string{datatype4.GetDatatypeID()},
		PrivateData:  testPrivateData,
		PublicData:   testPublicData,
		OwnerIds:     []string{callerId},
		Metadata:     metadata}

	assetManager = asset_mgmt_i.GetAssetManager(stub, caller)
	err = assetManager.AddAsset(asset2, key2, true)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")
	mstub.MockTransactionEnd("t1")

	// 5. add asset3
	logger.Debug("5. add asset3 to ledger (dataype2)")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	key3 := data_model.Key{}
	key3.ID = "key3"
	key3.Type = global.KEY_TYPE_SYM
	key3.KeyBytes = test_utils.GenerateSymKey()

	asset3 := data_model.Asset{
		AssetId:      asset_mgmt_i.GetAssetId("test", "asset3"),
		AssetKeyId:   key3.ID,
		AssetKeyHash: crypto.Hash(key3.KeyBytes),
		Datatypes:    []string{datatype2.GetDatatypeID()},
		PrivateData:  testPrivateData,
		PublicData:   testPublicData,
		OwnerIds:     []string{callerId},
		Metadata:     metadata}

	assetManager = asset_mgmt_i.GetAssetManager(stub, caller)
	err = assetManager.AddAsset(asset3, key3, true)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")
	mstub.MockTransactionEnd("t1")

	// 6. add asset4  -- datatype3
	logger.Debug("6. add asset4 to ledger (datatype3)")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	key4 := data_model.Key{}
	key4.ID = "key4"
	key4.Type = global.KEY_TYPE_SYM
	key4.KeyBytes = test_utils.GenerateSymKey()

	asset4 := data_model.Asset{
		AssetId:      asset_mgmt_i.GetAssetId("test", "asset4"),
		AssetKeyId:   key4.ID,
		AssetKeyHash: crypto.Hash(key4.KeyBytes),
		Datatypes:    []string{datatype3.GetDatatypeID()},
		PrivateData:  testPrivateData,
		PublicData:   testPublicData,
		OwnerIds:     []string{callerId},
		Metadata:     metadata}

	assetManager = asset_mgmt_i.GetAssetManager(stub, caller)
	err = assetManager.AddAsset(asset4, key4, true)
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")
	mstub.MockTransactionEnd("t1")

	// 7. Add consent read consent to user1, datatyp4
	logger.Debug("7. Add consent read consent to user1, datatype4")
	consentData := generateConsent(caller.ID, user.ID, global.ACCESS_READ, datatype4.GetDatatypeID())
	// Create consent key
	consentKey := test_utils.GenerateSymKey()
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = consent_mgmt.PutConsentWithParams(stub, caller, consentData, consentKey)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// 8. user has read access to asset4
	logger.Debug("8. user has read access to asset2")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam := GetUserAccessManager(stub, user)
	accessControl := data_model.AccessControl{UserId: user.ID, AssetId: asset2.AssetId, Access: global.ACCESS_READ}

	hasAccess, err := uam.CheckAccess(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed.")
	test_utils.AssertTrue(t, hasAccess, "Expected user to have access to asset.")
	mstub.MockTransactionEnd("t1")

	// 9. user has read access to asset1
	logger.Debug("9. user has read access to asset1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user)
	accessControl = data_model.AccessControl{UserId: user.ID, AssetId: asset1.AssetId, Access: global.ACCESS_READ, AssetKey: &data_model.Key{ID: key1.ID}}

	hasAccess, err = uam.CheckAccess(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed.")
	test_utils.AssertTrue(t, hasAccess, "Expected user to have access to asset.")
	mstub.MockTransactionEnd("t1")

	// 9b. user has read access to asset1
	logger.Debug("9b. user has no write access to asset1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user)
	accessControl = data_model.AccessControl{UserId: user.ID, AssetId: asset1.AssetId, Access: global.ACCESS_WRITE, AssetKey: &data_model.Key{ID: key1.ID}}

	hasAccess, err = uam.CheckAccess(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed.")
	test_utils.AssertTrue(t, !hasAccess, "Expected user to have access to asset.")
	mstub.MockTransactionEnd("t1")

	// 10. user has not read access to asset4
	logger.Debug("10. user has no read access to asset4")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user)
	accessControl = data_model.AccessControl{UserId: user.ID, AssetId: asset4.AssetId, Access: global.ACCESS_READ}

	hasAccess, err = uam.CheckAccess(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed.")
	test_utils.AssertTrue(t, !hasAccess, "Expected user to have access to asset.")
	mstub.MockTransactionEnd("t1")

	// 11. user has not read access to asset4
	logger.Debug("9. user has no read access to asset3")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user)
	accessControl = data_model.AccessControl{UserId: user.ID, AssetId: asset3.AssetId, Access: global.ACCESS_READ}

	hasAccess, err = uam.CheckAccess(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed.")
	test_utils.AssertTrue(t, !hasAccess, "Expected user to have access to asset.")
	mstub.MockTransactionEnd("t1")

	// 7. Add consent write consent to user1, datatype2
	logger.Debug("7. Add consent write consent to user1, datatype2")
	consentData2 := generateConsent(caller.ID, user.ID, global.ACCESS_WRITE, datatype2.GetDatatypeID())
	// Create consent key
	consentKey2 := test_utils.GenerateSymKey()
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = consent_mgmt.PutConsentWithParams(stub, caller, consentData2, consentKey2)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// 8. user has no write access to asset2
	logger.Debug("8. user has no write access to asset2")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user)
	accessControl = data_model.AccessControl{UserId: user.ID, AssetId: asset2.AssetId, Access: global.ACCESS_WRITE}

	hasAccess, err = uam.CheckAccess(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed.")
	test_utils.AssertTrue(t, !hasAccess, "Expected user to have access to asset.")
	mstub.MockTransactionEnd("t1")

	// 9. user has read access to asset1
	logger.Debug("9. user has read access to asset3")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user)
	accessControl = data_model.AccessControl{UserId: user.ID, AssetId: asset3.AssetId, Access: global.ACCESS_READ, AssetKey: &data_model.Key{ID: key3.ID}}

	hasAccess, err = uam.CheckAccess(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed.")
	test_utils.AssertTrue(t, hasAccess, "Expected user to have access to asset.")
	mstub.MockTransactionEnd("t1")

	// 9b. user has read access to asset1
	logger.Debug("9b. user has no write access to asset3")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user)
	accessControl = data_model.AccessControl{UserId: user.ID, AssetId: asset3.AssetId, Access: global.ACCESS_WRITE, AssetKey: &data_model.Key{ID: key3.ID}}

	hasAccess, err = uam.CheckAccess(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed.")
	test_utils.AssertTrue(t, hasAccess, "Expected user to have access to asset.")
	mstub.MockTransactionEnd("t1")

	// 10. user has not read access to asset4
	logger.Debug("10. user has no read access to asset4")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user)
	accessControl = data_model.AccessControl{UserId: user.ID, AssetId: asset4.AssetId, Access: global.ACCESS_READ}

	hasAccess, err = uam.CheckAccess(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed.")
	test_utils.AssertTrue(t, !hasAccess, "Expected user to have access to asset.")
	mstub.MockTransactionEnd("t1")

	// 11. user has not read access to asset4
	logger.Debug("9. user has no read access to asset1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user)
	accessControl = data_model.AccessControl{UserId: user.ID, AssetId: asset1.AssetId, Access: global.ACCESS_READ}

	hasAccess, err = uam.CheckAccess(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed.")
	test_utils.AssertTrue(t, hasAccess, "Expected user to have access to asset.")
	mstub.MockTransactionEnd("t1")

}

// Tests group ADMIN access to an asset which is owned by the group
func TestCheckAccess_groupAdmin(t *testing.T) {

	mstub := setup(t)

	// Create user & group
	user := test_utils.CreateTestUser("user")
	group := test_utils.CreateTestGroup("group")
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user_mgmt_i.RegisterUserWithParams(stub, user, user, false)
	user_mgmt_i.RegisterOrgWithParams(stub, group, group, false)
	mstub.MockTransactionEnd("t1")

	// Add user to group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user_mgmt_i.PutUserInGroup(stub, group, user.ID, group.ID, true) // user is admin of group
	mstub.MockTransactionEnd("t1")

	// Create asset
	asset := test_utils.CreateTestAsset(assetId)
	assetKey := test_utils.CreateSymKey("assetKey")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	asset_mgmt_i.GetAssetManager(stub, group).AddAsset(asset, assetKey, true)
	mstub.MockTransactionEnd("t1")

	// Check that user has write-access to asset
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam := GetUserAccessManager(stub, user)
	accessControl := data_model.AccessControl{UserId: user.ID, AssetId: asset.AssetId, Access: global.ACCESS_WRITE}
	hasAccess, err := uam.CheckAccess(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed.")
	test_utils.AssertTrue(t, hasAccess, "Expected user to have write access to asset.")
	mstub.MockTransactionEnd("t1")
}

// Tests INDIRECT group ADMIN access to an asset which is owned by the group
func TestCheckAccess_indirectGroupAdmin(t *testing.T) {

	mstub := setup(t)

	// Create user, group1, group2
	user := test_utils.CreateTestUser("user")
	group1 := test_utils.CreateTestGroup("group1")
	group2 := test_utils.CreateTestGroup("group2")
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user_mgmt_i.RegisterUserWithParams(stub, user, user, false)
	user_mgmt_i.RegisterOrgWithParams(stub, group1, group1, false)
	mstub.MockTransactionEnd("t1")

	// Register group2 as a subgroup of group1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user_mgmt_i.RegisterSubgroupWithParams(stub, group1, group2, group1.ID)
	mstub.MockTransactionEnd("t1")

	// Add user to group1 as ADMIN - this makes them an indirect admin of group2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user_mgmt_i.PutUserInGroup(stub, group1, user.ID, group1.ID, true) // user is admin of group1
	mstub.MockTransactionEnd("t1")

	// Create asset w/ group2 as owner
	asset := test_utils.CreateTestAsset(assetId)
	assetKey := test_utils.CreateSymKey("assetKey")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	asset_mgmt_i.GetAssetManager(stub, group2).AddAsset(asset, assetKey, true)
	mstub.MockTransactionEnd("t1")

	// Check that user has write-access to asset
	// this will fail since checkAccess only checks direct access
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam := GetUserAccessManager(stub, user)
	accessControl := data_model.AccessControl{UserId: user.ID, AssetId: asset.AssetId, Access: global.ACCESS_WRITE}
	hasAccess, err := uam.CheckAccess(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed.")
	test_utils.AssertTrue(t, !hasAccess, "Expected user to not have write access to asset.")
	mstub.MockTransactionEnd("t1")

	// Check that group1 has write-access to asset
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, group1)
	accessControl = data_model.AccessControl{UserId: group1.ID, AssetId: asset.AssetId, Access: global.ACCESS_WRITE}
	hasAccess, err = uam.CheckAccess(accessControl)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed.")
	test_utils.AssertTrue(t, hasAccess, "Expected user to not have write access to asset.")
	mstub.MockTransactionEnd("t1")
}

func generateConsent(ownerID, targetID, access, datatypeID string) data_model.Consent {
	consent := data_model.Consent{}
	consent.OwnerID = ownerID
	consent.TargetID = targetID
	consent.DatatypeID = datatypeID
	consent.Access = access
	consent.ConsentDate = time.Now().Unix()
	consent.ExpirationDate = consent.ConsentDate + 60*60*24
	consent.Data = make(map[string]interface{})
	return consent
}

func TestRemoveAccessByKey(t *testing.T) {
	logger.SetLevel(shim.LogDebug)

	t.Log("Running TestAccess")
	mstub := setup(t)

	// create caller
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser(callerId)
	user_mgmt_i.RegisterUserWithParams(stub, caller, caller, false)
	mstub.MockTransactionEnd("t1")

	// create user
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user := test_utils.CreateTestUser(userId)
	user_mgmt_i.RegisterUserWithParams(stub, user, user, false)
	mstub.MockTransactionEnd("t1")

	userKey := data_model.Key{}
	userKey.ID = user.GetPubPrivKeyId()
	userKey.Type = global.KEY_TYPE_PUBLIC
	userKey.KeyBytes = crypto.PublicKeyToBytes(user.PublicKey)

	// add asset to ledger
	asset := test_utils.CreateTestAsset(assetId)
	assetKey := test_utils.CreateSymKey("assetKey")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	asset_mgmt_i.GetAssetManager(stub, caller).AddAsset(asset, assetKey, true)
	mstub.MockTransactionEnd("t1")

	// verify that user does not yet have read access
	ac := data_model.AccessControl{}
	ac.UserId = user.ID
	ac.AssetId = asset.AssetId
	ac.Access = global.ACCESS_READ
	ac.AssetKey = &assetKey

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam := GetUserAccessManager(stub, user)
	hasAccess, err := uam.CheckAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	test_utils.AssertTrue(t, hasAccess == false, "user should not have read access")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user)
	err = uam.AddAccessByKey(userKey, assetKey)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccessByKey to succeed")
	mstub.MockTransactionEnd("t1")

	// verify that user now has read access
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user)
	hasAccess, err = uam.CheckAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	test_utils.AssertTrue(t, hasAccess == true, "user should have read access")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user)
	err = uam.RemoveAccessByKey(userKey.ID, assetKey.ID)
	test_utils.AssertTrue(t, err == nil, "Expected RemoveAccessByKey to succeed")
	mstub.MockTransactionEnd("t1")

	// verify that user no longer has read access
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user)
	hasAccess, err = uam.CheckAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	test_utils.AssertTrue(t, hasAccess == false, "user should not have read access")
	mstub.MockTransactionEnd("t1")
}

func TestRemoveAccessAsNonOwner(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	t.Log("Running TestRemoveAccess")
	mstub := setup(t)

	// create caller1
	caller1 := test_utils.CreateTestUser("caller1")
	caller1Bytes, _ := json.Marshal(&caller1)
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	_, err := user_mgmt_i.RegisterUser(stub, caller1, []string{string(caller1Bytes), "false"})
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")

	// create caller2
	caller2 := test_utils.CreateTestUser("caller2")
	caller2Bytes, _ := json.Marshal(&caller2)
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = user_mgmt_i.RegisterUser(stub, caller2, []string{string(caller2Bytes), "false"})
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")

	// create user
	user := test_utils.CreateTestUser(userId)
	userBytes, _ := json.Marshal(&user)
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = user_mgmt_i.RegisterUser(stub, caller1, []string{string(userBytes), "false"})
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")

	// add key1 to KeyGraph
	key1 = data_model.Key{}
	key1.ID = "key1"
	key1.Type = global.KEY_TYPE_SYM
	key1.KeyBytes = test_utils.GenerateSymKey()

	// gives caller1 access to key1
	caller1Key := data_model.Key{}
	caller1Key.ID = caller1.GetPubPrivKeyId()
	caller1Key.Type = global.KEY_TYPE_PUBLIC
	caller1Key.KeyBytes = crypto.PublicKeyToBytes(caller1.PublicKey)
	/*
		mstub.MockTransactionStart("t1")
		stub = cached_stub.NewCachedStub(mstub)
		err = key_mgmt_i.AddAccess(stub, caller1Key, key1)
		err = asset_mgmt_i.GetAssetManager(stub, caller1)
		mstub.MockTransactionEnd("t1")
		test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	*/

	// add asset to ledger as caller1 (asset owner)
	asset = data_model.Asset{
		AssetId:      assetId,
		AssetKeyId:   key1.ID, // key1 is assetKey
		AssetKeyHash: crypto.Hash(key1.KeyBytes),
		Datatypes:    []string{},
		PrivateData:  test_utils.CreateTestAssetData("private1"),
		PublicData:   test_utils.CreateTestAssetData("public1"),
		OwnerIds:     []string{caller1.ID},
		Metadata:     make(map[string]string),
	}
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt_i.GetAssetManager(stub, caller1)
	err = assetManager.AddAsset(asset, key1, true)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expected PutAsset to succeed")

	// add read access to user as caller1 (asset owner)
	ac := data_model.AccessControl{}
	ac.UserId = user.ID
	ac.AssetId = asset.AssetId
	ac.Access = global.ACCESS_READ
	ac.AssetKey = &key1

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam := GetUserAccessManager(stub, caller1)
	err = uam.AddAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller1)
	ok, err := uam.CheckAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	test_utils.AssertTrue(t, ok == true, "user should have read access")
	mstub.MockTransactionEnd("t1")

	// add read access to caller2 as caller1 (asset owner)
	ac.UserId = caller2.ID
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller1)
	err = uam.AddAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller1)
	ok, err = uam.CheckAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	test_utils.AssertTrue(t, ok == true, "user should have read access")
	mstub.MockTransactionEnd("t1")

	// attempt to remove read access from user as caller2 (with read access)
	ac.UserId = user.ID
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller2)
	err = uam.RemoveAccess(ac)
	test_utils.AssertTrue(t, err != nil, "Expected RemoveAccess to fail")
	mstub.MockTransactionEnd("t1")

	// add write access to caller2 as caller1 (asset owner)
	ac.UserId = caller2.ID
	ac.Access = global.ACCESS_WRITE
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller1)
	err = uam.AddAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller1)
	ok, err = uam.CheckAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected CheckAccess to succeed")
	test_utils.AssertTrue(t, ok == true, "user should have read access")
	mstub.MockTransactionEnd("t1")

	// remove read access from user as caller2 (with write access)
	// will fail since only owner of the asset can remove access
	ac.UserId = user.ID
	ac.Access = global.ACCESS_READ
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller2)
	err = uam.RemoveAccess(ac)
	test_utils.AssertTrue(t, err != nil, "Expected RemoveAccess to fail")
	mstub.MockTransactionEnd("t1")

	// remove read access from user as caller1 (owner)
	ac.UserId = user.ID
	ac.Access = global.ACCESS_READ
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, caller1)
	err = uam.RemoveAccess(ac)
	test_utils.AssertTrue(t, err == nil, "Expected RemoveAccess to succeed")
	mstub.MockTransactionEnd("t1")
}

func TestGetKey(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	t.Log("Running TestGetKey")
	mstub := setup(t)

	key1 := test_utils.CreateSymKey("key1")

	// register org
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	org := test_utils.CreateTestGroup("org")
	err := user_mgmt_i.RegisterOrgWithParams(stub, org, org, false)
	test_utils.AssertTrue(t, err == nil, "Expected RegisterOrgWithParams to succeed")
	mstub.MockTransactionEnd("t1")

	orgKey := data_model.Key{}
	orgKey.ID = org.GetPubPrivKeyId()
	orgKey.Type = global.KEY_TYPE_PRIVATE
	orgKey.KeyBytes = crypto.PrivateKeyToBytes(org.PrivateKey)

	// give org access to key1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam := GetUserAccessManager(stub, org)
	err = uam.AddAccessByKey(orgKey, key1)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccessByKey to succeed")
	mstub.MockTransactionEnd("t1")

	// register user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	err = user_mgmt_i.RegisterUserWithParams(stub, user1, user1, false)
	test_utils.AssertTrue(t, err == nil, "Expected RegisterOrgWithParams to succeed")
	mstub.MockTransactionEnd("t1")

	user1Key := data_model.Key{}
	user1Key.ID = user1.GetPubPrivKeyId()
	user1Key.Type = global.KEY_TYPE_PRIVATE
	user1Key.KeyBytes = crypto.PrivateKeyToBytes(user1.PrivateKey)

	// give user1 access to key1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, org)
	err = uam.AddAccessByKey(user1Key, key1)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccessByKey to succeed")
	mstub.MockTransactionEnd("t1")

	// register user2 (no access to key1)
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user2 := test_utils.CreateTestUser("user2")
	err = user_mgmt_i.RegisterUserWithParams(stub, user2, user2, false)
	test_utils.AssertTrue(t, err == nil, "Expected RegisterOrgWithParams to succeed")
	mstub.MockTransactionEnd("t1")

	// GetKey as user1, with access
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user1)
	key1Result, err := uam.GetKey("key1", []string{user1.GetPubPrivKeyId(), "key1"})
	test_utils.AssertTrue(t, err == nil, "Expected GetKey to succeed")
	test_utils.AssertTrue(t, reflect.DeepEqual(key1, key1Result), "Expected key1")
	mstub.MockTransactionEnd("t1")

	// GetKey as user2, no access
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user2)
	key1Result, err = uam.GetKey("key1", []string{user2.GetPubPrivKeyId(), "key1"})
	test_utils.AssertTrue(t, err != nil, "Expected GetKey to fail")
	mstub.MockTransactionEnd("t1")

	// put user2 in org as non-admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = user_mgmt_i.PutUserInGroup(stub, org, user2.ID, org.ID, false)
	test_utils.AssertTrue(t, err == nil, "Expected PutUserInGroup to succeed")
	mstub.MockTransactionEnd("t1")

	// GetKey as user2 without access
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user2)
	key1Result, err = uam.GetKey("key1", []string{user2.GetPubPrivKeyId(), org.GetPrivateKeyHashSymKeyId(), org.GetPubPrivKeyId(), "key1"})
	test_utils.AssertTrue(t, err != nil, "Expected GetKey to fail")
	mstub.MockTransactionEnd("t1")

	// put user2 in org as admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = user_mgmt_i.PutUserInGroup(stub, org, user2.ID, org.ID, true)
	test_utils.AssertTrue(t, err == nil, "Expected PutUserInGroup to succeed")
	mstub.MockTransactionEnd("t1")

	// GetKey as user2 with access
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	uam = GetUserAccessManager(stub, user2)
	key1Result, err = uam.GetKey("key1", []string{user2.GetPubPrivKeyId(), org.GetPrivateKeyHashSymKeyId(), org.GetPubPrivKeyId(), "key1"})
	test_utils.AssertTrue(t, err == nil, "Expected GetKey to succeed")
	test_utils.AssertTrue(t, reflect.DeepEqual(key1, key1Result), "Expected key1")
	mstub.MockTransactionEnd("t1")
}
