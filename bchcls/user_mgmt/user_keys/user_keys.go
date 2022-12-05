/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package user_keys handles user management functions related to user keys.
package user_keys

import (
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/internal/metering_i"
	"common/bchcls/internal/user_mgmt_i"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("user_keys")

// GetUserKeys returns a user's private, public, and sym keys.
// Caller must have access to the user's private key.
// keyPaths are optional parameters. If passed in, the first keyPath is used
// for getting the private key, and the second keyPath is for getting the symkey.
// If only one keyPath is passed in, it is for the private key, and the sym key is
// obtained from the private key.
func GetUserKeys(stub cached_stub.CachedStubInterface, caller data_model.User, userId string, keyPaths ...[]string) (*data_model.Keys, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GetUserKeys(stub, caller, userId, keyPaths...)
}

// GetUserPrivateKey returns a user's private key.
// Caller must have access to the user's private key.
// keyPath is an optional parameter; if passed in, this keyPath is used to get
// the privateKey. If not, a default key path will be used.
// Default keyPath = [caller privkey, user privhashkey, user privkey]
func GetUserPrivateKey(stub cached_stub.CachedStubInterface, caller data_model.User, userId string, keyPath ...[]string) (data_model.Key, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("userId: %v %v", userId, keyPath)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GetUserPrivateKey(stub, caller, userId, keyPath...)
}

// GetUserSymKey returns a user's sym key.
// Caller must have access to the user's private key.
// keyPath is an optional parameter; if passed in, this keyPath is used to get
// the symKey. If not, a default key path will be used.
// Default keyPath = [caller privkey, user symKey]
func GetUserSymKey(stub cached_stub.CachedStubInterface, caller data_model.User, userId string, keyPath ...[]string) (data_model.Key, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("callerId: %v, userId: %v", caller.ID, userId)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GetUserSymKey(stub, caller, userId, keyPath...)
}

// GetUserPublicKey returns the public key for a given userId.
// If a caller already has the user object, call the GetPublicKey(user) function instead.
func GetUserPublicKey(stub cached_stub.CachedStubInterface, caller data_model.User, userId string) (data_model.Key, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GetUserPublicKey(stub, caller, userId)
}

// ConvertAdminPathToPrivateKeyPath is a convenience function that returns a keyPath to a user's private key
// given an admin path.
// KeyPath can be passed to AssetManger's GetAssetKey function.
func ConvertAdminPathToPrivateKeyPath(adminPath []string) (keyPath []string, err error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return user_mgmt_i.ConvertAdminPathToPrivateKeyPath(adminPath)
}

// ConvertAdminPathToSymKeyPath is a convenience function that returns a keyPath to a user's sym key
// given an admin path.
// KeyPath can be passed to AssetManger's GetAssetKey function.
func ConvertAdminPathToSymKeyPath(adminPath []string) (keyPath []string, err error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return user_mgmt_i.ConvertAdminPathToSymKeyPath(adminPath)
}
