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
	"common/bchcls/datastore"
	"common/bchcls/datastore/datastore_manager"
	"common/bchcls/init_common"
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
	init_common.Init(stub, shim.LogDebug)

	// change the following Cloudant config to connect to a custom Cloudant and set environment variables:
	// CLOUDANT_USERNAME
	// CLOUDANT_PASSWORD
	// CLOUDANT_DATABASE
	// CLOUDANT_HOST
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

	_, err := init_common.InitDatastore(stub, username, password, database, host)
	test_utils.AssertTrue(t, err == nil, "Error Getting Cloudant datastore: Make sure to start cloudant docker before running this test")
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

func TestDefaultCloudantDatastore(t *testing.T) {

	mstub, caller := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	t1 := time.Now().UnixNano()
	cloudant, err := datastore_manager.GetDatastoreImpl(stub, global.DEFAULT_CLOUDANT_DATASTORE_ID)
	t2 := time.Now().UnixNano()
	logger.Debugf("Time to get cloudant first time: %v", t2-t1)
	//if err != nil {
	//	logger.Error("Error Getting Cloudant datastore: Make sure to start cloudant docker before running this test: %v", err)
	//	return
	//}
	test_utils.AssertTrue(t, err == nil, "Error Getting Cloudant datastore: Make sure to start cloudant docker before running this test")

	// just call it second time to check that this should be returned from memory
	// disabling this test since it's not reliable since could fail depending on your machine's resouce status
	/*
		t3 := time.Now().UnixNano()
		cloudant, err = datastore_manager.GetDatastoreImpl(stub, global.DEFAULT_CLOUDANT_DATASTORE_ID)
		t4 := time.Now().UnixNano()
		logger.Debugf("Time to get cloudant second time: %v", t4-t3)
		test_utils.AssertTrue(t, err == nil, "GetDatastoreImpl should be successful")
		// getting from cache should be much faster
		test_utils.AssertTrue(t, (t4-t3) < (t2-t1), "Cache should be faster")
		test_utils.AssertTrue(t, cloudant.IsReady(), "cloudant should be isReady true")
	*/

	// write and delete data
	for i := 0; i < 10; i++ {
		myData := []byte("this is test data for cloudant datastore " + strconv.Itoa(i))
		key, err := cloudant.Put(stub, myData)
		test_utils.AssertTrue(t, err == nil, "Put() should be successful")

		err = cloudant.Delete(stub, key)
		test_utils.AssertTrue(t, err == nil, "Delete() should be successful")
		time.Sleep(1 * time.Second)
	}

	// write data first time since we deleted data in the previous step
	keys := []string{}
	t1 = time.Now().UnixNano()
	for i := 0; i < 10; i++ {
		myData := []byte("this is test data for cloudant datastore " + strconv.Itoa(i))
		key, err := cloudant.Put(stub, myData)
		keys = append(keys, key)
		test_utils.AssertTrue(t, err == nil, "Put() should be successful")
		time.Sleep(1 * time.Second)
	}
	t2 = time.Now().UnixNano()

	// writing it second time should be much faster
	// write same data second time; should not re-write data since it already exists
	keys2 := []string{}
	t3 := time.Now().UnixNano()
	for i := 0; i < 10; i++ {
		myData := []byte("this is test data for cloudant datastore " + strconv.Itoa(i))
		key, err := cloudant.Put(stub, myData)
		keys2 = append(keys2, key)
		test_utils.AssertTrue(t, err == nil, "Put() should be successful")
		time.Sleep(1 * time.Second)
	}
	t4 := time.Now().UnixNano()
	test_utils.AssertListsEqual(t, keys, keys2)
	logger.Debugf("Time to get write first time: %v", t2-t1)
	logger.Debugf("Time to get write second time: %v", t4-t3)
	test_utils.AssertTrue(t, (t4-t3) < (t2-t1), "Second time should be faster")

	// read
	for i := 0; i < 10; i++ {
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

	username := "admin"
	database := "test"
	host := "http://127.0.0.1:9080"
	// Get values from environment variables
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_USERNAME")) {
		username = os.Getenv("CLOUDANT_USERNAME")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_DATABASE")) {
		database = os.Getenv("CLOUDANT_DATABASE")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_HOST")) {
		host = os.Getenv("CLOUDANT_HOST")
	}

	params := url.Values{}
	params.Add("username", username)
	params.Add("password", "wrong_password")
	params.Add("database", database)
	params.Add("host", host)

	connection3 := datastore.DatastoreConnection{
		ID:         "cloudant2",
		Type:       datastore.DATASTORE_TYPE_DEFAULT_CLOUDANT,
		ConnectStr: params.Encode(),
	}

	err = datastore_manager.PutDatastoreConnection(stub, caller, connection3)
	test_utils.AssertTrue(t, err == nil, "PutDatastoreConnection should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	cloudant, err = datastore_manager.GetDatastoreImpl(stub, "cloudant2")
	test_utils.AssertTrue(t, err != nil, "GetDatastoreImpl should fail")
	mstub.MockTransactionEnd("t1")

	// wrong ID for read
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	cloudant, err = datastore_manager.GetDatastoreImpl(stub, global.DEFAULT_CLOUDANT_DATASTORE_ID)
	test_utils.AssertTrue(t, err == nil, "GetDatastoreImpl should be successful")

	_, err = cloudant.Get(stub, "wrongid")
	test_utils.AssertTrue(t, err != nil, "Get() should fail")
	mstub.MockTransactionEnd("t1")

}
