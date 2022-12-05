/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package cloudant_index

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/datastore"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/datastore_i"
	"common/bchcls/test_utils"
	"common/bchcls/utils"

	"crypto/rsa"
	"encoding/json"
	"net/url"
	"os"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func setup(t *testing.T) (*test_utils.NewMockStub, data_model.User) {
	mstub := test_utils.CreateNewMockStub(t)

	mstub.MockTransactionStart("t0")
	stub := cached_stub.NewCachedStub(mstub)
	datastore_i.Init(stub, shim.LogDebug)
	Init(stub, shim.LogDebug)
	mstub.MockTransactionEnd("t0")

	caller := data_model.User{}
	caller.PrivateKey = test_utils.GeneratePrivateKey()
	caller.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller.PrivateKey))
	pub := crypto.PublicKeyToBytes(caller.PrivateKey.Public().(*rsa.PublicKey))
	caller.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller.ID = "sysadmin"
	caller.Role = global.ROLE_SYSTEM_ADMIN

	return mstub, caller
}
func createSimpleKey(stub cached_stub.CachedStubInterface, objectType string, attributes []string) (string, error) {
	compositeKey, err := stub.CreateCompositeKey(objectType, attributes)
	simpleKey := ""
	if len(compositeKey) > 0 {
		simpleKey = compositeKey[1:]
	}
	return simpleKey, err
}

func TestCloudantDatastore(t *testing.T) {
	mstub, caller := setup(t)

	//using default connection
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)

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

	connection2 := datastore.DatastoreConnection{
		ID:         "cloudant1",
		Type:       datastore.DATASTORE_TYPE_DEFAULT_CLOUDANT,
		ConnectStr: params.Encode(),
	}

	err := datastore_i.PutDatastoreConnection(stub, caller, connection2)
	test_utils.AssertTrue(t, err == nil, "PutDatastoreConnection should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t2")
	stub = cached_stub.NewCachedStub(mstub)
	//cloudant, err := GetDatastoreImpl(stub, "cloudant1")
	cIndex, err := GetIndexDatastoreImpl(stub, "cloudant1")
	test_utils.AssertTrue(t, err == nil, "Error Getting Cloudant index datastore")

	// write index
	dataKey, _ := createSimpleKey(stub, "testIndex", []string{"a1", "b1", "c0"})
	encryptedData := "{\"id\":\"c0\"}"
	hash, err := cIndex.PutIndex(stub, dataKey, []byte(encryptedData))
	logger.Debugf("hash: %v, err: %v", hash, err)
	test_utils.AssertTrue(t, err == nil, "Error PutIndex")

	dataKey, _ = createSimpleKey(stub, "testIndex", []string{"a1", "b5", "c1"})
	encryptedData = "{\"id\":\"c1\"}"
	hash, err = cIndex.PutIndex(stub, dataKey, []byte(encryptedData))
	logger.Debugf("hash: %v, err: %v", hash, err)
	test_utils.AssertTrue(t, err == nil, "Error PutIndex")

	dataKey, _ = createSimpleKey(stub, "testIndex", []string{"a2", "b2", "c2"})
	encryptedData = "{\"id\":\"c2\"}"
	hash, err = cIndex.PutIndex(stub, dataKey, []byte(encryptedData))
	logger.Debugf("hash: %v, err: %v", hash, err)
	test_utils.AssertTrue(t, err == nil, "Error PutIndex")

	dataKey, _ = createSimpleKey(stub, "testIndex", []string{"a3", "b3", "c3"})
	encryptedData = "{\"id\":\"c3\"}"
	hash, err = cIndex.PutIndex(stub, dataKey, []byte(encryptedData))
	logger.Debugf("hash: %v, err: %v", hash, err)
	test_utils.AssertTrue(t, err == nil, "Error PutIndex")

	dataKey, _ = createSimpleKey(stub, "testIndex", []string{"a1", "b2", "c4"})
	encryptedData = "{\"id\":\"c4\"}"
	hash, err = cIndex.PutIndex(stub, dataKey, []byte(encryptedData))
	logger.Debugf("hash: %v, err: %v", hash, err)
	test_utils.AssertTrue(t, err == nil, "Error PutIndex")

	dataKey, _ = createSimpleKey(stub, "testIndex", []string{"a2", "b1", "c5"})
	encryptedData = "{\"id\":\"c5\"}"
	hash, err = cIndex.PutIndex(stub, dataKey, []byte(encryptedData))
	logger.Debugf("hash: %v, err: %v", hash, err)
	test_utils.AssertTrue(t, err == nil, "Error PutIndex")

	mstub.MockTransactionEnd("t2")

	// read from cloudant
	mstub.MockTransactionStart("t3")
	stub = cached_stub.NewCachedStub(mstub)

	cIndex, err = GetIndexDatastoreImpl(stub, "cloudant1")
	test_utils.AssertTrue(t, err == nil, "Error Getting Cloudant index datastore")

	dataKey, _ = createSimpleKey(stub, "testIndex", []string{"a3", "b3", "c3"})
	encryptedData = "{\"id\":\"c3\"}"
	dataBytes, err := cIndex.GetIndex(stub, dataKey)
	logger.Debugf("hash: %v, err: %v", hash, err)
	test_utils.AssertTrue(t, err == nil, "Error GetIndex")
	test_utils.AssertTrue(t, encryptedData == string(dataBytes), "Wrong value")

	dataKey, _ = createSimpleKey(stub, "testIndex", []string{"a1", "b2", "c4"})
	encryptedData = "{\"id\":\"c4\"}"
	dataBytes, err = cIndex.GetIndex(stub, dataKey)
	logger.Debugf("hash: %v, err: %v", hash, err)
	test_utils.AssertTrue(t, err == nil, "Error GetIndex")
	test_utils.AssertTrue(t, encryptedData == string(dataBytes), "Wrong value")

	mstub.MockTransactionEnd("t3")

	// read iter from cloudant
	mstub.MockTransactionStart("t4")
	stub = cached_stub.NewCachedStub(mstub)

	cIndex, err = GetIndexDatastoreImpl(stub, "cloudant1")
	test_utils.AssertTrue(t, err == nil, "Error Getting Cloudant index datastore")

	startKey, _ := createSimpleKey(stub, "testIndex", []string{"a1"})
	endKey, _ := createSimpleKey(stub, "testIndex", []string{"a2"})
	limit := 10
	lastKey := ""
	iter, err := cIndex.GetIndexByRange(stub, startKey, endKey, limit, lastKey)

	logger.Debugf("err: %v", err)
	test_utils.AssertTrue(t, err == nil, "Error Getting iter")

	result := []string{}
	defer iter.Close()
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("Error reading row: %v", err)
			continue
		}
		rowBytes := KV.GetValue()
		var row map[string]string
		err = json.Unmarshal(rowBytes, &row)
		if err != nil {
			logger.Errorf("Error Umnarshal row: %v", err)
			continue
		}
		logger.Debugf("row ==> %v", row)
		result = append(result, row["id"])
	}

	test_utils.AssertListsEqual(t, []string{"c0", "c4", "c1"}, result)
	mstub.MockTransactionEnd("t4")
}
