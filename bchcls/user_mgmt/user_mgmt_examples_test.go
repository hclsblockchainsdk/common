/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package user_mgmt

import (
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/internal/common/global"
	"common/bchcls/simple_rule"
	"common/bchcls/test_utils"
	"common/bchcls/user_mgmt/user_groups"
	"common/bchcls/user_mgmt/user_keys"

	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
)

func ExampleGetPublicKey() {
	privateKey := test_utils.GeneratePrivateKey()
	publicKey := privateKey.Public().(*rsa.PublicKey)

	user := data_model.User{
		ID:         "user1",
		Role:       "system",
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		// other data_model.User fields
	}

	user.GetPublicKey()
}

func ExampleGetPrivateKey() {
	user := data_model.User{
		ID:         "user1",
		Role:       "system",
		PrivateKey: test_utils.GeneratePrivateKey(),
		// other data_model.User fields
	}

	user.GetPrivateKey()
}

func ExampleGetSymKey() {
	user := data_model.User{
		ID:     "user1",
		Role:   "system",
		SymKey: test_utils.GenerateSymKey(),
		// other data_model.User fields
	}

	user.GetSymKey()
}

func ExampleGetLogSymKeyFromUser() {
	user := data_model.User{
		ID:     "user1",
		Role:   "system",
		SymKey: test_utils.GenerateSymKey(),
		// other data_model.User fields
	}

	user.GetLogSymKey()
}

func ExampleGetPrivateKeyHash() {
	user := data_model.User{
		ID:         "user1",
		Role:       "system",
		PrivateKey: test_utils.GeneratePrivateKey(),
		// other data_model.User fields
	}

	user.GetPrivateKeyHashSymKey()
}

func ExampleGetUserData() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	GetUserData(stub, caller, "user1", true, true)
}

func ExamplePutUserInGroup() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	user_groups.PutUserInGroup(stub, caller, "user1", "group1", true)
}

func ExampleRegisterSubgroup() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	subGroup := test_utils.CreateTestGroup("subGroup1")
	subGroupBytes, _ := json.Marshal(&subGroup)

	user_groups.RegisterSubgroup(stub, caller, []string{string(subGroupBytes), "parentGroup1"})
}

func ExampleRemoveSubgroupFromGroup() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	user_groups.RemoveSubgroupFromGroup(stub, caller, "subGroup1", "parentGroup1")
}

func ExampleRemoveUserFromGroup() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	user_groups.RemoveUserFromGroup(stub, caller, []string{"user1", "group1"})
}

func ExampleIsUserInGroup() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	user_groups.IsUserInGroup(stub, "user1", "group1")
}

func ExampleIsUserAdminOfGroup() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	user_groups.IsUserAdminOfGroup(stub, "user1", "group1")
}

func ExampleSlowGetGroupMemberIDs() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	user_groups.SlowGetGroupMemberIDs(stub, "group1")
}

func ExampleGetGroupAdminIDs() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	user_groups.SlowGetGroupAdminIDs(stub, "group1")
}

func ExampleGetMyDirectGroupIDs() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	user_groups.GetMyDirectGroupIDs(stub, "user1")
}

func ExampleGetMyGroupIDs() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	user_groups.SlowGetMyGroupIDs(stub, caller, "user1", false)
}

func ExampleSlowGetSubgroups() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	user_groups.SlowGetSubgroups(stub, "group1")
}

func ExampleGetMyDirectAdminGroupIDs() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	user_groups.GetMyDirectAdminGroupIDs(stub, "user1")
}

func ExampleIsParentGroup() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	user_groups.IsParentGroup(stub, caller, "group1", "group2")
}

func ExampleGiveAdminPermissionOfGroup() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	user_groups.GiveAdminPermissionOfGroup(stub, caller, "user1", "group1")
}

func ExampleRemoveAdminPermissionOfGroup() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	user_groups.RemoveAdminPermissionOfGroup(stub, caller, []string{"user1", "group1"})
}

func ExampleGiveAuditorPermissionOfGroupById() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	user_groups.GiveAuditorPermissionOfGroupById(stub, caller, "auditor1", "group1")
}

func ExampleRemoveAuditorPermissionOfGroup() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	user_groups.RemoveAuditorPermissionOfGroup(stub, caller, "auditor1", "group1")
}

func ExampleGetUserPublicKey() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	user_keys.GetUserPublicKey(stub, caller, "user1")
}

func ExampleGetCallerData() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	GetCallerData(stub)
}

func ExampleRegisterUser() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	privateKey := test_utils.GeneratePrivateKey()
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	publicKey := privateKey.Public().(*rsa.PublicKey)
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(publicKey)
	symKey := test_utils.GenerateSymKey()

	user := data_model.User{
		ID:                  "user1",
		Name:                "Jo Smith",
		Role:                global.ROLE_USER,
		PublicKey:           publicKey,
		PublicKeyB64:        base64.StdEncoding.EncodeToString(publicKeyBytes),
		PrivateKey:          privateKey,
		PrivateKeyB64:       base64.StdEncoding.EncodeToString(privateKeyBytes),
		SymKey:              symKey,
		SymKeyB64:           base64.StdEncoding.EncodeToString(symKey),
		IsGroup:             false,
		Status:              "active",
		Email:               "email@mail.com",
		SolutionPublicData:  make(map[string]interface{}),
		SolutionPrivateData: make(map[string]interface{}),
		KmsPublicKeyId:      "kmspublickeyid",
		KmsPrivateKeyId:     "kmsprivatekeyid",
		KmsSymKeyId:         "kmssymkeyid",
		Secret:              "secret",
	}
	userBytes, _ := json.Marshal(&user)

	RegisterUser(stub, caller, []string{string(userBytes), "true"})
}

func ExampleGetUser() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	GetUser(stub, caller, []string{"user1"})
}

func ExampleGetUserIter() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	// filter rule to exclude user with ID of tom123
	rule := simple_rule.NewRule(simple_rule.R("!=",
		simple_rule.R("var", "asset_id"),
		"tom123"),
	)

	// only get users whose is_group is set to false and role set to user
	GetUserIter(
		stub,
		caller,
		[]string{"false", "user"},
		[]string{"false", "user"},
		false,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"",
		10,
		&rule)
}

func ExampleGetUserKeys() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	user_keys.GetUserKeys(stub, caller, "user1")
}

func ExampleRegisterSystemAdmin() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	user := data_model.User{
		ID:   "user1",
		Role: global.ROLE_SYSTEM_ADMIN,
		// other data_model.User fields
	}
	userBytes, _ := json.Marshal(&user)

	RegisterSystemAdmin(stub, caller, []string{string(userBytes), "true"})
}

func ExampleRegisterAuditor() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	user := data_model.User{
		ID:   "user1",
		Role: global.ROLE_AUDIT,
		// other data_model.User fields
	}
	userBytes, _ := json.Marshal(&user)

	RegisterAuditor(stub, caller, []string{string(userBytes), "true"})
}

func ExampleRegisterOrg() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	privateKey := test_utils.GeneratePrivateKey()
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	publicKey := privateKey.Public().(*rsa.PublicKey)
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(publicKey)
	symKey := test_utils.GenerateSymKey()
	solutionPrivateData := make(map[string]interface{})

	org := data_model.User{
		ID:                  "org1",
		Name:                "Org 1",
		Role:                global.ROLE_ORG,
		PublicKey:           publicKey,
		PublicKeyB64:        base64.StdEncoding.EncodeToString(publicKeyBytes),
		PrivateKey:          privateKey,
		PrivateKeyB64:       base64.StdEncoding.EncodeToString(privateKeyBytes),
		SymKey:              symKey,
		SymKeyB64:           base64.StdEncoding.EncodeToString(symKey),
		IsGroup:             true,
		Status:              "active",
		Email:               "email@mail.com",
		SolutionPublicData:  make(map[string]interface{}),
		SolutionPrivateData: solutionPrivateData,
		KmsPublicKeyId:      "kmspublickeyid",
		KmsPrivateKeyId:     "kmsprivatekeyid",
		KmsSymKeyId:         "kmssymkeyid",
		Secret:              "secret",
	}
	orgBytes, _ := json.Marshal(&org)

	RegisterOrg(stub, caller, []string{string(orgBytes), "true"})
}

func ExampleUpdateOrg() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	org1Bytes, _ := GetOrg(stub, caller, []string{"org1"})
	org1 := data_model.User{}
	json.Unmarshal(org1Bytes, &org1)

	// modify solution public data
	solutionPublicData := org1.SolutionPublicData.(map[string]interface{})
	solutionPublicData["age"] = 30
	org1.SolutionPublicData = solutionPublicData
	org1Bytes, _ = json.Marshal(org1)

	UpdateOrg(stub, caller, []string{string(org1Bytes)})
}

func ExampleGetOrg() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	GetOrg(stub, caller, []string{"org1"})
}

func ExampleGetOrgs() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	GetOrgs(stub, caller, []string{})
}

func ExampleGetUsers() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	// returns org1 member users
	GetUsers(stub, caller, []string{"org1"})

	// returns org1 member users with "user" role
	GetUsers(stub, caller, []string{"org1", global.ROLE_USER})
}

func ExamplePutUserInOrg() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")

	PutUserInOrg(stub, caller, []string{"user1", "org1", "false"})
}
