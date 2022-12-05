/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package datastore_i

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/datastore"
	"common/bchcls/internal/common/global"
	"common/bchcls/test_utils"
	"common/bchcls/utils"

	"crypto/rsa"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func setup(t *testing.T) (*test_utils.NewMockStub, data_model.User) {
	mstub := test_utils.CreateNewMockStub(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	Init(stub, shim.LogDebug)
	mstub.MockTransactionEnd("t1")

	caller := data_model.User{}
	caller.PrivateKey = test_utils.GeneratePrivateKey()
	caller.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller.PrivateKey))
	pub := crypto.PublicKeyToBytes(caller.PrivateKey.Public().(*rsa.PublicKey))
	caller.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller.ID = "sysadmin"
	caller.Role = global.ROLE_SYSTEM_ADMIN

	return mstub, caller
}

func TestDefaultDatastoreInitialization(t *testing.T) {

	mstub, caller := setup(t)

	//ledger
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	datastore1 := datastore.DatastoreConnection{
		ID: "testconnection", Type: global.DATASTORE_TYPE_DEFAULT_LEDGER}

	datastoreImpl, implErr := GetDatastoreImpl(stub, datastore1.ID)
	test_utils.AssertTrue(t, implErr != nil, "GetDatastoreImpl call before PutDatastoreConnection of that datastoreId should fail")
	// Put datastoreconnection as sysadmin
	err := PutDatastoreConnection(stub, caller, datastore1)
	test_utils.AssertNilError(t, err, "Register Datastore should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	thisDatastore, getErr := GetDatastoreConnection(stub, datastore1.ID)
	test_utils.AssertNilError(t, getErr, "GetDatastoreConnection should be successful")
	test_utils.AssertTrue(t, thisDatastore.Type == datastore1.Type, "Get DatastoreConnection type should match")

	datastoreImpl, implErr = GetDatastoreImpl(stub, datastore1.ID)
	logger.Debugf("GetDatastoreImpl for ledger: %v", datastoreImpl)

	test_utils.AssertNilError(t, err, "GetDatastoreImpl should be successful")
	test_utils.AssertTrue(t, datastoreImpl != nil, "Get of Datastore Impl should be successful")

	myData := []byte("this is my test data")
	myKey, err := datastoreImpl.Put(stub, myData)
	test_utils.AssertTrue(t, err == nil, "Put() using Datastore Impl should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	datastoreImpl, implErr = GetDatastoreImpl(stub, datastore1.ID)
	logger.Debugf("GetDatastoreImpl for ledger: %v", datastoreImpl)

	myData2, keyErr := datastoreImpl.Get(stub, myKey)
	test_utils.AssertTrue(t, keyErr == nil, "Get() using Datastore Impl should be successful")

	test_utils.AssertTrue(t, string(myData) == string(myData2), "Data from Get() should return the original value")

	mstub.MockTransactionEnd("t1")

}

func TestPutDatastoreConnection_MultipleConnections(t *testing.T) {

	mstub, caller := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	datastore1 := datastore.DatastoreConnection{
		ID:         "testledger1",
		ConnectStr: "connect string",
		Type:       global.DATASTORE_TYPE_DEFAULT_LEDGER}

	err := PutDatastoreConnection(stub, caller, datastore1)
	test_utils.AssertNilError(t, err, "Register datastore1 should be successful")
	mstub.MockTransactionEnd("t1")

	datastore2 := datastore.DatastoreConnection{
		ID:         "testledger2",
		ConnectStr: "connect string",
		Type:       global.DATASTORE_TYPE_DEFAULT_LEDGER}

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutDatastoreConnection(stub, caller, datastore2)
	test_utils.AssertNilError(t, err, "Register datastore2 should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	retDatastore, getErr := GetDatastoreConnection(stub, datastore1.ID)
	test_utils.AssertNilError(t, getErr, "GetDatastoreConnection should be successful")
	test_utils.AssertTrue(t, retDatastore.ID == datastore1.ID, "GetDatastoreConnection should retrieve dataId "+datastore1.ID)
	logger.Info("GetDatastoreConnection %v", retDatastore)

	retDatastore, getErr = GetDatastoreConnection(stub, datastore2.ID)
	test_utils.AssertNilError(t, getErr, "GetDatastoreConnection should be successful")
	test_utils.AssertTrue(t, len(retDatastore.ConnectStr) > 0, "GetDatastoreConnection should fetch encrypted data")

	//Put again, should replace fields
	newConnectStr := "new connect string"
	datastore1.ConnectStr = newConnectStr
	err = PutDatastoreConnection(stub, caller, datastore1)
	test_utils.AssertTrue(t, err == nil, "PutDatastoreConnection should be successful ")

	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	retDatastore, getErr = GetDatastoreConnection(stub, datastore1.ID)
	test_utils.AssertTrue(t, retDatastore.ConnectStr == newConnectStr, "ConnectStr should be updated to "+newConnectStr)

	datastoreImpl, implErr := GetDatastoreImpl(stub, datastore1.ID)
	test_utils.AssertNilError(t, implErr, "GetDatastoreImpl should be successful")

	test_utils.AssertTrue(t, datastoreImpl.GetDatastoreConnection().ID == datastore1.ID, "Datastore Impl ID should match")

	datastoreImpl, implErr = GetDatastoreImpl(stub, datastore2.ID)
	test_utils.AssertNilError(t, implErr, "GetDatastoreImpl should be successful")
	test_utils.AssertTrue(t, datastoreImpl.GetDatastoreConnection().ID == datastore2.ID, "Datastore Impl ID should match")

	mstub.MockTransactionEnd("t1")

}

func TestDeleteDatastoreConnections(t *testing.T) {

	mstub, caller := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	datastore1 := datastore.DatastoreConnection{
		ID:   "testledger1",
		Type: global.DATASTORE_TYPE_DEFAULT_LEDGER}

	err := PutDatastoreConnection(stub, caller, datastore1)
	test_utils.AssertNilError(t, err, "Register datastore1 should be successful")

	datastore2 := datastore.DatastoreConnection{
		ID:   "testledger2",
		Type: global.DATASTORE_TYPE_DEFAULT_LEDGER}

	err = PutDatastoreConnection(stub, caller, datastore2)
	test_utils.AssertNilError(t, err, "Register datastore2 should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	retDatastore, getErr := GetDatastoreConnection(stub, datastore1.ID)
	test_utils.AssertTrue(t, getErr == nil, "GetDatastoreConnection should succeed")
	test_utils.AssertTrue(t, retDatastore.ID == datastore1.ID, "ID should match, "+datastore1.ID)

	datastoreImpl, implErr := GetDatastoreImpl(stub, datastore1.ID)
	test_utils.AssertNilError(t, implErr, "GetDatastoreImpl should be successful")
	test_utils.AssertTrue(t, datastoreImpl.GetDatastoreConnection().ID == datastore1.ID, "Datastore Impl ID should match")

	//Delete Connection
	err = DeleteDatastoreConnection(stub, caller, datastore1.ID)
	test_utils.AssertNilError(t, err, "Delete datastore1 should be successful")

	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	retDatastore, getErr = GetDatastoreConnection(stub, datastore1.ID)
	test_utils.AssertTrue(t, getErr == nil, "GetDatastoreConnection should succeed")
	test_utils.AssertTrue(t, utils.IsStringEmpty(retDatastore.ID), "Connection should be deleted, ID:"+datastore1.ID)

	datastoreImpl, implErr = GetDatastoreImpl(stub, datastore1.ID)
	test_utils.AssertTrue(t, implErr != nil, "GetDatastoreImpl should fail on non-existing Connection")

	retDatastore, getErr = GetDatastoreConnection(stub, datastore2.ID)
	test_utils.AssertNilError(t, getErr, "GetDatastoreConnection should be successful")
	test_utils.AssertTrue(t, retDatastore.ID == datastore2.ID, "Datastore2 ID should match")

	mstub.MockTransactionEnd("t1")
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

	connection2 := datastore.DatastoreConnection{
		ID:         "cloudant1",
		Type:       datastore.DATASTORE_TYPE_DEFAULT_CLOUDANT,
		ConnectStr: params.Encode(),
	}

	err := PutDatastoreConnection(stub, caller, connection2)
	test_utils.AssertTrue(t, err == nil, "PutDatastoreConnection should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	cloudant, err := GetDatastoreImpl(stub, "cloudant1")
	test_utils.AssertTrue(t, err == nil, "Error Getting Cloudant datastore: Make sure to start cloudant docker before running this test")

	// write and delete data
	for i := 0; i < 5; i++ {
		myData := []byte("this is test data for cloudant datastore " + strconv.Itoa(i))
		key, err := cloudant.Put(stub, myData)
		test_utils.AssertTrue(t, err == nil, "Put() should be successful")
		time.Sleep(1 * time.Second)

		err = cloudant.Delete(stub, key)
		test_utils.AssertTrue(t, err == nil, "Delete() should be successful")
		time.Sleep(1 * time.Second)
	}

	// write data first time since we deleted data in the previous step
	keys := []string{}
	for i := 0; i < 5; i++ {
		myData := []byte("this is test data for cloudant datastore " + strconv.Itoa(i))
		key, err := cloudant.Put(stub, myData)
		keys = append(keys, key)
		test_utils.AssertTrue(t, err == nil, "Put() should be successful")
		time.Sleep(1 * time.Second)
	}

	// read
	for i := 0; i < 5; i++ {
		myData := []byte("this is test data for cloudant datastore " + strconv.Itoa(i))
		readData, err := cloudant.Get(stub, keys[i])
		test_utils.AssertTrue(t, err == nil, "Get() should be successful")
		test_utils.AssertTrue(t, string(myData) == string(readData), "Get() should return same data")
		time.Sleep(1 * time.Second)
	}
	mstub.MockTransactionEnd("t1")

	// non happy path
	// wrong connection param (wrong password)
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	params2 := url.Values{}
	params2.Add("username", username)
	params2.Add("password", "wrong_password")
	params2.Add("database", database)
	params2.Add("host", host)

	connection3 := datastore.DatastoreConnection{
		ID:         "cloudant2",
		Type:       datastore.DATASTORE_TYPE_DEFAULT_CLOUDANT,
		ConnectStr: params2.Encode(),
	}

	err = PutDatastoreConnection(stub, caller, connection3)
	test_utils.AssertTrue(t, err == nil, "PutDatastoreConnection should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	cloudant, err = GetDatastoreImpl(stub, "cloudant2")
	test_utils.AssertTrue(t, err != nil, "GetDatastoreImpl should fail")
	mstub.MockTransactionEnd("t1")

	// wrong ID for read
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	cloudant, err = GetDatastoreImpl(stub, "cloudant1")
	test_utils.AssertTrue(t, err == nil, "GetDatastoreImpl should be successful")

	_, err = cloudant.Get(stub, "wrongid")
	test_utils.AssertTrue(t, err != nil, "Get() should fail")
	mstub.MockTransactionEnd("t1")

}
