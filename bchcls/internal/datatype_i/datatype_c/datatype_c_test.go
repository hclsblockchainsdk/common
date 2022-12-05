/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package datatype_c

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/test_utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"

	"crypto/rsa"
	"encoding/json"
	"testing"
)

// Call this before each test for stub setup
func setup(t *testing.T) *test_utils.NewMockStub {
	mstub := test_utils.CreateNewMockStub(t)
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	Init(stub)
	mstub.MockTransactionEnd("t1")
	logger.SetLevel(shim.LogDebug)
	return mstub
}

// Adds a datatype to the ledger
func TestRegisterDatatype(t *testing.T) {
	logger.Info("TestRegisterDatatype function called")
	mstub := setup(t)

	caller := data_model.User{}
	caller.PrivateKey = test_utils.GeneratePrivateKey()
	caller.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller.PrivateKey))
	pub := crypto.PublicKeyToBytes(caller.PrivateKey.Public().(*rsa.PublicKey))
	caller.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller.ID = "sysadmin"
	caller.Role = "system"

	user := data_model.User{}
	user.PrivateKey = test_utils.GeneratePrivateKey()
	user.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(user.PrivateKey))
	pub = crypto.PublicKeyToBytes(user.PrivateKey.Public().(*rsa.PublicKey))
	user.PublicKeyB64 = crypto.EncodeToB64String(pub)
	user.ID = "user"
	user.Role = "user"

	datatype1 := data_model.Datatype{DatatypeID: "datatype1", Description: "datatype1", IsActive: true}
	datatype1Bytes, _ := json.Marshal(&datatype1)

	// attempt to register datatype with invalid parentDatatypeID
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	_, err := RegisterDatatype(stub, caller, []string{string(datatype1Bytes), "datatypeX"})
	test_utils.AssertTrue(t, err != nil, "RegisterDatatype should fail, invalid parentDatatypeID")
	mstub.MockTransactionEnd("t1")

	// register datatype as sysadmin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterDatatype(stub, caller, []string{string(datatype1Bytes), ""})
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// attempt to register datatype that already exists
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterDatatype(stub, caller, []string{string(datatype1Bytes), ""})
	test_utils.AssertTrue(t, err != nil, "RegisterDatatype should fail, datatype already exists")
	mstub.MockTransactionEnd("t1")

	// check parent datatypes
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	expectedParents1 := []string{}
	parents1, err := GetParentDatatypes(stub, datatype1.DatatypeID)
	test_utils.AssertTrue(t, err == nil, "GetParentDatatypes should be successful")
	test_utils.AssertSetsEqual(t, expectedParents1, parents1)
	mstub.MockTransactionEnd("t1")

	datatype2 := data_model.Datatype{DatatypeID: "datatype2", Description: "datatype2", IsActive: true}
	datatype2Bytes, _ := json.Marshal(&datatype2)

	// register second datatype as child of datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterDatatype(stub, caller, []string{string(datatype2Bytes), "datatype1"})
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// check parent datatypes
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	expectedParents2 := []string{"datatype1"}
	parents2, err := GetParentDatatypes(stub, datatype2.DatatypeID)
	test_utils.AssertTrue(t, err == nil, "GetParentDatatypes should be successful")
	test_utils.AssertSetsEqual(t, expectedParents2, parents2)

}

// Adds a datatype to the ledger
func TestRegisterDatatype_inActive(t *testing.T) {
	logger.Info("TestRegisterDatatype_inActive function called")
	mstub := setup(t)

	caller := data_model.User{}
	caller.PrivateKey = test_utils.GeneratePrivateKey()
	caller.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller.PrivateKey))
	pub := crypto.PublicKeyToBytes(caller.PrivateKey.Public().(*rsa.PublicKey))
	caller.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller.ID = "sysadmin"
	caller.Role = "system"

	user := data_model.User{}
	user.PrivateKey = test_utils.GeneratePrivateKey()
	user.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(user.PrivateKey))
	pub = crypto.PublicKeyToBytes(user.PrivateKey.Public().(*rsa.PublicKey))
	user.PublicKeyB64 = crypto.EncodeToB64String(pub)
	user.ID = "user"
	user.Role = "user"

	datatype1 := data_model.Datatype{DatatypeID: "datatype1", Description: "datatype1", IsActive: true}
	datatype1Bytes, _ := json.Marshal(&datatype1)

	// register datatype as sysadmin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	_, err := RegisterDatatype(stub, caller, []string{string(datatype1Bytes), ""})
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	datatype2 := data_model.Datatype{DatatypeID: "datatype2", Description: "datatype2", IsActive: false}
	datatype2Bytes, _ := json.Marshal(&datatype2)

	// register inactive datatype under active parent datatype
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterDatatype(stub, caller, []string{string(datatype2Bytes), "datatype1"})
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	datatype3 := data_model.Datatype{DatatypeID: "datatype3", Description: "datatype2", IsActive: true}
	datatype3Bytes, _ := json.Marshal(&datatype3)

	// register active datatype under inactive parent datatype (should fail)
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterDatatype(stub, caller, []string{string(datatype3Bytes), "datatype2"})
	test_utils.AssertTrue(t, err != nil, "RegisterDatatype should not be successful")
	mstub.MockTransactionEnd("t1")

	datatype3 = data_model.Datatype{DatatypeID: "datatype3", Description: "datatype2", IsActive: false}
	datatype3Bytes, _ = json.Marshal(&datatype3)

	// register inactive datatype under inactive parent datatype
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterDatatype(stub, caller, []string{string(datatype3Bytes), "datatype2"})
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

}

func TestUpdateDatatype(t *testing.T) {
	logger.Info("TestUpdateDatatype function called")
	mstub := setup(t)
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	Init(stub)
	mstub.MockTransactionEnd("t1")

	caller := data_model.User{}
	caller.PrivateKey = test_utils.GeneratePrivateKey()
	caller.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller.PrivateKey))
	pub := crypto.PublicKeyToBytes(caller.PrivateKey.Public().(*rsa.PublicKey))
	caller.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller.ID = "sysadmin"
	caller.Role = "system"

	user := data_model.User{}
	user.PrivateKey = test_utils.GeneratePrivateKey()
	user.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(user.PrivateKey))
	pub = crypto.PublicKeyToBytes(user.PrivateKey.Public().(*rsa.PublicKey))
	user.PublicKeyB64 = crypto.EncodeToB64String(pub)
	user.ID = "user"
	user.Role = "user"

	datatype1 := data_model.Datatype{DatatypeID: "datatype1", Description: "datatype1", IsActive: true}
	datatype1Bytes, _ := json.Marshal(&datatype1)

	// register datatype as sysadmin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err := RegisterDatatype(stub, caller, []string{string(datatype1Bytes), ""})
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// update datatype
	datatype1.Description = "new description"
	datatype1Bytes, _ = json.Marshal(&datatype1)
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = UpdateDatatype(stub, caller, []string{string(datatype1Bytes)})
	test_utils.AssertTrue(t, err == nil, "UpdateDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	datatype1Test, err := GetDatatypeWithParams(stub, datatype1.DatatypeID)
	test_utils.AssertTrue(t, err == nil, "GetDatatypeWithParams should be successful")
	test_utils.AssertTrue(t, datatype1Test.GetDescription() == "new description", "Datatype was not updated successfully")
	mstub.MockTransactionEnd("t1")
}

// get datatype from ledger
func TestGetDatatype(t *testing.T) {
	logger.Info("TestGetDatatype function called")
	mstub := setup(t)
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	Init(stub)
	mstub.MockTransactionEnd("t1")

	caller := data_model.User{}
	caller.PrivateKey = test_utils.GeneratePrivateKey()
	caller.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller.PrivateKey))
	pub := crypto.PublicKeyToBytes(caller.PrivateKey.Public().(*rsa.PublicKey))
	caller.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller.ID = "sysadmin"
	caller.Role = "system"

	user := data_model.User{}
	user.PrivateKey = test_utils.GeneratePrivateKey()
	user.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(user.PrivateKey))
	pub = crypto.PublicKeyToBytes(user.PrivateKey.Public().(*rsa.PublicKey))
	user.PublicKeyB64 = crypto.EncodeToB64String(pub)
	user.ID = "user"
	user.Role = "user"

	datatype1 := data_model.Datatype{DatatypeID: "datatype1", Description: "datatype1", IsActive: true}
	datatype1Bytes, _ := json.Marshal(&datatype1)

	// register datatype as sysadmin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err := RegisterDatatype(stub, caller, []string{string(datatype1Bytes)})
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// get datatype1 as sysadmin
	datatype1Test := data_model.Datatype{}
	datatype1TestBytes, err := GetDatatype(stub, caller, []string{"datatype1"})
	test_utils.AssertTrue(t, err == nil, "GetDatatype should succeed")
	json.Unmarshal(datatype1TestBytes, &datatype1Test)
	test_utils.AssertTrue(t, datatype1Test.DatatypeID == "datatype1", "GetDatatype should succeed")

	// get datatype1 as user
	datatype1Test = data_model.Datatype{}
	datatype1TestBytes, err = GetDatatype(stub, user, []string{"datatype1"})
	test_utils.AssertTrue(t, err == nil, "GetDatatype should succeed")
	json.Unmarshal(datatype1TestBytes, &datatype1Test)
	test_utils.AssertTrue(t, datatype1Test.DatatypeID == "datatype1", "GetDatatype should succeed")
	mstub.MockTransactionEnd("t1")
}

// Gets a datatype from the ledger
func TestGetDatatypeWithParams(t *testing.T) {
	logger.Info("TestGetDatatypeWithParams function called")

	mstub := setup(t)
	var dtype data_model.Datatype
	dtype.DatatypeID = "myDatatype"

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	_, err := RegisterDatatypeWithParams(stub, dtype.DatatypeID, dtype.Description, dtype.IsActive, ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatypeWithParams should not have returned an error")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	dtypeResult, err := GetDatatypeWithParams(stub, dtype.DatatypeID)
	test_utils.AssertTrue(t, err == nil, "GetDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, dtypeResult.GetDatatypeID() == dtype.DatatypeID, "GetDatatypeWithParams should not have returned an null byte")
	mstub.MockTransactionEnd("t1")
}

func TestGetAllDatatypes(t *testing.T) {
	logger.Info("TestGetAllDatatypes function called")

	mstub := setup(t)
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	Init(stub)
	mstub.MockTransactionEnd("t1")

	caller := data_model.User{}
	caller.PrivateKey = test_utils.GeneratePrivateKey()
	caller.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller.PrivateKey))
	pub := crypto.PublicKeyToBytes(caller.PrivateKey.Public().(*rsa.PublicKey))
	caller.PublicKeyB64 = crypto.EncodeToB64String(pub)
	caller.ID = "sysadmin"
	caller.Role = "system"

	datatype1 := data_model.Datatype{DatatypeID: "datatype1", Description: "datatype1"}
	datatype2 := data_model.Datatype{DatatypeID: "datatype2", Description: "datatype2"}
	datatype3 := data_model.Datatype{DatatypeID: "datatype3", Description: "datatype3"}
	datatype4 := data_model.Datatype{DatatypeID: "datatype4", Description: "datatype4"}

	// register datatypes
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err := RegisterDatatypeWithParams(stub, datatype1.DatatypeID, datatype1.Description, datatype1.IsActive, ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	_, err = RegisterDatatypeWithParams(stub, datatype2.DatatypeID, datatype2.Description, datatype2.IsActive, ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterDatatypeWithParams(stub, datatype3.DatatypeID, datatype3.Description, datatype3.IsActive, datatype1.DatatypeID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterDatatypeWithParams(stub, datatype4.DatatypeID, datatype4.Description, datatype4.IsActive, datatype3.DatatypeID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// get datatypes
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	datatypes := []data_model.Datatype{}
	datatypesBytes, err := GetAllDatatypes(stub, caller, []string{})
	test_utils.AssertTrue(t, err == nil, "GetAllDatatypes should succeed")
	json.Unmarshal(datatypesBytes, &datatypes)
	test_utils.AssertTrue(t, len(datatypes) == 4, "Expected 4 datatypes")

	expectedDatatypes := make(map[string]bool)
	expectedDatatypes[datatype1.DatatypeID] = true
	expectedDatatypes[datatype2.DatatypeID] = true
	expectedDatatypes[datatype3.DatatypeID] = true
	expectedDatatypes[datatype4.DatatypeID] = true
	returnedDatatypes := make(map[string]bool)
	for _, dtype := range datatypes {
		returnedDatatypes[dtype.DatatypeID] = true
	}
	test_utils.AssertMapsEqual(t, expectedDatatypes, returnedDatatypes, "Expected: datatype1, datatype2, datatype3, datatype4")
	mstub.MockTransactionEnd("t1")
}

// Tests GetChildDatatypes and IsChildOf funcs
func TestChildRelationships(t *testing.T) {
	logger.Info("TestChildRelationships function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	// enable putCache so that multiple transactions can be submitted
	stub := cached_stub.NewCachedStub(mstub, true, true)
	datatype1, err := RegisterDatatypeWithParams(stub, "myDatatype1", "", true, ROOT_DATATYPE_ID)
	datatype2, err2 := RegisterDatatypeWithParams(stub, "myDatatype2", "", true, datatype1.GetDatatypeID())
	datatype3, err3 := RegisterDatatypeWithParams(stub, "myDatatype3", "", true, datatype1.GetDatatypeID())
	datatype4, err4 := RegisterDatatypeWithParams(stub, "myDatatype4", "", true, datatype2.GetDatatypeID())
	datatype5, err5 := RegisterDatatypeWithParams(stub, "myDatatype5", "", true, datatype2.GetDatatypeID())
	mstub.MockTransactionEnd("t1")

	test_utils.AssertTrue(t, err == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err2 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err3 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err4 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err5 == nil, "RegisterDatatypeWithParams should not have returned an error")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	children, err := datatype1.GetChildDatatypes(stub)
	expectedChildren := []string{datatype2.GetDatatypeID(), datatype3.GetDatatypeID(), datatype4.GetDatatypeID(), datatype5.GetDatatypeID()}
	children2, err2 := datatype2.GetChildDatatypes(stub)
	expectedChildren2 := []string{datatype4.GetDatatypeID(), datatype5.GetDatatypeID()}
	children3, err3 := datatype3.GetChildDatatypes(stub)
	expectedChildren3 := []string{}

	relationship, err4 := datatype3.IsChildOf(stub, datatype1.GetDatatypeID())
	relationship2, err5 := datatype5.IsChildOf(stub, datatype1.GetDatatypeID())
	relationship3, err6 := datatype3.IsChildOf(stub, datatype5.GetDatatypeID())
	mstub.MockTransactionEnd("t1")

	test_utils.AssertSetsEqual(t, expectedChildren, children)
	test_utils.AssertSetsEqual(t, expectedChildren2, children2)
	test_utils.AssertSetsEqual(t, expectedChildren3, children3)
	test_utils.AssertTrue(t, relationship == true, "Is child")
	test_utils.AssertTrue(t, relationship2 == true, "Is child")
	test_utils.AssertTrue(t, relationship3 == false, "Is not child")
	test_utils.AssertTrue(t, err == nil, "GetChildDatatypes should not have returned an error")
	test_utils.AssertTrue(t, err2 == nil, "GetChildDatatypes should not have returned an error")
	test_utils.AssertTrue(t, err3 == nil, "GetChildDatatypes should not have returned an error")
	test_utils.AssertTrue(t, err4 == nil, "IsChildOf should not have returned an error")
	test_utils.AssertTrue(t, err5 == nil, "IsChildOf should not have returned an error")
	test_utils.AssertTrue(t, err6 == nil, "IsChildOf should not have returned an error")
}

// Tests GetChildParents and IsParentOf funcs
func TestParentRelationships(t *testing.T) {
	logger.Info("TestParentRelationships function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	// enable putCache so that multiple transactions can be submitted
	stub := cached_stub.NewCachedStub(mstub, true, true)
	datatype1, err := RegisterDatatypeWithParams(stub, "myDatatype1", "", true, ROOT_DATATYPE_ID)
	datatype2, err2 := RegisterDatatypeWithParams(stub, "myDatatype2", "", true, datatype1.GetDatatypeID())
	datatype3, err3 := RegisterDatatypeWithParams(stub, "myDatatype3", "", true, datatype1.GetDatatypeID())
	datatype4, err4 := RegisterDatatypeWithParams(stub, "myDatatype4", "", true, datatype2.GetDatatypeID())
	datatype5, err5 := RegisterDatatypeWithParams(stub, "myDatatype5", "", true, datatype2.GetDatatypeID())
	mstub.MockTransactionEnd("t1")

	test_utils.AssertTrue(t, err == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err2 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err3 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err4 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err5 == nil, "RegisterDatatypeWithParams should not have returned an error")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	parents, err := datatype5.GetParentDatatypes(stub)
	expectedParents := []string{datatype2.GetDatatypeID(), datatype1.GetDatatypeID()}
	parents2, err2 := datatype2.GetParentDatatypes(stub)
	expectedParents2 := []string{datatype1.GetDatatypeID()}
	parents3, err3 := datatype1.GetParentDatatypes(stub)
	expectedParents3 := []string{}

	relationship, err4 := datatype1.IsParentOf(stub, datatype5.GetDatatypeID())
	relationship2, err5 := datatype2.IsParentOf(stub, datatype5.GetDatatypeID())
	relationship3, err6 := datatype3.IsParentOf(stub, datatype5.GetDatatypeID())
	relationship4, err7 := datatype4.IsParentOf(stub, datatype5.GetDatatypeID())
	mstub.MockTransactionEnd("t1")

	test_utils.AssertSetsEqual(t, expectedParents, parents)
	test_utils.AssertSetsEqual(t, expectedParents2, parents2)
	test_utils.AssertSetsEqual(t, expectedParents3, parents3)
	test_utils.AssertTrue(t, relationship == true, "Is parent")
	test_utils.AssertTrue(t, relationship2 == true, "Is parent")
	test_utils.AssertTrue(t, relationship3 == false, "Is not parent")
	test_utils.AssertTrue(t, relationship4 == false, "Is not parent")
	test_utils.AssertTrue(t, err == nil, "GetParentDatatypes should not have returned an error")
	test_utils.AssertTrue(t, err2 == nil, "GetParentDatatypes should not have returned an error")
	test_utils.AssertTrue(t, err3 == nil, "GetParentDatatypes should not have returned an error")
	test_utils.AssertTrue(t, err4 == nil, "IsParentOf should not have returned an error")
	test_utils.AssertTrue(t, err5 == nil, "IsParentOf should not have returned an error")
	test_utils.AssertTrue(t, err6 == nil, "IsParentOf should not have returned an error")
	test_utils.AssertTrue(t, err7 == nil, "IsParentOf should not have returned an error")
}

// Tests remove datatype
func TestRemoveDatatype(t *testing.T) {
	logger.Info("TestRemoveDatatype function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	// enable putCache so that multiple transactions can be submitted
	stub := cached_stub.NewCachedStub(mstub, true, true)
	datatype1, err := RegisterDatatypeWithParams(stub, "myDatatype1", "", true, ROOT_DATATYPE_ID)
	datatype2, err2 := RegisterDatatypeWithParams(stub, "myDatatype2", "", true, datatype1.GetDatatypeID())
	datatype3, err3 := RegisterDatatypeWithParams(stub, "myDatatype3", "", true, datatype1.GetDatatypeID())
	datatype4, err4 := RegisterDatatypeWithParams(stub, "myDatatype4", "", true, datatype2.GetDatatypeID())
	datatype5, err5 := RegisterDatatypeWithParams(stub, "myDatatype5", "", true, datatype2.GetDatatypeID())
	mstub.MockTransactionEnd("t1")

	test_utils.AssertTrue(t, err == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err2 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err3 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err4 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err5 == nil, "RegisterDatatypeWithParams should not have returned an error")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	datatype2Impl := datatypeImpl{DatatypeID: datatype2.GetDatatypeID(), Description: datatype2.GetDescription(), Active: datatype2.IsActive(), deactivated: false}
	err = datatype2Impl.removeDatatype(stub, data_model.User{})
	mstub.MockTransactionEnd("t1")

	test_utils.AssertTrue(t, err == nil, "RemoveDatatype should not have returned an error")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	parents, err := datatype5.GetParentDatatypes(stub)
	expectedParents := []string{datatype1.GetDatatypeID()}
	parents2, err2 := datatype4.GetParentDatatypes(stub)
	expectedParents2 := []string{datatype1.GetDatatypeID()}
	parents3, err3 := datatype2.GetParentDatatypes(stub)
	expectedParents3 := []string{}

	relationship, err4 := datatype4.IsChildOf(stub, datatype1.GetDatatypeID())
	relationship2, err5 := datatype5.IsChildOf(stub, datatype1.GetDatatypeID())
	relationship3, err6 := datatype2.IsChildOf(stub, datatype1.GetDatatypeID())
	relationship4, err7 := datatype4.IsChildOf(stub, datatype2.GetDatatypeID())
	relationship5, err8 := datatype3.IsChildOf(stub, datatype1.GetDatatypeID())
	mstub.MockTransactionEnd("t1")

	test_utils.AssertSetsEqual(t, expectedParents, parents)
	test_utils.AssertSetsEqual(t, expectedParents2, parents2)
	test_utils.AssertSetsEqual(t, expectedParents3, parents3)
	test_utils.AssertTrue(t, relationship == true, "Is child")
	test_utils.AssertTrue(t, relationship2 == true, "Is child")
	test_utils.AssertTrue(t, relationship3 == false, "Is not child because datatype has been deleted")
	test_utils.AssertTrue(t, relationship4 == false, "Is not child because datatype has been deleted")
	test_utils.AssertTrue(t, relationship5 == true, "Is child")
	test_utils.AssertTrue(t, err == nil, "GetParentDatatypes should not have returned an error")
	test_utils.AssertTrue(t, err2 == nil, "GetParentDatatypes should not have returned an error")
	test_utils.AssertTrue(t, err3 == nil, "GetParentDatatypes should not have returned an error")
	test_utils.AssertTrue(t, err4 == nil, "IsChildOf should not have returned an error")
	test_utils.AssertTrue(t, err5 == nil, "IsChildOf should not have returned an error")
	test_utils.AssertTrue(t, err6 == nil, "IsChildOf should not have returned an error")
	test_utils.AssertTrue(t, err7 == nil, "IsChildOf should not have returned an error")
	test_utils.AssertTrue(t, err8 == nil, "IsChildOf should not have returned an error")
}

func TestNormalizeDatatypes(t *testing.T) {
	logger.Info("TestNormalizeDatatypes function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	// enable putCache so that multiple transactions can be submitted
	stub := cached_stub.NewCachedStub(mstub, true, true)
	datatype1, err := RegisterDatatypeWithParams(stub, "myDatatype1", "", true, ROOT_DATATYPE_ID)
	datatype2, err2 := RegisterDatatypeWithParams(stub, "myDatatype2", "", true, ROOT_DATATYPE_ID)
	datatype3, err3 := RegisterDatatypeWithParams(stub, "myDatatype3", "", true, datatype1.GetDatatypeID())
	datatype4, err4 := RegisterDatatypeWithParams(stub, "myDatatype4", "", true, datatype2.GetDatatypeID())
	datatype5, err5 := RegisterDatatypeWithParams(stub, "myDatatype5", "", true, datatype2.GetDatatypeID())
	mstub.MockTransactionEnd("t1")

	test_utils.AssertTrue(t, err == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err2 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err3 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err4 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err5 == nil, "RegisterDatatypeWithParams should not have returned an error")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	datatypesA := []string{datatype1.GetDatatypeID(), datatype3.GetDatatypeID(), datatype5.GetDatatypeID()}
	expectedNormalizedDatatypesA := []string{datatype3.GetDatatypeID(), datatype5.GetDatatypeID()}
	normalizedDatatypesA, err := NormalizeDatatypes(stub, datatypesA)
	test_utils.AssertTrue(t, err == nil, "NormalizeDatatypes should be successful")
	test_utils.AssertSetsEqual(t, expectedNormalizedDatatypesA, normalizedDatatypesA)

	datatypesB := []string{datatype1.GetDatatypeID(), datatype2.GetDatatypeID(), datatype3.GetDatatypeID(), datatype4.GetDatatypeID()}
	expectedNormalizedDatatypesB := []string{datatype3.GetDatatypeID(), datatype4.GetDatatypeID()}
	normalizedDatatypesB, err := NormalizeDatatypes(stub, datatypesB)
	test_utils.AssertTrue(t, err == nil, "NormalizeDatatypes should be successful")
	test_utils.AssertSetsEqual(t, expectedNormalizedDatatypesB, normalizedDatatypesB)

	datatypesC := []string{datatype2.GetDatatypeID(), datatype4.GetDatatypeID()}
	expectedNormalizedDatatypesC := []string{datatype4.GetDatatypeID()}
	normalizedDatatypesC, err := NormalizeDatatypes(stub, datatypesC)
	test_utils.AssertTrue(t, err == nil, "NormalizeDatatypes should be successful")
	test_utils.AssertSetsEqual(t, expectedNormalizedDatatypesC, normalizedDatatypesC)

	datatypesD := []string{datatype1.GetDatatypeID(), datatype4.GetDatatypeID(), datatype5.GetDatatypeID()}
	expectedNormalizedDatatypesD := []string{datatype1.GetDatatypeID(), datatype4.GetDatatypeID(), datatype5.GetDatatypeID()}
	normalizedDatatypesD, err := NormalizeDatatypes(stub, datatypesD)
	test_utils.AssertTrue(t, err == nil, "NormalizeDatatypes should be successful")
	test_utils.AssertSetsEqual(t, expectedNormalizedDatatypesD, normalizedDatatypesD)
	mstub.MockTransactionEnd("t1")
}

// Tests deactivate
func TestDataypeInterface_Deactivate(t *testing.T) {
	logger.Info("TestDataypeInterface_Deactivate function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	// enable putCache so that multiple transactions can be submitted
	stub := cached_stub.NewCachedStub(mstub, true, true)
	datatype1, err := RegisterDatatypeWithParams(stub, "myDatatype1", "", true, ROOT_DATATYPE_ID)
	datatype2, err2 := RegisterDatatypeWithParams(stub, "myDatatype2", "", true, datatype1.GetDatatypeID())
	datatype3, err3 := RegisterDatatypeWithParams(stub, "myDatatype3", "", true, datatype1.GetDatatypeID())
	datatype4, err4 := RegisterDatatypeWithParams(stub, "myDatatype4", "", true, datatype2.GetDatatypeID())
	datatype5, err5 := RegisterDatatypeWithParams(stub, "myDatatype5", "", true, datatype2.GetDatatypeID())
	mstub.MockTransactionEnd("t1")

	test_utils.AssertTrue(t, err == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err2 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err3 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err4 == nil, "RegisterDatatypeWithParams should not have returned an error")
	test_utils.AssertTrue(t, err5 == nil, "RegisterDatatypeWithParams should not have returned an error")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = datatype2.Deactivate()
	err = datatype2.PutDatatype(stub)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "PutDatatype should not have returned an error")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	datatype1, err = GetDatatypeWithParams(stub, "myDatatype1")
	test_utils.AssertTrue(t, datatype1.IsActive(), "myDatatype1 should be active")

	datatype2, err = GetDatatypeWithParams(stub, "myDatatype2")
	test_utils.AssertTrue(t, !datatype2.IsActive(), "myDatatype2 should be inactive")

	datatype3, err = GetDatatypeWithParams(stub, "myDatatype3")
	test_utils.AssertTrue(t, datatype3.IsActive(), "myDatatype3 should be active")

	datatype4, err = GetDatatypeWithParams(stub, "myDatatype4")
	test_utils.AssertTrue(t, !datatype4.IsActive(), "myDatatype4 should be inactive")

	datatype5, err = GetDatatypeWithParams(stub, "myDatatype5")
	test_utils.AssertTrue(t, !datatype5.IsActive(), "myDatatype5 should be inactive")
	mstub.MockTransactionEnd("t1")
}
