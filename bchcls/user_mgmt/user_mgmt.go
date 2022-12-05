/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package user_mgmt manages users and groups.
// It stores users and organizations as assets and maintains a graph of user
// and organization relationships.
// An organization can be a group or subgroup.
package user_mgmt

import (
	"common/bchcls/asset_mgmt/asset_manager"
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/metering_i"
	"common/bchcls/internal/user_mgmt_i"
	"common/bchcls/simple_rule"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("user_mgmt")

///////////////////////////////////////////////////////
// User management constants

// ROLE_SYSTEM_ADMIN is a User.Role option that specifies a system admin.
const ROLE_SYSTEM_ADMIN = global.ROLE_SYSTEM_ADMIN

// ROLE_USER is a User.Role option that specifies a user.
const ROLE_USER = global.ROLE_USER

// ROLE_ORG is a User.Role option that specifies an org.
const ROLE_ORG = global.ROLE_ORG

// ROLE_AUDIT is a User.Role option that specifies an auditor.
const ROLE_AUDIT = global.ROLE_AUDIT

// ------------------------------------------------------
// ---------------------- INIT FUNCTIONS ----------------
// ------------------------------------------------------

// Init sets up the user_mgmt package by building an index table for users.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return user_mgmt_i.Init(stub, logLevel...)
}

// ----------------- CONVERSION BETWEEN USER AND ASSET -----------------

// GetUserAssetID returns the asset ID for the stored user object identified by the given userID.
func GetUserAssetID(userID string) string {
	return user_mgmt_i.GetUserAssetID(userID)
}

// ConvertToAsset converts a user object to an asset object.
func ConvertToAsset(user data_model.User) data_model.Asset {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return user_mgmt_i.ConvertToAsset(user)
}

// ConvertFromAsset converts an asset object to a user object.
func ConvertFromAsset(asset *data_model.Asset) data_model.User {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return user_mgmt_i.ConvertFromAsset(asset)
}

// ----------------- GETUSER AND USER.PUTSTATE FUNCTIONS -----------------

// GetUserData finds, decrypts, and returns a User for the given userID.
// The user's public key will always be included.
// If the private and/or sym keys cannot be retrieved, they will be left blank, and no error will be returned.
// If userID is same as callerId, User object is copied from caller object.
//
// options can be passed in any of the following orders:
//
// keyPath []string
// keyPath []string, keyPath2 []string
// includePrivateAndSymKeys bool
// includePrivateAndSymKeys bool, keyPath []string
// includePrivateAndSymKeys bool, keyPath []string, keyPath2 []string
// includePrivateAndSymKeys bool, includePrivateData bool
// includePrivateAndSymKeys bool, includePrivateData bool, keyPath []string
// includePrivateAndSymKeys bool, includePrivateData bool, keyPath []string, keyPath2 []string
//
// If includePrivateAndSymKeys (default false) is true, this function will include the user's
// private and sym keys as well.
// If includePrivateData (default false) is false, the user's private data will not be decrypted.
// if keyPath (default nil) is passed in, user's symKey will be retrieved using this keyPath.
// The first element of keyPath must be a caller's key, and the last element must be the user's sym key.
// keyPaths is always the last option if it's specified.
// KeyPath2 is for the user's private key.
func GetUserData(stub cached_stub.CachedStubInterface, caller data_model.User, userID string, options ...interface{}) (data_model.User, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("GetUserData for userID: %v options: %v", userID, options)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GetUserData(stub, caller, userID, options...)
}

// ==================================================================================
// HIGHLEVEL API FUNCTIONS
// ==================================================================================

// GetCallerData gets keys from TMAP and returns the caller's data from the ledger.
func GetCallerData(stub cached_stub.CachedStubInterface) (data_model.User, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GetCallerData(stub)
}

// RegisterUser registers or updates a user.
//
// args = [ user, allowAccess ]
//
// user is the data_model.User to add or update.
// If allowAccess is true and a new user is being registered, gives the caller access to the user's private key.
// If allowAccess is true and a new group is being registered, makes the caller an admin of the group.
func RegisterUser(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("RegisterUser args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RegisterUser(stub, caller, args)
}

// RegisterUserWithParams registers or updates a user
// "WithParams" functions should only be called from within the chaincode.
//
// user        - the user object to add/update
// allowAccess - [users] if true, gives the caller access to the user's private key (only applies for a new user, not an update of an existing user)
// allowAccess - [groups] if true, makes the caller an admin of the group (only applies for a new group, not an update of an existing group)
func RegisterUserWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, user data_model.User, allowAccess bool) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v user: %v allowAccess: %v", caller.ID, user.ID, allowAccess)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RegisterUserWithParams(stub, caller, user, allowAccess)
}

// GetUser returns a user object.
// args = [userID]
func GetUser(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GetUser(stub, caller, args)
}

// RegisterSystemAdmin registers a system admin user.
// Callers role must be "system".
// System admin's role must be "system".
//
// args = [userBytes, allowAccess]
//
// When registering a new user, if allowAccess is true, the caller will be given access to the user's private key.
func RegisterSystemAdmin(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RegisterSystemAdmin(stub, caller, args)
}

// RegisterAuditor registers an auditor user.
// Caller's role must be "system".
// Auditor's role must be "audit".
//
// args = [userBytes, allowAccess]
//
// When registering a new user, if allowAccess is true, the caller will be given access to the user's private key.
func RegisterAuditor(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RegisterAuditor(stub, caller, args)
}

// RegisterOrg registers or updates an organization or a group.
// Also creates a default org admin user.
// When registering a new org, if the makeCallerAdmin flag is true, the caller will be added
// as an admin of the org.
//
// args = [ orgBytes, makeCallerAdmin ]
func RegisterOrg(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	// Encrypts org keys with org public key.
	// Encrypts org private key with org public key.
	// Saves org data with org sym key.
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RegisterOrg(stub, caller, args)
}

// RegisterOrgWithParams is the internal function for registering or updating an org.
// "WithParams" functions should only be called from within the chaincode.
//
// When registering a new org, if the makeCallerAdmin flag is true, the caller will be added
// as an admin of the org.
func RegisterOrgWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, org data_model.User, makeCallerAdmin bool) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RegisterOrgWithParams(stub, caller, org, makeCallerAdmin)
}

// UpdateOrg updates an organization.
//
// args = [orgBytes]
func UpdateOrg(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.UpdateOrg(stub, caller, args)
}

// GetOrg returns an organization.
//
// args = [orgId]
func GetOrg(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GetOrg(stub, caller, args)
}

// GetOrgs returns a list of all organizations.
//
// args = []
func GetOrgs(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GetOrgs(stub, caller, args)
}

// GetUsers returns a list of all members for a given orgId, optionally filtered by role.
//
// args = [orgId, role]
// role is an optional parameter.
func GetUsers(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GetUsers(stub, caller, args)
}

// PutUserInOrg is a proxy function for PutUserInGroup.
//
// args = [ userID, orgID, isAdmin]
func PutUserInOrg(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.PutUserInOrg(stub, caller, args)
}

// GetUserIter returns an interator of user objects
// This function is not meant to be called from outside of chaincode
func GetUserIter(
	stub cached_stub.CachedStubInterface,
	caller data_model.User,
	startValues []string,
	endValues []string,
	decryptPrivateData bool,
	returnOnlyPrivateAssets bool,
	assetKeyPath interface{},
	previousKey string,
	limit int,
	filterRule *simple_rule.Rule) (asset_manager.AssetIteratorInterface, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GetUserIter(stub, caller, startValues, endValues, decryptPrivateData, returnOnlyPrivateAssets, assetKeyPath, previousKey, limit, filterRule)
}
