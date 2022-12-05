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
	"common/bchcls/internal/datastore_i"
	"common/bchcls/internal/datatype_i"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/internal/user_mgmt_i"
	"common/bchcls/test_utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"

	"testing"
)

var logger = shim.NewLogger("datatype_i")

// Call this before each test for stub setup
func setup(t *testing.T) *test_utils.NewMockStub {
	logger.SetLevel(shim.LogDebug)
	mstub := test_utils.CreateNewMockStub(t)
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	datatype_i.Init(stub, shim.LogDebug)
	key_mgmt_i.Init(stub, shim.LogDebug)
	user_mgmt_i.Init(stub, shim.LogDebug)
	datastore_i.Init(stub, shim.LogDebug)
	mstub.MockTransactionEnd("t1")
	return mstub
}

// Tests AddDatatypeSymKey funcs
func TestAddDatatypeSymKey(t *testing.T) {
	logger.Info("TestAddDatatypeSymKey function called")
	mstub := setup(t)

	logger.Debug("Register datatypes")
	mstub.MockTransactionStart("t1")
	// enable putCache so that multiple transactions can be submitted
	stub := cached_stub.NewCachedStub(mstub, true, true)
	datatype1, err := datatype_i.RegisterDatatypeWithParams(stub, "myDatatype1", "", true, datatype_i.ROOT_DATATYPE_ID)
	datatype2, err2 := datatype_i.RegisterDatatypeWithParams(stub, "myDatatype2", "", true, datatype1.GetDatatypeID())
	datatype3, err3 := datatype_i.RegisterDatatypeWithParams(stub, "myDatatype3", "", true, datatype1.GetDatatypeID())
	datatype4, err4 := datatype_i.RegisterDatatypeWithParams(stub, "myDatatype4", "", true, datatype2.GetDatatypeID())
	datatype5, err5 := datatype_i.RegisterDatatypeWithParams(stub, "myDatatype5", "", true, datatype2.GetDatatypeID())
	mstub.MockTransactionEnd("t1")

	test_utils.AssertTrue(t, err == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err2 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err3 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err4 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err5 == nil, "RegisterDatatypeWithParams should not have returned an error")

	//creating users
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Create caller and add to ledger
	callerID := "callerID"
	caller := test_utils.CreateTestUser(callerID)
	err = user_mgmt_i.RegisterUserWithParams(stub, caller, caller, false)
	test_utils.AssertTrue(t, err == nil, "Register caller user should not have returned an error")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Create caller and add to ledger
	caller2ID := "caller2ID"
	caller2 := test_utils.CreateTestUser(caller2ID)
	err = user_mgmt_i.RegisterUserWithParams(stub, caller2, caller2, false)
	test_utils.AssertTrue(t, err == nil, "Register caller2 user should not have returned an error")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Create caller and add to ledger
	ownerID := "ownerID"
	owner := test_utils.CreateTestUser(ownerID)
	err = user_mgmt_i.RegisterUserWithParams(stub, caller, owner, true)
	test_utils.AssertTrue(t, err == nil, "Register owner user should not have returned an error")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	//enable put cache
	stub = cached_stub.NewCachedStub(mstub, true, true)

	//add key for datatype1
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, datatype1.GetDatatypeID(), owner.ID)
	test_utils.AssertTrue(t, err == nil, "Add DatatypeSymkey should not have returned an error")
	datatype1KeyID := datatype1.GetDatatypeKeyID(owner.ID)
	ok, err := key_mgmt_i.VerifyAccessPath(stub, []string{owner.GetSymKeyId(), datatype1KeyID})
	test_utils.AssertTrue(t, err == nil, "verify access path should not have returned an error")
	test_utils.AssertTrue(t, ok, "owner should have access to datatype1 key")

	//add key for datatype5
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, datatype5.GetDatatypeID(), owner.ID)
	test_utils.AssertTrue(t, err == nil, "Add DatatypeSymkey should not have returned an error")
	ok, err = key_mgmt_i.VerifyAccessPath(stub, []string{owner.GetSymKeyId(), datatype5.GetDatatypeKeyID(owner.ID)})
	test_utils.AssertTrue(t, err == nil, "verify access path should not have returned an error")
	test_utils.AssertTrue(t, ok, "owner should have access to datatype5 key")

	//this should also add key for datatype2 (parent)
	ok, err = key_mgmt_i.VerifyAccessPath(stub, []string{owner.GetSymKeyId(), datatype2.GetDatatypeKeyID(owner.ID)})
	test_utils.AssertTrue(t, err == nil, "verify access path should not have returned an error")
	test_utils.AssertTrue(t, ok, "owner should have access to datatype2 key")

	//fail case: this should not add key for datatype4 (sibling)
	ok, err = key_mgmt_i.VerifyAccessPath(stub, []string{owner.GetSymKeyId(), datatype4.GetDatatypeKeyID(owner.ID)})
	test_utils.AssertTrue(t, err == nil, "verify access path should not have returned an error")
	test_utils.AssertTrue(t, !ok, "owner should not have access to datatype4 key")

	//fail case: try to add key for datatype3 for owner by caller2 user
	_, err = datatype_i.AddDatatypeSymKey(stub, caller2, datatype3.GetDatatypeID(), owner.ID)
	test_utils.AssertTrue(t, err != nil, "Add DatatypeSymkey should have returned an error")

	mstub.MockTransactionEnd("t1")
}

// Tests GetDatatypeSymKey funcs
func TestGetDatatypeSymKey(t *testing.T) {
	logger.Info("TestAddDatatypeSymKey function called")
	mstub := setup(t)

	logger.Debug("Register datatypes")
	mstub.MockTransactionStart("t1")
	// enable putCache so that multiple transactions can be submitted
	stub := cached_stub.NewCachedStub(mstub, true, true)
	datatype1, err := datatype_i.RegisterDatatypeWithParams(stub, "myDatatype1", "", true, datatype_i.ROOT_DATATYPE_ID)
	mstub.MockTransactionEnd("t1")

	test_utils.AssertTrue(t, err == nil, "RegisterDatatypeWithParams should not have returned an error")

	//creating users
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Create caller and add to ledger
	callerID := "callerID"
	caller := test_utils.CreateTestUser(callerID)
	err = user_mgmt_i.RegisterUserWithParams(stub, caller, caller, false)
	test_utils.AssertTrue(t, err == nil, "Register caller user should not have returned an error")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Create caller and add to ledger
	caller2ID := "caller2ID"
	caller2 := test_utils.CreateTestUser(caller2ID)
	err = user_mgmt_i.RegisterUserWithParams(stub, caller2, caller2, false)
	test_utils.AssertTrue(t, err == nil, "Register caller2 user should not have returned an error")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Create caller and add to ledger
	ownerID := "ownerID"
	owner := test_utils.CreateTestUser(ownerID)
	err = user_mgmt_i.RegisterUserWithParams(stub, caller, owner, true)
	test_utils.AssertTrue(t, err == nil, "Register owner user should not have returned an error")
	mstub.MockTransactionEnd("t1")

	//add key for datatype1 by caller
	mstub.MockTransactionStart("t1")
	//enable put cache
	stub = cached_stub.NewCachedStub(mstub, true, true)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, datatype1.GetDatatypeID(), owner.ID)
	test_utils.AssertTrue(t, err == nil, "Register owner user should not have returned an error")
	datatype1KeyID := datatype1.GetDatatypeKeyID(owner.ID)
	ok, err := key_mgmt_i.VerifyAccessPath(stub, []string{owner.GetSymKeyId(), datatype1KeyID})
	test_utils.AssertTrue(t, err == nil, "verify access path should not have returned an error")
	test_utils.AssertTrue(t, ok, "owner should have access to datatype1 key")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	//read datatypeSymkey by caller
	symkey, err := datatype_i.GetDatatypeSymKey(stub, caller, datatype1.GetDatatypeID(), owner.ID)
	test_utils.AssertTrue(t, err == nil, "should not have returned an error")
	test_utils.AssertTrue(t, symkey.IsEmpty() == false, "key should not be empty")

	//read datatypeSymkey by owner
	symkey, err = datatype_i.GetDatatypeSymKey(stub, owner, datatype1.GetDatatypeID(), owner.ID)
	test_utils.AssertTrue(t, err == nil, "should not have returned an error")
	test_utils.AssertTrue(t, symkey.IsEmpty() == false, "key should not be empty")

	//read datatypeSymkey by caller2
	symkey, err = datatype_i.GetDatatypeSymKey(stub, caller2, datatype1.GetDatatypeID(), owner.ID)
	test_utils.AssertTrue(t, err != nil, "should have returned an error")
	test_utils.AssertTrue(t, symkey.IsEmpty() == true, "key should  be empty")

	mstub.MockTransactionEnd("t1")

	logger.Debug("Register datatype, add datatype key, and read key from cache within the same transaction")
	mstub.MockTransactionStart("t1")
	// enable putCache so that multiple transactions can be submitted
	stub = cached_stub.NewCachedStub(mstub, true, true)
	datatype2, err := datatype_i.RegisterDatatypeWithParams(stub, "myDatatype2", "", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatypeWithParams should not have returned an error")

	// Create caller and add to ledger
	owner2ID := "owner2ID"
	owner2 := test_utils.CreateTestUser(owner2ID)
	err = user_mgmt_i.RegisterUserWithParams(stub, caller, owner2, true)
	test_utils.AssertTrue(t, err == nil, "Register owner2 user should not have returned an error")

	// add datatypeSymKey by caller
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, datatype2.GetDatatypeID(), owner2.ID)
	test_utils.AssertTrue(t, err == nil, "Register owner2 user should not have returned an error")
	datatype2KeyID := datatype2.GetDatatypeKeyID(owner2.ID)
	keyPath := []string{owner2.GetPubPrivKeyId(), owner2.GetSymKeyId(), datatype2KeyID}
	ok, err = key_mgmt_i.VerifyAccessPath(stub, keyPath)
	test_utils.AssertTrue(t, err == nil, "verify access path should not have returned an error")
	test_utils.AssertTrue(t, ok, "owner2 should have access to datatype2 key")

	//read datatypeSymkey by owner2
	symkey, err = datatype_i.GetDatatypeSymKey(stub, owner2, datatype2.GetDatatypeID(), owner2.ID, keyPath)
	test_utils.AssertTrue(t, err == nil, "should not have returned an error")
	test_utils.AssertTrue(t, symkey.IsEmpty() == false, "key should not be empty")

	//read datatypeSymkey by owner2 again, should get it from cache
	symkey, err = datatype_i.GetDatatypeSymKey(stub, owner2, datatype2.GetDatatypeID(), owner2.ID, keyPath)
	test_utils.AssertTrue(t, err == nil, "should not have returned an error")
	test_utils.AssertTrue(t, symkey.IsEmpty() == false, "key should not be empty")
	mstub.MockTransactionEnd("t1")
}

// Tests DatatypeInterface funcs
func TestDatatypeInterface(t *testing.T) {
	logger.Info("TestAddDatatypeSymKey function called")
	mstub := setup(t)

	logger.Debug("Register datatypes")
	mstub.MockTransactionStart("t1")
	// enable putCache so that multiple transactions can be submitted
	stub := cached_stub.NewCachedStub(mstub, true, true)
	datatype1, err := datatype_i.RegisterDatatypeWithParams(stub, "myDatatype1", "", true, datatype_i.ROOT_DATATYPE_ID)
	mstub.MockTransactionEnd("t1")

	test_utils.AssertTrue(t, err == nil, "RegisterDatatypeWithParams should not have returned an error")

	//creating users
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Create caller and add to ledger
	callerID := "callerID"
	caller := test_utils.CreateTestUser(callerID)
	err = user_mgmt_i.RegisterUserWithParams(stub, caller, caller, false)
	test_utils.AssertTrue(t, err == nil, "Register caller user should not have returned an error")
	mstub.MockTransactionEnd("t1")

	// Deactivate
	datatype1.Deactivate()
	test_utils.AssertTrue(t, !datatype1.IsActive(), "should be inactive state")

	// Activate
	datatype1.Activate()
	test_utils.AssertTrue(t, datatype1.IsActive(), "should be active state")

	// SetDescription
	datatype1.SetDescription("New description")
	test_utils.AssertTrue(t, datatype1.GetDescription() == "New description", "should have changed the description")

	// PutDatatype
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = datatype1.PutDatatype(stub)
	test_utils.AssertTrue(t, err == nil, "PutDatatype should have succeeded")
	mstub.MockTransactionEnd("t1")

	// readback from the ledger and compare
	datatype2, err := datatype_i.GetDatatypeWithParams(stub, datatype1.GetDatatypeID())
	test_utils.AssertTrue(t, err == nil, "GetDatatypeWithParams should have succeeded")
	test_utils.AssertTrue(t, datatype2.GetDescription() == "New description", "should have changed the description")

}
