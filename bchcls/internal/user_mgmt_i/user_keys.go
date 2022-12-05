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
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/internal/user_mgmt_i/user_mgmt_c"
	"common/bchcls/utils"

	"github.com/pkg/errors"
)

// GetUserKeys returns a user's private, public, and sym keys.
// Caller must have access to the user's private key.
// keyPaths is optional.
// First keyPath is for private key, second keyPath is for symkey.
// If only one keyPath is passed in, it's for the private key. The sym key is obtained from the private key.
func GetUserKeys(stub cached_stub.CachedStubInterface, caller data_model.User, userId string, keyPaths ...[]string) (*data_model.Keys, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("userId: %v", userId)
	// parse options
	var keyPath1 []string = nil
	var keyPath2 []string = nil
	if len(keyPaths) > 0 {
		keyPath1 = keyPaths[0]
	}
	if len(keyPaths) > 1 {
		keyPath2 = keyPaths[1]
	}

	var keys = data_model.Keys{}

	if userId == caller.ID {
		keys.PrivateKey = caller.PrivateKeyB64
		keys.PublicKey = caller.PublicKeyB64
		keys.SymKey = caller.SymKeyB64
		logger.Debugf("returning userkeys from caller for %v", userId)
		return &keys, nil

	} else {
		var returnErr error = nil
		// get user private key
		privateKey, err := GetUserPrivateKey(stub, caller, userId, keyPath1)
		if err == nil && len(privateKey.KeyBytes) > 0 {
			keys.PrivateKey = crypto.EncodeToB64String(privateKey.KeyBytes)
		} else {
			logger.Errorf("Unable to get user's private key: %v", err)
			returnErr = errors.New("unable to get user's private key")
		}

		// sym key
		if len(privateKey.KeyBytes) > 0 {
			user := data_model.User{ID: userId}
			// use private key to get sym key
			keyPath2 := []string{user.GetPubPrivKeyId(), user.GetSymKeyId()}
			symKeyBytes, err := key_mgmt_i.GetKey(stub, keyPath2, privateKey.KeyBytes)
			if err != nil || symKeyBytes == nil {
				logger.Errorf("Unable to get user's sym key: %v", err)
				returnErr = errors.New("unable to get user's sym key")
			} else {
				keys.SymKey = crypto.EncodeToB64String(symKeyBytes)
			}
		} else if len(keyPath2) > 0 {
			// use key path to get sym key
			symKey, err := GetUserSymKey(stub, caller, userId, keyPath2)
			if err == nil && len(symKey.KeyBytes) > 0 {
				keys.SymKey = crypto.EncodeToB64String(symKey.KeyBytes)
			} else {
				logger.Errorf("Unable to get user's sym key: %v", err)
				returnErr = errors.New("unable to get user's sym key")
			}
		}

		// public key
		publicKey, err := GetUserPublicKey(stub, caller, userId)
		if err != nil || publicKey.KeyBytes == nil {
			logger.Errorf("unable to get user's public keys: %v", err)
			returnErr = errors.New("unable to get user's public keys")
		} else {
			keys.PublicKey = crypto.EncodeToB64String(publicKey.KeyBytes)
		}
		return &keys, returnErr
	}
}

// GetUserPrivateKey returns a user's private key.
// Caller must have access to the user's private key.
// keyPath is optional.
// Default keyPath = [caller privkey, user privhashkey, user privkey]
// If keyPath is passed in, use this keyPath to get the private key.
func GetUserPrivateKey(stub cached_stub.CachedStubInterface, caller data_model.User, userId string, keyPath ...[]string) (data_model.Key, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return user_mgmt_c.GetUserPrivateKey(stub, caller, userId, keyPath...)
}

// GetUserSymKey returns a user's sym key.
// keyPath is an optional parameter; if passed in, this keyPath is used to get the symKey.
// Caller must have access to the user's private key.
func GetUserSymKey(stub cached_stub.CachedStubInterface, caller data_model.User, userId string, keyPath ...[]string) (data_model.Key, error) {
	return user_mgmt_c.GetUserSymKey(stub, caller, userId, keyPath...)
}

// GetUserPublicKey returns the user's public key.
// If you already have the user, call GetPublicKey(user) instead.
func GetUserPublicKey(stub cached_stub.CachedStubInterface, caller data_model.User, userId string) (data_model.Key, error) {
	user, err := GetUserData(stub, caller, userId, false, false)
	if err != nil {
		logger.Errorf("Failed to get user \"%v\"", userId)
		return data_model.Key{}, errors.Wrapf(err, "Failed to get user \"%v\"", userId)
	} else if len(user.ID) == 0 {
		logger.Errorf("Failed to get user \"%v\"", userId)
		return data_model.Key{}, errors.Errorf("Failed to get user \"%v\"", userId)
	}
	return user.GetPublicKey(), nil
}

// ConvertAdminPathToPrivateKeyPath returns a keyPath to get a user's private key.
// KeyPath can be passed to AssetManger's GetAssetKey function.
func ConvertAdminPathToPrivateKeyPath(adminPath []string) (keyPath []string, err error) {
	keyPath = []string{}
	if len(adminPath) == 0 {
		logger.Error("Empty adminPath")
		return nil, errors.New("Empty adminPath")
	}

	var prev = data_model.User{}
	var curr = data_model.User{}
	prevId := ""
	for _, currId := range adminPath {
		curr.ID = currId

		if currId == prevId || len(currId) == 0 {
			continue
		}
		if len(prevId) > 0 {
			keyPath = append(keyPath, curr.GetPrivateKeyHashSymKeyId())
		}

		keyPath = append(keyPath, curr.GetPubPrivKeyId())
		prevId = currId
		prev.ID = currId
	}
	return keyPath, nil
}

// ConvertAdminPathToSymKeyPath returns a keyPath to get a user's sym key.
// KeyPath can be passed to AssetManger's GetAssetKey function.
func ConvertAdminPathToSymKeyPath(adminPath []string) (keyPath []string, err error) {
	keyPath = []string{}
	if len(adminPath) == 0 {
		logger.Error("Empty adminPath")
		return nil, errors.New("Empty adminPath")
	}

	var prev = data_model.User{}
	var curr = data_model.User{}
	if len(adminPath) == 1 {
		curr.ID = adminPath[0]
		return []string{curr.GetSymKeyId()}, nil
	}

	prev.ID = adminPath[0]
	curr.ID = adminPath[1]
	keyPath = append(keyPath, prev.GetPubPrivKeyId())
	keyPath = append(keyPath, curr.GetPrivateKeyHashSymKeyId())
	keyPath = append(keyPath, curr.GetPubPrivKeyId())
	for _, currId := range adminPath[1:] {
		curr.ID = currId
		if currId == prev.ID || len(currId) == 0 {
			continue
		}
		keyPath = append(keyPath, curr.GetSymKeyId())
		prev.ID = currId
	}
	return keyPath, nil
}
