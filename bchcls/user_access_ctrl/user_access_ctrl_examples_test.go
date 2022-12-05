/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package user_access_ctrl

import (
	"common/bchcls/asset_mgmt"
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/test_utils"
)

func ExampleGetUserAccessManager() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")

	GetUserAccessManager(stub, caller)
}

func ExampleUserAccessManagerImpl_GetStub() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")
	userAccessManager := GetUserAccessManager(stub, caller)

	userAccessManager.GetStub()
}

func ExampleUserAccessManagerImpl_GetCaller() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")
	userAccessManager := GetUserAccessManager(stub, caller)

	userAccessManager.GetCaller()
}

func ExampleUserAccessManagerImpl_AddAccess() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")
	userAccessManager := GetUserAccessManager(stub, caller)

	accessControl := data_model.AccessControl{
		UserId:  "user1",
		AssetId: asset_mgmt.GetAssetId("data_model.Asset", "asset1"),
		// AssetKey unnecessary, assuming asset1's asset key already exists in the key graph
		Access: ACCESS_WRITE,
	}
	mstub.MockTransactionStart("transaction1")
	userAccessManager.AddAccess(accessControl)
	mstub.MockTransactionEnd("transaction1")
}

func ExampleUserAccessManagerImpl_AddAccessByKey() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")
	assetManager := asset_mgmt.GetAssetManager(stub, caller)
	userAccessManager := GetUserAccessManager(stub, caller)

	user := data_model.User{
		ID:         "user1",
		PrivateKey: test_utils.GeneratePrivateKey(),
		// other data_model.User fields
	}
	userPrivateKey := user.GetPrivateKey()

	assetId := asset_mgmt.GetAssetId("data_model.Asset", "asset1")
	assetKeyID := "asset1KeyId"
	assetKey, _ := assetManager.GetAssetKey(assetId, []string{user.GetPubPrivKeyId(), assetKeyID})

	mstub.MockTransactionStart("transaction1")
	userAccessManager.AddAccessByKey(userPrivateKey, assetKey)
	mstub.MockTransactionEnd("transaction1")
}

func ExampleUserAccessManagerImpl_RemoveAccess() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")
	userAccessManager := GetUserAccessManager(stub, caller)

	accessControl := data_model.AccessControl{
		UserId:  "user1",
		AssetId: asset_mgmt.GetAssetId("data_model.Asset", "asset1"),
		// AssetKey unnecessary, assuming asset1's asset key already exists in the key graph
		Access: ACCESS_WRITE,
	}
	mstub.MockTransactionStart("transaction1")
	userAccessManager.RemoveAccess(accessControl)
	mstub.MockTransactionEnd("transaction1")
}

func ExampleUserAccessManagerImpl_RemoveAccessByKey() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")
	assetManager := asset_mgmt.GetAssetManager(stub, caller)
	userAccessManager := GetUserAccessManager(stub, caller)

	user := data_model.User{
		ID:         "user1",
		PrivateKey: test_utils.GeneratePrivateKey(),
		// other data_model.User fields
	}
	userPrivateKey := user.GetPrivateKey()

	assetId := asset_mgmt.GetAssetId("data_model.Asset", "asset1")
	assetKey, _ := assetManager.GetAssetKey(assetId, []string{})

	mstub.MockTransactionStart("transaction1")
	userAccessManager.RemoveAccessByKey(userPrivateKey.ID, assetKey.ID)
	mstub.MockTransactionEnd("transaction1")
}

func ExampleUserAccessManagerImpl_CheckAccess() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")
	userAccessManager := GetUserAccessManager(stub, caller)

	accessControl := data_model.AccessControl{
		UserId:  "user1",
		AssetId: asset_mgmt.GetAssetId("data_model.Asset", "asset1"),
		// AssetKey not required
		Access: ACCESS_WRITE,
	}

	userAccessManager.CheckAccess(accessControl)
}

func ExampleUserAccessManagerImpl_GetAccessData() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")
	userAccessManager := GetUserAccessManager(stub, caller)

	assetId := asset_mgmt.GetAssetId("data_model.Asset", "asset1")

	userAccessManager.GetAccessData("user1", assetId)
}

func ExampleUserAccessManagerImpl_CheckAccessToKey() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")
	userAccessManager := GetUserAccessManager(stub, caller)

	userAccessManager.SlowCheckAccessToKey("key1")
}

func ExampleUserAccessManagerImpl_GetKey() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")
	userAccessManager := GetUserAccessManager(stub, caller)

	userAccessManager.GetKey("key1", []string{})
}
