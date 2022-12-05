/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package bchcls

import (
	"common/bchcls/asset_mgmt"
	"common/bchcls/asset_mgmt/asset_key_func"
	"common/bchcls/cached_stub"
	"common/bchcls/consent_mgmt"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/datatype"
	"common/bchcls/history"
	"common/bchcls/index"
	"common/bchcls/internal/common/global"
	"common/bchcls/key_mgmt"
	"common/bchcls/simple_rule"
	"common/bchcls/test_utils"
	"common/bchcls/user_access_ctrl"
	"common/bchcls/user_mgmt"
	"common/bchcls/user_mgmt/user_groups"
	"common/bchcls/user_mgmt/user_keys"

	"crypto/rsa"
	"encoding/json"
	"strconv"
	"time"
)

func Example_assetMgmtOwnerActions() {
	// In this example, the owner of an asset will
	// create the asset, get the asset with and without private data,
	// update the asset with modified data, and delete the asset.
	mstub := test_utils.CreateExampleMockStub()

	// user owner
	owner := test_utils.CreateTestUser("owner")

	// datatype
	datatype1 := data_model.Datatype{DatatypeID: "datatype1", Description: "datatype1", IsActive: true}
	mstub.MockTransactionStart("transaction01")
	stub := cached_stub.NewCachedStub(mstub)
	datatype1Bytes, _ := json.Marshal(&datatype1)
	args := []string{string(datatype1Bytes)}
	datatype.RegisterDatatype(stub, owner, args)
	mstub.MockTransactionEnd("transaction01")

	// owner adds datatype symkey
	mstub.MockTransactionStart("transaction02")
	stub = cached_stub.NewCachedStub(mstub)
	datatype.AddDatatypeSymKey(stub, owner, datatype1.DatatypeID, owner.ID)
	mstub.MockTransactionEnd("transaction02")

	// asset key
	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = key_mgmt.KEY_TYPE_SYM
	assetKey.KeyBytes = test_utils.GenerateSymKey()

	// asset public data
	publicData := make(map[string]string)
	publicData["data_key1"] = "my public data"
	publicDataBytes, _ := json.Marshal(&publicData)

	// asset private data
	privateData := make(map[string]string)
	privateData["data_key2"] = "my private data"
	privateDataBytes, _ := json.Marshal(&privateData)

	// asset meta data
	metaData := make(map[string]string)
	metaData["nameOfAsset"] = "my asset"

	// create asset data
	assetData := data_model.Asset{
		AssetId:        asset_mgmt.GetAssetId("MyAssetNameSpace", "asset1"),
		Datatypes:      []string{"datatype1"},
		PublicData:     publicDataBytes,
		PrivateData:    privateDataBytes,
		OwnerIds:       []string{owner.ID},
		Metadata:       metaData,
		AssetKeyId:     assetKey.ID,
		AssetKeyHash:   crypto.Hash(assetKey.KeyBytes),
		IndexTableName: "CustomAssetIndex",
	}

	// save asset and give access to owner
	mstub.MockTransactionStart("transaction1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt.GetAssetManager(stub, owner)
	assetManager.AddAsset(assetData, assetKey, true)
	mstub.MockTransactionEnd("transaction1")

	// get asset without retrieving private data
	mstub.MockTransactionStart("transaction2")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	// passing empty key indicates there should be no attempt to get private portion of the asset
	assetManager.GetAsset(assetData.AssetId, data_model.Key{})
	mstub.MockTransactionEnd("transaction2")

	// get asset with private data
	mstub.MockTransactionStart("transaction3")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	assetManager.GetAsset(assetData.AssetId, assetKey)
	mstub.MockTransactionEnd("transaction3")

	// modify asset data
	publicData["data_key2"] = "more public data"
	publicDataBytes, _ = json.Marshal(&publicData)
	assetData.PublicData = publicDataBytes

	// get asset key and update existing asset
	mstub.MockTransactionStart("transaction4")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	keyPath := []string{owner.GetPubPrivKeyId(), assetData.AssetKeyId}
	assetKey, _ = assetManager.GetAssetKey(assetData.AssetId, keyPath)
	assetManager.UpdateAsset(assetData, assetKey)
	mstub.MockTransactionEnd("transaction4")

	// delete existing asset
	mstub.MockTransactionStart("transaction5")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	assetManager.DeleteAsset(assetData.AssetId, assetKey)
	mstub.MockTransactionEnd("transaction5")
}

func Example_assetMgmtAccess() {
	// In this example, the owner of an asset will
	// create the asset, grant read and write access to a user,
	// check the user's access, and remove this access.

	mstub := test_utils.CreateExampleMockStub()

	// owner is a user who is owner of the asset
	owner := test_utils.CreateTestUser("owner")

	// user is a user who will be given access by the owner
	user := test_utils.CreateTestUser("user")

	// datatype
	datatype1 := data_model.Datatype{DatatypeID: "datatype1", Description: "datatype1", IsActive: true}
	mstub.MockTransactionStart("transaction01")
	stub := cached_stub.NewCachedStub(mstub)
	datatype1Bytes, _ := json.Marshal(&datatype1)
	args := []string{string(datatype1Bytes)}
	datatype.RegisterDatatype(stub, owner, args)
	mstub.MockTransactionEnd("transaction01")

	// owner adds datatype symkey
	mstub.MockTransactionStart("transaction02")
	stub = cached_stub.NewCachedStub(mstub)
	datatype.AddDatatypeSymKey(stub, owner, datatype1.DatatypeID, owner.ID)
	mstub.MockTransactionEnd("transaction02")

	// asset key
	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = key_mgmt.KEY_TYPE_SYM
	assetKey.KeyBytes = test_utils.GenerateSymKey()

	// asset public data
	publicData := make(map[string]string)
	publicData["data_key1"] = "my public data"
	publicDataBytes, _ := json.Marshal(&publicData)

	// asset private data
	privateData := make(map[string]string)
	privateData["data_key2"] = "my private data"
	privateDataBytes, _ := json.Marshal(&privateData)

	// asset meta data
	metaData := make(map[string]string)
	metaData["nameOfAsset"] = "my asset"

	// create asset data
	assetData := data_model.Asset{
		AssetId:        asset_mgmt.GetAssetId("MyAssetNameSpace", "asset1"),
		Datatypes:      []string{"datatype1"},
		PublicData:     publicDataBytes,
		PrivateData:    privateDataBytes,
		OwnerIds:       []string{owner.ID},
		Metadata:       metaData,
		AssetKeyId:     assetKey.ID,
		AssetKeyHash:   crypto.Hash(assetKey.KeyBytes),
		IndexTableName: "CustomAssetIndex",
	}

	// save asset
	mstub.MockTransactionStart("transaction1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt.GetAssetManager(stub, owner)
	assetManager.AddAsset(assetData, assetKey, true)
	mstub.MockTransactionEnd("transaction1")

	// owner gives read access to user
	mstub.MockTransactionStart("transaction2")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	ac := data_model.AccessControl{}
	ac.UserId = user.ID
	ac.AssetId = assetData.AssetId
	ac.AssetKey = &assetKey
	ac.Access = user_access_ctrl.ACCESS_READ
	assetManager.AddAccessToAsset(ac)
	mstub.MockTransactionEnd("transaction2")

	// user checks access to asset
	mstub.MockTransactionStart("transaction3")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, user)
	assetManager.CheckAccessToAsset(ac)
	mstub.MockTransactionEnd("transaction3")

	// owner gives write access to user
	mstub.MockTransactionStart("transaction4")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	ac = data_model.AccessControl{}
	ac.UserId = user.ID
	ac.AssetId = assetData.AssetId
	ac.AssetKey = &assetKey
	ac.Access = user_access_ctrl.ACCESS_WRITE
	assetManager.AddAccessToAsset(ac)
	mstub.MockTransactionEnd("transaction4")

	// owner removes write access from user
	mstub.MockTransactionStart("transaction5")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	assetManager.RemoveAccessFromAsset(ac)
	mstub.MockTransactionEnd("transaction5")
}

func Example_assetMgmtGetAssetIter() {
	// In this example, the owner will create three vehicle assets
	// and query vehicles by each index.
	const vehicleTableName = "vehicleTable"
	const vehicleNamespace = "vehicle"

	type vehicle struct {
		ID        string  `json:"id"`
		MfrDate   int64   `json:"mfr_date"`
		NumMiles  int64   `json:"num_miles"`
		NumWheels int     `json:"num_wheels"`
		MPG       float64 `json:"mpg"`
		Cost      float64 `json:"cost"`
		Color     string  `json:"color"`
	}

	compact := vehicle{
		ID:        "compact",
		MfrDate:   time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		NumMiles:  10923,
		NumWheels: 4,
		MPG:       32.89432,
		Cost:      19000.99,
		Color:     "blue",
	}
	truck := vehicle{
		ID:        "truck",
		MfrDate:   time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		NumMiles:  30831,
		NumWheels: 18,
		MPG:       12.493,
		Cost:      39999.99,
		Color:     "blue",
	}
	van := vehicle{
		ID:        "van",
		MfrDate:   time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		NumMiles:  225000,
		NumWheels: 3,
		MPG:       20.94,
		Cost:      -1599,
		Color:     "green",
	}

	mstub := test_utils.CreateExampleMockStub()
	owner := test_utils.CreateTestUser("owner")
	assetKey := test_utils.CreateSymKey("key1")

	// create indices on the vehicle fields
	mstub.MockTransactionStart("transaction1")
	stub := cached_stub.NewCachedStub(mstub)
	vehicleTable := index.GetTable(stub, vehicleTableName, "id")
	vehicleTable.AddIndex([]string{"mfr_date", "id"}, false)
	vehicleTable.AddIndex([]string{"num_miles", "id"}, false)
	vehicleTable.AddIndex([]string{"num_wheels", "id"}, false)
	vehicleTable.AddIndex([]string{"mpg", "id"}, false)
	vehicleTable.AddIndex([]string{"cost", "id"}, false)
	vehicleTable.AddIndex([]string{"color", "id"}, false)
	vehicleTable.AddIndex([]string{"color", "mfr_date", "id"}, false)
	vehicleTable.SaveToLedger()
	mstub.MockTransactionEnd("transaction1")

	// save the compact
	privateDataBytes, _ := json.Marshal(compact)
	compactAsset := data_model.Asset{
		AssetId:        asset_mgmt.GetAssetId(vehicleNamespace, compact.ID),
		PrivateData:    privateDataBytes,
		OwnerIds:       []string{owner.ID},
		AssetKeyId:     assetKey.ID,
		AssetKeyHash:   crypto.Hash(assetKey.KeyBytes),
		IndexTableName: vehicleTableName,
	}
	mstub.MockTransactionStart("transaction2")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt.GetAssetManager(stub, owner)
	assetManager.AddAsset(compactAsset, assetKey, true)
	mstub.MockTransactionEnd("transaction2")

	// save the truck
	privateDataBytes, _ = json.Marshal(truck)
	truckAsset := data_model.Asset{
		AssetId:        asset_mgmt.GetAssetId(vehicleNamespace, truck.ID),
		PrivateData:    privateDataBytes,
		OwnerIds:       []string{owner.ID},
		AssetKeyId:     assetKey.ID,
		AssetKeyHash:   crypto.Hash(assetKey.KeyBytes),
		IndexTableName: vehicleTableName,
	}
	mstub.MockTransactionStart("transaction3")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	assetManager.AddAsset(truckAsset, assetKey, true)
	mstub.MockTransactionEnd("transaction3")

	// save the van
	privateDataBytes, _ = json.Marshal(van)
	vanAsset := data_model.Asset{
		AssetId:        asset_mgmt.GetAssetId(vehicleNamespace, van.ID),
		PrivateData:    privateDataBytes,
		OwnerIds:       []string{owner.ID},
		AssetKeyId:     assetKey.ID,
		AssetKeyHash:   crypto.Hash(assetKey.KeyBytes),
		IndexTableName: vehicleTableName,
	}
	mstub.MockTransactionStart("transaction4")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	assetManager.AddAsset(vanAsset, assetKey, true)
	mstub.MockTransactionEnd("transaction4")

	// key path function
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

	// query by mfr_date
	mstub.MockTransactionStart("transaction5")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	assetIter, _ := assetManager.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"mfr_date"},
		[]string{},
		[]string{},
		true,
		false,
		keyFunc,
		"",
		20,
		nil)
	assetIter.GetAssetPage()
	mstub.MockTransactionEnd("transaction5")

	// query by num_miles
	mstub.MockTransactionStart("transaction6")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	assetIter, _ = assetManager.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"num_miles"},
		[]string{},
		[]string{},
		true,
		false,
		[]string{owner.GetPubPrivKeyId()},
		"",
		20,
		nil)
	assetIter.GetAssetPage()
	mstub.MockTransactionEnd("transaction6")

	// query by num_wheels
	mstub.MockTransactionStart("transaction7")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	assetIter, _ = assetManager.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"num_wheels"},
		[]string{},
		[]string{},
		true,
		false,
		[]string{owner.GetPubPrivKeyId()},
		"",
		20,
		nil)
	assetIter.GetAssetPage()
	mstub.MockTransactionEnd("transaction7")

	// query by mpg
	mstub.MockTransactionStart("transaction8")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	assetIter, _ = assetManager.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"mpg"},
		[]string{},
		[]string{},
		true,
		false,
		[]string{owner.GetPubPrivKeyId()},
		"",
		20,
		nil)
	assetIter.GetAssetPage()
	mstub.MockTransactionEnd("transaction8")

	// query by cost
	mstub.MockTransactionStart("transaction9")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	assetIter, _ = assetManager.GetAssetIter(
		vehicleNamespace,
		vehicleTableName,
		[]string{"cost"},
		[]string{},
		[]string{},
		true,
		false,
		[]string{owner.GetPubPrivKeyId()},
		"",
		20,
		nil)
	assetIter.GetAssetPage()
	mstub.MockTransactionEnd("transaction9")

	// query by color
	mstub.MockTransactionStart("transaction10")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt.GetAssetManager(stub, owner)
	assetIter, _ = assetManager.GetAssetIter(vehicleNamespace, vehicleTableName,
		[]string{"color"},
		[]string{"blue"},
		[]string{"blue"},
		true,
		false,
		[]string{owner.GetPubPrivKeyId()},
		"",
		20,
		nil)
	assetIter.GetAssetPage()
	mstub.MockTransactionEnd("transaction10")
}

func Example_assetMgmtAddAssetByUserWithWriteOnlyAccess() {
	// In this example, the owner of an asset will give
	// write-only access to a user before the asset
	// is saved to the ledger.
	// In the following transaction, the user
	// saves the asset.
	// Note that the asset is needed when the owner gives
	// access to the user, and same asset key must
	// be used to create the asset.
	mstub := test_utils.CreateExampleMockStub()

	// owner is a user who is owner of the asset
	owner := test_utils.CreateTestUser("owner")

	// user is a user who will be given write-only access by the owner
	user := test_utils.CreateTestUser("user")

	// asset key
	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = key_mgmt.KEY_TYPE_SYM
	assetKey.KeyBytes = test_utils.GenerateSymKey()

	// asset public data
	publicData := make(map[string]string)
	publicData["data_key1"] = "my public data"
	publicDataBytes, _ := json.Marshal(&publicData)

	// asset private data
	privateData := make(map[string]string)
	privateData["data_key2"] = "my private data"
	privateDataBytes, _ := json.Marshal(&privateData)

	// asset meta data
	metaData := make(map[string]string)
	metaData["nameOfAsset"] = "my asset"

	// create asset data
	assetData := data_model.Asset{
		AssetId:        asset_mgmt.GetAssetId("MyAssetNameSpace", "asset1"),
		Datatypes:      []string{"datatype1"},
		PublicData:     publicDataBytes,
		PrivateData:    privateDataBytes,
		OwnerIds:       []string{owner.ID},
		Metadata:       metaData,
		AssetKeyId:     assetKey.ID,
		AssetKeyHash:   crypto.Hash(assetKey.KeyBytes),
		IndexTableName: "CustomAssetIndex",
	}

	// first transaction, give write-only access to a user
	// before the asset is created
	mstub.MockTransactionStart("transaction1")
	stub := cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt.GetAssetManager(stub, owner)
	// give access to the user with allowAddAccessBeforeAssetIsCreated
	// option set to true
	ac := data_model.AccessControl{}
	ac.UserId = user.ID
	ac.AssetId = assetData.AssetId
	ac.AssetKey = &assetKey
	ac.Access = user_access_ctrl.ACCESS_WRITE_ONLY
	assetManager.AddAccessToAsset(ac, true)
	mstub.MockTransactionEnd("transaction1")

	// second transaction, users saves the asset
	// without giving itself read access
	mstub.MockTransactionStart("transaction2")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager2 := asset_mgmt.GetAssetManager(stub, user)
	assetManager2.AddAsset(assetData, assetKey, false)
	mstub.MockTransactionEnd("transaction2")

}

func Example_cachedStub() {
	// In this example, the caller saves data to the cache,
	// retrieves cache data, updates cache data,
	// queries cache data by composite key and by range,
	// and deletes cache data.
	mstub := test_utils.CreateExampleMockStub()

	mstub.MockTransactionStart("transaction1")
	stub := cached_stub.NewCachedStub(mstub)
	// put state for composite key query
	for _, k1 := range []string{"tom", "jane", "alex"} {
		for _, k2 := range []string{"1", "2", "3", "4"} {
			key, _ := mstub.CreateCompositeKey("Test", []string{k1, k2})
			val := []byte("value for " + k1 + k2)
			stub.PutState(key, val)
		}
	}
	// put state for range query
	for i := 1; i < 30; i++ {
		key := strconv.Itoa(i)
		val := []byte("value for " + key)
		stub.PutState(key, val)
	}
	mstub.MockTransactionEnd("transaction1")

	mstub.MockTransactionStart("transaction2")
	stub = cached_stub.NewCachedStub(mstub)
	// returns "value for 11"
	stub.GetState("11")

	stub.PutState("11", []byte("new 11 val"))
	// returns "value for 11"
	stub.GetState("11")
	mstub.MockTransactionEnd("transaction2")

	mstub.MockTransactionStart("transaction3")
	stub = cached_stub.NewCachedStub(stub)
	// returns "new value 11"
	stub.GetState("11")
	mstub.MockTransactionEnd("transaction3")

	mstub.MockTransactionStart("transaction4")
	stub = cached_stub.NewCachedStub(stub)
	result := []string{}
	iter, _ := stub.GetStateByRange("18", "22")
	defer iter.Close()
	for iter.HasNext() {
		KV, _ := iter.Next()
		key := KV.GetKey()
		result = append(result, key)
	}
	// result: []string{"18", "19", "2", "20", "21", "22"}
	mstub.MockTransactionEnd("transaction4")

	mstub.MockTransactionStart("transaction5")
	stub = cached_stub.NewCachedStub(stub)
	result = []string{}
	iter, _ = stub.GetStateByPartialCompositeKey("Test", []string{"jane"})
	defer iter.Close()
	for iter.HasNext() {
		KV, _ := iter.Next()
		key := KV.GetKey()
		result = append(result, key)
	}
	// result: 4 composite keys for jane
	mstub.MockTransactionEnd("transaction5")

	mstub.MockTransactionStart("transaction6")
	// delete cache
	stub.DelCache("11")
	// returns nil
	stub.GetCache("11")
	mstub.MockTransactionEnd("transaction6")
}

func Example_consentMgmtConsentToDatatype() {
	// In this example, a datatype owner gives consent to a
	// target user and updates consent permissions. The target user
	// retrieves and validates this consent.
	mstub := test_utils.CreateExampleMockStub()

	// user owner
	owner := test_utils.CreateTestUser("owner")

	// user target
	target := test_utils.CreateTestUser("target")

	// datatype
	datatype1 := data_model.Datatype{DatatypeID: "datatype1", Description: "datatype1", IsActive: true}

	// register datatype
	mstub.MockTransactionStart("transaction01")
	stub := cached_stub.NewCachedStub(mstub)
	datatype1Bytes, _ := json.Marshal(&datatype1)
	args := []string{string(datatype1Bytes)}
	datatype.RegisterDatatype(stub, owner, args)
	mstub.MockTransactionEnd("transaction01")

	// owner adds datatype symkey
	mstub.MockTransactionStart("transaction02")
	stub = cached_stub.NewCachedStub(mstub)
	datatype.AddDatatypeSymKey(stub, owner, datatype1.DatatypeID, owner.ID)
	mstub.MockTransactionEnd("transaction02")

	// asset
	assetData := data_model.Asset{
		AssetId:   asset_mgmt.GetAssetId("MyAssetNameSpace", "asset1"),
		Datatypes: []string{"datatype1"},
		OwnerIds:  []string{owner.ID},
	}

	// create consent object
	consent := data_model.Consent{}
	consent.OwnerID = owner.ID
	consent.TargetID = target.ID
	consent.DatatypeID = datatype1.DatatypeID
	consent.Access = global.ACCESS_WRITE
	consent.ConsentDate = time.Now().Unix()
	consent.ExpirationDate = consent.ConsentDate + 60*60*24
	consent.Data = make(map[string]interface{})
	consentBytes, _ := json.Marshal(&consent)

	// create consent key
	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	// sysadmin gives write consent to target user
	mstub.MockTransactionStart("transaction1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	consent_mgmt.PutConsent(stub, owner, args)
	mstub.MockTransactionEnd("transaction1")

	// update consent permission to deny
	consent.Access = global.ACCESS_DENY
	consentBytes, _ = json.Marshal(&consent)

	mstub.MockTransactionStart("transaction2")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes)}
	consent_mgmt.PutConsent(stub, owner, args)
	mstub.MockTransactionEnd("transaction2")

	// update consent permission to read
	consent.Access = global.ACCESS_READ
	consentBytes, _ = json.Marshal(&consent)

	mstub.MockTransactionStart("transaction3")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes)}
	consent_mgmt.PutConsent(stub, owner, args)
	mstub.MockTransactionEnd("transaction3")

	// get consent as target user
	mstub.MockTransactionStart("transaction4")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{datatype1.DatatypeID, target.ID, owner.ID}
	consentResult := data_model.Consent{}
	consentResultBytes, _ := consent_mgmt.GetConsent(stub, target, args)
	json.Unmarshal(consentResultBytes, &consentResult)
	mstub.MockTransactionEnd("transaction4")

	// validate consent target user
	currTime := time.Now().Unix()
	args = []string{datatype1.DatatypeID, "", owner.ID, target.ID, global.ACCESS_READ, strconv.FormatInt(currTime, 10)}

	mstub.MockTransactionStart("transaction5")
	stub = cached_stub.NewCachedStub(mstub)
	filter, _, _ := consent_mgmt.ValidateConsent(stub, target, args)
	mstub.MockTransactionEnd("transaction5")

	// apply filter rules to check that consent's datatype and owner match asset
	assetDataBytes, _ := json.Marshal(assetData)
	assetDataMap := make(map[string]interface{})
	json.Unmarshal(assetDataBytes, &assetDataMap)
	appliedRuleResult, _ := filter.Apply(assetDataMap)
	logger.Debugf("%v %v", filter.GetExprJSON(), simple_rule.ToJSON(appliedRuleResult))
}

func Example_consentMgmtGetDatatypeConsents() {
	// In this example, consent owners and targets query datatype consents.
	mstub := test_utils.CreateExampleMockStub()

	// user owner
	owner := test_utils.CreateTestUser("owner")

	// user target
	target1 := test_utils.CreateTestUser("target1")
	target2 := test_utils.CreateTestUser("target2")

	// datatype1
	datatype1 := data_model.Datatype{DatatypeID: "datatype1", Description: "datatype1", IsActive: true}
	datatype1Bytes, _ := json.Marshal(&datatype1)

	// register datatype
	mstub.MockTransactionStart("transaction01")
	stub := cached_stub.NewCachedStub(mstub)
	args := []string{string(datatype1Bytes)}
	datatype.RegisterDatatype(stub, owner, args)
	mstub.MockTransactionEnd("transaction01")

	// owner adds datatype symkey
	mstub.MockTransactionStart("transaction02")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(datatype1Bytes)}
	datatype.AddDatatypeSymKey(stub, owner, datatype1.DatatypeID, owner.ID)
	mstub.MockTransactionEnd("transaction02")

	// datatype2
	datatype2 := data_model.Datatype{DatatypeID: "datatype2", Description: "datatype2", IsActive: true}
	datatype2Bytes, _ := json.Marshal(&datatype2)

	// register datatype
	mstub.MockTransactionStart("transaction03")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(datatype2Bytes)}
	datatype.RegisterDatatype(stub, owner, args)
	mstub.MockTransactionEnd("transaction03")

	// owner adds datatype symkey
	mstub.MockTransactionStart("transaction04")
	stub = cached_stub.NewCachedStub(mstub)
	datatype.AddDatatypeSymKey(stub, owner, datatype2.DatatypeID, owner.ID)
	mstub.MockTransactionEnd("transaction04")

	// give write consent to target1 for datatype1
	consent := data_model.Consent{}
	consent.OwnerID = owner.ID
	consent.TargetID = target1.ID
	consent.DatatypeID = datatype1.DatatypeID
	consent.Access = global.ACCESS_WRITE
	consent.ConsentDate = time.Now().Unix()
	consent.ExpirationDate = consent.ConsentDate + 60*60*24
	consent.Data = make(map[string]interface{})
	consentBytes, _ := json.Marshal(&consent)

	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	mstub.MockTransactionStart("transaction1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	consent_mgmt.PutConsent(stub, owner, args)
	mstub.MockTransactionEnd("transaction1")

	// give write consent to target2 for datatype1
	consent.TargetID = target2.ID
	consent.ConsentDate = time.Now().Unix()
	consent.ExpirationDate = consent.ConsentDate + 60*60*24
	consentBytes, _ = json.Marshal(&consent)

	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	mstub.MockTransactionStart("transaction2")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	consent_mgmt.PutConsent(stub, owner, args)
	mstub.MockTransactionEnd("transaction2")

	// give write consent to target1 for datatype2
	consent.TargetID = target1.ID
	consent.DatatypeID = datatype2.DatatypeID
	consent.ConsentDate = time.Now().Unix()
	consent.ExpirationDate = consent.ConsentDate + 60*60*24
	consentBytes, _ = json.Marshal(&consent)

	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	mstub.MockTransactionStart("transaction3")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	consent_mgmt.PutConsent(stub, owner, args)
	mstub.MockTransactionEnd("transaction3")

	// get consents by owner
	mstub.MockTransactionStart("transaction4")
	stub = cached_stub.NewCachedStub(mstub)
	// returns 2 consents
	consents := []data_model.Consent{}
	consentsBytes, _ := consent_mgmt.GetConsentsWithOwnerID(stub, owner, []string{owner.ID})
	json.Unmarshal(consentsBytes, &consents)
	mstub.MockTransactionEnd("transaction4")

	// get consents by target
	mstub.MockTransactionStart("transaction5")
	stub = cached_stub.NewCachedStub(mstub)
	// returns 2 consents
	consents = []data_model.Consent{}
	consentsBytes, _ = consent_mgmt.GetConsentsWithTargetID(stub, target1, []string{target1.ID})
	json.Unmarshal(consentsBytes, &consents)
	mstub.MockTransactionEnd("transaction5")

	// get consents by caller
	mstub.MockTransactionStart("transaction6")
	stub = cached_stub.NewCachedStub(mstub)
	// returns 3 consents
	consents = []data_model.Consent{}
	consentsBytes, _ = consent_mgmt.GetConsentsWithCallerID(stub, owner, []string{})
	json.Unmarshal(consentsBytes, &consents)
	mstub.MockTransactionEnd("transaction6")

	// get consents by owner and datatype
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// returns 2 consents
	consents = []data_model.Consent{}
	consentsBytes, _ = consent_mgmt.GetConsentsWithOwnerIDAndDatatypeID(stub, owner, []string{owner.ID, datatype1.DatatypeID})
	json.Unmarshal(consentsBytes, &consents)
	mstub.MockTransactionEnd("t1")

	// get consents by target and owner
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// returns 1 consent
	consents = []data_model.Consent{}
	consentsBytes, _ = consent_mgmt.GetConsentsWithTargetIDAndOwnerID(stub, owner, []string{target2.ID, owner.ID})
	json.Unmarshal(consentsBytes, &consents)
	mstub.MockTransactionEnd("t1")
}

func Example_cryptoRSAKeys() {
	// In this example, the caller generates an RSA key pair.
	// Each key is marshalled into a byte string, validated,
	// encoded to b64, and parsed. The keys are used to encrypt and
	// decrypt arbitrary data.

	// generate rsa key pair
	privateKey := crypto.GeneratePrivateKey()
	publicKey := privateKey.Public().(*rsa.PublicKey)

	// marshal private key into bytes
	// privateKeyBytes := crypto.MarshalPrivateKey(privateKey)
	privateKeyBytes := crypto.PrivateKeyToBytes(privateKey)
	// validate private key, returns true
	crypto.ValidatePrivateKey(privateKeyBytes)
	// parse private key bytes
	crypto.ParsePrivateKey(privateKeyBytes)
	// encode key to b64 string
	privateKeyB64 := crypto.EncodeToB64String(privateKeyBytes)
	// parse b64 encoded key
	crypto.ParsePrivateKeyB64(privateKeyB64)

	// marshal public key into bytes
	publicKeyBytes := crypto.PublicKeyToBytes(publicKey)
	// validate public key, returns true
	crypto.ValidatePublicKey(publicKeyBytes)
	// parse public key bytes
	crypto.ParsePublicKey(publicKeyBytes)
	// encode key to b64 string
	publicKeyB64 := crypto.EncodeToB64String(publicKeyBytes)
	// parse b64 encoded key
	crypto.ParsePublicKeyB64(publicKeyB64)

	data := []byte("data")
	// encrypt with public key
	encryptedData, _ := crypto.EncryptWithPublicKey(publicKey, data)
	// decrypt with private key
	crypto.DecryptWithPrivateKey(privateKey, encryptedData)
}

func Example_cryptoSymmetricKey() {
	// In this example, the caller generates a random symmetric key
	// and a symmetric key derived from the hash of an input seed.
	// This key is validated, encoded to b64, and parsed.  It is
	// used to encrypt and decrypt arbitrary data.

	// generate sym key
	symKey := crypto.GenerateSymKey()

	// generate sym key from hash of input seed
	symKeyBytes := []byte("seed")
	crypto.GetSymKeyFromHash(symKeyBytes)

	// validate sym key, returns true
	crypto.ValidateSymKey(symKey)

	// encode key to b64 string
	symKeyB64 := crypto.EncodeToB64String(symKey)
	// parse b64 encoded string
	crypto.ParseSymKeyB64(symKeyB64)

	data := []byte("data")
	// encrypt with sym key
	encryptedData, _ := crypto.EncryptWithSymKey(symKey, data)
	// decrypt with sym key
	crypto.DecryptWithSymKey(symKey, encryptedData)
}

func Example_datatype() {
	// In this example, the caller registers datatypes,
	// retrieves datatypes, retrieves datatype keys, adds
	// relationships between datatypes, checks those relationships,
	// and normalizes datatypes.

	mstub := test_utils.CreateExampleMockStub()
	caller := test_utils.CreateTestUser("caller")

	// register datatype1
	mstub.MockTransactionStart("transaction01")
	stub := cached_stub.NewCachedStub(mstub)
	datatype1, _ := datatype.RegisterDatatypeWithParams(stub, "datatyp1", "datatype1 description", true, "")
	mstub.MockTransactionEnd("transaction01")

	// caller adds datatype1 symkey
	mstub.MockTransactionStart("transaction02")
	stub = cached_stub.NewCachedStub(mstub)
	datatype.AddDatatypeSymKey(stub, caller, datatype1.GetDatatypeID(), caller.ID)
	mstub.MockTransactionEnd("transaction02")

	// register datatype2 as a sub-datatype of datatype1
	mstub.MockTransactionStart("transaction03")
	stub = cached_stub.NewCachedStub(mstub)
	datatype2, _ := datatype.RegisterDatatypeWithParams(stub, "datatyp2", "datatype2 description", true, "datatype1")
	mstub.MockTransactionEnd("transaction03")

	// caller adds datatype2 symkey
	mstub.MockTransactionStart("transaction04")
	stub = cached_stub.NewCachedStub(mstub)
	datatype.AddDatatypeSymKey(stub, caller, datatype2.GetDatatypeID(), caller.ID)
	mstub.MockTransactionEnd("transaction04")

	// register datatype3 as a sub-datatype of datatype2
	mstub.MockTransactionStart("transaction05")
	stub = cached_stub.NewCachedStub(mstub)
	datatype3, _ := datatype.RegisterDatatypeWithParams(stub, "datatyp3", "datatype3 description", true, "datatype2")
	mstub.MockTransactionEnd("transaction05")

	// owner adds datatype3 symkey
	mstub.MockTransactionStart("transaction06")
	stub = cached_stub.NewCachedStub(mstub)
	datatype.AddDatatypeSymKey(stub, caller, datatype3.GetDatatypeID(), caller.ID)
	mstub.MockTransactionEnd("transaction06")

	// get datatype data (data_type.Datatype)
	mstub.MockTransactionStart("transaction4")
	stub = cached_stub.NewCachedStub(mstub)
	datatype1Bytes, _ := datatype.GetDatatype(stub, caller, []string{"datatype1"})
	var datatype1Data data_model.Datatype
	json.Unmarshal(datatype1Bytes, &datatype1Data)
	mstub.MockTransactionEnd("transaction4")

	// get datatype
	mstub.MockTransactionStart("transaction4")
	stub = cached_stub.NewCachedStub(mstub)
	datatype1, _ = datatype.GetDatatypeWithParams(stub, "datatype1")
	mstub.MockTransactionEnd("transaction4")

	// get datatypes (list of data_model.Datatype)
	mstub.MockTransactionStart("transaction5")
	stub = cached_stub.NewCachedStub(mstub)
	// returns 2 datatypes
	datatypes := []data_model.Datatype{}
	datatypesBytes, _ := datatype.GetAllDatatypes(stub, caller, []string{})
	json.Unmarshal(datatypesBytes, &datatypes)
	mstub.MockTransactionEnd("transaction5")

	// get datatype1 sym key
	mstub.MockTransactionStart("transaction6")
	stub = cached_stub.NewCachedStub(mstub)
	// returns datatype1's key
	datatype.GetDatatypeSymKey(stub, caller, datatype1.GetDatatypeID(), caller.ID)
	mstub.MockTransactionEnd("transaction6")

	// update datatype
	mstub.MockTransactionStart("transaction7")
	stub = cached_stub.NewCachedStub(mstub)
	// change description
	datatype1.SetDescription("updated description")
	// change state
	datatype1.Activate()
	// save changes to the ledger
	datatype1.PutDatatype(stub)
	mstub.MockTransactionEnd("transaction7")

	// check isParent / isChild
	mstub.MockTransactionStart("transaction10")
	stub = cached_stub.NewCachedStub(mstub)
	// returns true
	datatype1.IsParentOf(stub, datatype2.GetDatatypeID())
	// returns false
	datatype3.IsParentOf(stub, datatype2.GetDatatypeID())
	// returns false
	datatype1.IsChildOf(stub, datatype2.GetDatatypeID())
	// returns true
	datatype3.IsChildOf(stub, "datatyp2")
	mstub.MockTransactionEnd("transaction10")

	// get datatype3's parent datatypes
	mstub.MockTransactionStart("transaction11")
	stub = cached_stub.NewCachedStub(mstub)
	// returns []string{"datatype1", "datatype2"}
	datatype3.GetParentDatatypes(stub)
	mstub.MockTransactionEnd("transaction11")

	// get datatype1's child datatypes
	mstub.MockTransactionStart("transaction12")
	stub = cached_stub.NewCachedStub(mstub)
	// returns []string{"datatype2", "datatype3"}
	datatype1.GetChildDatatypes(stub)
	mstub.MockTransactionEnd("transaction12")

	// normalize datatypes
	mstub.MockTransactionStart("transaction13")
	stub = cached_stub.NewCachedStub(mstub)
	// returns []string{"datatype3"}
	datatype.NormalizeDatatypes(stub, []string{"datatype1", "datatype2", "datatype3"})
	mstub.MockTransactionEnd("transaction13")
}

func Example_history() {
	// In this example, the caller saves an invoke transaction log
	// and a query transaction log and retrieves both.

	mstub := test_utils.CreateExampleMockStub()
	caller := test_utils.CreateTestUser("caller")

	// put invoke transaction log
	mstub.MockTransactionStart("transaction1")
	stub := cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt.GetAssetManager(stub, caller)
	historyManager := history.GetHistoryManager(assetManager)
	txTimestamp, _ := stub.GetTxTimestamp()
	invokeTransactionLog := data_model.TransactionLog{
		TransactionID: stub.GetTxID(),
		Namespace:     "namespace",
		FunctionName:  "invoke_function1",
		CallerID:      "caller",
		Timestamp:     txTimestamp.GetSeconds(),
		Data:          "any arbitrary data object",
		Field1:        "abc",
	}
	encryptionKey := caller.GetSymKey()
	historyManager.PutInvokeTransactionLog(invokeTransactionLog, encryptionKey)
	mstub.MockTransactionEnd("transaction1")

	// generate exportable transaction log
	// since query functions do not invoke the ledger, query transaction logs must first be exported
	// and then sent back in a separate invoke
	mstub.MockTransactionStart("transaction2")
	stub = cached_stub.NewCachedStub(mstub)
	txTimestamp, _ = stub.GetTxTimestamp()
	queryTransactionLog := data_model.TransactionLog{
		TransactionID: stub.GetTxID(),
		Namespace:     "namespace",
		FunctionName:  "query_function1",
		CallerID:      "caller",
		Timestamp:     txTimestamp.GetSeconds(),
		Data:          "any arbitrary data object",
		Field1:        "abc",
	}
	exportableLog, _ := history.GenerateExportableTransactionLog(stub, caller, queryTransactionLog, caller.GetSymKey())
	mstub.MockTransactionEnd("transaction2")

	// put query transaction log
	mstub.MockTransactionStart("transaction3")
	stub = cached_stub.NewCachedStub(mstub)
	exportableLogBytes, _ := json.Marshal(&exportableLog)
	history.PutQueryTransactionLog(stub, caller, []string{string(exportableLogBytes)})
	mstub.MockTransactionEnd("transaction3")

	// get transaction log
	mstub.MockTransactionStart("transaction4")
	stub = cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	assetManager = asset_mgmt.GetAssetManager(stub, caller)
	historyManager = history.GetHistoryManager(assetManager)
	logKey := caller.GetLogSymKey()
	historyManager.GetTransactionLog(invokeTransactionLog.TransactionID, logKey)
	mstub.MockTransactionEnd("transaction4")

	// get transaction logs
	mstub.MockTransactionStart("transaction5")
	stub = cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	assetManager = asset_mgmt.GetAssetManager(stub, caller)
	historyManager = history.GetHistoryManager(assetManager)
	rule := simple_rule.NewRule(
		simple_rule.R("==",
			simple_rule.R("var", "private_data.data.doctor_id"),
			"doc1"))
	logKeyId := caller.GetLogSymKeyId()
	historyManager.GetTransactionLogs("namespace", "field_1", "abc", 1000000002, -1, "", 10, &rule, logKeyId)
	mstub.MockTransactionEnd("transaction5")
}

func Example_index() {
	// In this example, the caller creates an index table,
	// saves an index, checks index table attributes, adds
	// entries (rows) to the table, retrieves an individual entry,
	// and queries by both partial key and range key.
	mstub := test_utils.CreateExampleMockStub()

	// create table and index
	mstub.MockTransactionStart("transaction1")
	stub := cached_stub.NewCachedStub(mstub)
	table := index.GetTable(stub, "TestIndex", "objectId")
	table.AddIndex([]string{"color", "objectId"}, false)
	table.SaveToLedger()
	mstub.MockTransactionEnd("transaction1")

	// check table attributes
	mstub.MockTransactionStart("transaction2")
	stub = cached_stub.NewCachedStub(mstub)
	table = index.GetTable(stub, "TestIndex", "objectId")
	// returns true
	table.HasIndex([]string{"color", "objectId"})
	// returns "color" and "objectId"
	table.GetIndexedFields()
	// returns "objectId"
	table.GetPrimaryKeyId()
	mstub.MockTransactionEnd("transaction2")

	// add table entries (rows)
	mstub.MockTransactionStart("transaction2")
	stub = cached_stub.NewCachedStub(mstub)
	table = index.GetTable(stub, "TestIndex", "objectId")

	row := make(map[string]string)
	row["color"] = "blue"
	row["objectId"] = "object1"
	table.UpdateRow(row)

	row["color"] = "blue"
	row["objectId"] = "object2"
	table.UpdateRow(row)

	row["color"] = "green"
	row["objectId"] = "object3"
	table.UpdateRow(row)

	row["color"] = "red"
	row["objectId"] = "object4"
	table.UpdateRow(row)
	mstub.MockTransactionEnd("transaction2")

	mstub.MockTransactionStart("transaction3")
	stub = cached_stub.NewCachedStub(mstub)
	table = index.GetTable(stub, "TestIndex", "objectId")
	table.UpdateAllRows()
	mstub.MockTransactionEnd("transaction3")

	// get table entry (row)
	mstub.MockTransactionStart("transaction4")
	stub = cached_stub.NewCachedStub(mstub)
	table = index.GetTable(stub, "TestIndex", "objectId")
	rowBytes, _ := table.GetRow("object1")
	json.Unmarshal(rowBytes, &row)
	mstub.MockTransactionEnd("transaction4")

	// query table by color
	mstub.MockTransactionStart("transaction5")
	stub = cached_stub.NewCachedStub(mstub)
	table = index.GetTable(stub, "TestIndex", "objectId")

	iter, _ := table.GetRowsByPartialKey([]string{"color"}, []string{"blue"})
	defer iter.Close()
	for iter.HasNext() {
		// retrieves object1 and object2
		KV, _ := iter.Next()
		rowBytes := KV.GetValue()
		json.Unmarshal(rowBytes, &row)
	}
	mstub.MockTransactionEnd("transaction5")

	// query table by color range
	mstub.MockTransactionStart("transaction6")
	stub = cached_stub.NewCachedStub(mstub)
	table = index.GetTable(stub, "TestIndex", "objectId")

	// filter on "blue"
	startKey, _ := table.CreateRangeKey([]string{"color", "id"}, []string{"blue"})
	endKey, _ := table.CreateRangeKey([]string{"color", "id"}, []string{"blue"})
	endKey = endKey + string(global.MAX_UNICODE_RUNE_VALUE)
	table.GetRowsByRange(startKey, endKey)

	// range starting with "blue"
	startKey, _ = table.CreateRangeKey([]string{"color", "id"}, []string{"blue"})
	endKey, _ = table.CreateRangeKey([]string{"color", "id"}, []string{""})
	endKey = endKey + string(global.MAX_UNICODE_RUNE_VALUE)
	table.GetRowsByRange(startKey, endKey)

	// range starting with "green" and ending with "red" (exclusive)
	startKey, _ = table.CreateRangeKey([]string{"color", "id"}, []string{"green"})
	endKey, _ = table.CreateRangeKey([]string{"color", "id"}, []string{"red"})
	table.GetRowsByRange(startKey, endKey)

	// range ending with "red" (exclusive)
	startKey, _ = table.CreateRangeKey([]string{"color", "id"}, []string{})
	endKey, _ = table.CreateRangeKey([]string{"color", "id"}, []string{"red"})
	table.GetRowsByRange(startKey, endKey)
	mstub.MockTransactionEnd("transaction6")

	// delete table entry (row)
	mstub.MockTransactionStart("transaction7")
	stub = cached_stub.NewCachedStub(mstub)
	table = index.GetTable(stub, "TestIndex", "objectId")
	table.DeleteRow("object3")
	mstub.MockTransactionEnd("transaction7")
}

func Example_keyMgmt() {
	// In this example, the caller retrieves properly formatted
	// key ids for a public/private key, a sym key, a log sym key, and
	// a private hash key. The caller checks to see if a sym key exists
	// and retrieves a private key using a key path.
	mstub := test_utils.CreateExampleMockStub()

	// get key ids
	mstub.MockTransactionStart("transaction1")
	keyId := "key1"
	key_mgmt.GetPubPrivKeyId(keyId)
	symKeyId := key_mgmt.GetSymKeyId(keyId)
	key_mgmt.GetLogSymKeyId(keyId)
	key_mgmt.GetPrivateKeyHashSymKeyId(keyId)
	mstub.MockTransactionEnd("transaction1")

	// check if key exists
	mstub.MockTransactionStart("transaction2")
	stub := cached_stub.NewCachedStub(mstub)
	key_mgmt.KeyExists(stub, symKeyId)
	mstub.MockTransactionEnd("transaction2")

	// get key
	mstub.MockTransactionStart("transaction3")
	privateStartKey := test_utils.GeneratePrivateKey()
	keyBytes := crypto.PrivateKeyToBytes(privateStartKey)
	listKeys := []string{"key1", "key2", "key5", "key6"}
	key_mgmt.GetKey(stub, listKeys, keyBytes)
	mstub.MockTransactionEnd("transaction3")
}

func Example_simpleRule() {
	// In this example, the caller creates an identical rule in three ways,
	// retrieves the rule expression, creates a rule using initial data,
	// retrieves the initial data, and applies a rule to arbitrary data.

	// create rule: {map[+:[1 2.4 6]] map[]}
	rule := simple_rule.NewRule(`{"+": [1, 2.4, 6]}`)

	e := make(map[string]interface{})
	e["+"] = []interface{}{1, 2.4, 6}
	rule = simple_rule.NewRule(e)

	rule = simple_rule.NewRule(simple_rule.R("+", 1, 2.4, 6))

	// expressions
	rule.GetExpr()
	rule.GetExprJSON()

	// init data
	init_data := simple_rule.M(simple_rule.D("my list", simple_rule.D(1, 2, 3, 4.0, "my val", 1, "test")))
	rule = simple_rule.NewRule(simple_rule.R("in", "my val", simple_rule.R("var", "my list")), init_data)
	rule.GetInitJSON()

	// apply rule
	data := `{
		"name": {"last": "Smith", "first": "Jo"},
		"age": 45
	  }`
	rule = simple_rule.NewRule(simple_rule.R("var", "age"))
	resultMap, _ := rule.Apply(data)
	simple_rule.ToJSON(resultMap)
}

func Example_userAccessCtrl() {
	// In this example, an asset owner gives and removes access to another user
	// with both an access control object and with keys. The caller also checks
	// this user's access. The user retrieves an asset key it has access to using
	// a key path.

	mstub := test_utils.CreateExampleMockStub()

	owner := test_utils.CreateTestUser("owner")

	user := test_utils.CreateTestUser("user")
	userKey := data_model.Key{}
	userKey.ID = user.GetPubPrivKeyId()
	userKey.Type = global.KEY_TYPE_PUBLIC
	userKey.KeyBytes = crypto.PublicKeyToBytes(user.PublicKey)

	// caller saves asset
	mstub.MockTransactionStart("transaction1")
	stub := cached_stub.NewCachedStub(mstub)
	assetKey := data_model.Key{}
	assetKey.ID = "key1"
	assetKey.Type = key_mgmt.KEY_TYPE_SYM
	assetKey.KeyBytes = test_utils.GenerateSymKey()
	asset := data_model.Asset{
		AssetId:      "asset1",
		AssetKeyId:   assetKey.ID,
		AssetKeyHash: crypto.Hash(assetKey.KeyBytes),
		OwnerIds:     []string{owner.ID},
	}
	assetManager := asset_mgmt.GetAssetManager(stub, owner)
	assetManager.AddAsset(asset, assetKey, true)
	mstub.MockTransactionEnd("transaction1")

	// give read access to user
	mstub.MockTransactionStart("transaction2")
	ac := data_model.AccessControl{}
	ac.UserId = user.ID
	ac.AssetId = asset.AssetId
	ac.Access = global.ACCESS_READ
	ac.AssetKey = &assetKey

	stub = cached_stub.NewCachedStub(mstub)
	uam := user_access_ctrl.GetUserAccessManager(stub, owner)
	uam.AddAccess(ac)
	mstub.MockTransactionEnd("transaction2")

	// get access data
	mstub.MockTransactionStart("transaction3")
	stub = cached_stub.NewCachedStub(mstub)
	uam = user_access_ctrl.GetUserAccessManager(stub, owner)
	uam.GetAccessData(user.ID, asset.AssetId)
	mstub.MockTransactionEnd("transaction3")

	// check user's access
	mstub.MockTransactionStart("transaction4")
	stub = cached_stub.NewCachedStub(mstub)
	uam = user_access_ctrl.GetUserAccessManager(stub, owner)
	uam.CheckAccess(ac)
	mstub.MockTransactionEnd("transaction4")

	// remove user's read access
	mstub.MockTransactionStart("transaction5")
	stub = cached_stub.NewCachedStub(mstub)
	uam = user_access_ctrl.GetUserAccessManager(stub, owner)
	uam.RemoveAccess(ac)
	mstub.MockTransactionEnd("transaction5")

	// give read access to user by key
	mstub.MockTransactionStart("transaction6")
	stub = cached_stub.NewCachedStub(mstub)
	uam = user_access_ctrl.GetUserAccessManager(stub, owner)
	uam.AddAccessByKey(userKey, assetKey)
	mstub.MockTransactionEnd("transaction6")

	// get key as user
	mstub.MockTransactionStart("transaction7")
	stub = cached_stub.NewCachedStub(mstub)
	uam = user_access_ctrl.GetUserAccessManager(stub, user)
	uam.GetKey(assetKey.ID, []string{user.GetPubPrivKeyId(), assetKey.ID})
	mstub.MockTransactionEnd("transaction7")

	// remove user's read access by key
	mstub.MockTransactionStart("transaction8")
	stub = cached_stub.NewCachedStub(mstub)
	uam = user_access_ctrl.GetUserAccessManager(stub, owner)
	uam.RemoveAccessByKey(userKey.ID, assetKey.ID)
	mstub.MockTransactionEnd("transaction8")
}

func Example_userMgmt() {
	// In this example, a sysadmin registers orgs and users.
	// User data is retrieved and updated. An org is assigned an admin
	// and members, and group membership (direct and indirect) is checked.
	// User permissions within a group are assigned, checked, and removed.
	// Subgroups are created, and group relationships are checked and updated.
	mstub := test_utils.CreateExampleMockStub()

	// register caller
	mstub.MockTransactionStart("transaction1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	user_mgmt.RegisterSystemAdmin(stub, caller, []string{string(callerBytes), "false"})
	mstub.MockTransactionEnd("transaction1")

	// register org
	mstub.MockTransactionStart("transaction2")
	stub = cached_stub.NewCachedStub(mstub)
	org1 := test_utils.CreateTestGroup("org1")
	org1Bytes, _ := json.Marshal(&org1)
	args := []string{string(org1Bytes), "true"}
	user_mgmt.RegisterOrg(stub, caller, args)
	mstub.MockTransactionEnd("transaction2")

	// register users
	mstub.MockTransactionStart("transaction3")
	stub = cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	user1Bytes, _ := json.Marshal(&user1)
	user_mgmt.RegisterUser(stub, caller, []string{string(user1Bytes), "true"})
	mstub.MockTransactionEnd("transaction3")

	mstub.MockTransactionStart("transaction4")
	stub = cached_stub.NewCachedStub(mstub)
	user2 := test_utils.CreateTestUser("user2")
	user2Bytes, _ := json.Marshal(&user2)
	user_mgmt.RegisterUser(stub, caller, []string{string(user2Bytes), "false"})
	mstub.MockTransactionEnd("transaction4")

	mstub.MockTransactionStart("transaction5")
	stub = cached_stub.NewCachedStub(mstub)
	user3 := test_utils.CreateTestUser("user3")
	user3Bytes, _ := json.Marshal(&user3)
	user_mgmt.RegisterUser(stub, caller, []string{string(user3Bytes), "false"})
	mstub.MockTransactionEnd("transaction5")

	mstub.MockTransactionStart("transaction6")
	stub = cached_stub.NewCachedStub(mstub)
	auditor := test_utils.CreateTestUser("auditor")
	auditor.Role = global.ROLE_AUDIT
	auditorBytes, _ := json.Marshal(&auditor)
	user_mgmt.RegisterAuditor(stub, caller, []string{string(auditorBytes), "false"})
	mstub.MockTransactionEnd("transaction6")

	// get caller's user data
	mstub.MockTransactionStart("transaction7")
	stub = cached_stub.NewCachedStub(mstub)
	user_mgmt.GetCallerData(stub)
	mstub.MockTransactionEnd("transaction7")

	// get org's user data
	mstub.MockTransactionStart("transaction8")
	stub = cached_stub.NewCachedStub(mstub)
	user_mgmt.GetUserData(stub, org1, org1.ID, false, true)
	mstub.MockTransactionEnd("transaction8")

	// get org
	mstub.MockTransactionStart("transaction9")
	stub = cached_stub.NewCachedStub(mstub)
	user_mgmt.GetOrg(stub, org1, []string{org1.ID})
	mstub.MockTransactionEnd("transaction9")

	// get user
	mstub.MockTransactionStart("transaction10")
	stub = cached_stub.NewCachedStub(mstub)
	user_mgmt.GetUser(stub, user1, []string{user1.ID})
	mstub.MockTransactionEnd("transaction10")

	// update org
	mstub.MockTransactionStart("transaction11")
	stub = cached_stub.NewCachedStub(mstub)
	org1.Name = "updatedOrgName"
	org1Bytes, _ = json.Marshal(&org1)
	args = []string{string(org1Bytes), "false"}
	user_mgmt.UpdateOrg(stub, caller, args)
	mstub.MockTransactionEnd("transaction11")

	// add org admin
	mstub.MockTransactionStart("transaction12")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.PutUserInGroup(stub, org1, user1.ID, org1.ID, true)
	mstub.MockTransactionEnd("transaction12")

	// add org members
	mstub.MockTransactionStart("transaction13")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.PutUserInGroup(stub, org1, user2.ID, org1.ID, false)
	mstub.MockTransactionEnd("transaction13")

	mstub.MockTransactionStart("transaction14")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.PutUserInGroup(stub, org1, user3.ID, org1.ID, false)
	mstub.MockTransactionEnd("transaction14")

	// check if users are in group
	mstub.MockTransactionStart("transaction15")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.IsUserInGroup(stub, user1.ID, org1.ID)
	user_groups.IsUserInGroup(stub, user2.ID, org1.ID)
	user_groups.IsUserInGroup(stub, user3.ID, org1.ID)
	mstub.MockTransactionEnd("transaction15")

	// check if users are direct group members
	mstub.MockTransactionStart("transaction16")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.IsUserMemberOfGroup(stub, user1.ID, org1.ID)
	user_groups.IsUserMemberOfGroup(stub, user2.ID, org1.ID)
	user_groups.IsUserMemberOfGroup(stub, user3.ID, org1.ID)
	mstub.MockTransactionEnd("transaction16")

	// get groups of which user is a direct member
	mstub.MockTransactionStart("transaction17")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.GetMyDirectGroupIDs(stub, user1.ID)
	mstub.MockTransactionEnd("transaction17")

	// get groups of which user is a direct or indirect member
	mstub.MockTransactionStart("transaction18")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.SlowGetMyGroupIDs(stub, user1, user1.ID, false)
	mstub.MockTransactionEnd("transaction18")

	// get groups of which user is a direct admin
	mstub.MockTransactionStart("transaction19")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.GetMyDirectAdminGroupIDs(stub, user1.ID)
	mstub.MockTransactionEnd("transaction19")

	// get group's admins
	mstub.MockTransactionStart("transaction20")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.SlowGetGroupAdminIDs(stub, org1.ID)
	mstub.MockTransactionEnd("transaction20")

	// get org's member users
	mstub.MockTransactionStart("transaction21")
	stub = cached_stub.NewCachedStub(mstub)
	user_mgmt.GetUsers(stub, org1, []string{org1.ID, "user"})
	mstub.MockTransactionEnd("transaction21")

	// check if user is org admin
	mstub.MockTransactionStart("transaction22")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.IsUserDirectAdminOfGroup(stub, user1.ID, org1.ID)
	mstub.MockTransactionEnd("transaction22")

	// get org data as admin
	mstub.MockTransactionStart("transaction23")
	_, adminPath, _ := user_groups.IsUserAdminOfGroup(stub, user1.ID, org1.ID)
	privkeyPath, _ := user_keys.ConvertAdminPathToPrivateKeyPath(adminPath)
	symkeyPath, _ := user_keys.ConvertAdminPathToSymKeyPath(adminPath)
	user_mgmt.GetUserData(stub, user1, org1.ID, true, false, symkeyPath, privkeyPath)
	mstub.MockTransactionEnd("transaction23")

	// give user admin permission
	mstub.MockTransactionStart("transaction24")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.GiveAdminPermissionOfGroup(stub, user1, user3.ID, org1.ID)
	mstub.MockTransactionEnd("transaction24")

	// remove user's admin permission
	mstub.MockTransactionStart("transaction25")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.RemoveAdminPermissionOfGroup(stub, user1, []string{user3.ID, org1.ID})
	mstub.MockTransactionEnd("transaction25")

	// give user audit permission
	mstub.MockTransactionStart("transaction26")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.GiveAuditorPermissionOfGroupById(stub, user1, auditor.ID, org1.ID)
	mstub.MockTransactionEnd("transaction26")

	// remove user's audit permission
	mstub.MockTransactionStart("transaction27")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.RemoveAuditorPermissionOfGroup(stub, user1, auditor.ID, org1.ID)
	mstub.MockTransactionEnd("transaction27")

	mstub.MockTransactionStart("transaction28")
	stub = cached_stub.NewCachedStub(mstub)
	user_keys.GetUserKeys(stub, caller, user1.ID)
	mstub.MockTransactionEnd("transaction28")

	mstub.MockTransactionStart("transaction29")
	stub = cached_stub.NewCachedStub(mstub)
	user_keys.GetUserPrivateKey(stub, caller, user1.ID)
	mstub.MockTransactionEnd("transaction29")

	mstub.MockTransactionStart("transaction30")
	stub = cached_stub.NewCachedStub(mstub)
	user_keys.GetUserSymKey(stub, caller, user1.ID)
	mstub.MockTransactionEnd("transaction30")

	// get user public key
	mstub.MockTransactionStart("transaction31")
	stub = cached_stub.NewCachedStub(mstub)
	user_keys.GetUserPublicKey(stub, org1, user1.ID)
	mstub.MockTransactionEnd("transaction31")

	// remove member user from group
	mstub.MockTransactionStart("transaction32")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.RemoveUserFromGroup(stub, org1, []string{user2.ID, org1.ID})
	mstub.MockTransactionEnd("transaction32")

	// update org as orgadmin
	mstub.MockTransactionStart("transaction33")
	stub = cached_stub.NewCachedStub(mstub)
	user_mgmt.UpdateOrg(stub, user1, args)
	mstub.MockTransactionEnd("transaction33")

	// register subgroups
	mstub.MockTransactionStart("transaction34")
	stub = cached_stub.NewCachedStub(mstub)
	org2 := test_utils.CreateTestGroup("org2")
	org2Bytes, _ := json.Marshal(&org2)
	user_groups.RegisterSubgroup(stub, org1, []string{string(org2Bytes), org1.ID})
	mstub.MockTransactionEnd("transaction34")

	mstub.MockTransactionStart("transaction35")
	stub = cached_stub.NewCachedStub(mstub)
	org3 := test_utils.CreateTestGroup("org3")
	org3Bytes, _ := json.Marshal(&org3)
	user_groups.RegisterSubgroup(stub, org1, []string{string(org3Bytes), org1.ID})
	mstub.MockTransactionEnd("transaction35")

	// check if subgroups are in group
	mstub.MockTransactionStart("transaction36")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.IsUserInGroup(stub, org2.ID, org1.ID)
	user_groups.IsUserInGroup(stub, org3.ID, org1.ID)
	mstub.MockTransactionEnd("transaction36")

	// check if org is parent group
	mstub.MockTransactionStart("transaction37")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.IsParentGroup(stub, org1, org1.ID, org2.ID)
	mstub.MockTransactionEnd("transaction37")

	// get all orgs
	mstub.MockTransactionStart("transaction38")
	stub = cached_stub.NewCachedStub(mstub)
	user_mgmt.GetOrgs(stub, caller, []string{})
	mstub.MockTransactionEnd("transaction38")

	// get org's member users filtered by role
	mstub.MockTransactionStart("transaction39")
	stub = cached_stub.NewCachedStub(mstub)
	user_mgmt.GetUsers(stub, user1, []string{org1.ID, "user"})
	mstub.MockTransactionEnd("transaction39")

	// get org's member users
	mstub.MockTransactionStart("transaction40")
	stub = cached_stub.NewCachedStub(mstub)
	user_groups.SlowGetGroupMemberIDs(stub, org1.ID)
	mstub.MockTransactionEnd("transaction40")
}
