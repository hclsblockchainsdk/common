/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package asset_mgmt

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/key_mgmt"
	"common/bchcls/simple_rule"
	"common/bchcls/test_utils"
	"common/bchcls/utils"

	"encoding/json"
	"fmt"
	"time"
)

func ExampleGetAssetManager() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	GetAssetManager(stub, caller)
}

func ExampleAssetManagerImpl_GetStub() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")
	assetManager := GetAssetManager(stub, caller)

	assetManager.GetStub()
}

func ExampleAssetManagerImpl_GetCaller() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")
	assetManager := GetAssetManager(stub, caller)

	amCaller := assetManager.GetCaller()

	fmt.Println(amCaller.ID)
	// Output: caller1
}

func ExampleAssetManagerImpl_AddAsset() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")

	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = key_mgmt.KEY_TYPE_SYM
	assetKey.KeyBytes = test_utils.GenerateSymKey()

	publicData, _ := json.Marshal(make(map[string]string))
	privateData, _ := json.Marshal(make(map[string]string))
	assetData := data_model.Asset{
		AssetId:        GetAssetId("data_model.Asset", "asset1"),
		Datatypes:      []string{"datatype1"},
		PublicData:     publicData,
		PrivateData:    privateData,
		OwnerIds:       []string{caller.ID},
		Metadata:       make(map[string]string),
		AssetKeyId:     assetKey.ID,
		AssetKeyHash:   crypto.Hash(assetKey.KeyBytes),
		IndexTableName: "CustomAssetIndex",
	}

	mstub.MockTransactionStart("transaction1")
	assetManager := GetAssetManager(stub, caller)
	assetManager.AddAsset(assetData, assetKey, true)
	mstub.MockTransactionEnd("transaction1")
}

func ExampleAssetManagerImpl_UpdateAsset() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")

	mstub.MockTransactionStart("transaction2")
	assetManager := GetAssetManager(stub, caller)

	// assume asset1 exists on the ledger
	assetId := GetAssetId("data_model.Asset", "asset1")
	assetKeyId, _ := GetAssetKeyId(stub, assetId)

	// get assetKey from key path
	keyPath := []string{caller.GetPubPrivKeyId(), assetKeyId}
	assetKey, _ := assetManager.GetAssetKey(assetId, keyPath)

	// get existing asset
	assetData, _ := assetManager.GetAsset(assetId, assetKey)

	// modify asset's public data
	publicDataMap := make(map[string]string)
	json.Unmarshal(assetData.PublicData, &publicDataMap)
	publicDataMap["age"] = "20"
	publicData, _ := json.Marshal(publicDataMap)
	assetData.PublicData = publicData

	// update asset
	assetManager.UpdateAsset(*assetData, assetKey)
	mstub.MockTransactionEnd("transaction2")
}

func ExampleAssetManagerImpl_GetAsset() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")
	assetManager := GetAssetManager(stub, caller)

	// assume asset1 exists on the ledger
	assetId := GetAssetId("data_model.Asset", "asset1")
	assetKeyId, _ := GetAssetKeyId(stub, assetId)

	// get assetKey from key path
	keyPath := []string{caller.GetPubPrivKeyId(), assetKeyId}
	assetKey, _ := assetManager.GetAssetKey(assetId, keyPath)

	assetManager.GetAsset(assetId, assetKey)
}

func ExampleAssetManagerImpl_GetAssetIter() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")
	assetManager := GetAssetManager(stub, caller)

	// convert numeric indices to strings for blue vehicles between 2002 and 2018
	mfrDateStart, _ := utils.ConvertToString(time.Date(2002, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	mfrDateEnd, _ := utils.ConvertToString(time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC).Unix())

	// filter rule to exclude vehicle1
	excludeAssetId := GetAssetId("vehicle", "vehicle1")
	rule := simple_rule.NewRule(simple_rule.R("!=",
		simple_rule.R("var", "asset_id"),
		excludeAssetId),
	)

	// assume color and mfr_date are key fields in vehicleTable
	assetManager.GetAssetIter(
		"vehicle",
		"vehicleTable",
		[]string{"color", "mfr_date"},
		[]string{"blue", mfrDateStart},
		[]string{"blue", mfrDateEnd},
		false,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		10,
		&rule)
}

func ExampleAssetManagerImpl_DeleteAsset() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller1")

	// assume asset1 exists on the ledger
	mstub.MockTransactionStart("transaction1")
	assetManager := GetAssetManager(stub, caller)

	// get assetKey from key path
	assetId := GetAssetId("data_model.Asset", "asset1")
	assetKeyId, _ := GetAssetKeyId(stub, assetId)
	keyPath := []string{caller.GetPubPrivKeyId(), assetKeyId}
	assetKey, _ := assetManager.GetAssetKey(assetId, keyPath)

	// delete asset
	assetManager.DeleteAsset(GetAssetId("data_model.Asset", "asset1"), assetKey)
	mstub.MockTransactionEnd("transaction1")
}

func ExampleAssetManagerImpl_GetAssetKey() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")
	assetManager := GetAssetManager(stub, caller)

	// assume asset1 exists on the ledger
	assetId := GetAssetId("data_model.Asset", "asset1")
	assetKeyId, _ := GetAssetKeyId(stub, assetId)

	// get assetKey from key path
	keyPath := []string{caller.GetPubPrivKeyId(), assetKeyId}
	assetManager.GetAssetKey(assetId, keyPath)
}

func ExampleGetAssetId() {
	assetId := GetAssetId("data_model.Asset", "asset1")

	fmt.Println(assetId)
	// Output: asset_hFOXaLsPZF3BnIi3pgDqC7u7yuLs2WL5RxvJRGm/Vbg=
}

func ExampleGetAssetKeyId() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	// assume asset1 exists in the asset index table
	GetAssetKeyId(stub, GetAssetId("data_model.Asset", "asset1"))
}

func ExampleConvertToString() {
	var b bool
	b = false
	bString, _ := utils.ConvertToString(b)
	fmt.Println(bString)

	var s string
	s = "hello world"
	sString, _ := utils.ConvertToString(s)
	fmt.Println(sString)

	var i int
	i = 491034234
	iString, _ := utils.ConvertToString(i)
	fmt.Println(iString)

	var i64 int64
	i64 = -2347289341
	i64String, _ := utils.ConvertToString(i64)
	fmt.Println(i64String)

	var f64 float64
	f64 = 1.2341
	f64String, _ := utils.ConvertToString(f64)
	fmt.Println(f64String)

	var n interface{}
	nString, _ := utils.ConvertToString(n)
	fmt.Println(nString)

	// Output: false
	// hello world
	// 1000491034234.0000
	// 0997652710659.0000
	// 1000000000001.2341
}
