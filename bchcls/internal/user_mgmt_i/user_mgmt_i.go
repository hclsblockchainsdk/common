/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package user_mgmt_i manages users and groups.
// It stores users/groups as assets and maintains a graph of user/group relationships.
package user_mgmt_i

import (
	"common/bchcls/asset_mgmt/asset_manager"
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/custom_errors"
	"common/bchcls/data_model"
	"common/bchcls/internal/asset_mgmt_i"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/internal/user_mgmt_i/user_mgmt_c"
	"common/bchcls/internal/user_mgmt_i/user_mgmt_c/user_mgmt_g"
	"common/bchcls/simple_rule"
	"common/bchcls/utils"

	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("user_mgmt_i")

// ------------------------------------------------------
// ---------------------- INIT FUNCTIONS ----------------
// ------------------------------------------------------

// Init sets up the user_mgmt package by building an index table for users.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	return user_mgmt_c.Init(stub, logLevel...)
}

// ----------------- CONVERSION BETWEEN USER AND ASSET -----------------

// GetUserAssetID returns the asset ID for the stored user object identified by the given userID.
func GetUserAssetID(userID string) string {
	return user_mgmt_g.GetUserAssetID(userID)
}

func getPublicData(user data_model.User) []byte {
	publicData := data_model.UserPublicData{}
	publicData.ID = user.ID
	publicData.Name = user.Name
	publicData.PublicKeyB64 = user.PublicKeyB64
	publicData.Role = user.Role
	publicData.Status = user.Status
	publicData.IsGroup = user.IsGroup
	publicData.SolutionPublicData = user.SolutionPublicData
	publicData.ConnectionID = user.ConnectionID
	publicBytes, _ := json.Marshal(&publicData)
	return publicBytes
}

func getPrivateData(user data_model.User) []byte {
	privateData := data_model.UserPrivateData{}
	privateData.SolutionPrivateData = user.SolutionPrivateData
	privateData.Email = user.Email
	privateData.KmsPrivateKeyId = user.KmsPrivateKeyId
	privateData.KmsPublicKeyId = user.KmsPublicKeyId
	privateData.KmsSymKeyId = user.KmsSymKeyId
	privateData.Secret = user.Secret
	privateBytes, _ := json.Marshal(&privateData)
	return privateBytes
}

// ConvertToAsset converts a user object to an asset object.
func ConvertToAsset(user data_model.User) data_model.Asset {
	asset := data_model.Asset{}
	asset.AssetId = user_mgmt_g.GetUserAssetID(user.ID)
	asset.Datatypes = []string{}
	metaData := make(map[string]string)
	metaData["namespace"] = global.USER_ASSET_NAMESPACE
	asset.Metadata = metaData
	asset.PublicData = user.GetPublicDataBytes()
	asset.PrivateData = user.GetPrivateDataBytes()
	asset.IndexTableName = global.INDEX_USER
	asset.OwnerIds = []string{user.ID}

	// if an off-chain datastore is specified, save the id so that the asset can be saved there
	if len(user.ConnectionID) != 0 {
		asset.SetDatastoreConnectionID(user.ConnectionID)
	}
	return asset
}

// ConvertFromAsset converts an asset object to a user object.
func ConvertFromAsset(asset *data_model.Asset) data_model.User {
	u := data_model.User{}
	var publicData data_model.UserPublicData
	var privateData data_model.UserPrivateData

	err := json.Unmarshal(asset.PublicData, &publicData)
	if err == nil {
		u.ID = publicData.ID
		u.Name = publicData.Name
		u.Role = publicData.Role
		u.PublicKeyB64 = publicData.PublicKeyB64
		u.IsGroup = publicData.IsGroup
		u.Status = publicData.Status
		u.SolutionPublicData = publicData.SolutionPublicData
		if datastoreConnectionID, ok := asset.Metadata[global.DATASTORE_CONNECTION_ID_METADATA_KEY]; ok {
			u.ConnectionID = datastoreConnectionID
		}
	}

	err = json.Unmarshal(asset.PrivateData, &privateData)
	if err == nil {
		u.Email = privateData.Email
		u.KmsPrivateKeyId = privateData.KmsPrivateKeyId
		u.KmsPublicKeyId = privateData.KmsPublicKeyId
		u.KmsSymKeyId = privateData.KmsSymKeyId
		u.Secret = privateData.Secret
		u.SolutionPrivateData = privateData.SolutionPrivateData
	}

	u.PublicKey, _ = crypto.ParsePublicKeyB64(u.PublicKeyB64)
	u.PrivateKey = nil
	u.SymKey = nil

	return u

}

// ----------------- GETUSER AND USER.PUTSTATE FUNCTIONS -----------------

// GetUserData finds, decrypts, and returns a User for the given userId.
// The user's public key will always be included.
// If the private and/or sym keys cannot be retrieved, they will be left blank, and no error will be returned.
// If userId is same as callerId, User object is copied from caller object.
//
// options can be passed in any of the following orders:
//
// keyPath []String
// keyPath []String, keyPath2 []string
// includePrivateAndSymKeys bool
// includePrivateAndSymKeys bool, keyPath []string
// includePrivateAndSymKeys bool, keyPath []string, keyPath2 []string
// includePrivateAndSymKeys bool, includePrivateData bool
// includePrivateAndSymKeys bool, includePrivateData bool, keyPath []string
// includePrivateAndSymKeys bool, includePrivateData bool, keyPath []string, keyPath2 []string
//
// If includePrivateAndSymKeys (default false) is true, attempts to include the user's private and sym keys as well.
// If includePrivateData (default false) is false, the user's private data will not be decrypted.
// if keyPath (default nil) is passed in, user's symKey will be retrieved using this keyPath.
// The first element of keyPath must be the caller's key, and the last element must be the user's sym key.
// keyPaths is always the last option if it's specified.
// KeyPath2 is for the user's private key.
func GetUserData(stub cached_stub.CachedStubInterface, caller data_model.User, userID string, options ...interface{}) (data_model.User, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("GetUserData for userID: %v options: %v", userID, options)

	// parse options
	var includePrivateAndSymKeys, includePrivateData bool
	var assetSymkeyPath []string = nil
	var assetPrivkeyPath []string = nil
	var lenOptions int = len(options)
	if lenOptions > 0 {
		// get keyPaths options
		var lastOption interface{}
		lastOption = options[lenOptions-1]
		if lastOption, ok := lastOption.([]string); ok {
			assetSymkeyPath = lastOption
			lenOptions = lenOptions - 1
		}
		if lenOptions > 1 {
			lastOption = options[lenOptions-1]
			if lastOption, ok := lastOption.([]string); ok {
				assetPrivkeyPath = assetSymkeyPath
				assetSymkeyPath = lastOption
				lenOptions = lenOptions - 1
			}
		}

		// get other bool options
		if lenOptions > 2 {
			logger.Errorf("invalid number of options: %v", len(options))
			return data_model.User{}, errors.New("Invalid number of options")
		}
		if lenOptions >= 1 {
			if option, ok := options[0].(bool); ok {
				includePrivateAndSymKeys = option
			} else {
				logger.Errorf("invalid option %v", options[0])
				return data_model.User{}, errors.New("Invalid option")
			}
		}
		if lenOptions >= 2 {
			if option, ok := options[1].(bool); ok {
				includePrivateData = option
			} else {
				logger.Errorf("invalid option %v", options[1])
				return data_model.User{}, errors.New("Invalid option")
			}
		}
	}

	logger.Debugf("includePrivateAndSymKeys: %v", includePrivateAndSymKeys)
	logger.Debugf("includePrivateData: %v", includePrivateData)
	logger.Debugf("assetSymkeyPath: %v", assetSymkeyPath)
	logger.Debugf("assetPrivkeyPath: %v", assetPrivkeyPath)

	userToReturn := data_model.User{}

	// key path
	user := data_model.User{ID: userID}
	if len(assetSymkeyPath) == 0 && len(assetPrivkeyPath) > 0 {
		assetSymkeyPath = append(assetPrivkeyPath, user.GetSymKeyId())
	}

	// get user asset key
	am := asset_mgmt_i.GetAssetManager(stub, caller)
	userAssetKey := data_model.Key{}
	if includePrivateData || includePrivateAndSymKeys {
		if caller.ID == userID {
			userAssetKey.ID = caller.GetSymKeyId()
			userAssetKey.KeyBytes = caller.SymKey
			userAssetKey.Type = global.KEY_TYPE_SYM
		} else {
			var err error
			userAssetKey, err = GetUserSymKey(stub, caller, userID, assetSymkeyPath)
			if err != nil {
				logger.Warningf("Failed to get user's sym key: privateData will not be decrypted: %v", userID)
			}
		}
	}

	userAsset, err := am.GetAsset(GetUserAssetID(userID), userAssetKey)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: userID}
		logger.Errorf("%v: %v", custom_err, err)
		return data_model.User{}, errors.Wrap(err, custom_err.Error())
	}
	if len(userAsset.AssetId) == 0 {
		custom_err := &custom_errors.GetUserError{ID: userID}
		logger.Errorf("%v", custom_err)
		return data_model.User{}, nil
	}

	userToReturn = ConvertFromAsset(userAsset)

	if includePrivateAndSymKeys {
		//symkey
		userToReturn.SymKey = userAssetKey.KeyBytes
		userToReturn.SymKeyB64 = crypto.EncodeToB64String(userAssetKey.KeyBytes)
		//privkey
		privKey, _ := GetUserPrivateKey(stub, caller, userID, assetPrivkeyPath)
		if len(privKey.KeyBytes) > 0 {
			userToReturn.PrivateKeyB64 = crypto.EncodeToB64String(privKey.KeyBytes)
			userToReturn.PrivateKey, _ = crypto.ParsePrivateKey(privKey.KeyBytes)
		} else {
			logger.Warningf("Unable to get private key of the user %v", userID)
		}
	}

	return userToReturn, nil
}

// commitToLedger encrypts and stores the user on the ledger.
func commitToLedger(stub cached_stub.CachedStubInterface,
	caller data_model.User,
	user data_model.User,
	userSymKeyID string,
	userSymKey []byte,
	existingOwnerIds []string,
	isNewUserAsset bool) error {

	// Call AddAsset to add user asset to the ledger
	asset := ConvertToAsset(user)
	asset.AssetKeyId = userSymKeyID
	asset.AssetKeyHash = crypto.Hash(userSymKey)

	assetKey := data_model.Key{}
	assetKey.ID = userSymKeyID
	assetKey.Type = global.KEY_TYPE_SYM
	assetKey.KeyBytes = userSymKey

	// preserve the owners
	asset.OwnerIds = existingOwnerIds
	// ownerIds cannot be empty
	if len(asset.OwnerIds) == 0 {
		asset.OwnerIds = []string{user.ID}
	}

	// add or update user object
	var err error
	am := asset_mgmt_i.GetAssetManager(stub, user)
	if isNewUserAsset {
		err = am.AddAsset(asset, assetKey, true)
	} else {
		err = am.UpdateAsset(asset, assetKey)
	}
	if err != nil {
		var errMsg = "PutState for " + user.ID + " error in user_mgmt"
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}

	return nil
}

// ==================================================================================
// HIGHLEVEL API FUNCTIONS
// ==================================================================================

// GetCallerData gets keys from TMAP and returns the caller's data from the ledger.
func GetCallerData(stub cached_stub.CachedStubInterface) (data_model.User, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	var caller = data_model.User{}

	var tmap map[string][]byte
	tmap, err := stub.GetTransient()
	if err != nil || tmap == nil {
		logger.Errorf("Unable to get Transient Map: %v", err)
		return caller, errors.New("Unable to get Transient Map")
	}

	// get priv key
	prk1, ok := tmap["prvkey"]
	if ok != true || prk1 == nil {
		logger.Error("Unable to get prvkey from transient")
		return caller, errors.New("Unable to parse private key from transitent")
	}
	privateKeyB64 := base64.StdEncoding.EncodeToString(prk1)
	privkey, err := crypto.ParsePrivateKeyB64(privateKeyB64)
	if err != nil || privkey == nil {
		logger.Errorf("Unable to parse prvkey from transient: %v", err)
		return caller, errors.New("Unable to parse private key from transitent")
	}

	// get pub key
	puk1, ok := tmap["pubkey"]
	if ok != true || puk1 == nil {
		logger.Error("Unable to get pubkey from transient")
		return caller, errors.New("Unable to get public key from transitent")
	}
	publicKeyB64 := base64.StdEncoding.EncodeToString(puk1)
	pubkey, err := crypto.ParsePublicKeyB64(publicKeyB64)
	if err != nil || pubkey == nil {
		logger.Errorf("Unable to parse pubkey from transient: %v", err)
		return caller, errors.New("Unable to parse public key from transitent")
	}

	//get sym key
	sym1, ok := tmap["symkey"]
	if ok != true || sym1 == nil {
		logger.Error("Unable to get symkey from transient")
		return caller, errors.New("Unable to get sym key from transitent")
	}
	symKeyB64 := base64.StdEncoding.EncodeToString(sym1)
	symkey, err := crypto.ParseSymKeyB64(symKeyB64)
	if err != nil || symkey == nil {
		logger.Errorf("Unable to parse symkey from transient: %v", err)
		return caller, errors.New("Unable to parse sym key from transitent")
	}

	certid, ok := tmap["id"]
	if ok != true || certid == nil {
		logger.Error("Unable to get id from transient")
		return caller, errors.New("Unable to get id from transitent")
	}

	caller.ID = string(certid[:])
	caller.Role = ""
	caller.PrivateKey = privkey
	caller.PrivateKeyB64 = privateKeyB64
	caller.PublicKey = pubkey
	caller.PublicKeyB64 = publicKeyB64
	caller.SymKey = symkey
	caller.SymKeyB64 = symKeyB64

	//get user Info
	var checkUserInfo = true
	function, args := stub.GetFunctionAndParameters()
	if function == "init" {
		checkUserInfo = false
	} else if function == "registerOrg" {
		var org = data_model.Org{}
		var orgBytes = []byte(args[0])
		err := json.Unmarshal(orgBytes, &org)
		if err == nil && org.Id == caller.ID {
			checkUserInfo = false
		}
	} else if function == "registerUser" {
		user := data_model.User{}
		userBytes := []byte(args[0])
		err := json.Unmarshal(userBytes, &user)
		if err == nil && user.ID == caller.ID {
			checkUserInfo = false
		}
	}
	logger.Debugf("checkUserInfo: %v %v", function, checkUserInfo)

	userInfo, err := GetUserData(stub, caller, caller.ID, false, true)

	if err != nil {
		logger.Errorf("Unable to get user: %v", err)
		if checkUserInfo {
			return caller, errors.New("Unable to get user")
		}

	} else {
		caller.Email = userInfo.Email
		caller.IsGroup = userInfo.IsGroup
		caller.Name = userInfo.Name
		caller.Role = userInfo.Role
		caller.KmsPrivateKeyId = userInfo.KmsPrivateKeyId
		caller.KmsPublicKeyId = userInfo.KmsPublicKeyId
		caller.KmsSymKeyId = userInfo.KmsSymKeyId
		caller.Secret = userInfo.Secret
		caller.Status = userInfo.Status
		caller.SolutionPublicData = userInfo.SolutionPublicData
		caller.SolutionPrivateData = userInfo.SolutionPrivateData
		caller.ConnectionID = userInfo.ConnectionID
	}

	logger.Debugf("caller sucess: %v %v", caller.ID, caller.Role)
	return caller, nil
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

	// Get parameters from args
	user := data_model.User{}
	userBytes := []byte(args[0])
	err := json.Unmarshal(userBytes, &user)
	if err != nil {
		logger.Errorf("Invalid input parameter: user, %v", err)
		return nil, errors.Wrap(err, "Invalid input parameter: user")
	}

	if user.Role != global.ROLE_SYSTEM_ADMIN && user.Role != global.ROLE_USER && user.Role != global.ROLE_AUDIT {
		errMsg := fmt.Sprintf("Invalid role.  Role must be %v, %v, or %v", global.ROLE_SYSTEM_ADMIN, global.ROLE_USER, global.ROLE_AUDIT)
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	allowAccess := false
	if len(args) >= 2 {
		allowAccess, err = strconv.ParseBool(args[1])
		logger.Debugf("allowAccess: %v, %v", allowAccess, err)
		if err != nil {
			allowAccess = false
		}
	}

	return nil, RegisterUserWithParams(stub, caller, user, allowAccess)

}

// RegisterUserWithParams registers or updates a user.
// user 	   - the user object to add/update
// allowAccess - [users] if true, gives the caller access to the user's private key (only applies for a new user, not an update of an existing user)
// allowAccess - [groups] if true, makes the caller an admin of the group (only applies for a new group, not an update of an existing group)
func RegisterUserWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, user data_model.User, allowAccess bool) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v user: %v allowAccess: %v", caller.ID, user.ID, allowAccess)
	// check user is not a group
	if user.IsGroup || user.Role == global.ROLE_ORG {
		logger.Error("User is a group or org; please use RegisterOrg or RegisterSubgroup function")
		return errors.New("User is a group or org; please use RegisterOrg or RegisterSubgroup function")
	}
	return registerUserInternal(stub, caller, user, allowAccess)
}

// registerUserInternal registers a user (user can be a person or a group)
func registerUserInternal(stub cached_stub.CachedStubInterface, caller data_model.User, user data_model.User, allowAccess bool) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v user: %v allowAccess: %v", caller.ID, user.ID, allowAccess)

	if len(user.ID) == 0 {
		logger.Errorf("Invalid user ID")
		return errors.New("Invalid user ID")
	}

	am := asset_mgmt_i.GetAssetManager(stub, caller)
	// Check if registering existing user (without asset key)
	existingUser := false
	userAsset, err := am.GetAsset(GetUserAssetID(user.ID), data_model.Key{})
	publicData := data_model.UserPublicData{}
	if err == nil && len(userAsset.AssetId) > 0 {
		json.Unmarshal(userAsset.PublicData, &publicData)
		logger.Debugf("Existing user found: %v", userAsset)

		// existing user's role cannot be changed
		user.Role = publicData.Role
		existingUser = true
	}
	logger.Debugf("existingUser=%v", existingUser)

	// Get and check user keys
	// If the caller user is not the user to be registered, you must provide user keys
	// or you must have access to user keys
	// 1. check if user is caller
	if caller.ID == user.ID {
		user.PrivateKey = caller.PrivateKey
		user.PrivateKeyB64 = caller.PrivateKeyB64
		user.PublicKey = caller.PublicKey
		user.PublicKeyB64 = caller.PublicKeyB64
		user.SymKey = caller.SymKey
		user.SymKeyB64 = caller.SymKeyB64
		logger.Debug("Got UserKey from caller")
	}

	// 2. if existing user, try to get symkey and public key from ledger
	if existingUser {
		logger.Debug("trying to get user sym keys from ledger")
		userSymKey, err := GetUserSymKey(stub, caller, user.ID)
		if err != nil {
			logger.Errorf("Unable to get user sym key for an existing user: %v", err)
			return errors.Wrap(err, "Unable to get user sym key for an existing user")
		}
		user.SymKeyB64 = crypto.EncodeToB64String(userSymKey.KeyBytes)
		user.SymKey = userSymKey.KeyBytes

		logger.Debug("trying to get user public keys from ledger")
		userPubKey, err := GetUserPublicKey(stub, caller, user.ID)
		if err != nil {
			logger.Errorf("Unable to get user public key for an existing user: %v", err)
			return errors.Wrap(err, "Unable to get user public key for an existing user")
		}
		user.PublicKeyB64 = crypto.EncodeToB64String(userPubKey.KeyBytes)
	}

	// you need to provide all userkeys for new user
	if !existingUser && (len(user.PublicKeyB64) == 0 || len(user.PrivateKeyB64) == 0 || len(user.SymKeyB64) == 0) {
		logger.Errorf("User keys (Private, Public, Symkey) not provided for the new user")
		return errors.New("User keys not provided for the new user")
	}

	// you need to provide symkey and pubkey for exsisting user
	if existingUser && (len(user.SymKeyB64) == 0 || len(user.PublicKeyB64) == 0) {
		logger.Errorf("User keys (Public, Symkey) not provided for the existing user")
		return errors.New("User keys not provided for the existing user")
	}

	// Now, parse keys
	if len(user.PrivateKeyB64) > 0 {
		privateKey, err := crypto.ParsePrivateKeyB64(user.PrivateKeyB64)
		if err != nil || privateKey == nil {
			logger.Errorf("Invalid privte key: %v", err)
			return errors.Wrap(err, "Invalid private key")
		} else {
			user.PrivateKey = privateKey
		}
	}
	if len(user.PublicKeyB64) > 0 {
		publicKey, err := crypto.ParsePublicKeyB64(user.PublicKeyB64)
		if err != nil || publicKey == nil {
			logger.Errorf("Invalid public key: %v", err)
			return errors.Wrap(err, "Invalid public key")
		} else {
			user.PublicKey = publicKey
		}
	}
	if len(user.SymKeyB64) > 0 {
		symKey, err := crypto.ParseSymKeyB64(user.SymKeyB64)
		if err != nil {
			logger.Errorf("Invalid sym key: %v", err)
			return errors.Wrap(err, "Invalid sym key")
		} else {
			user.SymKey = symKey
		}
	}

	// get private data of existing user
	privateData := data_model.UserPrivateData{}
	if existingUser {
		privateDataByte := userAsset.PrivateData
		if data_model.IsEncryptedData(privateDataByte) {
			// get encrypted data
			encData := data_model.EncryptedData{}
			err := encData.Load(privateDataByte)
			userAsset.PrivateData = encData.Encrypted

			// decrypt
			privateDataByte, err = asset_mgmt_i.GetAssetPrivateData(stub, *userAsset, user.SymKey)
			if err != nil {
				logger.Errorf("Unable to decypt private data of existing user asset: %v", err)
				return errors.Wrap(err, "Unable to decypt private data of existing user asset")
			}
		}

		err := json.Unmarshal(privateDataByte, &privateData)
		if err != nil {
			custom_err := &custom_errors.UnmarshalError{Type: "userAsset.PrivateData"}
			logger.Errorf("%v: %v", custom_err, err)
			return errors.Wrap(err, custom_err.Error())
		}
	}

	// Check if existing user, merge in updated fields
	if existingUser {
		user = getUpdatedUserObject(publicData, privateData, user)
		logger.Debugf("updated user object: %+v", user)
	}

	// Save user to ledger
	err = commitToLedger(stub, caller, user, user.GetSymKeyId(), user.SymKey, userAsset.OwnerIds, !existingUser)
	if err != nil {
		errMsg := "Failed to create new user " + user.ID + " in user_mgmt"
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}

	// add key access only for new user
	// existing user should already have these in place
	if !existingUser {
		// Get keys for user
		symKey := user.GetSymKey()
		priKey := user.GetPrivateKey()
		priKeyHashSym := user.GetPrivateKeyHashSymKey()
		logSymKey := user.GetLogSymKey()

		//encrypt sym key with public key
		edgeData := make(map[string]string)
		edgeData["type"] = global.KEY_TYPE_SYM
		err = key_mgmt_i.AddAccess(stub, priKey, symKey, edgeData)
		if err != nil {
			errMsg := "Failed saving symkey for " + user.ID
			logger.Errorf("%v: %v", errMsg, err)
			return errors.Wrap(err, errMsg)
		}
		logger.Infof("Success saving user sym key for user %v", user.ID)

		//encrypt private key with sym key derived from hash of private key
		edgeData = make(map[string]string)
		edgeData["type"] = global.KEY_TYPE_PRIVATE
		err = key_mgmt_i.AddAccess(stub, priKeyHashSym, priKey, edgeData)
		if err != nil {
			errMsg := "Failed saving private key for " + user.ID
			logger.Errorf("%v: %v", errMsg, err)
			return errors.Wrap(err, errMsg)
		}
		logger.Infof("Success saving user private key for user %v", user.ID)

		//encrypt sym key derived from hash of private key with public key
		edgeData = make(map[string]string)
		edgeData["type"] = global.KEY_TYPE_SYM
		err = key_mgmt_i.AddAccess(stub, priKey, priKeyHashSym, edgeData)
		if err != nil {
			errMsg := "Failed saving symkey for " + user.ID
			logger.Errorf("%v: %v", errMsg, err)
			return errors.Wrap(err, errMsg)
		}
		logger.Infof("Success saving user sym key for user %v", user.ID)

		//encrypt log sym key with sym key
		edgeData = make(map[string]string)
		edgeData["type"] = global.KEY_TYPE_SYM
		err = key_mgmt_i.AddAccess(stub, symKey, logSymKey, edgeData)
		if err != nil {
			errMsg := "Failed saving log sym key for " + user.ID
			logger.Errorf("%v: %v", errMsg, err)
			return errors.Wrap(err, errMsg)
		}
		logger.Infof("Success saving user log sym key for user %v", user.ID)

		// allow Access
		if allowAccess && caller.ID != user.ID {
			if !caller.IsGroup && user.IsGroup {
				// group case make caller an admin
				logger.Debugf("Making the caller an admin of the user %v %v", caller.ID, user.ID)
				err = putUserInGroup(stub, user, caller, user, true)
				if err != nil {
					logger.Errorf("Failed to make caller an admin of the user: %v", err)
					return errors.Wrap(err, "Failed to make caller an admin of the user")
				}
			} else {
				// user AddAccess()
				logger.Debugf("Saving user keys of %v to give access to %v", user.ID, caller.ID)
				callerPriKey := caller.GetPrivateKey()
				edgeData := make(map[string]string)
				edgeData["type"] = global.KEY_TYPE_PRIVATE
				//same access as the admin access
				err = key_mgmt_i.AddAccess(stub, callerPriKey, priKeyHashSym, edgeData)
				if err != nil {
					logger.Errorf("Failed to add access for caller's private hash key: %v", err)
					return errors.Wrap(err, "Failed to add access for caller's private hash key")
				}
				err = key_mgmt_i.AddAccess(stub, callerPriKey, symKey, edgeData)
				if err != nil {
					logger.Errorf("Failed to add access for caller sym key: %v", err)
					return errors.Wrap(err, "Failed to add access for caller sym key")
				}
			}
		}

	}
	logger.Infof("successfully registered user %v", user.ID)
	return nil
}

// GetUser returns a user.
//
// args = [userId]
func GetUser(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)
	userID := args[0]
	user, err := GetUserData(stub, caller, userID, false, true)
	if err != nil {
		return nil, err
	}
	if user.Equal(data_model.User{}) {
		var errMsg = userID + " does not exist"
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}
	return json.Marshal(&user)
}

// getUpdatedUserObject returns an updated user object, only allowing certain user fields to be changed.
func getUpdatedUserObject(existingPublicData data_model.UserPublicData, existingPrivateData data_model.UserPrivateData, newUser data_model.User) data_model.User {

	// These fields cannot be changed
	newUser.ID = existingPublicData.ID
	newUser.Role = existingPublicData.Role
	newUser.IsGroup = existingPublicData.IsGroup
	newUser.PublicKeyB64 = existingPublicData.PublicKeyB64
	newUser.PublicKey, _ = crypto.ParsePublicKeyB64(existingPublicData.PublicKeyB64)
	newUser.KmsPrivateKeyId = existingPrivateData.KmsPrivateKeyId
	newUser.KmsPublicKeyId = existingPrivateData.KmsPublicKeyId
	newUser.KmsSymKeyId = existingPrivateData.KmsSymKeyId
	newUser.Secret = existingPrivateData.Secret

	// These fields cannot be removed
	if utils.IsStringEmpty(newUser.Name) {
		newUser.Name = existingPublicData.Name
	}
	if utils.IsStringEmpty(newUser.Status) {
		newUser.Status = existingPublicData.Status
	}
	if utils.IsStringEmpty(newUser.Email) {
		newUser.Email = existingPrivateData.Email
	}
	if newUser.SolutionPublicData == nil {
		newUser.SolutionPublicData = existingPublicData.SolutionPublicData
	}
	if newUser.SolutionPrivateData == nil {
		newUser.SolutionPrivateData = existingPrivateData.SolutionPrivateData
	}

	return newUser
}

// RegisterSystemAdmin registers a system admin user.
// Caller's role must be "system".
//
// args = [userBytes, allowAccess]
//
// If allowAccess is true and a new user is being registered, gives the caller access to the user's private key.
func RegisterSystemAdmin(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	user := data_model.User{}
	userBytes := []byte(args[0])
	err := json.Unmarshal(userBytes, &user)
	if err != nil {
		logger.Errorf("Invalid input parameter: user: %v", err)
		return nil, errors.New("Invalid input parameter: user")
	}

	if user.Role != global.ROLE_SYSTEM_ADMIN {
		logger.Error("Invalid user role:" + user.Role)
		return nil, errors.New("Invalid user role:" + user.Role)
	}

	if caller.Role != global.ROLE_SYSTEM_ADMIN {
		logger.Error("Permission error:" + caller.Role)
		return nil, errors.New("Permission error:" + caller.Role)
	}

	return RegisterUser(stub, caller, args)
}

// RegisterAuditor registers an auditor user.
// Caller's role must be "system".
//
// args = [userBytes, allowAccess]
//
// If allowAccess is true and a new user is being registered, gives the caller access to the user's private key.
func RegisterAuditor(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	user := data_model.User{}
	userBytes := []byte(args[0])
	err := json.Unmarshal(userBytes, &user)
	if err != nil {
		logger.Errorf("Invalid input parameter: user: %v", err)
		return nil, errors.New("Invalid input parameter: user")
	}

	if user.Role != global.ROLE_AUDIT {
		logger.Error("Invalid user role (should be audit):" + user.Role)
		return nil, errors.New("Invalid user role (should be audit):" + user.Role)
	}

	// Check permission
	if caller.ID != user.ID && !caller.IsSystemAdmin() {
		logger.Error("Only system admin can register an auditor")
		return nil, errors.New("Permission error: not authorized to register auditor, not system admin")
	}

	// ---------------------------------------------------
	// perform any auditor specific setting here
	// ---------------------------------------------------

	// register user
	return RegisterUser(stub, caller, args[0:])
}

// RegisterOrgAdmin registers an org admin user.
// Caller's role must be "system".
//
// args: [userBytes, allowAccess]
//
// If allowAccess is true and a new user is being registered, gives the caller access to the user's private key.
// DEPRECATED use RegisterOrg() and/or GiveAdminPermissionOfGroup()
func RegisterOrgAdmin(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	user := data_model.User{}
	userBytes := []byte(args[0])
	err := json.Unmarshal(userBytes, &user)
	if err != nil {
		logger.Errorf("Invalid input parameter: user: %v", err)
		return nil, errors.New("Invalid input parameter: user")
	}

	if user.Role != global.ROLE_ORG {
		logger.Error("Invalid user role (should be org):" + user.Role)
		return nil, errors.New("Invalid user role (should be org):" + user.Role)
	}

	// Check permission
	if caller.ID != user.ID && !caller.IsSystemAdmin() {
		logger.Error("Only system admin can register an org admin")
		return nil, errors.New("Permission error: not authorized to register org admin, not system admin")
	}

	// ---------------------------------------------------
	// perform any org admin specific setting here
	// ---------------------------------------------------

	// register user
	return RegisterUser(stub, caller, args[0:])
}

// RegisterOrg registers or updates an organization (a group user).
// Encrypts org keys with org public key.
// Encrypts org private key with org public key.
// Saves org data with org sym key.
// Creates org admin user.
// If makeCaller is true and this is a new org, caller will be added as an admin of the org.
//
// args = [ orgBytes, makeCallerAdmin ]
func RegisterOrg(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)
	if len(args) != 2 {
		custom_err := &custom_errors.LengthCheckingError{Type: "user_mgmt.RegisterOrg args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	// Parse org object from args[0]
	org := data_model.User{}
	orgBytes := []byte(args[0])
	err := json.Unmarshal(orgBytes, &org)
	if err != nil {
		custom_err := &custom_errors.UnmarshalError{Type: "User object for org"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	makeCallerAdmin := false
	if len(args) >= 2 {
		makeCallerAdmin, err = strconv.ParseBool(args[1])
		logger.Debugf("makeCallerAdmin: %v, %v", makeCallerAdmin, err)
		if err != nil {
			makeCallerAdmin = false
		}
	}

	return nil, RegisterOrgWithParams(stub, caller, org, makeCallerAdmin)
}

// RegisterOrgWithParams validates and creates/updates an org.
// The caller will be added as an admin of the org if this is a new org and makeCallerAdmin is true.
func RegisterOrgWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, org data_model.User, makeCallerAdmin bool) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	// only admin can create/update a new org
	if !caller.IsSystemAdmin() && caller.ID != org.ID {
		if isAdmin, _, _ := user_mgmt_c.IsUserAdminOfGroup(stub, caller.ID, org.ID); !isAdmin {
			logger.Errorf("Caller must be a admin")
			return errors.New("Caller must be a admin")
		}
	}
	return registerOrgInternal(stub, caller, org, makeCallerAdmin)
}

// registerOrgInternal registers an org without checking the caller's permission.
func registerOrgInternal(stub cached_stub.CachedStubInterface, caller data_model.User, org data_model.User, makeCallerAdmin bool) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	valid, err := validateOrg(stub, caller, org)
	if err != nil {
		logger.Errorf("Error validating org: %v", err)
		return errors.Wrap(err, "Error validating org")
	} else if !valid {
		logger.Errorf("Invalid org!")
		return errors.New("Invalid org!")

	}
	return registerUserInternal(stub, caller, org, makeCallerAdmin)
}

// validateOrg checks that the org is valid.
func validateOrg(stub cached_stub.CachedStubInterface, caller data_model.User, org data_model.User) (bool, error) {

	// org.IsGroup should be true
	if org.IsGroup != true {
		custom_err := &custom_errors.RegisterOrgInvalidFieldError{ID: org.ID, Field: "IsGroup"}
		logger.Errorf("%v: %v", custom_err.Error(), org.IsGroup)
		return false, errors.New(custom_err.Error())
	}
	// org.Role should be "org"
	if org.Role != global.ROLE_ORG {
		custom_err := &custom_errors.RegisterOrgInvalidFieldError{ID: org.ID, Field: "Role"}
		logger.Errorf("%v: %v", custom_err.Error(), org.Role)
		return false, errors.New(custom_err.Error())
	}

	return true, nil
}

// UpdateOrg updates an organization.
//
// args = [orgBytes]
func UpdateOrg(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return RegisterOrg(stub, caller, args)
}

// GetOrg returns an organization.
//
// args = [orgId]
func GetOrg(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return GetUser(stub, caller, args)
}

// GetOrgs returns a list of all organizations.
//
// args = []
func GetOrgs(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	orgs := []data_model.User{}

	// Use index to find all orgs
	am := asset_mgmt_i.GetAssetManager(stub, caller)
	iter, err := am.GetAssetIter(
		global.USER_ASSET_NAMESPACE,
		global.INDEX_USER,
		[]string{"is_group", "role"},
		[]string{"true", "org"},
		[]string{"true", "org"},
		true,
		false,
		[]string{caller.GetPubPrivKeyId()},
		"", -1, nil)
	if err != nil {
		logger.Errorf("GetAssets failed: %v", err)
		return nil, errors.Wrap(err, "GetAssets failed")
	}
	// Iterate over all orgs
	defer iter.Close()
	for iter.HasNext() {
		assetData, err := iter.Next()
		if err != nil {
			custom_err := &custom_errors.IterError{}
			logger.Errorf("%v: %v", custom_err, err)
			continue
		}
		org := ConvertFromAsset(assetData)
		orgs = append(orgs, org)
	}

	return json.Marshal(&orgs)
}

// GetUsers returns a list of all member users for a given orgId, optionally filtered by role.
//
// args = [orgId, role]
func GetUsers(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)
	if len(args) != 1 && len(args) != 2 {
		custom_err := &custom_errors.LengthCheckingError{Type: "user_mgmt.GetUsers args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.New(custom_err.Error())
	}

	orgId := args[0]
	role := ""
	if len(args) == 2 {
		role = args[1]
	}

	// TODO: This is inefficient. We are finding all org members, then manually filtering on role. I don't think there's a
	// better way given the current implementation, unfortunately.

	// Get all memberIds of this group, then search for each one and append to list
	memberIds, err := SlowGetGroupMemberIDs(stub, orgId)

	if err != nil {
		logger.Errorf("GetGroupMemberIDS returned error: %v", err)
		return nil, errors.WithStack(err)
	}
	userList := []data_model.User{}
	for _, userId := range memberIds {

		// Get the user
		user, err := GetUserData(stub, caller, userId, false, true)
		if len(user.ID) == 0 {
			msg := "Failed to get user with ID: " + userId
			logger.Errorf(msg)
			return nil, errors.Wrap(err, msg)
		}

		// Check that the roles match
		if len(role) > 0 && user.Role != role {
			continue
		}

		userList = append(userList, user)
	}

	return json.Marshal(&userList)
}

// GetUserIter returns an interator of user objects
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

	am := asset_mgmt_i.GetAssetManager(stub, caller)
	iter, err := am.GetAssetIter(
		global.USER_ASSET_NAMESPACE,
		global.INDEX_USER,
		[]string{"is_group", "role", "id"},
		startValues,
		endValues,
		decryptPrivateData,
		returnOnlyPrivateAssets,
		assetKeyPath,
		previousKey,
		limit,
		filterRule)
	if err != nil {
		logger.Errorf("GetUserIter failed: %v", err)
		return nil, errors.Wrap(err, "GetUserIter failed")
	}

	return iter, nil
}

// PutUserInOrg is a proxy function for PutUserInGroup.
// Call if you need to call PutUserInGroup directly from Invoke in a solution.
//
// args = [ userID, orgID, isAdmin]
func PutUserInOrg(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)
	if len(args) != 3 {
		custom_err := &custom_errors.LengthCheckingError{Type: "user_mgmt.PutUserInOrg args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	userID := args[0]
	orgID := args[1]
	isAdminStr := args[2]
	isAdmin := false
	if isAdminStr == "true" {
		isAdmin = true
	}

	if len(userID) == 0 {
		errMsg := "userID cannot be empty"
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}
	if len(orgID) == 0 {
		errMsg := "orgID cannot be empty"
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	return nil, PutUserInGroup(stub, caller, userID, orgID, isAdmin)
}
