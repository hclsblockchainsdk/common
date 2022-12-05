/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// common package contains global_data and functions to be shared across bchcls common packages.
package user_mgmt_c

import (
	"github.com/pkg/errors"

	"common/bchcls/cached_stub"
	"common/bchcls/custom_errors"
	"common/bchcls/data_model"
	"common/bchcls/index"
	"common/bchcls/internal/asset_mgmt_i/asset_mgmt_c"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/common/graph"
	"common/bchcls/internal/key_mgmt_i/key_mgmt_c"
	"common/bchcls/internal/user_mgmt_i/user_mgmt_c/user_mgmt_g"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("user_mgmt_c")

// Init sets up the user_mgmt package by building an index table for users.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	logger.Debug("Init user_mgmt")
	//User Index
	userTable := index.GetTable(stub, global.INDEX_USER)
	userTable.AddIndex([]string{"is_group", "role", "id"}, false)
	err := userTable.SaveToLedger()
	return nil, err
}

// GetUserWithoutPrivateData gets a user object without encrypting private data.
func GetUserDataWithoutPrivateData(stub cached_stub.CachedStubInterface, userID string) (data_model.User, error) {

	userAsset, err := asset_mgmt_c.GetEncryptedAssetData(stub, GetUserAssetID(userID))
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: userID}
		logger.Errorf("%v: %v", custom_err, err)
		return data_model.User{ID: userID}, errors.WithMessage(err, custom_err.Error())
	}
	userAsset.PrivateData = data_model.GetEncryptedDataBytes(userAsset.PrivateData)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: userID}
		logger.Errorf("%v: %v", custom_err, err)
		return data_model.User{ID: userID}, errors.WithMessage(err, custom_err.Error())
	}
	user := ConvertUserFromAsset(&userAsset)
	return user, nil
}

// IsUserInGroup returns true if a user is in a group, directly or indirectly.
func IsUserInGroup(stub cached_stub.CachedStubInterface, userID string, groupID string) (bool, error) {
	// User is Group User
	if userID == groupID {
		return true, nil
	}

	parents, err := graph.GetDirectParents(stub, global.USER_GRAPH, userID)
	if err != nil {
		logger.Errorf("Failed to get direct parents of user: %v", err)
		return false, errors.Wrap(err, "Failed to get direct parents of user")
	}
	for _, parent := range parents {
		//if groupID is admin of parent of user, user is in groupID group
		isAdmin, _, err2 := IsUserAdminOfGroup(stub, groupID, parent)
		if isAdmin {
			return true, nil
		}
		// even if error happend, return this error in the end
		if err2 != nil {
			err = err2
		}
	}

	return false, err
}

// IsUserMemberOfGroup returns true if a user is in a group.
// Thsi function does not check indirect membership.
func IsUserMemberOfGroup(stub cached_stub.CachedStubInterface, userID string, groupID string) (bool, error) {
	isMember, err := graph.HasEdge(stub, global.USER_GRAPH, groupID, userID)
	if err != nil {
		logger.Errorf("Failed to check if %v is member of %v", userID, groupID)
		return false, errors.Wrapf(err, "Failed to check if %v is member of %v", userID, groupID)
	}
	return isMember, nil
}

// IsUserDirectAdminOfGroup checks if userID matches any admin ID of group's direct children, or is direct parent of a subgroup.
func IsUserDirectAdminOfGroup(stub cached_stub.CachedStubInterface, userID string, groupID string) (bool, error) {

	edgeValue, _, err := graph.GetEdge(stub, global.USER_GRAPH, groupID, userID)
	edgeString := string(edgeValue[:])
	if edgeString == global.ADMIN_EDGE {
		return true, nil
	}

	parents, err := graph.GetDirectParents(stub, global.USER_GRAPH, groupID)
	for _, parent := range parents {
		if parent == userID {
			return true, err
		}
	}
	return false, err
}

// IsUserAdminOfGroup returns []string user admin chain if user is an admin (or parent group) of a group.
// If user is not an admin of a group, returns empty list or nil.
func IsUserAdminOfGroup(stub cached_stub.CachedStubInterface, userID string, groupID string) (bool, []string, error) {
	// User is Group User
	if userID == groupID {
		return true, []string{userID}, nil
	}

	// ckeck cache
	valueCache, err := getCacheIsUserAdminOfGroup(stub, userID, groupID)
	if err == nil {
		return len(valueCache) > 0, valueCache, nil
	}

	//return path
	path := []string{groupID}

	isAdmin, err := IsUserDirectAdminOfGroup(stub, userID, groupID)
	if err != nil {
		logger.Errorf("Failed to check if user %v is admin of %v : %v", userID, groupID, err)
		//return false, nil, errors.Wrap(err, "Failed to check isDirectAdmin")
	}
	if isAdmin {
		path = append([]string{userID}, path...)
		putCacheIsUserAdminOfGroup(stub, userID, groupID, path)
		return true, path, nil
	}

	parents, err := graph.GetDirectParents(stub, global.USER_GRAPH, groupID)
	for err == nil && len(parents) > 0 && len(parents[0]) > 0 {
		currID := parents[0]
		path = append([]string{currID}, path...)
		// user is parent group
		if userID == currID {
			putCacheIsUserAdminOfGroup(stub, userID, groupID, path)
			return true, path, nil
		}
		isAdmin, err = IsUserDirectAdminOfGroup(stub, userID, currID)
		if err != nil {
			logger.Errorf("Failed to check if user %v is admin of %v : %v", userID, currID, err)
			//return false, nil, errors.Wrap(err, "Failed to check isDirectAdmin")
		}
		// user is admin
		if isAdmin {
			path = append([]string{userID}, path...)
			putCacheIsUserAdminOfGroup(stub, userID, groupID, path)
			return true, path, nil
		}
		parents, err = graph.GetDirectParents(stub, global.USER_GRAPH, currID)
	}

	putCacheIsUserAdminOfGroup(stub, userID, groupID, nil)
	return false, nil, err
}

func getCacheKeyIsUserAdminOfGroup(userID string, groupID string) string {
	return "user_isAdmin_" + userID + "_" + groupID
}

func putCacheIsUserAdminOfGroup(stub cached_stub.CachedStubInterface, userID string, groupID string, adminPath []string) error {
	cachekey := getCacheKeyIsUserAdminOfGroup(userID, groupID)
	pathCopy := make([]string, len(adminPath))
	copy(pathCopy, adminPath)
	return stub.PutCache(cachekey, pathCopy)
}

func getCacheIsUserAdminOfGroup(stub cached_stub.CachedStubInterface, userID string, groupID string) ([]string, error) {

	cachekey := getCacheKeyIsUserAdminOfGroup(userID, groupID)
	keyCache, err := stub.GetCache(cachekey)
	if err != nil {
		return nil, err
	}
	if value, ok := keyCache.([]string); ok {
		valueCopy := make([]string, len(value))
		copy(valueCopy, value)
		logger.Debugf("Get adminPath from cache: %v %v", userID, groupID)
		return valueCopy, nil
	} else {
		return nil, errors.New("Invalid cache value")
	}
}

// GetMyDirectGroupIDs returns a list of group ids of which user is a direct member.
func GetMyDirectGroupIDs(stub cached_stub.CachedStubInterface, userID string) ([]string, error) {
	groupIDs := []string{}
	directParents, err := graph.GetDirectParents(stub, global.USER_GRAPH, userID)
	if err != nil {
		var errMsg = "Failed to get direct parents of " + userID + " in common.global.USER_GRAPH"
		logger.Errorf("%v: %v", errMsg, err)
		return groupIDs, errors.Wrap(err, errMsg)
	}
	for _, directParent := range directParents {
		groupIDs = append(groupIDs, directParent)
	}
	return groupIDs, nil
}

// GetMyDirectAdminGroupIDs returns a list of group ids of which user is a direct admin.
func GetMyDirectAdminGroupIDs(stub cached_stub.CachedStubInterface, userID string) ([]string, error) {

	groupIDs := []string{}

	// make sure userID is not a group
	// don't need to check here
	// let's assume that this should be checked by caller
	// if user is group, it will return parent group, which is okay

	//get IDs
	directParents, err := graph.GetDirectParents(stub, global.USER_GRAPH, userID)
	if err != nil {
		var errMsg = "Failed to get direct parents of " + userID + " in common.global.USER_GRAPH"
		logger.Errorf("%v: %v", errMsg, err)
		return groupIDs, errors.Wrap(err, errMsg)
	}
	for _, directParent := range directParents {
		edgeValueBytesResult, _, err := graph.GetEdge(stub, global.USER_GRAPH, directParent, userID)
		edgeString := string(edgeValueBytesResult[:])
		if err != nil {
			custom_err := &custom_errors.GetEdgeError{ParentNode: directParent, ChildNode: userID}
			logger.Errorf("%v: %v", custom_err, err)
			return groupIDs, errors.Wrap(err, custom_err.Error())
		}
		if edgeString == global.ADMIN_EDGE || edgeString == global.SUBGROUP_EDGE {
			groupIDs = append(groupIDs, directParent)
		}
	}

	return groupIDs, nil
}

// IsParentGroup returns true if parentGroup is a direct or indirect parent of childGroup, false otherwise.
func IsParentGroup(stub cached_stub.CachedStubInterface, caller data_model.User, parentGroupID string, childGroupID string) bool {

	//make sure parentGroupID and childGroupID are groups and not users
	parentGroup, err := GetUserDataWithoutPrivateData(stub, parentGroupID)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: parentGroupID}
		logger.Errorf("%v: %v", custom_err, err)
		return false
	}
	if parentGroup.IsGroup != true {
		var errMsg = "parentGroupID cannot be user: " + parentGroupID
		logger.Error(errMsg)
		return false
	}
	childGroup, err := GetUserDataWithoutPrivateData(stub, childGroupID)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: childGroupID}
		logger.Errorf("%v: %v", custom_err, err)
		return false
	}
	if childGroup.IsGroup != true {
		var errMsg = "childGroupID cannot be user: " + childGroupID
		logger.Error(errMsg)
		return false
	}

	isParent, err := IsUserInGroup(stub, childGroupID, parentGroupID)
	if err != nil {
		var errMsg = "IsUserInGroup Error: " + err.Error()
		logger.Error(errMsg)
		return false
	}

	return isParent
}

// GetUserAssetID returns the asset ID for the stored user object identified by the given userID.
func GetUserAssetID(userID string) string {
	return user_mgmt_g.GetUserAssetID(userID)
}

// ConvertUserToAsset converts a user to an asset.
func ConvertUserToAsset(user data_model.User) data_model.Asset {
	return user.ConvertToAsset()
}

// ConvertUserFromAsset converts an asset to a user object.
func ConvertUserFromAsset(asset *data_model.Asset) data_model.User {
	user := data_model.User{}
	user.LoadFromAsset(asset)
	return user
}

// get user's public key
func GetUserPublicKey(stub cached_stub.CachedStubInterface, userID string) (data_model.Key, error) {
	userData, err := GetUserDataWithoutPrivateData(stub, userID)
	if err != nil {
		logger.Errorf("Failed to get user data: %v", err)
		return data_model.Key{}, err
	} else if len(userData.ID) == 0 {
		logger.Errorf("Failed to get user \"%v\"", userID)
		return data_model.Key{}, errors.Errorf("Failed to get user \"%v\"", userID)
	} else if len(userData.PublicKeyB64) == 0 {
		logger.Errorf("Failed to get user public key for a user \"%v\"", userID)
		return data_model.Key{}, errors.Errorf("Failed to get user public key for a user \"%v\"", userID)
	}
	return userData.GetPublicKey(), nil
}

// GetUserSymKey returns a user's sym key.
// keyPath is an optional parameter; if passed in, this keyPath is used to get the symKey.
// Caller must have access to the user's private key.
func GetUserSymKey(stub cached_stub.CachedStubInterface, caller data_model.User, userId string, keyPath ...[]string) (data_model.Key, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("callerId: %v, userId: %v", caller.ID, userId)
	user := data_model.User{ID: userId}
	symkey := data_model.Key{ID: user.GetSymKeyId(), Type: global.KEY_TYPE_SYM}
	if userId == caller.ID {
		logger.Debugf("returning user symkey from caller for %v", userId)
		symkey.KeyBytes = caller.SymKey
		return symkey, nil
	} else {
		symkeyPath := []string{}
		symkeyPath2 := []string{} //retury key
		if len(keyPath) > 0 && len(keyPath[0]) > 0 {
			// keyPath passed in
			symkeyPath = keyPath[0]
			if !utils.EqualStringArrays(symkeyPath, []string{caller.GetPubPrivKeyId(), user.GetSymKeyId()}) {
				symkeyPath2 = []string{caller.GetPubPrivKeyId(), user.GetSymKeyId()}
			} else if !utils.EqualStringArrays(symkeyPath, []string{caller.GetPubPrivKeyId(), user.GetPrivateKeyHashSymKeyId(), user.GetPubPrivKeyId(), user.GetSymKeyId()}) {
				symkeyPath2 = []string{caller.GetPubPrivKeyId(), user.GetPrivateKeyHashSymKeyId(), user.GetPubPrivKeyId(), user.GetSymKeyId()}
			}
		} else {
			// default keys
			symkeyPath = []string{caller.GetPubPrivKeyId(), user.GetSymKeyId()}
			symkeyPath2 = []string{caller.GetPubPrivKeyId(), user.GetPrivateKeyHashSymKeyId(), user.GetPubPrivKeyId(), user.GetSymKeyId()}
		}
		startKeyBytes := []byte{}
		symKeyBytes := []byte{}
		//check user sym key id
		if symkeyPath[len(symkeyPath)-1] != user.GetSymKeyId() {
			logger.Error("last element of key path must be user's symKeyId")
			return symkey, errors.New("last element of key path must be user's symKeyId")
		}
		// check start key id
		if symkeyPath[0] == caller.GetPubPrivKeyId() {
			startKey := caller.GetPrivateKey()
			startKeyBytes = startKey.KeyBytes
		} else if symkeyPath[0] == caller.GetPrivateKeyHashSymKeyId() {
			startKey := caller.GetPrivateKeyHashSymKey()
			startKeyBytes = startKey.KeyBytes
		} else if symkeyPath[0] == caller.GetSymKeyId() {
			startKeyBytes = caller.SymKey
		} else {
			logger.Error("first element of key path must be caller's key")
			return symkey, errors.New("first element of key path must be caller's key")
		}
		// get key using the key path
		symKeyBytes, err := key_mgmt_c.GetKey(stub, symkeyPath, startKeyBytes)
		if err == nil && len(symKeyBytes) > 0 {
			symkey.KeyBytes = symKeyBytes
			return symkey, nil
		}
		if len(symkey.KeyBytes) == 0 && len(symkeyPath2) > 0 {
			// check start key id
			startKey := caller.GetPrivateKey()
			startKeyBytes = startKey.KeyBytes
			// get key using the retry key path
			symKeyBytes, err := key_mgmt_c.GetKey(stub, symkeyPath2, startKeyBytes)
			if err == nil && len(symKeyBytes) > 0 {
				symkey.KeyBytes = symKeyBytes
				return symkey, nil
			} else {
				logger.Errorf("Unable to get user's sym key: %v", err)
				return symkey, errors.New("unable to get user's sym key")
			}
		} else {
			logger.Errorf("Unable to get user's sym key: %v", err)
			return symkey, errors.New("unable to get user's sym key")
		}
	}
	return symkey, nil
}

// GetUserPrivateKey returns a user's private key.
// Caller must have access to the user's private key.
// keyPath is optional.
// Default keyPath = [caller privkey, user privhashkey, user privkey]
// If keyPath is passed in, use this keyPath to get the private key.
func GetUserPrivateKey(stub cached_stub.CachedStubInterface, caller data_model.User, userId string, keyPath ...[]string) (data_model.Key, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("userId: %v %v", userId, keyPath)

	user := data_model.User{ID: userId}
	privkey := data_model.Key{ID: user.GetPubPrivKeyId(), Type: global.KEY_TYPE_PRIVATE}
	if userId == caller.ID {
		logger.Debugf("returning user private from caller for %v", userId)
		return caller.GetPrivateKey(), nil
	} else {
		var privkeyPath []string
		if len(keyPath) > 0 && len(keyPath[0]) > 0 {
			privkeyPath = keyPath[0]
		} else {
			privkeyPath = []string{caller.GetPubPrivKeyId(), user.GetPrivateKeyHashSymKeyId(), user.GetPubPrivKeyId()}
		}
		//check user sym key id
		if privkeyPath[len(privkeyPath)-1] != user.GetPubPrivKeyId() {
			logger.Errorf("last element of key path must be user's PubPrivKeyId: %v", user.GetPubPrivKeyId())
			return privkey, errors.New("last element of key path must be user's PubPrivKeyId")
		}
		// check start key id
		var startKeyBytes []byte = nil
		if privkeyPath[0] == caller.GetPubPrivKeyId() {
			startKey := caller.GetPrivateKey()
			startKeyBytes = startKey.KeyBytes
		} else if privkeyPath[0] == caller.GetPrivateKeyHashSymKeyId() {
			startKey := caller.GetPrivateKeyHashSymKey()
			startKeyBytes = startKey.KeyBytes
		} else if privkeyPath[0] == caller.GetSymKeyId() {
			startKeyBytes = caller.SymKey
		} else {
			logger.Error("first element of key path must be caller's privKeyId or privSymKeyId")
			return privkey, errors.New("first element of key path must be caller's privKeyId or privSymKeyId")
		}
		// get key using the key path
		privkeyBytes, err := key_mgmt_c.GetKey(stub, privkeyPath, startKeyBytes)
		if err == nil && len(privkeyBytes) > 0 {
			privkey.KeyBytes = privkeyBytes
			return privkey, nil
		} else {
			logger.Errorf("Unable to get user's private key: %v", err)
			return privkey, errors.New("unable to get user's private key")
		}
	}
}
