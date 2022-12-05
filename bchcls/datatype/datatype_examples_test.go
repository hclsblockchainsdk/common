/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package datatype

import (
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"

	//"common/bchcls/internal/datatype_i/datatype_interface"
	"common/bchcls/test_utils"

	"encoding/json"
	"fmt"
	"testing"
)

func ExampleGetDatatypeKeyID() {
	datatypeKeyId := GetDatatypeKeyID("datatype1", "owner1")

	fmt.Println(datatypeKeyId)
	// Output: sym-owner1-datatype1
}

func ExampleRegisterDatatype(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	caller := test_utils.CreateTestUser("caller")

	datatype1 := data_model.Datatype{DatatypeID: "datatype1", Description: "description", IsActive: true}
	datatype1Bytes, _ := json.Marshal(&datatype1)
	parentDatatypeID := "datatype0"

	// registers datatype
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	RegisterDatatype(stub, caller, []string{string(datatype1Bytes), parentDatatypeID})
	mstub.MockTransactionEnd("t1")
}

func ExampleRegisterDatatypeWithParam(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)

	datatype1 := data_model.Datatype{DatatypeID: "datatype1", Description: "description", IsActive: true}
	parentDatatypeID := "datatype0"

	// registers datatype
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	RegisterDatatypeWithParams(stub, datatype1.DatatypeID, datatype1.Description, datatype1.IsActive, parentDatatypeID)
	mstub.MockTransactionEnd("t1")
}

func ExampleUpdateDatatype(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	caller := test_utils.CreateTestUser("caller")

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	var datatype1 data_model.Datatype
	datatype1Bytes, _ := GetDatatype(stub, caller, []string{"datatype1"})
	json.Unmarshal(datatype1Bytes, &datatype1)

	datatype1.Description = "updated description"
	datatype1Bytes, _ = json.Marshal(datatype1)
	mstub.MockTransactionEnd("t1")

	// updates datatype
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	UpdateDatatype(stub, caller, []string{string(datatype1Bytes)})
	mstub.MockTransactionEnd("t1")
}

func ExampleGetDatatype(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	caller := test_utils.CreateTestUser("caller")
	stub := cached_stub.NewCachedStub(mstub)

	// returns datatype
	GetDatatype(stub, caller, []string{"datatype1"})
}

func ExampleGetDatatypeWithParam(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	// returns datatype
	GetDatatypeWithParams(stub, "datatype1")
}

func ExampleGetAllDatatypes(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	caller := test_utils.CreateTestUser("caller")
	stub := cached_stub.NewCachedStub(mstub)

	// returns all datatypes
	GetAllDatatypes(stub, caller, []string{})
}

func ExampleAddDatatypeSymKey(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	caller := test_utils.CreateTestUser("caller")
	stub := cached_stub.NewCachedStub(mstub)

	// add datatype sym key for owner1
	AddDatatypeSymKey(stub, caller, "datatype1", "owner1")
}

func ExampleGetDatatypeSymKey(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	caller := test_utils.CreateTestUser("caller")
	stub := cached_stub.NewCachedStub(mstub)

	// returns datatype sym key for owner1
	GetDatatypeSymKey(stub, caller, "datatype1", "owner1")
}

func ExampleGetParentDatatype(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	// returns datatypeID of the parent datatype
	GetParentDatatype(stub, "datatype1")
}

func ExampleNormalizeDatatypes(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	RegisterDatatypeWithParams(stub, "datatype1", "datatype1 description", true, ROOT_DATATYPE_ID)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	RegisterDatatypeWithParams(stub, "datatype2", "datatype2 description", true, "datatype1")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	RegisterDatatypeWithParams(stub, "datatype3", "datatype3 description", true, "datatype2")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	RegisterDatatypeWithParams(stub, "datatype4", "datatype4 description", true, ROOT_DATATYPE_ID)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	datatypes := []string{"datatype1", "datatype2", "datatype3", "datatype4"}

	// returns []string{"datatype3", "datatype4"}
	NormalizeDatatypes(stub, datatypes)
	mstub.MockTransactionEnd("t1")
}

func ExampleDatatypeImpl_GetDatatypeStruct(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	datatype1, _ := GetDatatypeWithParams(stub, "datatype1")

	// returns Datatype object
	datatype1.GetDatatypeStruct()
}

func ExampleDatatypeImpl_GetDatatypeID(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	datatype1, _ := GetDatatypeWithParams(stub, "datatype1")

	// returns DatatypeID
	datatype1.GetDatatypeID()
}

func ExampleDatatypeImpl_GetDescription(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	datatype1, _ := GetDatatypeWithParams(stub, "datatype1")

	// returns Description
	datatype1.GetDescription()
}

func ExampleDatatypeImpl_SetDescription(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	datatype1, _ := GetDatatypeWithParams(stub, "datatype1")

	// sets Description
	datatype1.SetDescription("new description")
}

func ExampleDatatypeImpl_Activate(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	datatype1, _ := GetDatatypeWithParams(stub, "datatype1")

	// changes state to active
	datatype1.Activate()
}

func ExampleDatatypeImpl_Deactivate(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	datatype1, _ := GetDatatypeWithParams(stub, "datatype1")

	// changes state to inactive
	datatype1.Deactivate()
}

func ExampleDatatypeImpl_GetChildDatatypes(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	datatype1, _ := GetDatatypeWithParams(stub, "datatype1")

	// returns child datatypes of datatype1
	datatype1.GetChildDatatypes(stub)
}

func ExampleDatatypeImpl_GetParentDatatypeID(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	datatype1, _ := GetDatatypeWithParams(stub, "datatype1")

	// returns child datatypes of datatype1
	datatype1.GetParentDatatypeID(stub)
}

func ExampleDatatypeImpl_GetParentDatatypes(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	datatype1, _ := GetDatatypeWithParams(stub, "datatype1")

	// returns parent datatypes of datatype1
	datatype1.GetParentDatatypes(stub)
}

func ExampleDatatypeImpl_IsChildOf(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	datatype1, _ := RegisterDatatypeWithParams(stub, "datatype1", "datatype1 description", true, ROOT_DATATYPE_ID)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	datatype2, _ := RegisterDatatypeWithParams(stub, "datatype2", "datatype2 description", true, "datatype1")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// returns false
	datatype1.IsChildOf(stub, "datatype2")

	// returns true
	datatype2.IsChildOf(stub, "datatype1")
	mstub.MockTransactionEnd("t1")
}

func ExampleDatatypeImpl_IsParentOf(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	datatype1, _ := RegisterDatatypeWithParams(stub, "datatype1", "datatype1 description", true, ROOT_DATATYPE_ID)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	datatype2, _ := RegisterDatatypeWithParams(stub, "datatype2", "datatype2 description", true, "datatype1")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	// returns true
	datatype1.IsParentOf(stub, "datatype2")

	// returns false
	datatype2.IsParentOf(stub, "datatype1")
	mstub.MockTransactionEnd("t1")
}

func ExampleDatatypeImpl_GetDatatypeKeyID(t *testing.T) {
	mstub := test_utils.CreateNewMockStub(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	datatype1, _ := RegisterDatatypeWithParams(stub, "datatype1", "datatype1 description", true, ROOT_DATATYPE_ID)
	mstub.MockTransactionEnd("t1")

	// returns sym-owner1-datatype1
	datatype1.GetDatatypeKeyID("owner1")
}
