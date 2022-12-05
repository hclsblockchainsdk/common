/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package datastore_manager

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/datastore"
	"common/bchcls/internal/common/global"
	"common/bchcls/test_utils"
	"crypto/rsa"
	"testing"
)

func ExamplePutDatastoreConnection(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	caller := createAdminCaller()

	datastore1 := datastore.DatastoreConnection{
		ID:         "testcloudant1",
		Type:       datastore.DATASTORE_TYPE_DEFAULT_CLOUDANT,
		ConnectStr: "username=yourid&password=yourpassword&database=database1&host=https://yourid.cloudantnosqldb.appdomain.cloud",
	}

	// Puts a new connection or replaces existing connection
	mstub.MockTransactionStart("transaction1")
	stub := cached_stub.NewCachedStub(mstub)
	PutDatastoreConnection(stub, caller, datastore1)
	mstub.MockTransactionEnd("transaction1")
}

func ExampleGetDatastoreConnection(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)

	datastoreConnectionID := "testcloudant" //an existing connectionId to deactivate

	mstub.MockTransactionStart("transaction1")
	stub := cached_stub.NewCachedStub(mstub)
	//Returns Connecton details
	GetDatastoreConnection(stub, datastoreConnectionID)

	mstub.MockTransactionEnd("transaction1")
}

func ExampleDeleteDatastoreConnection(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	caller := createAdminCaller()

	datastoreConnectionID := "testcloudant" //an existing connectionId to delete

	mstub.MockTransactionStart("transaction1")
	stub := cached_stub.NewCachedStub(mstub)
	DeleteDatastoreConnection(stub, caller, datastoreConnectionID)

	mstub.MockTransactionEnd("transaction1")
}

func ExampleGetDatastoreImpl(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)

	datastoreConnectionID := "testcloudant" //an existing connectionId to deactivate

	mstub.MockTransactionStart("transaction1")
	stub := cached_stub.NewCachedStub(mstub)
	dsImpl, _ := GetDatastoreImpl(stub, datastoreConnectionID)

	//Use Implementation to Get a data like below
	data, _ := dsImpl.Get(stub, "datakey")
	logger.Debugf("%v", data)
	mstub.MockTransactionEnd("transaction1")
}

func createAdminCaller() data_model.User {

	caller := data_model.User{}
	caller.PrivateKey = test_utils.GeneratePrivateKey()
	caller.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller.PrivateKey))
	pub := crypto.PublicKeyToBytes(caller.PrivateKey.Public().(*rsa.PublicKey))
	caller.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller.ID = "sysadmin"
	caller.Role = global.ROLE_SYSTEM_ADMIN

	return caller
}
