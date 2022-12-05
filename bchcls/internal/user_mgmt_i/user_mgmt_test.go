/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package user_mgmt_i

import (
	"common/bchcls/asset_mgmt/asset_manager"
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/index"
	"common/bchcls/internal/asset_mgmt_i"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/datastore_i"
	"common/bchcls/internal/datastore_i/datastore_c/cloudant/cloudant_datastore_test_utils"
	"common/bchcls/internal/datatype_i"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/internal/user_mgmt_i/user_mgmt_c"
	"common/bchcls/test_utils"
	"common/bchcls/utils"

	"crypto/rsa"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	//"github.com/pkg/errors"
)

// Call this before each test for stub setup
func setup(t *testing.T) *test_utils.NewMockStub {
	mstub := test_utils.CreateNewMockStub(t)
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	Init(stub)
	asset_mgmt_i.Init(stub)
	datatype_i.Init(stub)
	datastore_i.Init(stub)
	mstub.MockTransactionEnd("t1")
	logger.SetLevel(shim.LogDebug)
	return mstub
}

func RegisterUserForTest(t *testing.T, mstub *test_utils.NewMockStub, caller, user data_model.User, allowAccess bool) {
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	userBytes, _ := json.Marshal(&user)
	switch role := user.Role; role {
	case global.ROLE_USER:
		_, err := RegisterUser(stub, caller, []string{string(userBytes), strconv.FormatBool(allowAccess)})
		test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	case global.ROLE_ORG:
		//caller have to be system admin
		_, err := RegisterOrg(stub, caller, []string{string(userBytes), strconv.FormatBool(allowAccess)})
		test_utils.AssertTrue(t, err == nil, "Expected RegisterOrg to succeed")
	case global.ROLE_SYSTEM_ADMIN:
		_, err := RegisterSystemAdmin(stub, caller, []string{string(userBytes), strconv.FormatBool(allowAccess)})
		test_utils.AssertTrue(t, err == nil, "Expected RegisterSystemAdmin to succeed")
	case global.ROLE_AUDIT:
		_, err := RegisterAuditor(stub, caller, []string{string(userBytes), strconv.FormatBool(allowAccess)})
		test_utils.AssertTrue(t, err == nil, "Expected RegisterAuditor to succeed")
	default:
		test_utils.AssertTrue(t, false, "RegisterUserForTest was passed a user with invalid role")
	}
	mstub.MockTransactionEnd("t1")
}

func RegisterSubgroupForTest(t *testing.T, mstub *test_utils.NewMockStub, caller, subgroup data_model.User, parentGroupID string) {
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	subgroupBytes, _ := json.Marshal(&subgroup)
	args := []string{string(subgroupBytes), parentGroupID}
	_, err := RegisterSubgroup(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "Expected RegisterSubgroup to succeed")
	mstub.MockTransactionEnd("t1")
}

func TestCommitToLedgerAndGetUserData(t *testing.T) {
	logger.Info("TestCommitToLedgerAndGetUserData function called")
	// Create a MockStub
	mstub := setup(t)

	testUser := test_utils.CreateTestUser("testUserID")
	symKey1 := testUser.GetSymKey()

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)

	// Test commit new user to ledger, and test get user
	err := commitToLedger(stub, testUser, testUser, symKey1.ID, symKey1.KeyBytes, []string{}, true)
	test_utils.AssertTrue(t, err == nil, "Expected CommitToLedger to succeed")

	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	returnedUser, err := GetUserData(stub, testUser, testUser.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected GetUserData to succeed "+testUser.ID)
	test_utils.AssertTrue(t, userEqual(returnedUser, testUser), "Expected to get correct new user data")
	mstub.MockTransactionEnd("t1")

	// Test update user to the ledger

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	testUser.Name = "modifiedName"
	testUser.IsGroup = false
	testUser.Status = "inactive"
	err = commitToLedger(stub, testUser, testUser, symKey1.ID, symKey1.KeyBytes, []string{testUser.ID}, false)
	test_utils.AssertTrue(t, err == nil, "Expected CommitToLedger to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	returnedUser, err = GetUserData(stub, testUser, testUser.ID, false, true)
	test_utils.AssertTrue(t, userEqual(returnedUser, testUser), "Expected to get correct updated user data")
	test_utils.AssertTrue(t, returnedUser.Email == "email@mail.com", "Expected to get private email field")

	// Test Get user does not exist
	returnedUser, err = GetUserData(stub, testUser, "noExistID", false, true)
	test_utils.AssertTrue(t, err == nil, "Expected GetUserData to pass")
	test_utils.AssertTrue(t, len(returnedUser.ID) == 0, "ID should be empty")

	// test get user data without private data fields
	returnedUser, err = GetUserData(stub, testUser, testUser.ID, false, false)
	test_utils.AssertTrue(t, len(returnedUser.Email) == 0, "Expected to not get private email field")

	mstub.MockTransactionEnd("t1")
}

// owners of the asset are the user and caller and owners should not be updated when user is updated
func TestCommitToLedger_AssetOwners(t *testing.T) {
	logger.Info("TestCommitToLedger_AssetOwners function called")
	mstub := setup(t)

	//register self; owner should be only user
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	testUser1 := test_utils.CreateTestUser("testUser1")
	symKey1 := testUser1.GetSymKey()
	err := commitToLedger(stub, testUser1, testUser1, symKey1.ID, symKey1.KeyBytes, []string{}, true)
	test_utils.AssertTrue(t, err == nil, "Expected CommitToLedger to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	am := asset_mgmt_i.GetAssetManager(stub, testUser1)
	asset, err := am.GetAsset(asset_mgmt_i.GetAssetId(global.USER_ASSET_NAMESPACE, testUser1.ID), data_model.Key{})
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, len(asset.OwnerIds) == 1, "Expected 1 owner")
	test_utils.AssertTrue(t, utils.InList(asset.OwnerIds, testUser1.ID), "Expected user to be an owner")
	mstub.MockTransactionEnd("t1")

	//register other user; owner should be only user
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	testUser2 := test_utils.CreateTestUser("testUser2")
	symKey2 := testUser2.GetSymKey()
	err = commitToLedger(stub, testUser1, testUser2, symKey2.ID, symKey2.KeyBytes, []string{}, true)
	test_utils.AssertTrue(t, err == nil, "Expected CommitToLedger to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	asset, err = am.GetAsset(asset_mgmt_i.GetAssetId(global.USER_ASSET_NAMESPACE, testUser2.ID), data_model.Key{})
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, len(asset.OwnerIds) == 1, "Expected 1 owners")
	test_utils.AssertTrue(t, utils.InList(asset.OwnerIds, testUser2.ID), "Expected user to be an owner")
	mstub.MockTransactionEnd("t1")

	//register group; owner should be the group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	testGroup1 := test_utils.CreateTestGroup("testGroup1")
	symKey1 = testGroup1.GetSymKey()
	err = commitToLedger(stub, testUser1, testGroup1, symKey1.ID, symKey1.KeyBytes, []string{}, true)
	test_utils.AssertTrue(t, err == nil, "Expected CommitToLedger to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	asset, err = am.GetAsset(asset_mgmt_i.GetAssetId(global.USER_ASSET_NAMESPACE, testGroup1.ID), data_model.Key{})
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, len(asset.OwnerIds) == 1, "Expected 1 owner")
	test_utils.AssertTrue(t, utils.InList(asset.OwnerIds, testGroup1.ID), "Expected group to be an owner")
	mstub.MockTransactionEnd("t1")
}

func TestAssetManagerImpl_UpdateAsset_Indices(t *testing.T) {
	// Setup
	mstub := setup(t)
	caller := test_utils.CreateTestUser("caller")

	// Create indexes on User asset
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	table := index.GetTable(stub, "user")
	table.AddIndex([]string{"name", "role", "id"}, false)
	table.AddIndex([]string{"email", "status", "id"}, false)
	table.SaveToLedger()
	mstub.MockTransactionEnd("t1")

	// Add User asset
	assetKey := data_model.Key{ID: "symKeyId", Type: global.KEY_TYPE_SYM, KeyBytes: caller.SymKey}
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	am := asset_mgmt_i.GetAssetManager(stub, caller)
	am.AddAsset(ConvertToAsset(caller), assetKey, true)
	mstub.MockTransactionEnd("t1")

	// Search index 1
	iter1, _ := table.GetRowsByPartialKey([]string{"name", "role", "id"}, []string{caller.Name, caller.Role, caller.ID})
	KV1, _ := iter1.Next()
	fmt.Printf("index 1 search result: %v\n", string(KV1.GetValue()))

	// Search index 2
	iter2, _ := table.GetRowsByPartialKey([]string{"email", "status", "id"}, []string{caller.Email, caller.Status, caller.ID})
	KV2, _ := iter2.Next()
	fmt.Printf("index 2 search result: %v\n", string(KV2.GetValue()))

	// Update the User and confirm that indices are updated
	caller.Name = "new name"
	caller.Role = "new role"
	caller.Email = "new email"
	caller.Status = "new status"

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	am = asset_mgmt_i.GetAssetManager(stub, caller)
	am.UpdateAsset(ConvertToAsset(caller), assetKey)
	mstub.MockTransactionEnd("t1")

	// Search index 1
	iter1, _ = table.GetRowsByPartialKey([]string{"name", "role", "id"}, []string{caller.Name, caller.Role, caller.ID})
	KV1, _ = iter1.Next()
	fmt.Printf("updated index 1 search result: %v\n", string(KV1.GetValue()))

	// Search index 2
	iter2, _ = table.GetRowsByPartialKey([]string{"email", "status", "id"}, []string{caller.Email, caller.Status, caller.ID})
	KV2, _ = iter2.Next()
	fmt.Printf("updated index 2 search result: %v\n", string(KV2.GetValue()))
}

// Test update user from JSON string
func TestUpdateOrgWithJSON(t *testing.T) {
	logger.SetLevel(shim.LogDebug)

	mstub := setup(t)

	// Create caller and register, and also test its data can be retrieved from ledger
	caller := test_utils.CreateTestUser("callerID")
	caller.Role = global.ROLE_SYSTEM_ADMIN
	stub := cached_stub.NewCachedStub(mstub)
	RegisterUserForTest(t, mstub, caller, caller, false)

	// Create org1 and register org by caller, also allow add access from caller to org1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	orgID1 := "orgID1"
	org1 := test_utils.CreateTestGroup(orgID1)
	org1.SolutionPrivateData = map[string]interface{}{"tax_id": "1", "address": "some address"}
	orgBytes, _ := json.Marshal(&org1)
	_, err := RegisterOrg(stub, caller, []string{string(orgBytes), "true"})
	logger.Debugf("err: %v", err)
	test_utils.AssertTrue(t, err == nil, "Expected RegisterOrg to succeed")
	mstub.MockTransactionEnd("t1")

	// Call RegisterOrg with JSON string, should update org data
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	orgBytes2 := []byte(`{"id":"orgID1","name":"modifiedOrgName", "is_group":true, "role":"org", "solution_private_data":{"tax_id":"2", "address":"new address"}}`)
	_, err = RegisterOrg(stub, caller, []string{string(orgBytes2), "true"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterOrg to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// make sure to reload from ledger
	returnedOrg, err := GetUserData(stub, org1, org1.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected getOrg to succeed")
	test_utils.AssertTrue(t, returnedOrg.Name == "modifiedOrgName", "Expected org name to be updated")
	test_utils.AssertTrue(t, returnedOrg.Email == org1.Email, "Expected org email to remain the same as before")
	returnedOrgData := returnedOrg.SolutionPrivateData.(map[string]interface{})
	test_utils.AssertTrue(t, returnedOrgData["address"] == "new address", "Expected org address to be updated")
	mstub.MockTransactionEnd("t1")

	// Call RegisterOrg with wrong field IsGroup = false, should be rejected
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	orgBytes3 := []byte(`{"id":"orgID1","name":"modifiedOrgName", "is_group":false, "role":"org", "solution_private_data":{"tax_id":"2", "address":"new address"}}`)
	_, err = RegisterOrg(stub, org1, []string{string(orgBytes3), "true"})
	test_utils.AssertTrue(t, err != nil, "Expected RegisterOrg to fail")
	mstub.MockTransactionEnd("t1")

	// Call RegisterOrg without tax_id and address, for existing org should pass, for new org should be rejected
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	orgBytes3 = []byte(`{"id":"orgID1","name":"some name", "is_group":true, "role":"org"}`)
	_, err = RegisterOrg(stub, caller, []string{string(orgBytes3), "true"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterOrg to succedd")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	orgBytes4 := []byte(`{"id":"orgID2","name":"some name", "is_group":true, "role":"org", "solution_private_data":{}}`)
	_, err = RegisterOrg(stub, caller, []string{string(orgBytes4), "true"})
	test_utils.AssertTrue(t, err != nil, "Expected RegisterOrg to fail")
	mstub.MockTransactionEnd("t1")

}

// Tests register org and update org functionalities
func TestRegisterOrg(t *testing.T) {
	logger.Info("TestRegisterOrg function called")
	mstub := setup(t)

	// Register caller
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Register org
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	testOrg := test_utils.CreateTestGroup("newOrg")
	testOrgBytes, _ := json.Marshal(&testOrg)
	args := []string{string(testOrgBytes), "true"}
	_, err = RegisterOrg(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "RegisterOrg should be successful")
	mstub.MockTransactionEnd("t1")

	// Make sure org asset got created
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	org, err := GetUserData(stub, testOrg, testOrg.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Got testOrg asset from ledger after testOrg was registered")
	test_utils.AssertTrue(t, org.IsGroup, "testOrg User object is a group")
	// Make sure tax id and address are in priv data
	var privateOrgData map[string]interface{}
	orgDataBytes, _ := json.Marshal(org.SolutionPrivateData)
	_ = json.Unmarshal(orgDataBytes, &privateOrgData)
	//Make sure there is access to sym key from priv key
	ok, err := key_mgmt_i.SlowVerifyAccess(stub, testOrg.GetPubPrivKeyId(), testOrg.GetSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(ok) > 0, "Access to org sym key from org private key was added properly")
	mstub.MockTransactionEnd("t1")

	// Test update org
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	testOrg.SolutionPrivateData.(map[string]interface{})["tax_id"] = "testOrgTaxId2"
	testOrg.Name = "updatedOrgName"
	//testOrg.Role = "changedRole"
	testOrgBytes, _ = json.Marshal(&testOrg)
	args = []string{string(testOrgBytes), "false"}
	_, err = UpdateOrg(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "Update org should have been successful")
	mstub.MockTransactionEnd("t1")

	// Make sure org asset got updated
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	org, err = GetUserData(stub, testOrg, testOrg.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Got testOrg asset from ledger after testOrg was updated")
	orgDataBytes, _ = json.Marshal(org.SolutionPrivateData)
	_ = json.Unmarshal(orgDataBytes, &privateOrgData)
	test_utils.AssertTrue(t, org.IsGroup, "testOrg User object is a group")
	test_utils.AssertTrue(t, org.Name == "updatedOrgName", "Name updated successfully")
	test_utils.AssertTrue(t, privateOrgData["tax_id"] == "testOrgTaxId2", "Private data, tax_id, was updated properly")
	test_utils.AssertTrue(t, org.Role == "org", "Role was not updated")
	mstub.MockTransactionEnd("t1")

	// make sure admin can update org
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	testUser := test_utils.CreateTestUser("testUserID")
	testUserBytes, _ := json.Marshal(&testUser)
	_, err = RegisterUser(stub, testUser, []string{string(testUserBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, testOrg, testUser.ID, testOrg.ID, true)
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = UpdateOrg(stub, testUser, args)
	test_utils.AssertTrue(t, err == nil, "Update org should have been successful")
	mstub.MockTransactionEnd("t1")
}

// Tests RegisterSubgroup
func TestRegisterSubgroup(t *testing.T) {
	logger.Info("TestRegisterSubgroup function called")
	mstub := setup(t)

	// Register parentOrg
	parentOrg := test_utils.CreateTestGroup("parentOrg")
	parentOrgBytes, _ := json.Marshal(&parentOrg)
	args := []string{string(parentOrgBytes), "false"}
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	_, err := RegisterOrg(stub, parentOrg, args)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "RegisterOrg should be successful")

	// Test RegisterSubgroup
	subOrg := test_utils.CreateTestGroup("subOrg")
	subOrgBytes, _ := json.Marshal(&subOrg)
	args = []string{string(subOrgBytes), parentOrg.ID}
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterSubgroup(stub, parentOrg, args)
	logger.Debugf("err: %v", err)
	test_utils.AssertTrue(t, err == nil, "RegisterSubgroup should be successful")
	mstub.MockTransactionEnd("t1")

	// Confirm that subgroup exists
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	subgroupIDs, err := SlowGetSubgroups(stub, parentOrg.ID)
	test_utils.AssertTrue(t, err == nil, "GetSubgroups should be successful")
	test_utils.AssertTrue(t, len(subgroupIDs) == 1, "GetSubgroups should have returned one ID")
	test_utils.AssertTrue(t, subgroupIDs[0] == subOrg.ID, "GetSubgroups should have returned subOrg.ID")
	ledgerSubgroup, err := GetUserData(stub, parentOrg, subOrg.ID, true, true)
	test_utils.AssertTrue(t, err == nil, "GetUserData should be successful")
	test_utils.AssertTrue(t, ledgerSubgroup.ID == subOrg.ID, "GetUserData should have returned the subOrg")
	mstub.MockTransactionEnd("t1")
}

// Tests RegisterSubgroup
func TestRegisterSubgroup_MissingKeys(t *testing.T) {
	logger.Info("TestRegisterSubgroup function called")
	mstub := setup(t)

	// Register parentOrg
	parentOrg := test_utils.CreateTestGroup("parentOrg")
	parentOrgBytes, _ := json.Marshal(&parentOrg)
	args := []string{string(parentOrgBytes), "false"}
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	_, err := RegisterOrg(stub, parentOrg, args)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "RegisterOrg should be successful")

	// Test RegisterSubgroup without PrivateKeyB64
	subOrg := test_utils.CreateTestGroup("subOrg")
	subOrg.PrivateKeyB64 = ""
	subOrgBytes, _ := json.Marshal(&subOrg)
	args = []string{string(subOrgBytes), parentOrg.ID}
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterSubgroup(stub, parentOrg, args)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err != nil, "RegisterSubgroup should fail")

	// Test RegisterSubgroup without PublicKeyB64
	subOrg = test_utils.CreateTestGroup("subOrg")
	subOrg.PublicKeyB64 = ""
	subOrgBytes, _ = json.Marshal(&subOrg)
	args = []string{string(subOrgBytes), parentOrg.ID}
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterSubgroup(stub, parentOrg, args)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err != nil, "RegisterSubgroup should fail")

	// Test RegisterSubgroup without SymKeyB64
	subOrg = test_utils.CreateTestGroup("subOrg")
	subOrg.SymKeyB64 = ""
	subOrgBytes, _ = json.Marshal(&subOrg)
	args = []string{string(subOrgBytes), parentOrg.ID}
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterSubgroup(stub, parentOrg, args)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err != nil, "RegisterSubgroup should fail")
}

func TestGetOrg(t *testing.T) {
	logger.Info("TestGetOrg function called")
	mstub := setup(t)

	// Create caller
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := data_model.User{}
	caller.ID = "caller"
	caller.PrivateKey = test_utils.GeneratePrivateKey()
	caller.PrivateKeyB64 = crypto.EncodeToB64String(crypto.PrivateKeyToBytes(caller.PrivateKey))
	caller.PublicKey = caller.PrivateKey.Public().(*rsa.PublicKey)
	caller.PublicKeyB64 = crypto.EncodeToB64String(crypto.PublicKeyToBytes(caller.PublicKey))
	caller.SymKey = test_utils.GenerateSymKey()
	caller.SymKeyB64 = crypto.EncodeToB64String(caller.SymKey)
	caller.Email = "none"
	caller.Name = "caller"
	caller.IsGroup = false
	caller.Status = "active"
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Make org keys
	orgPrivateKey := test_utils.GeneratePrivateKey()
	orgPublicKey := orgPrivateKey.Public().(*rsa.PublicKey)
	orgSymKey := test_utils.GenerateSymKey()
	orgPrivateKeyBytes := crypto.PrivateKeyToBytes(orgPrivateKey)
	orgPublicKeyBytes := crypto.PublicKeyToBytes(orgPublicKey)

	// Create org object
	orgId := "newOrg"
	testOrg := data_model.User{ID: orgId, Name: "newOrgName", Role: "org"}
	testOrg.Email = "newOrg@newOrg.com"
	testOrg.IsGroup = true
	testOrgData := make(map[string]string)
	testOrgData["tax_id"] = "testOrgTaxId"
	testOrgData["address"] = "123 testOrg Street"
	testOrg.SolutionPrivateData = testOrgData
	testOrg.KmsPublicKeyId = "kmspubkeyid"
	testOrg.KmsPrivateKeyId = "kmsprivkeyid"
	testOrg.KmsSymKeyId = "kmssymkeyid"
	testOrg.PrivateKeyB64 = crypto.EncodeToB64String(orgPrivateKeyBytes)
	testOrg.PublicKeyB64 = crypto.EncodeToB64String(orgPublicKeyBytes)
	testOrg.SymKeyB64 = crypto.EncodeToB64String(orgSymKey)
	testOrg.Status = "active"
	testOrg.PrivateKey = orgPrivateKey
	testOrg.PublicKey = orgPublicKey
	testOrg.SymKey = orgSymKey
	testOrgBytes, _ := json.Marshal(&testOrg)

	// Register org
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{string(testOrgBytes), "false"}
	_, err = RegisterOrg(stub, caller, args)
	test_utils.AssertTrue(t, err == nil, "RegisterOrg should be successful")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Get org
	args = []string{"newOrg"}
	orgBytes, err := GetOrg(stub, testOrg, args)
	var org = data_model.User{}
	_ = json.Unmarshal(orgBytes, &org)
	test_utils.AssertTrue(t, err == nil, "GetOrg succcessful")
	test_utils.AssertTrue(t, org.IsGroup, "testOrg User object is a group")
	test_utils.AssertTrue(t, org.Name == "newOrgName", "testOrg User object is a group")
	// Make sure got private data of org successfully
	var privateOrgData map[string]string
	orgDataBytes, _ := json.Marshal(org.SolutionPrivateData)
	_ = json.Unmarshal(orgDataBytes, &privateOrgData)
	test_utils.AssertTrue(t, privateOrgData["tax_id"] == "testOrgTaxId", "Private data, tax_id, was saved properly")
	test_utils.AssertTrue(t, privateOrgData["address"] == "123 testOrg Street", "Private data, address, was saved properly")
	mstub.MockTransactionEnd("t1")
}

func TestGetUserDataByCaller(t *testing.T) {
	logger.Info("TestGetUserDataByCaller function called")
	mstub := setup(t)

	// Create mock users and keys and put users to ledger
	testUserID1 := "testUser1"
	testUser1 := test_utils.CreateTestUser(testUserID1)

	testUserID2 := "testUser2"
	testUser2 := test_utils.CreateTestUser(testUserID2)

	symkey1 := data_model.Key{}
	symkey1.ID = testUser1.GetSymKeyId()
	symkey1.Type = global.KEY_TYPE_SYM
	symkey1.KeyBytes = testUser1.SymKey

	pubkey1 := data_model.Key{}
	pubkey1.ID = testUser1.GetPubPrivKeyId()
	pubkey1.Type = global.KEY_TYPE_PUBLIC
	pubkey1.KeyBytes = crypto.PublicKeyToBytes(testUser1.PublicKey)

	symkey2 := data_model.Key{}
	symkey2.ID = testUser2.GetSymKeyId()
	symkey2.Type = global.KEY_TYPE_SYM
	symkey2.KeyBytes = testUser2.SymKey

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	err := registerUserInternal(stub, testUser1, testUser1, true)
	test_utils.AssertTrue(t, err == nil, "Expected CommitToLedger to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = registerUserInternal(stub, testUser2, testUser2, true)
	test_utils.AssertTrue(t, err == nil, "Expected CommitToLedger to succeed")
	mstub.MockTransactionEnd("t1")

	testUser1.Name = "modifiedName"
	testUser1.IsGroup = false
	testUser1.Status = "inactive"
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = registerUserInternal(stub, testUser1, testUser1, true)
	test_utils.AssertTrue(t, err == nil, "Expected CommitToLedger to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Test user1 get user1's data

	returnedUser, err := GetUserData(stub, testUser1, testUserID1, false, true)
	test_utils.AssertTrue(t, userEqual(returnedUser, testUser1), "Expected to get correct user1 data")

	// Test user1 get user2's data, before adding key access,
	// our current design is user can get public data, but no private data

	returnedUser, err = GetUserData(stub, testUser1, testUserID2, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected GetUserDataByCaller to succeed even without key access")
	test_utils.AssertTrue(t, len(returnedUser.Email) == 0, "Expected GetUserDataByCaller does not have private data without key access")
	mstub.MockTransactionEnd("t1")

	// Test user1 get user2's data, after adding key access, this one should succeed

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = key_mgmt_i.AddAccess(stub, pubkey1, symkey2)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t1")

	// pass key path [pubkey1.ID, symkey2.ID]
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	returnedUser, err = GetUserData(stub, testUser1, testUserID2, false, true, []string{pubkey1.ID, symkey2.ID})
	test_utils.AssertTrue(t, err == nil, "Expected GetUserDataByCaller to succeed with key access")
	test_utils.AssertTrue(t, userEqual(returnedUser, testUser2), "Expected to get correct user2 public data")
	test_utils.AssertTrue(t, len(returnedUser.Email) > 0, "Expected GetUserDataByCaller to have private data with key access")
	mstub.MockTransactionEnd("t1")
}

// Test isUserInGroup, also contains test for RegisterSubgroupWithParams
func TestIsUserInGroup(t *testing.T) {
	logger.Info("TestIsUserInGroup function called")

	mstub := setup(t)

	testUserID1 := "testUser1"
	testUser1 := test_utils.CreateTestUser(testUserID1)

	testUserID2 := "testUser2"
	testUser2 := test_utils.CreateTestUser(testUserID2)

	testGroupID1 := "testGroup1"
	testGroup1 := test_utils.CreateTestGroup(testGroupID1)

	testGroupID2 := "testGroup2"
	testGroup2 := test_utils.CreateTestGroup(testGroupID2)

	// Add test users and groups to ledger, and put user1 in group1
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	registerUserInternal(stub, testUser1, testUser1, true)
	registerUserInternal(stub, testUser2, testUser2, true)
	registerUserInternal(stub, testGroup1, testGroup1, true)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err := PutUserInGroup(stub, testGroup1, testUserID1, testGroupID1, false)
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Check user1 should be in group1 after the PutUserInGroup. Also check user2 not in group1
	isInGroup, err := user_mgmt_c.IsUserInGroup(stub, testUserID1, testGroupID1)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect IsUserInGroup to return true.")
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, testUserID2, testGroupID1)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect IsUserInGroup to return false.")
	mstub.MockTransactionEnd("t1")

	// Put group2 in group1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = RegisterSubgroupWithParams(stub, testGroup1, testGroup2, testGroupID1)
	test_utils.AssertTrue(t, err == nil, "Expect RegisterSubgroupWithParams to be successful.")
	mstub.MockTransactionEnd("t1")

	// Put user2 in group2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, testGroup2, testUserID2, testGroupID2, false)
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Now user2 should be both in group2 and group1, user1 should be in group1
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, testGroupID2, testGroupID1)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect group2 in group2.")
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, testUserID2, testGroupID1)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect user2 in group1.")
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, testUserID2, testGroupID2)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect user2 in group2.")
	mstub.MockTransactionEnd("t1")
}

func TestIsUserInGroup_Indirect(t *testing.T) {
	logger.Info("TestIsUserInGroup_Indirect function called")
	mstub := setup(t)

	//make groups
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	group1 := test_utils.CreateTestGroup("group1")
	group2 := test_utils.CreateTestGroup("group2")
	group3 := test_utils.CreateTestGroup("group3")
	registerUserInternal(stub, group1, group1, true)
	// group1.GetSymKeyId(), group1.SymKey, []string{}, true)
	mstub.MockTransactionEnd("t1")

	//group1 <- group2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	RegisterSubgroupWithParams(stub, group1, group2, group1.ID)
	mstub.MockTransactionEnd("t1")

	//group2 <- group3
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	RegisterSubgroupWithParams(stub, group2, group3, group2.ID)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	//valid
	isInGroup, err := user_mgmt_c.IsUserInGroup(stub, group2.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect group2 in group1.")
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, group3.ID, group2.ID)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect group3 in group2.")
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, group3.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect group3 in group1.")

	//invalid
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, group1.ID, group2.ID)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect group1 not in group2.")
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, group2.ID, group3.ID)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect group2 not in group3.")
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, group1.ID, group3.ID)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect group1 not in group3.")
	mstub.MockTransactionEnd("t1")
}

func TestPutUserInGroupAndRemove(t *testing.T) {
	logger.Info("TestPutUserInGroupAndRemove function called")
	mstub := setup(t)

	// add test users and groups to ledger
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	testGroup := test_utils.CreateTestGroup("testGroupID")
	testGroupBytes, _ := json.Marshal(&testGroup)
	_, err := RegisterOrg(stub, testGroup, []string{string(testGroupBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	testUser := test_utils.CreateTestUser("testUserID")
	testUserBytes, _ := json.Marshal(&testUser)
	_, err = RegisterUser(stub, testUser, []string{string(testUserBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// put user in group as a member
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, testGroup, testUser.ID, testGroup.ID, false)
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// check that user has access to the group sym key and not group private key
	keyPath, err := key_mgmt_i.SlowVerifyAccess(stub, testUser.GetPubPrivKeyId(), testGroup.GetSymKeyId())
	logger.Debugf("keyPath to symkey: %v", keyPath)
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) > 0, "Access to group sym key from user private key was added properly")

	keyPath, err = key_mgmt_i.SlowVerifyAccess(stub, testUser.GetPubPrivKeyId(), testGroup.GetPubPrivKeyId())
	logger.Debugf("keyPath to private key: %v", keyPath)
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) == 0, "Access to group private key from user private key does not exist")

	// check that user is in group and is not admin
	isInGroup, err := user_mgmt_c.IsUserInGroup(stub, testUser.ID, testGroup.ID)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect IsUserInGroup to return true.")
	isAdmin, err := user_mgmt_c.IsUserDirectAdminOfGroup(stub, testUser.ID, testGroup.ID)
	test_utils.AssertTrue(t, err == nil && !isAdmin, "Expect IsDirectGroupAdmin to return false.")
	mstub.MockTransactionEnd("t1")

	// remove user from group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{testUser.ID, testGroup.ID}
	_, err = RemoveUserFromGroup(stub, testGroup, args)
	test_utils.AssertTrue(t, err == nil, "Expect RemoveUserFromGroup to be successful.")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// check that user access to the group sym key has been removed
	keyPath, err = key_mgmt_i.SlowVerifyAccess(stub, testUser.GetPubPrivKeyId(), testGroup.GetSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) == 0, "Access to group sym key from user private key was removed properly")
	keyPath, err = key_mgmt_i.SlowVerifyAccess(stub, testUser.GetPubPrivKeyId(), testGroup.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) == 0, "Access to group private key from user private key does not exist")

	// check that user is not in group and is not admin
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, testUser.ID, testGroup.ID)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect IsUserInGroup to return false.")
	isAdmin, err = user_mgmt_c.IsUserDirectAdminOfGroup(stub, testUser.ID, testGroup.ID)
	logger.Debugf("isAdmin: %v, err:%v", isAdmin, err)
	test_utils.AssertTrue(t, err == nil && !isAdmin, "Expect IsDirectGroupAdmin to return false.")

	mstub.MockTransactionEnd("t1")
}

func TestPutUserInGroupAndRemove_subgroup(t *testing.T) {
	logger.Info("TestPutUserInGroupAndRemove_subgroup function called")
	mstub := setup(t)

	// add test users and groups to ledger
	testGroup := test_utils.CreateTestGroup("testGroupID")
	testGroupBytes, _ := json.Marshal(&testGroup)
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	_, err := RegisterOrg(stub, testGroup, []string{string(testGroupBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	testUser := test_utils.CreateTestUser("testUserID")
	testUserBytes, _ := json.Marshal(&testUser)
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterUser(stub, testUser, []string{string(testUserBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	subgroupUser1 := test_utils.CreateTestUser("subgroupUser1")
	testSubgroupUser1Bytes, _ := json.Marshal(&subgroupUser1)
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterUser(stub, subgroupUser1, []string{string(testSubgroupUser1Bytes), "false"})
	mstub.MockTransactionEnd("t1")

	subgroupUser2 := test_utils.CreateTestUser("subgroupUser2")
	testSubgroupUser2Bytes, _ := json.Marshal(&subgroupUser2)
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterUser(stub, subgroupUser2, []string{string(testSubgroupUser2Bytes), "false"})
	mstub.MockTransactionEnd("t1")

	// put testUser in testGroup as admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, testGroup, testUser.ID, testGroup.ID, true)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")

	// Register subgroup1 and subgroup2, they are both children of testGroup
	subGroup1 := test_utils.CreateTestGroup("subGroup1")
	subGroup1Bytes, _ := json.Marshal(&subGroup1)
	args := []string{string(subGroup1Bytes), testGroup.ID}

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterSubgroup(stub, testGroup, args)
	test_utils.AssertTrue(t, err == nil, "RegisterSubgroup should be successful")
	mstub.MockTransactionEnd("t1")

	subGroup2 := test_utils.CreateTestGroup("subGroup2")
	subGroup2Bytes, _ := json.Marshal(&subGroup2)
	args = []string{string(subGroup2Bytes), testGroup.ID}
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterSubgroup(stub, testGroup, args)
	test_utils.AssertTrue(t, err == nil, "RegisterSubgroup should be successful")
	mstub.MockTransactionEnd("t1")

	//------------------ Test for orginal parent group admin
	// Test original group admin can put user in subgroup
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, testGroup, subgroupUser1.ID, subGroup1.ID, false)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")

	// check that user is in group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isInGroup, err := user_mgmt_c.IsUserInGroup(stub, subgroupUser1.ID, subGroup1.ID)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect IsUserInGroup to return true.")
	mstub.MockTransactionEnd("t1")

	// remove user from group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{subgroupUser1.ID, subGroup1.ID}
	_, err = RemoveUserFromGroup(stub, testGroup, args)
	test_utils.AssertTrue(t, err == nil, "Expect RemoveUserFromGroup to be successful.")
	mstub.MockTransactionEnd("t1")

	// check that user is not in group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, subgroupUser1.ID, subGroup1.ID)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect IsUserInGroup to return false.")
	mstub.MockTransactionEnd("t1")

	//------------------ Test for parent group admin
	// Test group admin can put user in subgroup
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)

	/*
		INFO 065 Adding access edge from "pub-priv-testUserID" to "private-hash-testGroupID"
		INFO 00d Adding access edge from "private-hash-testGroupID" to "pub-priv-testGroupID"
		INFO 072 Adding access edge from "pub-priv-testGroupID" to "private-hash-subGroup1"
		INFO 082 Adding access edge from "private-hash-subGroup1" to "pub-priv-subGroup1"
		INFO 07e Adding access edge from "pub-priv-subGroup1" to "sym-subGroup1"
	*/

	keyPathSymKey := []string{"pub-priv-testUserID", "private-hash-testGroupID", "pub-priv-testGroupID", "private-hash-subGroup1", "pub-priv-subGroup1", "sym-subGroup1"}
	keyPathPrivKey := []string{"pub-priv-testUserID", "private-hash-testGroupID", "pub-priv-testGroupID", "private-hash-subGroup1", "pub-priv-subGroup1"}
	err = PutUserInGroup(stub, testUser, subgroupUser1.ID, subGroup1.ID, false, keyPathSymKey, keyPathPrivKey)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")

	// check that user is in group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, subgroupUser1.ID, subGroup1.ID)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect IsUserInGroup to return true.")
	mstub.MockTransactionEnd("t1")

	// remove user from group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{subgroupUser1.ID, subGroup1.ID}
	_, err = RemoveUserFromGroup(stub, testUser, args)
	test_utils.AssertTrue(t, err == nil, "Expect RemoveUserFromGroup to be successful.")
	mstub.MockTransactionEnd("t1")

	// check that user is not in group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, subgroupUser1.ID, subGroup1.ID)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect IsUserInGroup to return false.")
	mstub.MockTransactionEnd("t1")

	//------------------ Test for original subgroup admin
	// Test original subgroup admin can put user in subgroup
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, subGroup1, subgroupUser1.ID, subGroup1.ID, false)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")

	// check that user is in group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, subgroupUser1.ID, subGroup1.ID)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect IsUserInGroup to return true.")
	mstub.MockTransactionEnd("t1")

	//------------------ Test for another subgroup admin
	// Test another subgroup admin put user in subgroup, should fail
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, subGroup2, subgroupUser2.ID, subGroup1.ID, false)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err != nil, "Expect PutUserInGroup to fail.")

	// check that user is in group and is not admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, subgroupUser2.ID, subGroup1.ID)
	test_utils.AssertFalse(t, err == nil && isInGroup, "Expect IsUserInGroup to return false.")
	mstub.MockTransactionEnd("t1")

	// remove user from subgroup by another subgroup admin, should fail
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args = []string{subgroupUser1.ID, subGroup1.ID}
	_, err = RemoveUserFromGroup(stub, subGroup2, args)
	test_utils.AssertTrue(t, err != nil, "Expect RemoveUserFromGroup to fail.")
	mstub.MockTransactionEnd("t1")

	// check that user is not in group and is not admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, subgroupUser1.ID, subGroup1.ID)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect IsUserInGroup to return true.")
	mstub.MockTransactionEnd("t1")
}

func TestPutAdminInGroupAndRemove(t *testing.T) {
	logger.Info("TestPutAdminInGroupAndRemove function called")
	mstub := setup(t)

	// add test users and groups to ledger
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	testGroup := test_utils.CreateTestGroup("testGroupID")
	testGroupBytes, _ := json.Marshal(&testGroup)
	_, err := RegisterOrg(stub, testGroup, []string{string(testGroupBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	testUser := test_utils.CreateTestUser("testUserID")
	testUserBytes, _ := json.Marshal(&testUser)
	_, err = RegisterUser(stub, testUser, []string{string(testUserBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// add user to group as admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, testGroup, testUser.ID, testGroup.ID, true)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// check that user has access to the group sym key and group private key
	keyPath, err := key_mgmt_i.SlowVerifyAccess(stub, testUser.GetPubPrivKeyId(), testGroup.GetSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) > 0, "Access to group sym key from user private key was added properly")
	keyPath, err = key_mgmt_i.SlowVerifyAccess(stub, testUser.GetPubPrivKeyId(), testGroup.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) > 0, "Access to group private key from user private key was added properly")

	// check that user is in group and is admin
	isInGroup, err := user_mgmt_c.IsUserInGroup(stub, testUser.ID, testGroup.ID)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect IsUserInGroup to return true.")
	isAdmin, err := user_mgmt_c.IsUserDirectAdminOfGroup(stub, testUser.ID, testGroup.ID)
	test_utils.AssertTrue(t, err == nil && isAdmin, "Expect IsDirectGroupAdmin to return true.")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{testUser.ID, testGroup.ID}
	_, err = RemoveUserFromGroup(stub, testGroup, args)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expect RemoveUserFromGroup to be successful.")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	//check that user access to the group sym key and private key has been removed
	keyPath, err = key_mgmt_i.SlowVerifyAccess(stub, testUser.GetPubPrivKeyId(), testGroup.GetSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) == 0, "Access to group sym key from user private key was removed properly")
	keyPath, err = key_mgmt_i.SlowVerifyAccess(stub, testUser.GetPubPrivKeyId(), testGroup.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) == 0, "Access to group private key from user private key was removed properly")

	// check that user is not in group and is not admin
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, testUser.ID, testGroup.ID)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect IsUserInGroup to return false.")
	isAdmin, err = user_mgmt_c.IsUserDirectAdminOfGroup(stub, testUser.ID, testGroup.ID)
	test_utils.AssertTrue(t, err == nil && !isAdmin, "Expect IsDirectGroupAdmin to return false.")

	// check that user has been removed as an owner of the group user asset
	am := asset_mgmt_i.GetAssetManager(stub, testGroup)
	asset, err := am.GetAsset(asset_mgmt_i.GetAssetId(global.USER_ASSET_NAMESPACE, testGroup.ID), data_model.Key{})
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, len(asset.OwnerIds) == 1, "Expected 1 owner")
	test_utils.AssertTrue(t, utils.InList(asset.OwnerIds, testGroup.ID), "Expected group to be an owner")
	mstub.MockTransactionEnd("t1")
}

func TestPutUserInGroupAndRemove_Promotion(t *testing.T) {
	logger.Info("TestPutUserInGroupAndRemove_Promotion function called")
	mstub := setup(t)

	// add test users and groups to ledger
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	testGroup := test_utils.CreateTestGroup("testGroupID")
	testGroupBytes, _ := json.Marshal(&testGroup)
	_, err := RegisterOrg(stub, testGroup, []string{string(testGroupBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	testUser := test_utils.CreateTestUser("testUserID")
	testUserBytes, _ := json.Marshal(&testUser)
	_, err = RegisterUser(stub, testUser, []string{string(testUserBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// add user to group as admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, testGroup, testUser.ID, testGroup.ID, false)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, testGroup, testUser.ID, testGroup.ID, true)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// check that user has access to the group sym key and group private key
	keyPath, err := key_mgmt_i.SlowVerifyAccess(stub, testUser.GetPubPrivKeyId(), testGroup.GetSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) > 0, "Access to group sym key from user private key was added properly")
	keyPath, err = key_mgmt_i.SlowVerifyAccess(stub, testUser.GetPubPrivKeyId(), testGroup.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) > 0, "Access to group private key from user private key was added properly")

	// check that user is in group and is admin
	isInGroup, err := user_mgmt_c.IsUserInGroup(stub, testUser.ID, testGroup.ID)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect IsUserInGroup to return true.")
	isAdmin, err := user_mgmt_c.IsUserDirectAdminOfGroup(stub, testUser.ID, testGroup.ID)
	test_utils.AssertTrue(t, err == nil && isAdmin, "Expect IsDirectGroupAdmin to return true.")
	mstub.MockTransactionEnd("t1")
}

func TestPutUserInGroupAndRemove_Demotion(t *testing.T) {
	logger.Info("TestPutUserInGroupAndRemove_Demotion function called")
	mstub := setup(t)

	// add test users and groups to ledger
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	testGroup := test_utils.CreateTestGroup("testGroupID")
	testGroupBytes, _ := json.Marshal(&testGroup)
	_, err := RegisterOrg(stub, testGroup, []string{string(testGroupBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	testUser := test_utils.CreateTestUser("testUserID")
	testUserBytes, _ := json.Marshal(&testUser)
	_, err = RegisterUser(stub, testUser, []string{string(testUserBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// add user to group as admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, testGroup, testUser.ID, testGroup.ID, true)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, testGroup, testUser.ID, testGroup.ID, false)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// check that user has access to the group sym key and not group private key
	keyPath, err := key_mgmt_i.SlowVerifyAccess(stub, testUser.GetPubPrivKeyId(), testGroup.GetSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) > 0, "Access to group sym key from user private key was added properly")
	keyPath, err = key_mgmt_i.SlowVerifyAccess(stub, testUser.GetPubPrivKeyId(), testGroup.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) == 0, "Access to group private key from user private key was removed properly")

	// check that user is in group and is not admin
	isInGroup, err := user_mgmt_c.IsUserInGroup(stub, testUser.ID, testGroup.ID)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect IsUserInGroup to return true.")
	isAdmin, err := user_mgmt_c.IsUserDirectAdminOfGroup(stub, testUser.ID, testGroup.ID)
	test_utils.AssertTrue(t, err == nil && !isAdmin, "Expect IsDirectGroupAdmin to return false.")

	// check that user has been removed as an owner of the group user asset
	am := asset_mgmt_i.GetAssetManager(stub, testGroup)
	asset, err := am.GetAsset(asset_mgmt_i.GetAssetId(global.USER_ASSET_NAMESPACE, testGroup.ID), data_model.Key{})
	test_utils.AssertTrue(t, err == nil, "Expected GetAsset to succeed")
	test_utils.AssertTrue(t, len(asset.OwnerIds) == 1, "Expected 1 owner")
	test_utils.AssertTrue(t, utils.InList(asset.OwnerIds, testGroup.ID), "Expected group to be an owner")
	mstub.MockTransactionEnd("t1")
}

func TestPutUserInGroupAndRemove_CallerNotAdmin(t *testing.T) {
	logger.Info("TestPutUserInGroupAndRemove_CallerNotAdmin function called")
	mstub := setup(t)

	// create a group and two users
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	testGroup := test_utils.CreateTestGroup("testGroupID")
	testGroupBytes, _ := json.Marshal(&testGroup)
	_, err := RegisterOrg(stub, testGroup, []string{string(testGroupBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	testUser := test_utils.CreateTestUser("testUserID")
	testUserBytes, _ := json.Marshal(&testUser)
	_, err = RegisterUser(stub, testUser, []string{string(testUserBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	testUser2 := test_utils.CreateTestUser("testUserID2")
	testUser2Bytes, _ := json.Marshal(&testUser2)
	_, err = RegisterUser(stub, testUser2, []string{string(testUser2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, testUser2, testUser.ID, testGroup.ID, false)
	test_utils.AssertTrue(t, err != nil, "Expect PutUserInGroup to fail.")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, testGroup, testUser.ID, testGroup.ID, false)
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to succeed.")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{testUser.ID, testGroup.ID}
	_, err = RemoveUserFromGroup(stub, testUser2, args)
	test_utils.AssertTrue(t, err != nil, "Expect RemoveUserFromGroup to fail.")
	mstub.MockTransactionEnd("t1")
}

func TestPutUserInGroupAndRemove_RemoveFromAllSubgroups(t *testing.T) {
	logger.Info("TestPutUserInGroupAndRemove_RemoveFromAllSubgroups function called")
	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	//create group1
	group1 := test_utils.CreateTestGroup("group1")
	group1Bytes, _ := json.Marshal(&group1)
	_, err := RegisterOrg(stub, group1, []string{string(group1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")
	//create subgroup of group1 called group2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	group2 := test_utils.CreateTestGroup("group2")
	group2Bytes, _ := json.Marshal(&group2)
	_, err = RegisterSubgroup(stub, group1, []string{string(group2Bytes), group1.ID})
	test_utils.AssertTrue(t, err == nil, "RegisterSubgroup should be successful")
	mstub.MockTransactionEnd("t1")
	//create subgroup of group2 called group3
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	group3 := test_utils.CreateTestGroup("group3")
	group3Bytes, _ := json.Marshal(&group3)
	_, err = RegisterSubgroup(stub, group2, []string{string(group3Bytes), group2.ID})
	test_utils.AssertTrue(t, err == nil, "RegisterSubgroup should be successful")
	//create subgroup of group1 called group4
	group4 := test_utils.CreateTestGroup("group4")
	group4Bytes, _ := json.Marshal(&group4)
	_, err = RegisterSubgroup(stub, group1, []string{string(group4Bytes), group1.ID})
	test_utils.AssertTrue(t, err == nil, "RegisterSubgroup should be successful")
	//create user1 and user2
	user1 := test_utils.CreateTestUser("user1")
	user1Bytes, _ := json.Marshal(&user1)
	_, err = RegisterUser(stub, user1, []string{string(user1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	user2 := test_utils.CreateTestUser("user2")
	user2Bytes, _ := json.Marshal(&user2)
	_, err = RegisterUser(stub, user2, []string{string(user2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")
	//add user1 to group3 and group4 and add user2 to group2, group3, and group4
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, group3, user1.ID, group3.ID, true)
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")
	err = PutUserInGroup(stub, group4, user1.ID, group4.ID, true)
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")
	err = PutUserInGroup(stub, group2, user2.ID, group2.ID, true)
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")
	err = PutUserInGroup(stub, group3, user2.ID, group3.ID, true)
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")
	err = PutUserInGroup(stub, group4, user2.ID, group4.ID, true)
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")
	mstub.MockTransactionEnd("t1")

	//remove user1 from group1 and expect they are not part of any group anymore
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RemoveUserFromGroup(stub, group1, []string{user1.ID, group1.ID, "true"})
	test_utils.AssertTrue(t, err == nil, "Expect RemoveUserFromGroup to be successful.")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isInGroup, err := user_mgmt_c.IsUserInGroup(stub, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect IsUserInGroup to return false.")
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, user1.ID, group2.ID)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect IsUserInGroup to return false.")
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, user1.ID, group3.ID)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect IsUserInGroup to return false.")
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, user1.ID, group4.ID)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect IsUserInGroup to return false.")
	mstub.MockTransactionEnd("t1")

	//remove user2 from group2 and expect they are still part of group1 and group4
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RemoveUserFromGroup(stub, group1, []string{user2.ID, group2.ID, "true"})
	test_utils.AssertTrue(t, err == nil, "Expect RemoveUserFromGroup to be successful.")
	mstub.MockTransactionEnd("t1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, user2.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect IsUserInGroup to return true.")
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, user2.ID, group2.ID)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect IsUserInGroup to return false.")
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, user2.ID, group3.ID)
	test_utils.AssertTrue(t, err == nil && !isInGroup, "Expect IsUserInGroup to return false.")
	isInGroup, err = user_mgmt_c.IsUserInGroup(stub, user2.ID, group4.ID)
	test_utils.AssertTrue(t, err == nil && isInGroup, "Expect IsUserInGroup to return true.")

	mstub.MockTransactionEnd("t1")
}

func TestAdminsAndMembersOfGroup(t *testing.T) {
	logger.Info("TestAdminsAndMembersOfGroup function called")
	mstub := setup(t)

	testUserID1 := "testUser1"
	testUser1 := test_utils.CreateTestUser(testUserID1)

	testUserID2 := "testUser2"
	testUser2 := test_utils.CreateTestUser(testUserID2)

	testUserID3 := "testUser3"
	testUser3 := test_utils.CreateTestUser(testUserID3)

	testGroupID1 := "testGroup1"
	testGroup1 := test_utils.CreateTestGroup(testGroupID1)

	testGroupID2 := "testGroup2"
	testGroup2 := test_utils.CreateTestGroup(testGroupID2)

	// Add test users and groups to ledger, and put user1 in group1
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	registerUserInternal(stub, testUser1, testUser1, true)
	registerUserInternal(stub, testUser2, testUser2, true)
	registerUserInternal(stub, testUser3, testUser3, true)
	registerUserInternal(stub, testGroup1, testGroup1, true)
	mstub.MockTransactionEnd("t1")

	// user1 is admin of group1, group2 is subgroup of group1, user2 is member of group2, user3 is admin of group2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err := PutUserInGroup(stub, testGroup1, testUserID1, testGroupID1, true)
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")
	err = RegisterSubgroupWithParams(stub, testGroup1, testGroup2, testGroupID1)
	test_utils.AssertTrue(t, err == nil, "Expect RegisterSubgroupWithParams to be successful.")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, testGroup2, testUserID2, testGroupID2, false)
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")
	err = PutUserInGroup(stub, testGroup2, testUserID3, testGroupID2, true)
	test_utils.AssertTrue(t, err == nil, "Expect PutUserInGroup to be successful.")
	mstub.MockTransactionEnd("t1")

	// Test user_mgmt_c.IsUserAdminOfGroup
	logger.Info("Test isUserAdminOfGroup function")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// check user1 admin of group1 and group2
	isAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, testUserID1, testGroupID1)
	test_utils.AssertTrue(t, err == nil && isAdmin, "Expect user_mgmt_c.IsUserAdminOfGroup to return true.")
	isAdmin, _, err = user_mgmt_c.IsUserAdminOfGroup(stub, testUserID1, testGroupID2)
	test_utils.AssertTrue(t, err == nil && isAdmin, "Expect user_mgmt_c.IsUserAdminOfGroup to return true.")

	// check user2 not admin of group1 and group2
	isAdmin, _, err = user_mgmt_c.IsUserAdminOfGroup(stub, testUserID2, testGroupID1)
	test_utils.AssertTrue(t, err == nil && !isAdmin, "Expect user_mgmt_c.IsUserAdminOfGroup to return false.")
	isAdmin, _, err = user_mgmt_c.IsUserAdminOfGroup(stub, testUserID2, testGroupID2)
	test_utils.AssertTrue(t, err == nil && !isAdmin, "Expect user_mgmt_c.IsUserAdminOfGroup to return false.")

	// check user3 admin of group2 but not group1
	isAdmin, _, err = user_mgmt_c.IsUserAdminOfGroup(stub, testUserID3, testGroupID1)
	test_utils.AssertTrue(t, err == nil && !isAdmin, "Expect user_mgmt_c.IsUserAdminOfGroup to return false.")
	isAdmin, _, err = user_mgmt_c.IsUserAdminOfGroup(stub, testUserID3, testGroupID2)
	test_utils.AssertTrue(t, err == nil && isAdmin, "Expect user_mgmt_c.IsUserAdminOfGroup to return true.")

	// Test GetGroupMemberIDs
	logger.Info("Test GetGroupMemberIDs function")

	// members of group1: user1, user2, user3, group2, group1(itself)
	memberlist, err := SlowGetGroupMemberIDs(stub, testGroupID1)
	test_utils.AssertTrue(t, err == nil, "Expect GetGroupMemberIDs to be successful.")
	expectlist := []string{testGroupID1, testGroupID2, testUserID1, testUserID2, testUserID3}
	test_utils.AssertTrue(t, reflect.DeepEqual(memberlist, expectlist), "Expect memberlist to be correct.")

	// members of group2: user2, user3, group2(itself)
	memberlist, err = SlowGetGroupMemberIDs(stub, testGroupID2)
	test_utils.AssertTrue(t, err == nil, "Expect GetGroupMemberIDs to be successful.")
	expectlist = []string{testGroupID2, testUserID2, testUserID3}
	test_utils.AssertTrue(t, reflect.DeepEqual(memberlist, expectlist), "Expect memberlist to be correct.")

	// Test GetGroupAdminIDs
	logger.Info("Test GetGroupAdminIDs function")

	// Admins of group1: user1, group1
	adminlist, err := SlowGetGroupAdminIDs(stub, testGroupID1)
	test_utils.AssertTrue(t, err == nil, "Expect GetGroupAdminIDs to be successful.")
	expectlist = []string{testGroupID1, testUserID1}
	test_utils.AssertTrue(t, reflect.DeepEqual(adminlist, expectlist), "Expect adminlist to be correct.")

	// Admins of group2: user1, user3, group1, group2
	adminlist, err = SlowGetGroupAdminIDs(stub, testGroupID2)
	test_utils.AssertTrue(t, err == nil, "Expect GetGroupAdminIDs to be successful.")
	expectlist = []string{testGroupID1, testGroupID2, testUserID1, testUserID3}
	test_utils.AssertTrue(t, reflect.DeepEqual(adminlist, expectlist), "Expect adminlist to be correct.")
	mstub.MockTransactionEnd("t1")
}

func userEqual(u1 data_model.User, u2 data_model.User) bool {
	// if u1 == nil && u2 == nil {
	// 	return true
	// } else if u1 == nil {
	// 	return false
	// } else if u2 == nil {
	// 	return false
	// }

	// check two user object's public data fields equal
	if u1.ID != u2.ID {
		return false
	}
	if u1.Name != u2.Name {
		return false
	}
	if u1.PublicKeyB64 != u2.PublicKeyB64 {
		return false
	}
	if u1.Role != u2.Role {
		return false
	}
	if u1.IsGroup != u2.IsGroup {
		return false
	}
	if u1.Status != u2.Status {
		return false
	}
	return true
}

// Test user_mgmt.RegisterUser function in different scenarios
func TestRegisterUser(t *testing.T) {
	//logger.SetLevel(shim.LogDebug)

	logger.Info("TestRegisterUser function called")
	mstub := setup(t)

	// Create caller and register, and also test its data can be retrieved from ledger
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("callerID")
	callerBytes, _ := json.Marshal(&caller)
	_, err := RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	//check key relationships
	keyPath, err := key_mgmt_i.SlowVerifyAccess(stub, caller.GetPubPrivKeyId(), caller.GetSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) > 0, "Access to user sym key from user private key was not added properly")
	keyPath, err = key_mgmt_i.SlowVerifyAccess(stub, caller.GetPrivateKeyHashSymKeyId(), caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) > 0, "Access to user sym key from user private key was not added properly")
	keyPath, err = key_mgmt_i.SlowVerifyAccess(stub, caller.GetSymKeyId(), caller.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Verify Access was successful")
	test_utils.AssertTrue(t, len(keyPath) > 0, "Access to user log sym key from user sym key was not added properly")

	newuser, err := GetUserData(stub, caller, caller.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected getUserData to succeed")
	test_utils.AssertTrue(t, newuser.ID == caller.ID, "Expected getUserData to succeed")
	mstub.MockTransactionEnd("t1")

	// Create user1 and register user by caller, also allow add access from caller to user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("userID1")
	userBytes, _ := json.Marshal(&user1)
	_, err = RegisterUser(stub, caller, []string{string(userBytes), "true"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create user2 which is another user and has no access to user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user2 := test_utils.CreateTestUser("userID2")
	userBytes2, _ := json.Marshal(&user2)
	_, err = RegisterUser(stub, user2, []string{string(userBytes2), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Call RegisterUser with caller.ID equals user.ID, should update user data
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user1.Name = "modifiedUserName"
	user1.Email = "modified@someaddress.com"
	userBytes, _ = json.Marshal(&user1)
	_, err = RegisterUser(stub, user1, []string{string(userBytes), "true"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	returnedUser, err := GetUserData(stub, user1, user1.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected getUserData to succeed")
	test_utils.AssertTrue(t, userEqual(returnedUser, user1), "Expected user public data to be updated")
	test_utils.AssertTrue(t, returnedUser.Email == user1.Email, "Expected user private data to be updated")
	mstub.MockTransactionEnd("t1")

	// Call RegisterUser to update user1 by user2, should fail because user2 does not have access to user1 sym key
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user1.Name = "modifiedUserName2"
	user1.Email = "modified2@someaddress.com"
	userBytes, _ = json.Marshal(&user1)
	_, err = RegisterUser(stub, user2, []string{string(userBytes), "true"})
	test_utils.AssertTrue(t, err != nil, "Expected RegisterUser to fail")
	mstub.MockTransactionEnd("t1")

	// Call RegisterUser by creator of user1 (caller), should succeed
	// NOT TRUE ANYMORE: even if caller is previously given access with allowAccess, they do not have write access
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user1.Name = "modifiedUserName3"
	user1.Email = "modified3@someaddress.com"
	userBytes, _ = json.Marshal(&user1)
	_, err = RegisterUser(stub, caller, []string{string(userBytes), "true"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to fail")
	mstub.MockTransactionEnd("t1")

	// Call RegisterUser with wrong user, should fail
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterUser(stub, user2, []string{string(userBytes), "true"})
	test_utils.AssertTrue(t, err != nil, "Expected RegisterUser to fail")
	mstub.MockTransactionEnd("t1")
}

func TestRegisterUser_OffChain(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestRegisterUser_OffChain function called")
	mstub := setup(t)

	// Register caller
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Setup cloudant datastore
	datastoreConnectionID := "cloudant1"
	err = cloudant_datastore_test_utils.SetupDatastore(mstub, caller, datastoreConnectionID)
	test_utils.AssertTrue(t, err == nil, "SetupDatastore should be successful")

	// Register user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("userID1")
	user1.ConnectionID = datastoreConnectionID
	userBytes, _ := json.Marshal(&user1)
	_, err = RegisterUser(stub, caller, []string{string(userBytes), "true"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create user2 which has no access to user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user2 := test_utils.CreateTestUser("userID2")
	userBytes2, _ := json.Marshal(&user2)
	_, err = RegisterUser(stub, user2, []string{string(userBytes2), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Get user1 data (as user1)
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	returnedUser, err := GetUserData(stub, user1, user1.ID, true, true)
	test_utils.AssertTrue(t, err == nil, "Expected getUserData to succeed")
	test_utils.AssertTrue(t, userEqual(returnedUser, user1), "Expected user public data to be updated")
	test_utils.AssertTrue(t, returnedUser.ID == user1.ID, "Expected access to user public ID")
	test_utils.AssertTrue(t, returnedUser.Name == user1.Name, "Expected access to user public Name")
	test_utils.AssertTrue(t, returnedUser.Role == user1.Role, "Expected access to user public Role")
	test_utils.AssertTrue(t, returnedUser.PublicKeyB64 == user1.PublicKeyB64, "Expected access to user public PublicKeyB64")
	test_utils.AssertTrue(t, returnedUser.IsGroup == user1.IsGroup, "Expected access to user public IsGroup")
	test_utils.AssertTrue(t, returnedUser.Status == user1.Status, "Expected access to user public Status")
	test_utils.AssertTrue(t, reflect.DeepEqual(returnedUser.SolutionPublicData, user1.SolutionPublicData), "Expected access to user public SolutionPublicData")
	test_utils.AssertTrue(t, returnedUser.ConnectionID == user1.ConnectionID, "Expected access to user public ConnectionID")

	test_utils.AssertTrue(t, returnedUser.Email == user1.Email, "Expected access to user private Email")
	test_utils.AssertTrue(t, returnedUser.SymKeyB64 == user1.SymKeyB64, "Expected access to user private SymKeyB64")
	test_utils.AssertTrue(t, returnedUser.PrivateKeyB64 == user1.PrivateKeyB64, "Expected access to user private PrivateKeyB64")
	test_utils.AssertTrue(t, returnedUser.KmsPublicKeyId == user1.KmsPublicKeyId, "Expected access to user private KmsPublicKeyId")
	test_utils.AssertTrue(t, returnedUser.KmsPrivateKeyId == user1.KmsPrivateKeyId, "Expected access to user private KmsPrivateKeyId")
	test_utils.AssertTrue(t, returnedUser.KmsSymKeyId == user1.KmsSymKeyId, "Expected access to user private KmsSymKeyId")
	test_utils.AssertTrue(t, returnedUser.Secret == user1.Secret, "Expected access to user private Secret")
	test_utils.AssertTrue(t, reflect.DeepEqual(returnedUser.SolutionPrivateData, user1.SolutionPrivateData), "Expected access to user public SolutionPrivateData")
	mstub.MockTransactionEnd("t1")

	// Attempt to get user1 data (as user2)
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	returnedUser, err = GetUserData(stub, user2, user1.ID, true, true)
	test_utils.AssertTrue(t, err == nil, "Expected getUserData to succeed")
	test_utils.AssertTrue(t, returnedUser.ID == user1.ID, "Expected access to user public ID")
	test_utils.AssertTrue(t, returnedUser.Name == user1.Name, "Expected access to user public Name")
	test_utils.AssertTrue(t, returnedUser.Role == user1.Role, "Expected access to user public Role")
	test_utils.AssertTrue(t, returnedUser.PublicKeyB64 == user1.PublicKeyB64, "Expected access to user public PublicKeyB64")
	test_utils.AssertTrue(t, returnedUser.IsGroup == user1.IsGroup, "Expected access to user public IsGroup")
	test_utils.AssertTrue(t, returnedUser.Status == user1.Status, "Expected access to user public Status")
	test_utils.AssertTrue(t, reflect.DeepEqual(returnedUser.SolutionPublicData, user1.SolutionPublicData), "Expected access to user public SolutionPublicData")
	test_utils.AssertTrue(t, returnedUser.ConnectionID == user1.ConnectionID, "Expected access to user public ConnectionID")

	test_utils.AssertTrue(t, returnedUser.Email == "", "Expected no access to user private Email")
	test_utils.AssertTrue(t, returnedUser.SymKeyB64 == "", "Expected no access to user private SymKeyB64")
	test_utils.AssertTrue(t, returnedUser.PrivateKeyB64 == "", "Expected no access to user private PrivateKeyB64")
	test_utils.AssertTrue(t, returnedUser.KmsPublicKeyId == "", "Expected no access to user private KmsPublicKeyId")
	test_utils.AssertTrue(t, returnedUser.KmsPrivateKeyId == "", "Expected no access to user private KmsPrivateKeyId")
	test_utils.AssertTrue(t, returnedUser.KmsSymKeyId == "", "Expected no access to user private KmsSymKeyId")
	test_utils.AssertTrue(t, returnedUser.Secret == "", "Expected no access to user private Secret")
	test_utils.AssertTrue(t, returnedUser.SolutionPrivateData == nil, "Expected no access to user public SolutionPrivateData")
	mstub.MockTransactionEnd("t1")

	// Update user1 as user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user1.Status = "inactive"
	user1.Email = "modified@someaddress.com"
	userBytes, _ = json.Marshal(&user1)
	_, err = RegisterUser(stub, user1, []string{string(userBytes), "true"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	returnedUser, err = GetUserData(stub, user1, user1.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected getUserData to succeed")
	test_utils.AssertTrue(t, returnedUser.Status == user1.Status, "Expected user public data to be updated")
	test_utils.AssertTrue(t, returnedUser.Email == user1.Email, "Expected user private data to be updated")
	mstub.MockTransactionEnd("t1")

	// Attempt to update user1 by user2 (should fail)
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user1.Status = "active"
	user1.Email = "modified2@someaddress.com"
	userBytes, _ = json.Marshal(&user1)
	_, err = RegisterUser(stub, user2, []string{string(userBytes), "true"})
	test_utils.AssertTrue(t, err != nil, "Expected RegisterUser to fail")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	returnedUser, err = GetUserData(stub, user1, user1.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected getUserData to succeed")
	test_utils.AssertTrue(t, returnedUser.Status != user1.Status, "Expected user public data to not be updated")
	test_utils.AssertTrue(t, returnedUser.Email != user1.Email, "Expected user private data to not be updated")
	mstub.MockTransactionEnd("t1")
}

func TestRegisterUser_OffChain_NoDatastore(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestRegisterUser_OffChain_NoDatastore function called")
	mstub := setup(t)

	// Register caller
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	_, err := RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Attempt to register user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("userID1")
	datastoreConnectionID := "cloudant1" // not set up
	user1.ConnectionID = datastoreConnectionID
	userBytes, _ := json.Marshal(&user1)
	_, err = RegisterUser(stub, caller, []string{string(userBytes), "true"})
	test_utils.AssertTrue(t, err != nil, "Expected RegisterUser to fail")
	expectedErrorMsg := fmt.Sprintf("DatastoreConnection with ID %v does not exist.", datastoreConnectionID)
	test_utils.AssertTrue(t, strings.Contains(err.Error(), expectedErrorMsg), fmt.Sprintf("Expected error message: %v", expectedErrorMsg))
	mstub.MockTransactionEnd("t1")

	// Attempt to get user1 data
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	returnedUser, err := GetUserData(stub, user1, user1.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected getUserData to succeed")
	test_utils.AssertTrue(t, userEqual(returnedUser, data_model.User{}), "Expected user data to be empty")
	mstub.MockTransactionEnd("t1")
}

func TestRegisterUser_Roles(t *testing.T) {
	logger.Info("TestRegisterUser_Roles function called")
	mstub := setup(t)

	// Create user with Role = user
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user := test_utils.CreateTestUser("user")
	user.Role = global.ROLE_USER
	userBytes, _ := json.Marshal(&user)
	_, err := RegisterUser(stub, user, []string{string(userBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create user with Role = system
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	system := test_utils.CreateTestUser("system")
	system.Role = global.ROLE_SYSTEM_ADMIN
	systemBytes, _ := json.Marshal(&system)
	_, err = RegisterUser(stub, system, []string{string(systemBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create user with Role = org
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	org := test_utils.CreateTestUser("org")
	org.Role = global.ROLE_ORG
	orgBytes, _ := json.Marshal(&org)
	_, err = RegisterUser(stub, org, []string{string(orgBytes), "false"})
	test_utils.AssertTrue(t, err != nil, "Expected RegisterUser to fail")
	mstub.MockTransactionEnd("t1")

	// Create user with Role = audit
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	audit := test_utils.CreateTestUser("audit")
	audit.Role = global.ROLE_AUDIT
	auditBytes, _ := json.Marshal(&audit)
	_, err = RegisterUser(stub, audit, []string{string(auditBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create user with Role = ""
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	noRole := test_utils.CreateTestUser("noRole")
	noRole.Role = ""
	noRoleBytes, _ := json.Marshal(&noRole)
	_, err = RegisterUser(stub, noRole, []string{string(noRoleBytes), "false"})
	test_utils.AssertTrue(t, err != nil, "Expected RegisterUser to fail")
	mstub.MockTransactionEnd("t1")

	// Create user with Role = "abc"
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	abc := test_utils.CreateTestUser("abc")
	abc.Role = "abc"
	abcBytes, _ := json.Marshal(&abc)
	_, err = RegisterUser(stub, abc, []string{string(abcBytes), "false"})
	test_utils.AssertTrue(t, err != nil, "Expected RegisterUser to fail")
	mstub.MockTransactionEnd("t1")
}

// Test update user from JSON string
func TestUpdateUserWithJSON(t *testing.T) {
	//logger.SetLevel(shim.LogDebug)

	mstub := setup(t)

	// Create caller and register, and also test its data can be retrieved from ledger
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	callerID := "callerID"
	caller := test_utils.CreateTestUser(callerID)
	callerBytes, _ := json.Marshal(&caller)
	_, err := RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	newuser, err := GetUserData(stub, caller, caller.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected getUserData to succeed")
	test_utils.AssertTrue(t, newuser.ID == caller.ID, "Expected getUserData to succeed")
	mstub.MockTransactionEnd("t1")

	// Create user1 and register user by caller, also allow add access from caller to user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	userID1 := "userID1"
	user1 := test_utils.CreateTestUser(userID1)
	user1.SolutionPrivateData = map[string]interface{}{"ssn": "111-11-1111", "address": "some address"}
	userBytes, _ := json.Marshal(&user1)
	_, err = RegisterUser(stub, caller, []string{string(userBytes), "true"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Call RegisterUser with JSON string, should update user data
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	userBytes2 := []byte(`{"id":"userID1","role":"user","name":"modifiedUserName", "solution_private_data":{"ssn":"111-11-1112", "address":"new address"}}`)
	_, err = RegisterUser(stub, user1, []string{string(userBytes2), "true"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// make sure to reload from ledger
	returnedUser, err := GetUserData(stub, user1, user1.ID, false, true)
	test_utils.AssertTrue(t, err == nil, "Expected getUserData to succeed")
	test_utils.AssertTrue(t, returnedUser.Name == "modifiedUserName", "Expected user name to be updated")
	test_utils.AssertTrue(t, returnedUser.Email == user1.Email, "Expected user email to remain the same as before")
	returnedUserData := returnedUser.SolutionPrivateData.(map[string]interface{})
	test_utils.AssertTrue(t, returnedUserData["address"] == "new address", "Expected user address to be updated")
	mstub.MockTransactionEnd("t1")
}

func TestGetUserKeys(t *testing.T) {
	logger.Info("TestGetUserKeys function called")
	mstub := setup(t)

	// Create caller and add to ledger
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	callerID := "callerID"
	caller := test_utils.CreateTestUser(callerID)
	registerUserInternal(stub, caller, caller, true)
	mstub.MockTransactionEnd("t1")

	// Create user and add to ledger
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	userID := "userID"
	user := test_utils.CreateTestUser(userID)
	registerUserInternal(stub, user, user, true)
	mstub.MockTransactionEnd("t1")

	symkey_caller := data_model.Key{}
	symkey_caller.ID = caller.GetSymKeyId()
	symkey_caller.Type = global.KEY_TYPE_SYM
	symkey_caller.KeyBytes = caller.SymKey

	privkey_user := data_model.Key{}
	privkey_user.ID = user.GetPubPrivKeyId()
	privkey_user.Type = global.KEY_TYPE_PRIVATE
	privkey_user.KeyBytes = crypto.PrivateKeyToBytes(user.PrivateKey)

	// Before adding key access, GetUserKeys should fail
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	userKeys, err := GetUserKeys(stub, caller, user.ID)
	test_utils.AssertTrue(t, err != nil, "Expected GetUserKeys to fail")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = key_mgmt_i.AddAccess(stub, symkey_caller, privkey_user)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	userKeys, err = GetUserKeys(stub, caller, user.ID, []string{symkey_caller.ID, privkey_user.ID})
	test_utils.AssertTrue(t, err == nil, "Expected GetUserKeys to succeed")
	test_utils.AssertTrue(t, userKeys.PublicKey == user.PublicKeyB64, "Expected to get correct public key")
	test_utils.AssertTrue(t, userKeys.PrivateKey == user.PrivateKeyB64, "Expected to get correct private key")
	test_utils.AssertTrue(t, userKeys.SymKey == user.SymKeyB64, "Expected to get correct sym key")
	mstub.MockTransactionEnd("t1")
}

func TestGetUserPublicKey(t *testing.T) {
	logger.Info("TestGetUserPublicKey function called")
	mstub := setup(t)
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	// Create caller and add to ledger
	callerID := "callerID"
	caller := test_utils.CreateTestUser(callerID)
	registerUserInternal(stub, caller, caller, true)

	// Create user and add to ledger
	userID := "userID"
	user := test_utils.CreateTestUser(userID)
	registerUserInternal(stub, user, user, true)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// Test get user public key
	userPublicKey, err := GetUserPublicKey(stub, caller, user.ID)
	test_utils.AssertTrue(t, err == nil, "Expected GetUserPublicKey to succeed")
	test_utils.AssertTrue(t, userPublicKey.ID == user.GetPubPrivKeyId(), "Expected to return correct key ID")
	test_utils.AssertTrue(t, userPublicKey.Type == global.KEY_TYPE_PUBLIC, "Expected to return correct key type")
	test_utils.AssertTrue(t, reflect.DeepEqual(userPublicKey.KeyBytes, crypto.PublicKeyToBytes(user.PublicKey)), "Expected to get correct public key")

	// Test get user public key by wrong user ID, should throw error
	userPublicKey, err = GetUserPublicKey(stub, caller, "no-existing-id")
	test_utils.AssertTrue(t, err != nil, "Expected GetUserPublicKey to fail")
	mstub.MockTransactionEnd("t1")
}

func TestGetUsers(t *testing.T) {
	logger.Info("TestGetUsers function called")
	mstub := setup(t)

	// Create sys admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	systemAdmin := test_utils.CreateTestUser("systemAdmin")
	systemAdmin.Role = "system"
	systemAdminBytes, _ := json.Marshal(&systemAdmin)
	_, err := RegisterUser(stub, systemAdmin, []string{string(systemAdminBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	user1Bytes, _ := json.Marshal(&user1)
	_, err = RegisterUser(stub, user1, []string{string(user1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create user2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user2 := test_utils.CreateTestUser("user2")
	user2Bytes, _ := json.Marshal(&user2)
	_, err = RegisterUser(stub, user2, []string{string(user2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create org1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	org1 := test_utils.CreateTestGroup("org1")
	org1Bytes, _ := json.Marshal(&org1)
	_, err = RegisterOrg(stub, org1, []string{string(org1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterOrg to succeed")
	mstub.MockTransactionEnd("t1")

	// Add users to org1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, org1, user1.ID, org1.ID, false)
	PutUserInGroup(stub, org1, user2.ID, org1.ID, false)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// test GetUsers
	users := []data_model.User{}
	usersBytes, err := GetUsers(stub, systemAdmin, []string{org1.ID, "user"})
	test_utils.AssertTrue(t, err == nil, "Expected GetUsers to succeed")
	err = json.Unmarshal(usersBytes, &users)
	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal usersBytes to succeed")
	test_utils.AssertTrue(t, len(users) == 2, "Expected to get exactly 2 users")
	test_utils.AssertFalse(t, users[0].IsGroup, "Expected to get only users not groups")
	test_utils.AssertFalse(t, users[1].IsGroup, "Expected to get only users not groups")

	// make sure caller can only see their own private data for GetUsers
	usersBytes, err = GetUsers(stub, user2, []string{org1.ID, "user"})
	test_utils.AssertTrue(t, err == nil, "Expected GetUsers to succeed")
	err = json.Unmarshal(usersBytes, &users)
	for _, user := range users {
		if user.ID == "user2" {
			test_utils.AssertTrue(t, len(user.Email) > 0, "Expected to be able to see user2 private data")
		} else {
			test_utils.AssertFalse(t, len(user.Email) > 0, "Expected not to be able to see "+user.ID+" private data")
		}
	}
	mstub.MockTransactionEnd("t1")
}

func TestGetOrgs(t *testing.T) {
	logger.Info("TestGetOrgs function called")
	mstub := setup(t)

	// Create sys admin
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	systemAdmin := test_utils.CreateTestUser("systemAdmin")
	systemAdmin.Role = "system"
	systemAdminBytes, _ := json.Marshal(&systemAdmin)
	_, err := RegisterUser(stub, systemAdmin, []string{string(systemAdminBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	user1Bytes, _ := json.Marshal(&user1)
	_, err = RegisterUser(stub, user1, []string{string(user1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create user2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user2 := test_utils.CreateTestUser("user2")
	user2Bytes, _ := json.Marshal(&user2)
	_, err = RegisterUser(stub, user2, []string{string(user2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create org1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	org1 := test_utils.CreateTestUser("org1")
	org1.Role = "org"
	org1.IsGroup = true
	//org1Bytes, _ := json.Marshal(&org1)
	err = registerUserInternal(stub, org1, org1, false)
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create org2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	org2 := test_utils.CreateTestUser("org2")
	org2.Role = "org"
	org2.IsGroup = true
	//org2Bytes, _ := json.Marshal(&org2)
	err = registerUserInternal(stub, org2, org2, false)
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// test GetOrgs
	orgs := []data_model.User{}
	orgsBytes, err := GetOrgs(stub, systemAdmin, []string{})
	test_utils.AssertTrue(t, err == nil, "Expected GetOrgs to succeed")
	err = json.Unmarshal(orgsBytes, &orgs)
	test_utils.AssertTrue(t, err == nil, "Expected Unmarshal orgsBytes to succeed")
	test_utils.AssertTrue(t, len(orgs) == 2, "Expected to get exactly 2 orgs")
	test_utils.AssertTrue(t, orgs[0].IsGroup, "Expected to get only groups not users")
	test_utils.AssertTrue(t, orgs[1].IsGroup, "Expected to get only groups not users")

	// make sure caller can only see their own private data for GetOrgs
	orgsBytes, err = GetOrgs(stub, org1, []string{})
	test_utils.AssertTrue(t, err == nil, "Expected GetOrgs to succeed")
	err = json.Unmarshal(orgsBytes, &orgs)
	for _, org := range orgs {
		if org.ID == "org1" {
			test_utils.AssertTrue(t, len(org.Email) > 0, "Expected to be able to see org1 private data")
		} else {
			test_utils.AssertFalse(t, len(org.Email) > 0, "Expected not to be able to see "+org.ID+" private data")
		}
	}
	mstub.MockTransactionEnd("t1")
}

func TestGetUsers_Filter(t *testing.T) {
	logger.Info("TestGetUsers function called")
	mstub := setup(t)

	// Create org1
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	org1 := test_utils.CreateTestGroup("org1")
	org1Bytes, _ := json.Marshal(&org1)
	args := []string{string(org1Bytes), "false"}
	_, err := RegisterOrg(stub, org1, args)
	test_utils.AssertTrue(t, err == nil, "RegisterOrg should be successful")
	mstub.MockTransactionEnd("t1")

	// Create org2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	org2 := test_utils.CreateTestGroup("org2")
	org2Bytes, _ := json.Marshal(&org2)
	args = []string{string(org2Bytes), "false"}
	_, err = RegisterOrg(stub, org2, args)
	test_utils.AssertTrue(t, err == nil, "RegisterOrg should be successful")
	mstub.MockTransactionEnd("t1")

	// Create user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	user1Bytes, _ := json.Marshal(&user1)
	_, err = RegisterUser(stub, user1, []string{string(user1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Create user2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	user2 := test_utils.CreateTestUser("user2")
	user2.Role = global.ROLE_AUDIT
	user2Bytes, _ := json.Marshal(&user2)
	_, err = RegisterUser(stub, user2, []string{string(user2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Put user1 in org1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, org1, user1.ID, org1.ID, false)
	mstub.MockTransactionEnd("t1")

	// Put user2 in org2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, org2, user2.ID, org2.ID, false)
	mstub.MockTransactionEnd("t1")

	users := []data_model.User{}

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	// get only org1 users
	usersBytes, err := GetUsers(stub, user1, []string{"org1", "user"})
	test_utils.AssertTrue(t, err == nil, "Expected GetUsers to succeed")
	err = json.Unmarshal(usersBytes, &users)
	test_utils.AssertTrue(t, len(users) == 1, "Expected to get exactly 1 users")
	test_utils.AssertTrue(t, users[0].ID == "user1", "Expected to get user1")

	// get org1 users (including the org1 group itself)
	usersBytes, err = GetUsers(stub, user1, []string{"org1"})
	test_utils.AssertTrue(t, err == nil, "Expected GetUsers to succeed")
	err = json.Unmarshal(usersBytes, &users)
	test_utils.AssertTrue(t, len(users) == 2, "Expected to get exactly 2 users")
	test_utils.AssertTrue(t, users[0].ID == "org1", "Expected to get org1")
	test_utils.AssertTrue(t, users[1].ID == "user1", "Expected to get user1")

	// get only org2 users with role = global.ROLE_AUDIT
	usersBytes, err = GetUsers(stub, user2, []string{"org2", global.ROLE_AUDIT})
	test_utils.AssertTrue(t, err == nil, "Expected GetUsers to succeed")
	err = json.Unmarshal(usersBytes, &users)
	test_utils.AssertTrue(t, len(users) == 1, "Expected to get exactly 1 users")
	test_utils.AssertTrue(t, users[0].ID == "user2", "Expected to get user2")
	mstub.MockTransactionEnd("t1")
}

func TestIsParentGroup_DirectParent(t *testing.T) {
	logger.Info("TestIsParentGroup_DirectParent function called")
	mstub := setup(t)

	// Create groups
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	group1 := test_utils.CreateTestGroup("group1")
	group2 := test_utils.CreateTestGroup("group2")
	registerUserInternal(stub, group1, group1, true)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	RegisterSubgroupWithParams(stub, group1, group2, group1.ID)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	test_utils.AssertTrue(t, IsParentGroup(stub, group1, group1.ID, group2.ID), "Expected group1 to be a parent group of group2")
	test_utils.AssertTrue(t, IsParentGroup(stub, group2, group1.ID, group2.ID), "Expected group1 to be a parent group of group2")
	mstub.MockTransactionEnd("t1")
}

func TestIsParentGroup_InvalidInput(t *testing.T) {
	logger.Info("TestIsParentGroup_InvalidInput function called")
	mstub := setup(t)

	// Create users and group
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	user2 := test_utils.CreateTestUser("user2")
	group1 := test_utils.CreateTestGroup("group1")
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, user2, user2, true)
	registerUserInternal(stub, group1, group1, true)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	test_utils.AssertFalse(t, IsParentGroup(stub, user1, user1.ID, user2.ID), "Expected false because a user was passed in to IsParentGroup")
	test_utils.AssertFalse(t, IsParentGroup(stub, group1, group1.ID, user2.ID), "Expected false because a user was passed in to IsParentGroup")
	test_utils.AssertFalse(t, IsParentGroup(stub, group1, group1.ID, user1.ID), "Expected false because a user was passed in to IsParentGroup")
	mstub.MockTransactionEnd("t1")

	//add users to group and make sure the IsParentGroup still returns false because IsParentGroup does not allow you to pass in users
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group1, user1.ID, group1.ID, false)
	PutUserInGroup(stub, group1, user2.ID, group1.ID, false)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	test_utils.AssertFalse(t, IsParentGroup(stub, user1, user1.ID, user2.ID), "Expected false because a user was passed in to IsParentGroup")
	test_utils.AssertFalse(t, IsParentGroup(stub, group1, group1.ID, user2.ID), "Expected false because a user was passed in to IsParentGroup")
	test_utils.AssertFalse(t, IsParentGroup(stub, group1, group1.ID, user1.ID), "Expected false because a user was passed in to IsParentGroup")
	mstub.MockTransactionEnd("t1")
}

func TestGetMyDirectGroupIDs(t *testing.T) {
	logger.Info("TestGetMyDirectGroupIDs function called")
	mstub := setup(t)

	//create user and groups
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	group1 := test_utils.CreateTestGroup("group1")
	group2 := test_utils.CreateTestGroup("group2")
	group3 := test_utils.CreateTestGroup("group3")
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, group1, group1, true)
	registerUserInternal(stub, group2, group2, true)
	registerUserInternal(stub, group3, group3, true)
	mstub.MockTransactionEnd("t1")

	//user1 is currently not a member of any group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	groupIDs, err := user_mgmt_c.GetMyDirectGroupIDs(stub, user1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.GetMyDirectGroupIDs to succeed")
	test_utils.AssertTrue(t, len(groupIDs) == 0, "Expected user_mgmt_c.GetMyDirectGroupIDs to return 0 groupIDs")
	mstub.MockTransactionEnd("t1")

	//make user a member of group1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group1, user1.ID, group1.ID, false)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionEnd("t1")
	groupIDs, err = user_mgmt_c.GetMyDirectGroupIDs(stub, user1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.GetMyDirectGroupIDs to succeed")
	test_utils.AssertTrue(t, len(groupIDs) == 1, "Expected user_mgmt_c.GetMyDirectGroupIDs to return 1 groupIDs")
	test_utils.AssertTrue(t, groupIDs[0] == "group1", "Expected groupIDs to contain group1")
	mstub.MockTransactionEnd("t1")

	//make user an admin of group2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group2, user1.ID, group2.ID, true)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	groupIDs, err = user_mgmt_c.GetMyDirectGroupIDs(stub, user1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.GetMyDirectGroupIDs to succeed")
	test_utils.AssertTrue(t, len(groupIDs) == 2, "Expected user_mgmt_c.GetMyDirectGroupIDs to return 2 groupIDs")
	mstub.MockTransactionEnd("t1")

	//remove user from group1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	args := []string{user1.ID, group1.ID}
	RemoveUserFromGroup(stub, group1, args)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	groupIDs, err = user_mgmt_c.GetMyDirectGroupIDs(stub, user1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.GetMyDirectGroupIDs to succeed")
	test_utils.AssertTrue(t, len(groupIDs) == 1, "Expected user_mgmt_c.GetMyDirectGroupIDs to return 1 groupIDs")
	test_utils.AssertTrue(t, groupIDs[0] == "group2", "Expected groupIDs to contain group2")
	mstub.MockTransactionEnd("t1")
}

func TestGetMyDirectGroupIDs_GetParentGroupsOfSubgroup(t *testing.T) {
	logger.Info("TestGetMyDirectGroupIDs_GetParentGroupsOfSubgroup function called")
	mstub := setup(t)

	//create user and groups
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	group1 := test_utils.CreateTestGroup("group1")
	group2 := test_utils.CreateTestGroup("group2")
	registerUserInternal(stub, group1, group1, true)
	mstub.MockTransactionEnd("t1")

	//make group2 a subgroup of group1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	RegisterSubgroupWithParams(stub, group1, group2, group1.ID)
	mstub.MockTransactionEnd("t1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	groupIDs, err := user_mgmt_c.GetMyDirectGroupIDs(stub, group2.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.GetMyDirectGroupIDs to succeed")
	test_utils.AssertTrue(t, len(groupIDs) == 1, "Expected user_mgmt_c.GetMyDirectGroupIDs to return 1 groupIDs")
	test_utils.AssertTrue(t, groupIDs[0] == "group1", "Expected groupIDs to contain group1")
	mstub.MockTransactionEnd("t1")
}

// Tests calling GetMyGroupIDs for a user
func TestGetMyGroupIDs_User(t *testing.T) {
	logger.Info("TestGetMyGroupIDs_User function called")
	mstub := setup(t)

	//create user and groups
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	group1 := test_utils.CreateTestGroup("group1")
	group2 := test_utils.CreateTestGroup("group2")
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, group1, group1, true)
	mstub.MockTransactionEnd("t1")

	//user1 is currently not a member of any group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	groupIDs, err := SlowGetMyGroupIDs(stub, user1, user1.ID, false)
	test_utils.AssertTrue(t, err == nil, "Expected GetMyGroupIDs to succeed")
	test_utils.AssertTrue(t, len(groupIDs) == 0, "Expected GetMyGroupIDs to return 0 groupIDs")
	mstub.MockTransactionEnd("t1")

	//make group2 a member of group1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	RegisterSubgroupWithParams(stub, group1, group2, group1.ID)
	mstub.MockTransactionEnd("t1")

	//make user a member of group2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group2, user1.ID, group2.ID, false)
	mstub.MockTransactionEnd("t1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	groupIDs, err = SlowGetMyGroupIDs(stub, user1, user1.ID, false)
	test_utils.AssertTrue(t, err == nil, "Expected GetMyGroupIDs to succeed")
	test_utils.AssertTrue(t, len(groupIDs) == 2, "Expected GetMyGroupIDs to return 2 groupIDs")
	test_utils.AssertInLists(t, "group1", groupIDs, "Expected groupIDs to contain group1")
	test_utils.AssertInLists(t, "group2", groupIDs, "Expected groupIDs to contain group2")
	mstub.MockTransactionEnd("t1")
}

// Tests calling GetMyGroupIDs for a group
func TestGetMyGroupIDs_Group(t *testing.T) {
	logger.Info("TestGetMyGroupIDs_Group function called")
	mstub := setup(t)

	// Create group1
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	group1 := test_utils.CreateTestGroup("group1")
	registerOrgInternal(stub, group1, group1, false)
	mstub.MockTransactionEnd("t1")

	// Add group2 to group1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	group2 := test_utils.CreateTestGroup("group2")
	RegisterSubgroupWithParams(stub, group1, group2, group1.ID)
	mstub.MockTransactionEnd("t1")

	// Add group3 to group2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	group3 := test_utils.CreateTestGroup("group3")
	RegisterSubgroupWithParams(stub, group2, group3, group2.ID)
	mstub.MockTransactionEnd("t1")

	// Call GetMyGroupIDs for group3
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	groupIDs, err := SlowGetMyGroupIDs(stub, group3, group3.ID, false)
	logger.Debugf("groupIDs: %v, err: %v", groupIDs, err)
	test_utils.AssertTrue(t, err == nil, "Expected GetMyGroupIDs to succeed")
	test_utils.AssertTrue(t, len(groupIDs) == 3, "Expected GetMyGroupIDs to return 3 groupIDs")
	test_utils.AssertInLists(t, "group1", groupIDs, "Expected groupIDs to contain group1")
	test_utils.AssertInLists(t, "group2", groupIDs, "Expected groupIDs to contain group2")
	test_utils.AssertInLists(t, "group2", groupIDs, "Expected groupIDs to contain grou32")
	mstub.MockTransactionEnd("t1")
}

func TestGetMyDirectAdminGroupIDs(t *testing.T) {
	logger.Info("TestGetMyDirectAdminGroupIDs function called")
	mstub := setup(t)

	//create user and groups
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	group1 := test_utils.CreateTestGroup("group1")
	group2 := test_utils.CreateTestGroup("group2")
	group3 := test_utils.CreateTestGroup("group3")
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, group1, group1, true)
	registerUserInternal(stub, group2, group2, true)
	registerUserInternal(stub, group3, group3, true)
	mstub.MockTransactionEnd("t1")

	//make user1 a member of group1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group1, user1.ID, group1.ID, false)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	groupIDs, err := user_mgmt_c.GetMyDirectAdminGroupIDs(stub, user1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.GetMyDirectAdminGroupIDs to succeed")
	test_utils.AssertTrue(t, len(groupIDs) == 0, "Expected user_mgmt_c.GetMyDirectAdminGroupIDs to return 0 groupIDs")
	mstub.MockTransactionEnd("t1")

	//make user1 an admin of group2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group2, user1.ID, group2.ID, true)
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	groupIDs, err = user_mgmt_c.GetMyDirectAdminGroupIDs(stub, user1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.GetMyDirectAdminGroupIDs to succeed")
	test_utils.AssertTrue(t, len(groupIDs) == 1, "Expected user_mgmt_c.GetMyDirectAdminGroupIDs to return 1 groupIDs")
	test_utils.AssertTrue(t, groupIDs[0] == "group2", "Expected groupIDs to contain group2")
	mstub.MockTransactionEnd("t1")

	//make user1 admin of group1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group1, user1.ID, group1.ID, true)
	mstub.MockTransactionEnd("t1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	groupIDs, err = user_mgmt_c.GetMyDirectAdminGroupIDs(stub, user1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.GetMyDirectAdminGroupIDs to succeed")
	test_utils.AssertTrue(t, len(groupIDs) == 2, "Expected user_mgmt_c.GetMyDirectAdminGroupIDs to return 2 groupIDs")
	mstub.MockTransactionEnd("t1")

	//make user a member of group2 (remove admin status)
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group2, user1.ID, group2.ID, false)
	mstub.MockTransactionEnd("t1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	groupIDs, err = user_mgmt_c.GetMyDirectAdminGroupIDs(stub, user1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.GetMyDirectAdminGroupIDs to succeed")
	test_utils.AssertTrue(t, len(groupIDs) == 1, "Expected user_mgmt_c.GetMyDirectAdminGroupIDs to return 1 groupIDs")
	test_utils.AssertTrue(t, groupIDs[0] == "group1", "Expected groupIDs to contain group1")
	mstub.MockTransactionEnd("t1")
}

func TestGetMyDirectAdminGroupIDs_subGroup(t *testing.T) {
	logger.Info("TestGetMyDirectAdminGroupIDs_InvalidInput function called")
	mstub := setup(t)

	// Register parentOrg
	parentOrg := test_utils.CreateTestGroup("parentOrg")
	parentOrgBytes, _ := json.Marshal(&parentOrg)
	args := []string{string(parentOrgBytes), "false"}
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	_, err := RegisterOrg(stub, parentOrg, args)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "RegisterOrg should be successful")

	// Test RegisterSubgroup
	subOrg := test_utils.CreateTestGroup("subOrg")
	subOrgBytes, _ := json.Marshal(&subOrg)
	args = []string{string(subOrgBytes), parentOrg.ID}
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterSubgroup(stub, parentOrg, args)
	test_utils.AssertTrue(t, err == nil, "RegisterSubgroup should be successful")
	mstub.MockTransactionEnd("t1")

	//try to get admin group ids of a group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	ids, err := user_mgmt_c.GetMyDirectAdminGroupIDs(stub, subOrg.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.GetMyDirectAdminGroupIDs to succeed")
	test_utils.AssertTrue(t, ids[0] == parentOrg.ID, "Should be parentOrg.ID")

	mstub.MockTransactionEnd("t1")
}

func TestGetMyDirectAdminGroupIDs_CallerIsDifferentFromSubject(t *testing.T) {
	logger.Info("TestGetMyDirectAdminGroupIDs_CallerIsDifferentFromSubject function called")
	mstub := setup(t)

	//create user and groups
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	user2 := test_utils.CreateTestUser("user2")
	group1 := test_utils.CreateTestGroup("group1")
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, user2, user2, true)
	registerUserInternal(stub, group1, group1, true)
	mstub.MockTransactionEnd("t1")

	//make user an admin of group1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group1, user1.ID, group1.ID, true)
	mstub.MockTransactionEnd("t1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	groupIDs, err := user_mgmt_c.GetMyDirectAdminGroupIDs(stub, user1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.GetMyDirectAdminGroupIDs to succeed")
	test_utils.AssertTrue(t, len(groupIDs) == 1, "Expected user_mgmt_c.GetMyDirectAdminGroupIDs to return 1 groupIDs")
	test_utils.AssertTrue(t, groupIDs[0] == "group1", "Expected groupIDs to contain group1")
	mstub.MockTransactionEnd("t1")
}

func TestGiveAdminPermissionOfGroup(t *testing.T) {
	logger.Info("TestGiveAdminPermissionOfGroup function called")
	mstub := setup(t)

	// Create users and group
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	group1 := test_utils.CreateTestGroup("group1")
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, group1, group1, true)
	mstub.MockTransactionEnd("t1")

	//put user1 in group as member
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group1, user1.ID, group1.ID, false)
	mstub.MockTransactionEnd("t1")

	//give admin permission to user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err := GiveAdminPermissionOfGroup(stub, group1, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected GiveAdminPermission to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertTrue(t, isAdmin, "Expected user1 to be admin of group1")
	mstub.MockTransactionEnd("t1")
}

func TestGiveAdminPermissionOfGroup_NonGroupUser(t *testing.T) {
	logger.Info("TestGiveAdminPermissionOfGroup_NonGroupUser function called")
	mstub := setup(t)

	// Create users and group
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	group1 := test_utils.CreateTestGroup("group1")
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, group1, group1, true)
	mstub.MockTransactionEnd("t1")

	//give admin permission to user1; user1 is not a member of group1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err := GiveAdminPermissionOfGroup(stub, group1, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected GiveAdminPermission to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertTrue(t, isAdmin, "Expected user1 to be admin of group1")
	mstub.MockTransactionEnd("t1")
}

func TestGiveAdminPermissionOfGroup_CallerIsNewAdmin(t *testing.T) {
	logger.Info("TestGiveAdminPermissionOfGroup_CallerIsNewAdmin function called")
	mstub := setup(t)

	// Create users and group
	user1 := test_utils.CreateTestUser("user1")
	user2 := test_utils.CreateTestUser("user2")
	group1 := test_utils.CreateTestGroup("group1")
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1Bytes, _ := json.Marshal(&user1)
	_, err := RegisterUser(stub, user1, []string{string(user1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	user2Bytes, _ := json.Marshal(&user2)
	_, err = RegisterUser(stub, user2, []string{string(user2Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	group1Bytes, _ := json.Marshal(&group1)
	_, err = RegisterOrg(stub, group1, []string{string(group1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	//give admin permission to user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = GiveAdminPermissionOfGroup(stub, group1, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected GiveAdminPermission to succeed")
	mstub.MockTransactionEnd("t1")

	//give admin permission to user2 as user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = GiveAdminPermissionOfGroup(stub, user1, user2.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected GiveAdminPermission to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, user2.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertTrue(t, isAdmin, "Expected user2 to be admin of group1")
	mstub.MockTransactionEnd("t1")
}

func TestGiveAdminPermissionOfGroup_CallerNotAdmin(t *testing.T) {
	logger.Info("TestGiveAdminPermissionOfGroup_CallerNotAdmin function called")
	mstub := setup(t)

	// Create users and group
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	user2 := test_utils.CreateTestUser("user2")
	group1 := test_utils.CreateTestGroup("group1")
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, user2, user2, true)
	registerUserInternal(stub, group1, group1, true)
	mstub.MockTransactionEnd("t1")

	//give permission to user2 as user1; user1 is not an admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err := GiveAdminPermissionOfGroup(stub, user1, user2.ID, group1.ID)
	test_utils.AssertFalse(t, err == nil, "Expected GiveAdminPermission to fail")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, user2.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertFalse(t, isAdmin, "Expected user2 not to be admin of group1")
	mstub.MockTransactionEnd("t1")
}

func TestRemoveAdminPermissionOfGroup(t *testing.T) {
	logger.Info("TestRemoveAdminPermissionOfGroup function called")
	mstub := setup(t)

	// Create users and group
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	group1 := test_utils.CreateTestGroup("group1")
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, group1, group1, true)
	mstub.MockTransactionEnd("t1")

	//put user1 in group as admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group1, user1.ID, group1.ID, true)
	mstub.MockTransactionEnd("t1")

	//remove admin permission from user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err := RemoveAdminPermissionOfGroup(stub, group1, []string{user1.ID, group1.ID})
	test_utils.AssertTrue(t, err == nil, "Expected RemoveAdminPermission to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertFalse(t, isAdmin, "Expected user1 not to be admin of group1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err = user_mgmt_c.IsUserAdminOfGroup(stub, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertFalse(t, isAdmin, "Expected user1 not to be admin of group1")

	//make sure user is still in the group after losing admin permission
	inGroup, err := user_mgmt_c.IsUserInGroup(stub, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected IsUserInGroup to succeed")
	test_utils.AssertTrue(t, inGroup, "Expected user1 to be in group1")
	mstub.MockTransactionEnd("t1")
}

func TestRemoveAdminPermissionOfGroup_NonGroupUser(t *testing.T) {
	logger.Info("TestRemoveAdminPermissionOfGroup_NonGroupUser function called")
	mstub := setup(t)

	// Create users and group
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	group1 := test_utils.CreateTestGroup("group1")
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, group1, group1, true)
	mstub.MockTransactionEnd("t1")

	//remove admin permission from user1; user1 is not in group1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err := RemoveAdminPermissionOfGroup(stub, group1, []string{user1.ID, group1.ID})
	test_utils.AssertFalse(t, err == nil, "Expected RemoveAdminPermission to fail")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertFalse(t, isAdmin, "Expected user1 not to be admin of group1")

	//make sure removing permission from a user that was not initially in the group
	//does not result in adding the user to the group
	inGroup, err := user_mgmt_c.IsUserInGroup(stub, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected IsUserInGroup to succeed")
	test_utils.AssertFalse(t, inGroup, "Expected user1 to not be in group1")
	mstub.MockTransactionEnd("t1")
}

func TestRemoveAdminPermissionOfGroup_CallerIsNewAdmin(t *testing.T) {
	logger.Info("TestRemoveAdminPermissionOfGroup_CallerIsNewAdmin function called")
	mstub := setup(t)

	// Create users and group
	user1 := test_utils.CreateTestUser("user1")
	user2 := test_utils.CreateTestUser("user2")
	group1 := test_utils.CreateTestGroup("group1")

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, user2, user2, true)
	registerUserInternal(stub, group1, group1, true)
	mstub.MockTransactionEnd("t1")

	//put user1 in group as admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group1, user1.ID, group1.ID, true)
	mstub.MockTransactionEnd("t1")

	//give admin permission to user2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err := GiveAdminPermissionOfGroup(stub, group1, user2.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected GiveAdminPermission to succeed")
	mstub.MockTransactionEnd("t1")

	//remove permission from user1 as user2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RemoveAdminPermissionOfGroup(stub, user2, []string{user1.ID, group1.ID})
	test_utils.AssertTrue(t, err == nil, "Expected RemoveAdminPermission to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertFalse(t, isAdmin, "Expected user1 to not be admin of group1")

	//make sure removing permission from a user does not kick them out of the group
	inGroup, err := user_mgmt_c.IsUserInGroup(stub, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected IsUserInGroup to succeed")
	test_utils.AssertTrue(t, inGroup, "Expected user1 to be in group1")
	mstub.MockTransactionEnd("t1")
}

func TestRemoveAdminPermissionOfGroup_CallerNotAdmin(t *testing.T) {
	logger.Info("TestRemoveAdminPermissionOfGroup_CallerNotAdmin function called")
	mstub := setup(t)

	// Create users and group
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	user2 := test_utils.CreateTestUser("user2")
	group1 := test_utils.CreateTestGroup("group1")
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, user2, user2, true)
	registerUserInternal(stub, group1, group1, true)
	mstub.MockTransactionEnd("t1")

	//put user1 in group as admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group1, user1.ID, group1.ID, true)
	mstub.MockTransactionEnd("t1")

	//try to remove permission from user1 as user2; user2 is not in the group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err := RemoveAdminPermissionOfGroup(stub, user2, []string{user1.ID, group1.ID})
	test_utils.AssertFalse(t, err == nil, "Expected RemoveAdminPermission to fail")
	mstub.MockTransactionEnd("t1")

	//put user2 in group as member
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group1, user2.ID, group1.ID, false)
	mstub.MockTransactionEnd("t1")

	//try to remove permission from user1 as user2; user2 is in the group but not an admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RemoveAdminPermissionOfGroup(stub, user2, []string{user1.ID, group1.ID})
	test_utils.AssertFalse(t, err == nil, "Expected RemoveAdminPermission to fail")
	mstub.MockTransactionEnd("t1")

	//make sure user1 is still admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, user1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertTrue(t, isAdmin, "Expected user1 to be admin of group1")
	mstub.MockTransactionEnd("t1")
}

func TestRemoveAdminPermissionOfGroup_RemovePermissionFromGroup(t *testing.T) {
	logger.Info("TestRemoveAdminPermissionOfGroup_RemovePermissionFromGroup function called")
	mstub := setup(t)

	// Create users and group
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	group1 := test_utils.CreateTestGroup("group1")
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, group1, group1, true)
	mstub.MockTransactionEnd("t1")

	//put user1 in group as admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, group1, user1.ID, group1.ID, true)
	mstub.MockTransactionEnd("t1")

	//try to remove permission from group1 as user1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err := RemoveAdminPermissionOfGroup(stub, user1, []string{group1.ID, group1.ID})
	test_utils.AssertFalse(t, err == nil, "Expected RemoveAdminPermissionOfGroup to fail")
	mstub.MockTransactionEnd("t1")

	//make sure group1 is still admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, group1.ID, group1.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertTrue(t, isAdmin, "Expected group1 to be admin of group1")
	mstub.MockTransactionEnd("t1")
}

func TestRemoveAdminPermissionOfGroup_Subgroup(t *testing.T) {
	logger.Info("TestRemoveAdminPermissionOfGroup_Subgroup function called")
	mstub := setup(t)

	// Create users and group
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	user1 := test_utils.CreateTestUser("user1")
	orgCreator1 := test_utils.CreateTestUser("orgCreator1")
	orgCreator1.Role = global.ROLE_SYSTEM_ADMIN
	subgroupUser1 := test_utils.CreateTestUser("subgroupUser1")
	subgroupUser2 := test_utils.CreateTestUser("subgroupUser2")
	group1 := test_utils.CreateTestGroup("group1")
	registerUserInternal(stub, user1, user1, true)
	registerUserInternal(stub, orgCreator1, orgCreator1, true)
	registerUserInternal(stub, subgroupUser1, subgroupUser1, true)
	registerUserInternal(stub, subgroupUser2, subgroupUser2, true)
	mstub.MockTransactionEnd("t1")

	// Register group1 from orgCreator1
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	group1Bytes, _ := json.Marshal(&group1)
	_, err := RegisterOrg(stub, orgCreator1, []string{string(group1Bytes), "false"})
	test_utils.AssertTrue(t, err == nil, "registerUserInternal should be successful")
	mstub.MockTransactionEnd("t1")

	// Register subGroup
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	subGroup := test_utils.CreateTestGroup("subGroup")
	subGroupBytes, _ := json.Marshal(&subGroup)
	args := []string{string(subGroupBytes), group1.ID}
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RegisterSubgroup(stub, group1, args)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "RegisterSubgroup should be successful")
	mstub.MockTransactionEnd("t1")

	// put user1 in group as admin
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, group1, user1.ID, group1.ID, true)
	test_utils.AssertTrue(t, err == nil, "should be successful")
	mstub.MockTransactionEnd("t1")

	// put subgroupUser1 and subgroupUser2 to subGroup, also give subgroup admin permission to them
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = GiveAdminPermissionOfGroup(stub, subGroup, subgroupUser1.ID, subGroup.ID)
	test_utils.AssertTrue(t, err == nil, "Expected GiveAdminPermission to succeed")
	err = GiveAdminPermissionOfGroup(stub, subGroup, subgroupUser2.ID, subGroup.ID)
	test_utils.AssertTrue(t, err == nil, "Expected GiveAdminPermission to succeed")
	mstub.MockTransactionEnd("t1")

	// Org admin remove subgroup admin's admin permission
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RemoveAdminPermissionOfGroup(stub, user1, []string{subgroupUser1.ID, subGroup.ID})
	test_utils.AssertTrue(t, err == nil, "Expected RemoveAdminPermission to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, subgroupUser1.ID, subGroup.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertFalse(t, isAdmin, "Expected subgroupUser1 to not be admin of subGroup")

	// make sure removing permission from a user does not kick them out of the group
	inGroup, err := user_mgmt_c.IsUserInGroup(stub, subgroupUser1.ID, subGroup.ID)
	test_utils.AssertTrue(t, err == nil, "Expected IsUserInGroup to succeed")
	test_utils.AssertTrue(t, inGroup, "Expected subgroupUser1 to be in subGroup")
	mstub.MockTransactionEnd("t1")

	// give subgroupUser1 back the admin permission of subGroup
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, subGroup, subgroupUser1.ID, subGroup.ID, true)
	mstub.MockTransactionEnd("t1")

	// Original group1 admin remove subgroupUser1's admin permission
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RemoveAdminPermissionOfGroup(stub, group1, []string{subgroupUser1.ID, subGroup.ID})
	test_utils.AssertTrue(t, err == nil, "Expected RemoveAdminPermission to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err = user_mgmt_c.IsUserAdminOfGroup(stub, subgroupUser1.ID, subGroup.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertFalse(t, isAdmin, "Expected subgroupUser1 to not be admin of subGroup")

	// make sure removing permission from a user does not kick them out of the group
	inGroup, err = user_mgmt_c.IsUserInGroup(stub, subgroupUser1.ID, subGroup.ID)
	test_utils.AssertTrue(t, err == nil, "Expected IsUserInGroup to succeed")
	test_utils.AssertTrue(t, inGroup, "Expected subgroupUser1 to be in subGroup")
	mstub.MockTransactionEnd("t1")

	// give subgroupUser1 back the admin permission of subGroup
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	PutUserInGroup(stub, subGroup, subgroupUser1.ID, subGroup.ID, true)
	mstub.MockTransactionEnd("t1")

	// Subgroup admin remove another admin's admin permission
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RemoveAdminPermissionOfGroup(stub, subgroupUser1, []string{subgroupUser2.ID, subGroup.ID})
	test_utils.AssertTrue(t, err == nil, "Expected RemoveAdminPermission to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err = user_mgmt_c.IsUserAdminOfGroup(stub, subgroupUser2.ID, subGroup.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertFalse(t, isAdmin, "Expected subgroupUser2 to not be admin of subGroup")

	// make sure removing permission from a user does not kick them out of the group
	inGroup, err = user_mgmt_c.IsUserInGroup(stub, subgroupUser2.ID, subGroup.ID)
	test_utils.AssertTrue(t, err == nil, "Expected IsUserInGroup to succeed")
	test_utils.AssertTrue(t, inGroup, "Expected subgroupUser2 to be in subGroup")
	mstub.MockTransactionEnd("t1")

	// Original subgroup admin remove another admin's admin permission
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	_, err = RemoveAdminPermissionOfGroup(stub, subGroup, []string{subgroupUser1.ID, subGroup.ID})
	test_utils.AssertTrue(t, err == nil, "Expected RemoveAdminPermission to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err = user_mgmt_c.IsUserAdminOfGroup(stub, subgroupUser1.ID, subGroup.ID)
	test_utils.AssertTrue(t, err == nil, "Expected user_mgmt_c.IsUserAdminOfGroup to succeed")
	test_utils.AssertFalse(t, isAdmin, "Expected subgroupUser1 to not be admin of subGroup")

	// make sure removing permission from a user does not kick them out of the group
	inGroup, err = user_mgmt_c.IsUserInGroup(stub, subgroupUser1.ID, subGroup.ID)
	test_utils.AssertTrue(t, err == nil, "Expected IsUserInGroup to succeed")
	test_utils.AssertTrue(t, inGroup, "Expected subgroupUser1 to be in subGroup")
	mstub.MockTransactionEnd("t1")
}

func TestGiveAndRemoveAuditorPermissionOfGroupById(t *testing.T) {
	logger.Info("TestGiveAndRemoveAditorPermissionOfGroupById function called")
	mstub := setup(t)

	// create auditor, org, and org admin
	auditor := test_utils.CreateTestUser("auditor")
	auditor.Role = global.ROLE_AUDIT
	RegisterUserForTest(t, mstub, auditor, auditor, false)
	orgAdmin := test_utils.CreateTestUser("orgAdmin")
	orgAdmin.Role = global.ROLE_SYSTEM_ADMIN
	RegisterUserForTest(t, mstub, orgAdmin, orgAdmin, false)
	org := test_utils.CreateTestGroup("org")
	RegisterUserForTest(t, mstub, orgAdmin, org, true)

	// give audit permission to org
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	err := GiveAuditorPermissionOfGroupById(stub, orgAdmin, auditor.ID, org.ID)
	mstub.MockTransactionEnd("t1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	test_utils.AssertTrue(t, err == nil, "Expected GiveAuditorPermissionOfGroupById to succeed")
	path, err := key_mgmt_i.SlowVerifyAccess(stub, auditor.GetPubPrivKeyId(), org.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected VerifyAccess to succeed")
	test_utils.AssertTrue(t, len(path) > 0, "Expected auditor to have access to the org log sym key")
	mstub.MockTransactionEnd("t1")

	// remove permission
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = RemoveAuditorPermissionOfGroup(stub, orgAdmin, auditor.ID, org.ID)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expected RemoveAuditorPermissionOfGroup to succeed")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	path, err = key_mgmt_i.SlowVerifyAccess(stub, auditor.GetPubPrivKeyId(), org.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected VerifyAccess to succeed")
	test_utils.AssertTrue(t, len(path) == 0, "Expected auditor to have no access to the org log sym key")
	mstub.MockTransactionEnd("t1")
}

// test permission transiability from parent to subgroup but not vis versa
func TestGiveAuditorPermissionOfGroupById_TransitivePermission(t *testing.T) {
	logger.Info("TestGiveAditorPermissionOfGroupById_InvalidRole function called")
	mstub := setup(t)

	// create auditor, and 4 orgs
	auditor := test_utils.CreateTestUser("auditor")
	auditor.Role = global.ROLE_AUDIT
	RegisterUserForTest(t, mstub, auditor, auditor, false)

	orgAdmin := test_utils.CreateTestUser("orgAdmin")
	orgAdmin.Role = global.ROLE_SYSTEM_ADMIN
	RegisterUserForTest(t, mstub, orgAdmin, orgAdmin, false)

	org := test_utils.CreateTestGroup("org")
	RegisterUserForTest(t, mstub, orgAdmin, org, true)

	subgroup1 := test_utils.CreateTestGroup("subgroup1")
	subgroup2 := test_utils.CreateTestGroup("subgroup2")
	subgroup3 := test_utils.CreateTestGroup("subgroup3")

	RegisterSubgroupForTest(t, mstub, orgAdmin, subgroup1, org.ID)
	RegisterSubgroupForTest(t, mstub, orgAdmin, subgroup2, subgroup1.ID)
	RegisterSubgroupForTest(t, mstub, orgAdmin, subgroup3, subgroup2.ID)

	// give audit permission to second org from top
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	err := GiveAuditorPermissionOfGroupById(stub, orgAdmin, auditor.ID, subgroup1.ID)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expected GiveAuditorPermissionOfGroupById to succeed")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	path, err := key_mgmt_i.SlowVerifyAccess(stub, auditor.GetPubPrivKeyId(), org.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected VerifyAccess to succeed")
	test_utils.AssertTrue(t, len(path) == 0, "Expected auditor to have no access to the org log sym key")
	path, err = key_mgmt_i.SlowVerifyAccess(stub, auditor.GetPubPrivKeyId(), subgroup1.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected VerifyAccess to succeed")
	test_utils.AssertTrue(t, len(path) > 0, "Expected auditor to have access to the subgroup1 log sym key")
	path, err = key_mgmt_i.SlowVerifyAccess(stub, auditor.GetPubPrivKeyId(), subgroup2.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected VerifyAccess to succeed")
	test_utils.AssertTrue(t, len(path) > 0, "Expected auditor to have access to the subgroup2 log sym key")
	path, err = key_mgmt_i.SlowVerifyAccess(stub, auditor.GetPubPrivKeyId(), subgroup3.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected VerifyAccess to succeed")
	test_utils.AssertTrue(t, len(path) > 0, "Expected auditor to have access to the subgroup3 log sym key")
	mstub.MockTransactionEnd("t1")

	// remove permission
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = RemoveAuditorPermissionOfGroup(stub, orgAdmin, auditor.ID, subgroup1.ID)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expected RemoveAuditorPermissionOfGroup to succeed")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	path, err = key_mgmt_i.SlowVerifyAccess(stub, auditor.GetPubPrivKeyId(), org.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected VerifyAccess to succeed")
	test_utils.AssertTrue(t, len(path) == 0, "Expected auditor to have no access to the org log sym key")
	path, err = key_mgmt_i.SlowVerifyAccess(stub, auditor.GetPubPrivKeyId(), subgroup1.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected VerifyAccess to succeed")
	test_utils.AssertTrue(t, len(path) == 0, "Expected auditor to have no access to the subgroup1 log sym key")
	path, err = key_mgmt_i.SlowVerifyAccess(stub, auditor.GetPubPrivKeyId(), subgroup2.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected VerifyAccess to succeed")
	test_utils.AssertTrue(t, len(path) == 0, "Expected auditor to have no access to the subgroup2 log sym key")
	path, err = key_mgmt_i.SlowVerifyAccess(stub, auditor.GetPubPrivKeyId(), subgroup3.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected VerifyAccess to succeed")
	test_utils.AssertTrue(t, len(path) == 0, "Expected auditor to have no access to the subgroup3 log sym key")
	mstub.MockTransactionEnd("t1")
}

// test non auditor
func TestGiveAuditorPermissionOfGroupById_InvalidRole(t *testing.T) {
	logger.Info("TestGiveAditorPermissionOfGroupById_InvalidRole function called")
	mstub := setup(t)

	// create orgs, user, and system admin
	orgAdmin := test_utils.CreateTestUser("orgAdmin")
	orgAdmin.Role = global.ROLE_SYSTEM_ADMIN
	RegisterUserForTest(t, mstub, orgAdmin, orgAdmin, false)
	org := test_utils.CreateTestGroup("org")
	RegisterUserForTest(t, mstub, orgAdmin, org, true)
	org2 := test_utils.CreateTestGroup("org2")
	RegisterUserForTest(t, mstub, orgAdmin, org2, true)
	user := test_utils.CreateTestUser("user")
	RegisterUserForTest(t, mstub, user, user, false)
	system := test_utils.CreateTestUser("system")
	RegisterUserForTest(t, mstub, system, system, false)

	// try and fail to give audit permission to user, org, and system
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	err := GiveAuditorPermissionOfGroupById(stub, orgAdmin, user.ID, org.ID)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err != nil, "Expected GiveAuditorPermissionOfGroupById to fail")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = GiveAuditorPermissionOfGroupById(stub, orgAdmin, org2.ID, org.ID)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err != nil, "Expected GiveAuditorPermissionOfGroupById to fail")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = GiveAuditorPermissionOfGroupById(stub, orgAdmin, system.ID, org.ID)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err != nil, "Expected GiveAuditorPermissionOfGroupById to fail")
}

func TestGetUserIter(t *testing.T) {
	// create a MockStub
	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	// Create caller
	caller := test_utils.CreateTestUser("caller")
	// Create users
	user1 := test_utils.CreateTestUser("user1")
	user2 := test_utils.CreateTestUser("user2")
	user3 := test_utils.CreateTestUser("user3")
	user4 := test_utils.CreateTestUser("user4")
	user5 := test_utils.CreateTestUser("user5")
	registerUserInternal(stub, user1, user1, false)
	registerUserInternal(stub, user2, user2, false)
	registerUserInternal(stub, user3, user3, false)
	registerUserInternal(stub, user4, user4, false)
	registerUserInternal(stub, user5, user5, false)
	// Create system admin
	systemAdmin := test_utils.CreateTestUser("systemAdmin")
	systemAdmin.Role = global.ROLE_SYSTEM_ADMIN
	registerUserInternal(stub, systemAdmin, systemAdmin, false)
	// Create default org admin
	org := test_utils.CreateTestGroup("org")
	registerUserInternal(stub, systemAdmin, org, false)
	mstub.MockTransactionEnd("t1")

	// query for role type of users only, should not include org admin or system admin
	mstub.MockTransactionStart("t2")
	stub = cached_stub.NewCachedStub(mstub)
	userIter, err := GetUserIter(stub, caller,
		[]string{"false", global.ROLE_USER},
		[]string{"false", global.ROLE_USER},
		false,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		20,
		nil)
	test_utils.AssertTrue(t, err == nil, "Expected GetUserIter to succeed")
	assertUserIterListsEqual(t, []data_model.User{user1, user2, user3, user4, user5}, userIter)
	mstub.MockTransactionEnd("t2")
}

// Asserts that two lists of users are equal
func assertUserIterListsEqual(t *testing.T, expectedAssetList []data_model.User, actualListIter asset_manager.AssetIteratorInterface) {
	idx := 0
	defer actualListIter.Close()
	for actualListIter.HasNext() {
		if len(expectedAssetList) < idx+1 {
			// expectedAssetList is too short
			debug.PrintStack()
			t.Fatalf("Expected user list was shorter than actual user list. Index: %v", idx)
		}
		_, err := actualListIter.Next()
		if err != nil {
			debug.PrintStack()
			t.Fatalf("Error getting actualListIter.Next(): %v", err)
		}
		idx++
	}
	if idx != len(expectedAssetList) {
		// expectedAssetList is too long
		debug.PrintStack()
		t.Fatalf("Expected user list was longer than actual user list. Expected %v, got %v", len(expectedAssetList), idx)
	}
}

func TestRemoveSubgroupFromGroup(t *testing.T) {
	logger.Info("TestRemoveSubgroupFromGroup function called")
	mstub := setup(t)

	// register parent group
	group := test_utils.CreateTestGroup("group")
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	err := RegisterOrgWithParams(stub, group, group, false)
	mstub.MockTransactionEnd("t1")
	test_utils.AssertTrue(t, err == nil, "Expected RegisterOrgWithParams to succeed")

	// register user1, add as admin of group
	user1 := test_utils.CreateTestUser("user1")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = RegisterUserWithParams(stub, user1, user1, false)
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUserWithParams to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, group, user1.ID, group.ID, true)
	test_utils.AssertTrue(t, err == nil, "Expected PutUserInGroup to succeed")
	mstub.MockTransactionEnd("t1")

	// register subgroup
	subgroup := test_utils.CreateTestGroup("subgroup")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = RegisterSubgroupWithParams(stub, group, subgroup, group.ID)
	test_utils.AssertTrue(t, err == nil, "Expected RegisterSubgroupWithParams to succeed")
	mstub.MockTransactionEnd("t1")

	// confirm that subgroup exists
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	subgroupIDs, err := SlowGetSubgroups(stub, group.ID)
	test_utils.AssertTrue(t, err == nil, "Expected SlowGetSubgroups to succeed")
	test_utils.AssertTrue(t, len(subgroupIDs) == 1, "SlowGetSubgroups should have returned one ID")
	test_utils.AssertTrue(t, subgroupIDs[0] == subgroup.ID, "SlowGetSubgroups should have returned subgroup.ID")
	subgroupData, err := GetUserData(stub, group, subgroup.ID, true, true)
	test_utils.AssertTrue(t, err == nil, "Expected GetUserData to succeed")
	test_utils.AssertTrue(t, subgroupData.ID == subgroup.ID, "GetUserData should have returned the subgroup")
	mstub.MockTransactionEnd("t1")

	// confirm that user1 (admin of group) is admin of subgroup
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, user1.ID, subgroup.ID)
	test_utils.AssertTrue(t, err == nil, "Expected IsUserAdminOfGroup to succeed")
	test_utils.AssertTrue(t, isAdmin, "Expected isAdmin to be true")
	mstub.MockTransactionEnd("t1")

	// register auditor
	auditor := test_utils.CreateTestUser("auditor")
	auditor.Role = global.ROLE_AUDIT
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = RegisterUserWithParams(stub, auditor, auditor, false)
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUserWithParams to succeed")
	mstub.MockTransactionEnd("t1")

	// give auditor audit permission for group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = GiveAuditorPermissionOfGroupById(stub, group, auditor.ID, group.ID)
	test_utils.AssertTrue(t, err == nil, "Expected GiveAuditorPermissionOfGroupById to succeed")
	mstub.MockTransactionEnd("t1")

	// confirm that auditor has audit permission for group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	path, err := key_mgmt_i.SlowVerifyAccess(stub, auditor.GetPubPrivKeyId(), group.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected VerifyAccess to succeed")
	test_utils.AssertTrue(t, len(path) > 0, "Expected auditor to have access to the group log sym key")
	mstub.MockTransactionEnd("t1")

	// confirm that auditor has audit permission for subgroup
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	path, err = key_mgmt_i.SlowVerifyAccess(stub, auditor.GetPubPrivKeyId(), subgroup.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected VerifyAccess to succeed")
	test_utils.AssertTrue(t, len(path) > 0, "Expected auditor to have access to the subgroup log sym key")
	mstub.MockTransactionEnd("t1")

	// register user2, add as member of subgroup
	user2 := test_utils.CreateTestUser("user2")
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = RegisterUserWithParams(stub, user2, user2, false)
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUserWithParams to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = PutUserInGroup(stub, subgroup, user2.ID, subgroup.ID, false)
	test_utils.AssertTrue(t, err == nil, "Expected PutUserInGroup to succeed")
	mstub.MockTransactionEnd("t1")

	// confirm that user2 is member of subgroup
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isMember, err := user_mgmt_c.IsUserInGroup(stub, user2.ID, subgroup.ID)
	test_utils.AssertTrue(t, err == nil, "Expected IsUserInGroup to succeed")
	test_utils.AssertTrue(t, isMember, "Expected user2 to be member of subgroup")
	mstub.MockTransactionEnd("t1")

	// confirm that user2 is member of group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isMember, err = user_mgmt_c.IsUserInGroup(stub, user2.ID, group.ID)
	test_utils.AssertTrue(t, err == nil, "Expected IsUserInGroup to succeed")
	test_utils.AssertTrue(t, isMember, "Expected user2 to be member of group")
	mstub.MockTransactionEnd("t1")

	// remove subgroup from group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = RemoveSubgroupFromGroup(stub, group, subgroup.ID, group.ID)
	test_utils.AssertTrue(t, err == nil, "Expected RemoveSubgroupFromGroup to succeed")
	mstub.MockTransactionEnd("t1")

	// confirm that subgroup has been removed
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	subgroupIDs, err = SlowGetSubgroups(stub, group.ID)
	test_utils.AssertTrue(t, err == nil, "Expected SlowGetSubgroups to succeed")
	test_utils.AssertTrue(t, len(subgroupIDs) == 0, "SlowGetSubgroups should have returned 0 IDs")
	mstub.MockTransactionEnd("t1")

	// confirm that user1 is no longer admin of subgroup
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isAdmin, _, err = user_mgmt_c.IsUserAdminOfGroup(stub, user1.ID, subgroup.ID)
	test_utils.AssertTrue(t, err == nil, "Expected IsUserAdminOfGroup to succeed")
	test_utils.AssertFalse(t, isAdmin, "Expected isAdmin to be false")
	mstub.MockTransactionEnd("t1")

	// confirm that auditor no longer has audit permission for subgroup
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	path, err = key_mgmt_i.SlowVerifyAccess(stub, auditor.GetPubPrivKeyId(), subgroup.GetLogSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected VerifyAccess to succeed")
	test_utils.AssertTrue(t, len(path) == 0, "Expected auditor to no longer have access to the subgroup log sym key")
	mstub.MockTransactionEnd("t1")

	// confirm that user2 is no longer member of group
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	isMember, err = user_mgmt_c.IsUserInGroup(stub, user2.ID, group.ID)
	test_utils.AssertTrue(t, err == nil, "Expected IsUserInGroup to succeed")
	test_utils.AssertFalse(t, isMember, "Expected user2 to no longer be member of group")
	mstub.MockTransactionEnd("t1")
}
