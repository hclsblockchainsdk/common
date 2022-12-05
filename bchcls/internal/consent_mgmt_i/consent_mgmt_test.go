/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package consent_mgmt_i

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/internal/asset_mgmt_i"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/datastore_i/datastore_c"
	"common/bchcls/internal/datastore_i/datastore_c/cloudant/cloudant_datastore_test_utils"
	"common/bchcls/internal/datatype_i"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/simple_rule"
	"common/bchcls/test_utils"
	"common/bchcls/user_mgmt"

	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func generateConsent(ownerID, targetID, access, datatypeID, connectionID string) data_model.Consent {
	consent := data_model.Consent{}
	consent.OwnerID = ownerID
	consent.TargetID = targetID
	consent.DatatypeID = datatypeID
	consent.Access = access
	consent.ConsentDate = time.Now().Unix()
	consent.ExpirationDate = consent.ConsentDate + 60*60*24
	consent.Data = make(map[string]interface{})
	consent.ConnectionID = connectionID
	return consent
}

func setup(t *testing.T) *test_utils.NewMockStub {
	logger.SetLevel(shim.LogDebug)
	mstub := test_utils.CreateNewMockStub(t)
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user_mgmt.Init(stub, shim.LogDebug)
	asset_mgmt_i.Init(stub, shim.LogDebug)
	datatype_i.Init(stub, shim.LogDebug)
	datastore_c.Init(stub, shim.LogDebug)
	key_mgmt_i.Init(stub, shim.LogDebug)
	Init(stub)
	mstub.MockTransactionEnd("t1")
	return mstub
}

func TestPutConsent(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutConsent function called")

	// create a MockStub
	mstub := setup(t)

	// Create caller - system admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// add datatype sym key
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype1", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create target user
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target := test_utils.CreateTestUser("target1")
	targetBytes, _ := json.Marshal(&target)
	_, err = user_mgmt.RegisterUser(stub, target, []string{string(targetBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	consent := generateConsent("caller", "target1", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ := json.Marshal(&consent)

	// Create consent key
	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Update consent permission to deny
	consent = generateConsent("caller", "target1", global.ACCESS_DENY, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes)}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Update consent permission to read
	consent = generateConsent("caller", "target1", global.ACCESS_READ, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes)}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Update consent permission to write
	consent = generateConsent("caller", "target1", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes)}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Create unauthorized caller
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	badCaller := test_utils.CreateTestUser("badCaller")
	badCallerBytes, _ := json.Marshal(&badCaller)
	_, err = user_mgmt.RegisterUser(stub, badCaller, []string{string(badCallerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// add datatype sym key
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, badCaller, "datatype1", badCaller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Update consent permission to read
	consent = generateConsent("badCaller", "target1", global.ACCESS_READ, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes)}
	_, err = PutConsent(stub, badCaller, args)
	test_utils.AssertTrue(t, err != nil, "PutConsent should fail as bad caller has no access to consentKey")
	mstub.MockTransactionEnd("t1")

	// Register another caller and give consent target for datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	caller2 := test_utils.CreateTestUser("caller2")
	caller2Bytes, _ := json.Marshal(&caller2)
	_, err = user_mgmt.RegisterUser(stub, caller, []string{string(caller2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// add datatype sym key
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller2, "datatype1", caller2.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	consent2 := generateConsent("caller2", "target1", global.ACCESS_WRITE, "datatype1", "")
	consent2Bytes, _ := json.Marshal(&consent2)

	// Create consent key
	consentKey2 := test_utils.GenerateSymKey()
	consentKey2B64 := crypto.EncodeToB64String(consentKey2)

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consent2Bytes), consentKey2B64}
	_, err = PutConsent(stub, caller2, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")
}

func TestPutConsent_InactiveDatatype(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutConsent_InactiveDatatype function called")

	// create a MockStub
	mstub := setup(t)

	// Create caller - system admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register inactive datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", false, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// add datatype sym key
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype1", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create target user
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target := test_utils.CreateTestUser("target1")
	targetBytes, _ := json.Marshal(&target)
	_, err = user_mgmt.RegisterUser(stub, target, []string{string(targetBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add write consent to datatype1
	consent := generateConsent("caller", "target1", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ := json.Marshal(&consent)

	// Create consent key
	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	// This should fail
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err != nil, "PutConsent should not be successful")
	mstub.MockTransactionEnd("t1")

	// Update consent permission to deny
	consent = generateConsent("caller", "target1", global.ACCESS_DENY, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Update consent permission to read (this should fail)
	consent = generateConsent("caller", "target1", global.ACCESS_READ, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes)}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err != nil, "PutConsent should not be successful")
	mstub.MockTransactionEnd("t1")

}

func TestPutConsent_OffChain(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutConsent_OffChain function called")

	// create a MockStub
	mstub := setup(t)

	// Create caller - system admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// setup cloudant datastore
	datastoreConnectionID := "cloudant1"
	err = cloudant_datastore_test_utils.SetupDatastore(mstub, caller, datastoreConnectionID)
	test_utils.AssertTrue(t, err == nil, "SetupDatastore should be successful")

	// Register datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// add datatype sym key
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype1", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create target user
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target := test_utils.CreateTestUser("target1")
	targetBytes, _ := json.Marshal(&target)
	_, err = user_mgmt.RegisterUser(stub, target, []string{string(targetBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	consent := generateConsent("caller", "target1", global.ACCESS_WRITE, "datatype1", datastoreConnectionID)
	consentBytes, _ := json.Marshal(&consent)

	// Create consent key
	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	// Save new consent offchain
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")
}

func TestPutConsent_OffChain_NoDatastore(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutConsent_OffChain_NoDatastore function called")

	// create a MockStub
	mstub := setup(t)

	// Create caller - system admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// add datatype sym key
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype1", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create target user
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target := test_utils.CreateTestUser("target1")
	targetBytes, _ := json.Marshal(&target)
	_, err = user_mgmt.RegisterUser(stub, target, []string{string(targetBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	datastoreConnectionID := "invalid"
	consent := generateConsent("caller", "target1", global.ACCESS_WRITE, "datatype1", datastoreConnectionID)
	consentBytes, _ := json.Marshal(&consent)

	// Create consent key
	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	// Attempt to save new consent offchain
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	args := []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err != nil, "PutConsent should fail")
	expectedErrorMsg := fmt.Sprintf("DatastoreConnection with ID %v does not exist.", datastoreConnectionID)
	test_utils.AssertTrue(t, strings.Contains(err.Error(), expectedErrorMsg), fmt.Sprintf("Expected error message: %v", expectedErrorMsg))
	mstub.MockTransactionEnd("t1")
}

func TestGetConsent_OffChain(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetConsent_OffChain function called")

	// create a MockStub
	mstub := setup(t)

	// Create caller - system admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Setup cloudant datastore
	datastoreConnectionID := "cloudant1"
	err = cloudant_datastore_test_utils.SetupDatastore(mstub, caller, datastoreConnectionID)
	test_utils.AssertTrue(t, err == nil, "SetupDatastore should be successful")

	// Register datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// add datatype sym key
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype1", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create target user
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target := test_utils.CreateTestUser("target1")
	targetBytes, _ := json.Marshal(&target)
	_, err = user_mgmt.RegisterUser(stub, target, []string{string(targetBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	consent := generateConsent("caller", "target1", global.ACCESS_WRITE, "datatype1", datastoreConnectionID)
	consentBytes, _ := json.Marshal(&consent)

	// Create consent key
	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	// Create new consent
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// GetConsent as target
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{"datatype1", "target1", "caller"}
	consentResult := data_model.Consent{}
	consentResultBytes, err := GetConsent(stub, target, args)
	json.Unmarshal(consentResultBytes, &consentResult)
	test_utils.AssertTrue(t, err == nil, "GetConsent should be successful")
	test_utils.AssertTrue(t, consentResult.Access == global.ACCESS_WRITE, "Expected private data Access")
	test_utils.AssertTrue(t, consentResult.ConsentDate > 0, "Expected private data ConsentDate")
	test_utils.AssertTrue(t, consentResult.ExpirationDate > 0, "Expected private data ExpirationDate")
	test_utils.AssertTrue(t, consentResult.ConnectionID == datastoreConnectionID, "Expected public ConnectionID")
	mstub.MockTransactionEnd("t1")
}

func TestGetConsent(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetConsent function called")

	// create a MockStub
	mstub := setup(t)

	// Create caller - system admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// Add datatype sym keys
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype1", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create random user
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	random := test_utils.CreateTestUser("random")
	randombytes, _ := json.Marshal(&random)
	_, err = user_mgmt.RegisterUser(stub, random, []string{string(randombytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create target user
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target := test_utils.CreateTestUser("target1")
	targetBytes, _ := json.Marshal(&target)
	_, err = user_mgmt.RegisterUser(stub, target, []string{string(targetBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	consent := generateConsent("caller", "target1", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ := json.Marshal(&consent)

	// Create consent key
	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// GetConsent as random user, should fail
	args = []string{"datatype1", "target1", "caller"}
	_, err = GetConsent(stub, random, args)
	test_utils.AssertTrue(t, err != nil, "GetConsent should not be successful")

	// GetConsent as target
	args = []string{"datatype1", "target1", "caller"}
	consentResult := data_model.Consent{}
	consentResultBytes, err := GetConsent(stub, target, args)
	json.Unmarshal(consentResultBytes, &consentResult)
	test_utils.AssertTrue(t, err == nil, "GetConsent should be successful")
	test_utils.AssertTrue(t, consentResult.Access == global.ACCESS_WRITE, "Expected private data Access")
	test_utils.AssertTrue(t, consentResult.ConsentDate > 0, "Expected private data ConsentDate")
	test_utils.AssertTrue(t, consentResult.ExpirationDate > 0, "Expected private data ExpirationDate")
	mstub.MockTransactionEnd("t1")
}

func TestValidateConsent(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestValidateConsent function called")

	// create a MockStub
	mstub := setup(t)

	// Create caller - system admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create target user
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target := test_utils.CreateTestUser("target1")
	targetBytes, _ := json.Marshal(&target)
	_, err = user_mgmt.RegisterUser(stub, target, []string{string(targetBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// Register datatype2	mstub.MockTransactionStart("t1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype2", "datatype2", true, "datatype1")
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// Register datatype3
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype3", "datatype3", true, "datatype1")
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// Register datatype4
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype4", "datatype4", true, "datatype1")
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// Register datatype5, parent is ROOT
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype5", "datatype5", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// Add datatype sym keys
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype1", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype2", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype3", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype4", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype5", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Add consent to datatype1
	consent := generateConsent("caller", "target1", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ := json.Marshal(&consent)

	// Create consent key
	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// ValidateConsent as target for datatype1
	currTime := time.Now().Unix()
	args = []string{"datatype1", "caller", "target1", global.ACCESS_READ, strconv.FormatInt(currTime, 10)}

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	filter, key, err := ValidateConsent(stub, target, args)
	test_utils.AssertTrue(t, err == nil, "ValidateConsent as target should be successful")
	test_utils.AssertTrue(t, len(key.ID) > 0, "ValidateConsent as target should be successful")

	// Create asset for testing
	assetData := test_utils.CreateTestAsset("asset1")
	assetData.OwnerIds = []string{"caller", "other_owner"}
	assetData.Datatypes = []string{"datatype1", "datatype2"}

	// Apply rules
	assetDataBytes, _ := json.Marshal(assetData)
	assetDataMap := make(map[string]interface{})
	_ = json.Unmarshal(assetDataBytes, &assetDataMap)
	m, e := filter.Apply(assetDataMap)
	logger.Debugf("%v %v %v", filter.GetExprJSON(), simple_rule.ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == simple_rule.D(true), "ok")

	// Right owner wrong datatypes
	assetData = test_utils.CreateTestAsset("asset2")
	assetData.OwnerIds = []string{"caller"}
	assetData.Datatypes = []string{"datatype5"}

	// Apply rules, should fail
	assetDataBytes, _ = json.Marshal(assetData)
	assetDataMap = make(map[string]interface{})
	_ = json.Unmarshal(assetDataBytes, &assetDataMap)
	m, e = filter.Apply(assetDataMap)
	logger.Debugf("%v %v %v", filter.GetExprJSON(), simple_rule.ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == simple_rule.D(false), "ok")

	// Right datatypes wrong owner
	assetData = test_utils.CreateTestAsset("asset2")
	assetData.OwnerIds = []string{"caller2"}
	assetData.Datatypes = []string{"datatype1"}

	// Apply rules, should fail
	assetDataBytes, _ = json.Marshal(assetData)
	assetDataMap = make(map[string]interface{})
	_ = json.Unmarshal(assetDataBytes, &assetDataMap)
	m, e = filter.Apply(assetDataMap)
	logger.Debugf("%v %v %v", filter.GetExprJSON(), simple_rule.ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == simple_rule.D(false), "ok")

	// Intersection logic
	assetData = test_utils.CreateTestAsset("asset2")
	assetData.OwnerIds = []string{"caller"}
	assetData.Datatypes = []string{"datatype2", " datatype5"}

	// Apply rules
	assetDataBytes, _ = json.Marshal(assetData)
	assetDataMap = make(map[string]interface{})
	_ = json.Unmarshal(assetDataBytes, &assetDataMap)
	m, e = filter.Apply(assetDataMap)
	logger.Debugf("%v %v %v", filter.GetExprJSON(), simple_rule.ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == simple_rule.D(true), "ok")

	// ValidateConsent as target for datatype2, child of datatype1 where consent was given
	currTime = time.Now().Unix()
	args = []string{"datatype2", "caller", "target1", global.ACCESS_READ, strconv.FormatInt(currTime, 10)}

	filter, key, err = ValidateConsent(stub, target, args)
	logger.Debugf("filter: %v, key: %v, err: %v", filter, key, err)
	test_utils.AssertTrue(t, err == nil, "ValidateConsent as target should be successful")
	test_utils.AssertTrue(t, len(key.ID) > 0, "ValidateConsent as target should be successful")

	// Create asset for testing
	assetData = test_utils.CreateTestAsset("asset1")
	assetData.OwnerIds = []string{"caller", "other_owner"}
	assetData.Datatypes = []string{"datatype2"}

	// Apply rules
	assetDataBytes, _ = json.Marshal(assetData)
	assetDataMap = make(map[string]interface{})
	_ = json.Unmarshal(assetDataBytes, &assetDataMap)
	m, e = filter.Apply(assetDataMap)
	logger.Debugf("%v %v %v", filter.GetExprJSON(), simple_rule.ToJSON(m), e)
	test_utils.AssertTrue(t, m["$result"] == simple_rule.D(true), "ok")
	mstub.MockTransactionEnd("t1")

	// doing one more time, should get consentID from cache
	currTime = time.Now().Unix()
	args = []string{"datatype2", "caller", "target1", global.ACCESS_READ, strconv.FormatInt(currTime, 10)}

	filter, key, err = ValidateConsent(stub, target, args)
	logger.Debugf("filter: %v, key: %v, err: %v", filter, key, err)
	test_utils.AssertTrue(t, err == nil, "ValidateConsent as target should be successful")
	test_utils.AssertTrue(t, len(key.ID) > 0, "ValidateConsent as target should be successful")
}

func TestGetConsentsWithOwnerID(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetConsentsWithOwnerID function called")

	// create a MockStub
	mstub := setup(t)

	// Create caller - system admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	// Add datatype sym key
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype1", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create target user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target1 := test_utils.CreateTestUser("target1")
	target1Bytes, _ := json.Marshal(&target1)
	_, err = user_mgmt.RegisterUser(stub, target1, []string{string(target1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	consent := generateConsent("caller", "target1", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ := json.Marshal(&consent)

	// Create consent key
	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	// Add consent for target1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Create target user2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target2 := test_utils.CreateTestUser("target2")
	target2Bytes, _ := json.Marshal(&target2)
	_, err = user_mgmt.RegisterUser(stub, target2, []string{string(target2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	consent = generateConsent("caller", "target2", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Create target user3
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target3 := test_utils.CreateTestUser("target3")
	target3Bytes, _ := json.Marshal(&target3)
	_, err = user_mgmt.RegisterUser(stub, target3, []string{string(target3Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	consent = generateConsent("caller", "target3", global.ACCESS_DENY, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target3
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// test GetConsents
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	consents := []data_model.Consent{}
	consentsBytes, err := GetConsentsWithOwnerID(stub, caller, []string{"caller"})
	test_utils.AssertTrue(t, err == nil, "Expected GetConsentsWithOwnerID to succeed")
	err = json.Unmarshal(consentsBytes, &consents)
	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal consentsBytes to succeed")
	test_utils.AssertTrue(t, len(consents) == 3, "Expected to get exactly 3 consents")
	for _, consent := range consents {
		test_utils.AssertTrue(t, consent.OwnerID == "caller", "Expected OwnerID  to match")
	}
	mstub.MockTransactionEnd("t1")

	// add new consent as a different owner to same target
	// Create caller2 - system admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	caller2 := test_utils.CreateTestUser("caller2")
	caller2.Role = "system"
	caller2Bytes, _ := json.Marshal(&caller2)
	_, err = user_mgmt.RegisterUser(stub, caller2, []string{string(caller2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype2", "datatype2", true, datatype_i.ROOT_DATATYPE_ID)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller2, "datatype2", caller2.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Add consent to datatype2
	consent = generateConsent("caller2", "target1", global.ACCESS_WRITE, "datatype2", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller2, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// test GetConsents again
	// should still get the same result
	consentsBytes, err = GetConsentsWithOwnerID(stub, caller, []string{"caller"})
	test_utils.AssertTrue(t, err == nil, "Expected GetConsentsWithOwnerID to succeed")
	err = json.Unmarshal(consentsBytes, &consents)
	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal consentsBytes to succeed")
	test_utils.AssertTrue(t, len(consents) == 3, "Expected to get exactly 3 consents")
	for _, consent := range consents {
		test_utils.AssertTrue(t, consent.OwnerID == "caller", "Expected OwnerID  to match")
	}
}

func TestGetConsentsWithTargetID(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetConsentsWithTargetID function called")

	// create a MockStub
	mstub := setup(t)

	// Create caller1 - system admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller1 := test_utils.CreateTestUser("caller1")
	caller1.Role = "system"
	caller1Bytes, _ := json.Marshal(&caller1)
	_, err := user_mgmt.RegisterUser(stub, caller1, []string{string(caller1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller1, "datatype1", caller1.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create caller2 - system admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	caller2 := test_utils.CreateTestUser("caller2")
	caller2.Role = "system"
	caller2Bytes, _ := json.Marshal(&caller2)
	_, err = user_mgmt.RegisterUser(stub, caller2, []string{string(caller2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype2", "datatype2", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller2, "datatype2", caller2.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create target user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target1 := test_utils.CreateTestUser("target1")
	target1Bytes, _ := json.Marshal(&target1)
	_, err = user_mgmt.RegisterUser(stub, target1, []string{string(target1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Generate consent
	consent := generateConsent("caller1", "target1", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ := json.Marshal(&consent)

	// Create consent key
	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	// Add consent for target1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller1, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Generate consent
	consent = generateConsent("caller2", "target1", global.ACCESS_READ, "datatype2", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller2, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// test GetConsents
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	consents := []data_model.Consent{}
	consentsBytes, err := GetConsentsWithTargetID(stub, target1, []string{"target1"})
	test_utils.AssertTrue(t, err == nil, "Expected GetConsentsWithTargetID to succeed")
	err = json.Unmarshal(consentsBytes, &consents)
	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal consentsBytes to succeed")
	test_utils.AssertTrue(t, len(consents) == 2, "Expected to get exactly 2 consents")
	for _, consent := range consents {
		test_utils.AssertTrue(t, consent.TargetID == "target1", "Expected targetID  to match")
	}
	// Add consent to a different target
	// Create target user2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target2 := test_utils.CreateTestUser("target2")
	target2Bytes, _ := json.Marshal(&target2)
	_, err = user_mgmt.RegisterUser(stub, target2, []string{string(target2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Generate consent
	consent = generateConsent("caller1", "target2", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller1, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// test GetConsents again
	// Expecting same results as before
	consentsBytes, err = GetConsentsWithTargetID(stub, target1, []string{"target1"})
	test_utils.AssertTrue(t, err == nil, "Expected GetConsentsWithTargetID to succeed")
	err = json.Unmarshal(consentsBytes, &consents)
	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal consentsBytes to succeed")
	test_utils.AssertTrue(t, len(consents) == 2, "Expected to get exactly 2 consents")
	test_utils.AssertTrue(t, consents[0].Access == global.ACCESS_WRITE, "Expected consent 1's access to be write")
	test_utils.AssertTrue(t, consents[1].Access == global.ACCESS_READ, "Expected consent 2's access to be read")

}

func TestGetConsentsWithCallerID(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetConsentsWithCallerID function called")

	// create a MockStub
	mstub := setup(t)

	// Create caller - system admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype1", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create target user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target1 := test_utils.CreateTestUser("target1")
	target1Bytes, _ := json.Marshal(&target1)
	_, err = user_mgmt.RegisterUser(stub, target1, []string{string(target1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	consent := generateConsent("caller", "target1", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ := json.Marshal(&consent)

	// Create consent key
	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	// Add consent for target1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Create target user2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target2 := test_utils.CreateTestUser("target2")
	target2Bytes, _ := json.Marshal(&target2)
	_, err = user_mgmt.RegisterUser(stub, target2, []string{string(target2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	consent = generateConsent("caller", "target2", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Create target user3
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target3 := test_utils.CreateTestUser("target3")
	target3Bytes, _ := json.Marshal(&target3)
	_, err = user_mgmt.RegisterUser(stub, target3, []string{string(target3Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	consent = generateConsent("caller", "target3", global.ACCESS_DENY, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target3
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// test GetConsents
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	consents := []data_model.Consent{}
	consentsBytes, err := GetConsentsWithCallerID(stub, caller, []string{})
	test_utils.AssertTrue(t, err == nil, "Expected GetConsentsWithCallerID to succeed")
	err = json.Unmarshal(consentsBytes, &consents)
	logger.Debugf("Consents: %v", string(consentsBytes[:]))
	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal consentsBytes to succeed")
	test_utils.AssertTrue(t, len(consents) == 3, "Expected to get exactly 3 consents")
	for _, consent := range consents {
		logger.Debugf("ConcentID: %v, Access: %v, OwnerID: %v", consent.ConsentID, consent.Access, consent.OwnerID)
		test_utils.AssertTrue(t, consent.OwnerID == "caller", "Expected ownerID is caller")
	}
	mstub.MockTransactionEnd("t1")

	// Create another caller - system admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	caller2 := test_utils.CreateTestUser("caller2")
	caller2.Role = "system"
	caller2Bytes, _ := json.Marshal(&caller2)
	_, err = user_mgmt.RegisterUser(stub, caller2, []string{string(caller2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype2", "datatype2", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")
	// Add datatype sym key
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller2, "datatype2", caller2.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Add consent to datatype2 as caller2
	consent = generateConsent("caller2", "target3", global.ACCESS_DENY, "datatype2", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target3
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller2, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// test GetConsents again
	// expecting same results
	consentsBytes, err = GetConsentsWithCallerID(stub, caller, []string{})
	test_utils.AssertTrue(t, err == nil, "Expected GetConsentsWithCallerID to succeed")
	err = json.Unmarshal(consentsBytes, &consents)
	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal consentsBytes to succeed")
	test_utils.AssertTrue(t, len(consents) == 3, "Expected to get exactly 3 consents")
	for _, consent := range consents {
		test_utils.AssertTrue(t, consent.CreatorID == caller.ID, "Expected CreatorID is caller")
	}

}

func TestGetConsentsWithOwnerIDAndDatatypeID(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetConsentsWithOwnerIDAndDatatypeID function called")

	// create a MockStub
	mstub := setup(t)

	// Create caller - system admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	datatype1, err := datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")
	// Add datatype sym key
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller, "datatype1", caller.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create another owner - system admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	caller2 := test_utils.CreateTestUser("caller2")
	caller2.Role = "system"
	caller2Bytes, _ := json.Marshal(&caller2)
	_, err = user_mgmt.RegisterUser(stub, caller2, []string{string(caller2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	datatype2, err := datatype_i.RegisterDatatypeWithParams(stub, "datatype2", "datatype2", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")
	// Add datatype sym key
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller2, "datatype2", caller2.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create target user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target1 := test_utils.CreateTestUser("target1")
	target1Bytes, _ := json.Marshal(&target1)
	_, err = user_mgmt.RegisterUser(stub, target1, []string{string(target1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	consent := generateConsent("caller", "target1", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ := json.Marshal(&consent)

	// Create consent key
	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	// Add consent for target1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Create target user2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target2 := test_utils.CreateTestUser("target2")
	target2Bytes, _ := json.Marshal(&target2)
	_, err = user_mgmt.RegisterUser(stub, target2, []string{string(target2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	consent = generateConsent("caller", "target2", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Create target user3
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target3 := test_utils.CreateTestUser("target3")
	target3Bytes, _ := json.Marshal(&target3)
	_, err = user_mgmt.RegisterUser(stub, target3, []string{string(target3Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Add consent to datatype1
	consent = generateConsent("caller", "target3", global.ACCESS_DENY, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target3
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// test GetConsents
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	consents := []data_model.Consent{}
	consentsBytes, err := GetConsentsWithOwnerIDAndDatatypeID(stub, caller, []string{caller.ID, datatype1.GetDatatypeID()})
	test_utils.AssertTrue(t, err == nil, "Expected GetConsentsWithOwnerID to succeed")
	err = json.Unmarshal(consentsBytes, &consents)

	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal consentsBytes to succeed")
	test_utils.AssertTrue(t, len(consents) == 3, "Expected to get exactly 3 consents")

	for _, consent := range consents {
		test_utils.AssertTrue(t, consent.OwnerID == caller.ID && consent.DatatypeID == datatype1.GetDatatypeID(), "Expected OwnerID and DatatypeID to match")
	}
	mstub.MockTransactionEnd("t1")

	// try again with different datatypeID
	consentsBytes, err = GetConsentsWithOwnerIDAndDatatypeID(stub, caller, []string{caller.ID, datatype2.GetDatatypeID()})
	test_utils.AssertTrue(t, err == nil, "Expected GetConsentsWithOwnerID to succeed")
	err = json.Unmarshal(consentsBytes, &consents)
	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal consentsBytes to succeed")
	test_utils.AssertTrue(t, len(consents) == 0, "Expected to get no consent")

	// Add consent to datatype2 as caller2
	consent = generateConsent("caller2", "target3", global.ACCESS_DENY, "datatype2", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target3
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller2, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// test GetConsents
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// expecting same result as before
	consentsBytes, err = GetConsentsWithOwnerIDAndDatatypeID(stub, caller, []string{caller.ID, datatype1.GetDatatypeID()})
	test_utils.AssertTrue(t, err == nil, "Expected GetConsentsWithOwnerID to succeed")
	err = json.Unmarshal(consentsBytes, &consents)

	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal consentsBytes to succeed")
	test_utils.AssertTrue(t, len(consents) == 3, "Expected to get exactly 3 consents")

	for _, consent := range consents {
		test_utils.AssertTrue(t, consent.OwnerID == caller.ID && consent.DatatypeID == datatype1.GetDatatypeID(), "Expected OwnerID and DatatypeID to match")
	}
	mstub.MockTransactionEnd("t1")
}

func TestGetConsentsWithTargetIDAndOwnerID(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetConsentsWithTargetIDAndOwnerID function called")

	// create a MockStub
	mstub := setup(t)

	// Create caller1 - system admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller1 := test_utils.CreateTestUser("caller1")
	caller1.Role = "system"
	caller1Bytes, _ := json.Marshal(&caller1)
	_, err := user_mgmt.RegisterUser(stub, caller1, []string{string(caller1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype1", "datatype1", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")
	// Add datatype symkey
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller1, "datatype1", caller1.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create caller2 - system admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	caller2 := test_utils.CreateTestUser("caller2")
	caller2.Role = "system"
	caller2Bytes, _ := json.Marshal(&caller2)
	_, err = user_mgmt.RegisterUser(stub, caller2, []string{string(caller2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register datatype2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype2", "datatype2", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")
	// Add datatype symkeys
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller2, "datatype2", caller2.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller1, "datatype2", caller1.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Register datatype3
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.RegisterDatatypeWithParams(stub, "datatype3", "datatype3", true, datatype_i.ROOT_DATATYPE_ID)
	test_utils.AssertTrue(t, err == nil, "RegisterDatatype should be successful")
	mstub.MockTransactionEnd("t1")
	// Add datatype symkey
	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = datatype_i.AddDatatypeSymKey(stub, caller1, "datatype3", caller1.ID)
	test_utils.AssertTrue(t, err == nil, "AddDatatypeSymKey should be successful")
	mstub.MockTransactionEnd("t123")

	// Create target user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target1 := test_utils.CreateTestUser("target1")
	target1Bytes, _ := json.Marshal(&target1)
	_, err = user_mgmt.RegisterUser(stub, target1, []string{string(target1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Generate consent
	consent := generateConsent("caller1", "target1", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ := json.Marshal(&consent)

	// Create consent key
	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	// Add consent for target1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller1, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Generate consent
	consent = generateConsent("caller1", "target1", global.ACCESS_WRITE, "datatype2", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller1, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Generate consent
	consent = generateConsent("caller1", "target1", global.ACCESS_READ, "datatype3", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller1, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// Generate consent
	consent = generateConsent("caller2", "target1", global.ACCESS_READ, "datatype2", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target1, caller2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller2, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// test GetConsents
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	consents := []data_model.Consent{}
	consentsBytes, err := GetConsentsWithTargetIDAndOwnerID(stub, caller1, []string{"target1", "caller1"})
	test_utils.AssertTrue(t, err == nil, "Expected GetConsentsWithTargetIDAndOwnerID to succeed")
	err = json.Unmarshal(consentsBytes, &consents)
	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal consentsBytes to succeed")
	test_utils.AssertTrue(t, len(consents) == 3, "Expected to get exactly 3 consents")
	for _, consent := range consents {
		test_utils.AssertTrue(t, consent.OwnerID == "caller1" && consent.TargetID == "target1", "Expected OwnerID and TargetID to match")
	}

	consentsBytes, err = GetConsentsWithTargetIDAndOwnerID(stub, caller2, []string{"target1", "caller2"})
	test_utils.AssertTrue(t, err == nil, "Expected GetConsentsWithTargetIDAndOwnerID to succeed")
	err = json.Unmarshal(consentsBytes, &consents)
	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal consentsBytes to succeed")
	test_utils.AssertTrue(t, len(consents) == 1, "Expected to get exactly 1 consents")
	test_utils.AssertTrue(t, consents[0].Access == global.ACCESS_READ, "Expected consent 1's access to be read")
	mstub.MockTransactionEnd("t1")

	// Add consent to a different target
	// Create target user2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	target2 := test_utils.CreateTestUser("target2")
	target2Bytes, _ := json.Marshal(&target2)
	_, err = user_mgmt.RegisterUser(stub, target2, []string{string(target2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Generate consent
	consent = generateConsent("caller1", "target2", global.ACCESS_WRITE, "datatype1", "")
	consentBytes, _ = json.Marshal(&consent)

	// Create consent key
	consentKey = test_utils.GenerateSymKey()
	consentKeyB64 = crypto.EncodeToB64String(consentKey)

	// Add consent for target2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{string(consentBytes), consentKeyB64}
	_, err = PutConsent(stub, caller1, args)
	test_utils.AssertTrue(t, err == nil, "PutConsent should be successful")
	mstub.MockTransactionEnd("t1")

	// test GetConsents again
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Expecting same results as before
	consentsBytes, err = GetConsentsWithTargetIDAndOwnerID(stub, caller1, []string{"target1", "caller1"})
	test_utils.AssertTrue(t, err == nil, "Expected GetConsentsWithTargetIDAndOwnerID to succeed")
	err = json.Unmarshal(consentsBytes, &consents)
	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal consentsBytes to succeed")
	test_utils.AssertTrue(t, len(consents) == 3, "Expected to get exactly 3 consents")
	for _, consent := range consents {
		test_utils.AssertTrue(t, consent.OwnerID == "caller1" && consent.TargetID == "target1", "Expected OwnerID and TargetID to match")
	}
	mstub.MockTransactionEnd("t1")
}
