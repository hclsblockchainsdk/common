/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package asset_mgmt_i is responsible for storing any type of asset on the ledger.
// It handles encryption/decryption as well as indexing.
package asset_mgmt_i

import (
	"common/bchcls/asset_mgmt/asset_key_func"
	"common/bchcls/asset_mgmt/asset_manager"
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/custom_errors"
	"common/bchcls/data_model"
	"common/bchcls/index"
	"common/bchcls/index/table_interface"
	"common/bchcls/internal/asset_mgmt_i/asset_mgmt_c"
	"common/bchcls/internal/asset_mgmt_i/asset_mgmt_c/asset_mgmt_g"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/consent_mgmt_i/consent_mgmt_c"
	"common/bchcls/internal/datastore_i/datastore_c"
	"common/bchcls/internal/datatype_i"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/internal/user_mgmt_i/user_mgmt_c"
	"common/bchcls/simple_rule"
	"common/bchcls/test_utils"
	"common/bchcls/utils"

	"bytes"
	"encoding/json"
	"fmt"

	"reflect"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("asset_mgmt_i")

// assetManagerImpl is the default implementation of the AssetManager interface.
type assetManagerImpl struct {
	stub   cached_stub.CachedStubInterface
	caller data_model.User
}

// assetIter is used for iterating over assets.
type assetIter struct {
	LedgerIter              shim.StateQueryIteratorInterface
	IndexTable              table_interface.Table
	AssetManager            asset_manager.AssetManager
	AssetNamespace          string
	DecryptPrivateData      bool
	ReturnPrivateAssetsOnly bool
	AssetKeyPath            interface{}
	PreviousLedgerKey       string
	Limit                   int
	FilterRule              *simple_rule.Rule
	count                   int
	nextAsset               *data_model.Asset
	closed                  bool
}

// if defaultDatastoreConnectionID is set, it will be used when DatastoreConnectionID is not set in asset's metadata.
// if this var is not set, it will not use datastore when DatastoreConnectionID is not set in asset's metadata.
var defaultDatastoreConnectionID = ""

// ------------------------------------------------------
// ---------------------- INIT FUNCTIONS ----------------
// ------------------------------------------------------

// Init sets up the asset_mgmt package.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}

	return nil, nil
}

// ------------------------------------------------------
// ----------------- TOP-LEVEL FUNCTIONS ----------------
// ------------------------------------------------------

// GetAssetManager constructs and returns an assetManagerImpl instance.
func GetAssetManager(stub cached_stub.CachedStubInterface, caller data_model.User) asset_manager.AssetManager {
	return assetManagerImpl{stub: stub, caller: caller}
}

// CreateAssetId returns the assetId given the object's type and unique identifier.
// This fuction calls data_model.GetAssetId().
func GetAssetId(assetNamespace string, id string) string {
	return asset_mgmt_c.GetAssetId(assetNamespace, id)
}

// GetAssetKeyId returns the asset key id for the given assetId.
// Returns an error if asset with assetId does not exist.
func GetAssetKeyId(stub cached_stub.CachedStubInterface, assetId string) (string, error) {
	asset, err := getAssetByKey(stub, assetId, nil)
	if err != nil {
		return "", err
	}
	if len(asset.AssetKeyId) == 0 {
		return "", errors.New("Asset doesn't exist")
	}
	return asset.AssetKeyId, nil
}

// GetAssetKey returns decrypted asset key given a key path.
// Caller must supply the key path.
// If keyPath is valid, it returns decrypted asset key.
// If keyPath is invalid, it returns nil for asset key and the error.
func GetAssetKey(stub cached_stub.CachedStubInterface, assetId string, keyPath []string, startKey []byte) ([]byte, error) {
	// verify keyPath
	valid, err := key_mgmt_i.VerifyAccessPath(stub, keyPath)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, errors.New("key path is not valid")
	}

	// verify that the target key is correct asset key for the assetId
	assetKeyId, err := GetAssetKeyId(stub, assetId)
	if assetKeyId != keyPath[len(keyPath)-1] {
		logger.Debugf("target key is not asset key of the asset: %v", assetId)
		return nil, errors.New("target key is not asset key of the asset " + assetId)
	}

	// try to get key
	return key_mgmt_i.GetKey(stub, keyPath, startKey)
}

// GetAssetKeyFromInterface returns asset key bytes.
// It is the solution caller's responsibility to verify the key.
// args:
//     []bytes 			--> asset key bytes
//     data_model.Key 	--> asset key
//     []string, []byte --> asset key path and start key byte
//     asset_key_func.AssetKeyPathFunc(), data_model.User, data_model.Asset, []byte --> asset key path function, caller object, asset object, and start key byte
// Returns asset key byte []byte.
// Returns an error if assetKey is the wrong type or failed to retrieve the asset.
func GetAssetKeyFromInterface(stub cached_stub.CachedStubInterface, args ...interface{}) ([]byte, error) {
	// parse args
	if len(args) == 1 {
		switch arg := args[0].(type) {
		case []byte:
			return arg, nil
		case data_model.Key:
			return arg.KeyBytes, nil
		default:
			logger.Errorf("Invalid assetkey %v", args)
			return nil, errors.New("Invalid assetKey")
		}
	} else if len(args) == 2 {
		if keyPath, ok := args[0].([]string); ok {
			if startKey, ok := args[1].([]byte); ok {
				return key_mgmt_i.GetKey(stub, keyPath, startKey)
			}
		}
	} else if len(args) == 4 {
		if keyFunc, ok := args[0].(asset_key_func.AssetKeyPathFunc); ok {
			if caller, ok := args[1].(data_model.User); ok {
				if asset, ok := args[2].(data_model.Asset); ok {
					if startKey, ok := args[3].([]byte); ok {
						keyPath, err := keyFunc(stub, caller, asset)
						if err != nil {
							logger.Errorf("Failed to run asset key path function: %v", err)
							return nil, errors.Wrap(err, "Failed to run asset key path function")
						}
						return key_mgmt_i.GetKey(stub, keyPath, startKey)
					}
				}
			}
		}
	}
	logger.Errorf("Invalid args")
	return nil, errors.New("Invalid args")
}

// GetEncryptedAssetData returns an asset object with encrypted PrivateData.
// If the AssetId passed in does not exist, it returns an empty data_model.Asset object.
// It is the caller's responsibility to check if the return object is empty.
func GetEncryptedAssetData(stub cached_stub.CachedStubInterface, assetId string) (data_model.Asset, error) {

	// check cache with assetId
	assetCache, err := getEncryptedAssetFromCache(stub, assetId)
	if err == nil && assetCache != nil {
		return *assetCache, nil
	}

	assetData := data_model.Asset{}
	// get assetLedgerKey using assetId
	assetLedgerKey := assetId

	// get assetData using assetLedgerKey
	assetBytes, err := stub.GetState(assetLedgerKey)
	if err != nil {
		custom_err := &custom_errors.GetLedgerError{LedgerKey: assetLedgerKey, LedgerItem: "assetBytes"}
		logger.Errorf("%v: %v", custom_err, err)
		return assetData, errors.Wrap(err, custom_err.Error())
	}

	if assetBytes == nil {
		logger.Debugf("Asset not found with ledger key: \"%v\"", assetLedgerKey)
		return assetData, nil
	}

	err = json.Unmarshal(assetBytes, &assetData)
	if err != nil {
		custom_err := &custom_errors.UnmarshalError{Type: "assetData"}
		logger.Errorf("%v: %v", custom_err, err)
		return assetData, errors.Wrap(err, custom_err.Error())
	}
	//save to cache
	putEncryptedAssetToCache(stub, assetData)
	// return asset data
	return assetData, nil
}

// GetAssetPrivateData returns an asset's private data bytes.
// Caller should provide proper assetKey for decryption.
// If decryption is successful, it returns decrypted private data bytes.
// If assetKey is nil, it returns encrypted private data bytes and nil for error.
// If assetKey is invalid, it returns encrypted private data bytes and an error.
func GetAssetPrivateData(stub cached_stub.CachedStubInterface, assetData data_model.Asset, assetKey []byte) ([]byte, error) {
	return getPrivateData(stub, assetData, assetKey)
}

// ------------------------------------------------------
// ------------- assetManagerImpl FUNCTIONS -------------
// ------------------------------------------------------

// GetStub documentation can be found in asset_mgmt_interfaces.go
func (assetManager assetManagerImpl) GetStub() cached_stub.CachedStubInterface {
	return assetManager.stub
}

// GetCaller documentation can be found in asset_mgmt_interfaces.go
func (assetManager assetManagerImpl) GetCaller() data_model.User {
	return assetManager.caller
}

// AddAsset documentation can be found in asset_mgmt_interfaces.go
func (assetManager assetManagerImpl) AddAsset(asset data_model.Asset, assetKey data_model.Key, giveAccessToCaller bool) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("assetId: \"%v\", assetKeyId: \"%v\", giveAccessToCaller \"%v\"", asset.AssetId, assetKey.ID, giveAccessToCaller)

	if !IsValidAssetId(asset.AssetId) {
		errMsg := "Invalid AssetID: Use asset_mgmt.GetAssetId to generate AssetID"
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	// if ownerId is not defined, caller becomes owner
	if len(asset.OwnerIds) == 0 {
		asset.OwnerIds = []string{assetManager.caller.ID}
	}
	// only single owner is allowed
	if len(asset.OwnerIds) > 1 {
		asset.OwnerIds = []string{asset.OwnerIds[0]}
	}
	// fill missing assetKeyId
	if len(asset.AssetKeyId) == 0 {
		asset.AssetKeyId = assetKey.ID
	}

	// check for write access
	hasWriteAccess, err := hasUserWriteAccessToAsset(assetManager.stub, assetManager.caller, asset, true, true)
	if !hasWriteAccess {
		logger.Errorf("Caller %v does not have write access to asset %v", assetManager.caller.ID, asset.AssetId)
		return errors.New("Caller does not have write access to the asset")
	}

	// giveAccess to Caller
	var privateKey []byte = nil
	var publicKey []byte = nil
	if giveAccessToCaller {
		privateKey, err = crypto.DecodeStringB64(assetManager.caller.PrivateKeyB64)
		if err != nil || privateKey == nil {
			custom_err := &custom_errors.InvalidPrivateKeyError{}
			logger.Errorf("%v: %v", custom_err, err)
			return errors.WithStack(custom_err)
		}
		publicKey, err = crypto.DecodeStringB64(assetManager.caller.PublicKeyB64)
		if err != nil || publicKey == nil {
			custom_err := &custom_errors.InvalidPublicKeyError{}
			logger.Errorf("%v: %v", custom_err, err)
			return errors.WithStack(custom_err)
		}
	}

	// validation done in putAssetByKey:
	// asset key id and keybyte are validated
	// asset should not already exist

	return putAssetByKey(
		assetManager.stub,
		assetManager.caller,
		asset,
		assetKey.ID,
		assetKey.KeyBytes,
		assetManager.caller.GetPubPrivKeyId(),
		privateKey,
		publicKey,
		false)

}

// UpdateAsset documentation can be found in asset_mgmt_interfaces.go
func (assetManager assetManagerImpl) UpdateAsset(asset data_model.Asset, assetKey data_model.Key, strictUpdate ...bool) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("assetId: \"%v\", assetKeyId: \"%v\", strictUpdate \"%v\"", asset.AssetId, assetKey.ID, strictUpdate)

	if !IsValidAssetId(asset.AssetId) {
		errMsg := "Invalid AssetID: Use asset_mgmt.GetAssetId to generate AssetID"
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	// if ownerId is not defined, caller becomes owner
	if len(asset.OwnerIds) == 0 {
		asset.OwnerIds = []string{assetManager.caller.ID}
	}
	// only single owner is allowed
	if len(asset.OwnerIds) > 1 {
		asset.OwnerIds = []string{asset.OwnerIds[0]}
	}
	// fill missing assetKeyId
	if len(asset.AssetKeyId) == 0 {
		asset.AssetKeyId = assetKey.ID
	}

	// check for write access
	hasWriteAccess, _ := hasUserWriteAccessToAsset(assetManager.stub, assetManager.caller, asset, true, true)
	if !hasWriteAccess {
		logger.Errorf("Caller %v does not have write access to asset %v", assetManager.caller.ID, asset.AssetId)
		return errors.New("Caller does not have write access to the asset")
	}

	// validation done in putAssetByKey:
	// asset key id and keybyte are validated
	// asset should not already exist

	//do not pass optional caller key, since you should already have access
	//to the assert in order to update it
	if len(strictUpdate) != 0 && len(strictUpdate) != 1 {
		errMsg := "Invalid strictUpdate: it should be a boolean value"
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}
	if len(strictUpdate) == 1 && !strictUpdate[0] {
		return putAssetByKey(
			assetManager.stub,
			assetManager.caller,
			asset,
			assetKey.ID,
			assetKey.KeyBytes,
			"",
			nil,
			nil)

	}

	return putAssetByKey(
		assetManager.stub,
		assetManager.caller,
		asset,
		assetKey.ID,
		assetKey.KeyBytes,
		"",
		nil,
		nil,
		true)
}

// DeleteAsset documentation can be found in asset_mgmt_interfaces.go
func (assetManager assetManagerImpl) DeleteAsset(assetId string, assetKey data_model.Key) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("assetId: \"%v\", assetKeyId: \"%v\"", assetId, assetKey.ID)
	if !IsValidAssetId(assetId) {
		errMsg := "Invalid AssetID: Use asset_mgmt.GetAssetId to generate AssetID"
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	// find existing asset from ledger
	assetData, err := GetEncryptedAssetData(assetManager.stub, assetId)
	if err != nil {
		custom_err := &custom_errors.GetAssetDataError{AssetId: assetId}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}
	if len(assetData.AssetKeyId) == 0 || assetData.AssetKeyId != assetKey.ID {
		custom_err := &custom_errors.GetAssetDataError{AssetId: assetId}
		logger.Errorf("%v", custom_err)
		return errors.WithStack(custom_err)
	}

	// verify asset key by hash
	assetKeyHash := crypto.Hash(assetKey.KeyBytes)
	if !bytes.Equal(assetData.AssetKeyHash, assetKeyHash) {
		logger.Error("Invalid Asset Key: Hash does not match")
		return errors.New("Invalid Asset Key: Hash does not match")
	}

	// check for write access
	hasWriteAccess, err := hasUserWriteAccessToAsset(assetManager.stub, assetManager.caller, assetData, true, true)
	if !hasWriteAccess {
		logger.Errorf("Caller %v does not have write access to asset %v", assetManager.caller.ID, assetId)
		return errors.New("Caller does not have write access to the asset")
	}

	// delete asset
	err = assetManager.stub.DelState(assetId)
	if err != nil {
		custom_err := &custom_errors.DeleteLedgerError{LedgerKey: assetId}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	// delete asset's index values
	if len(assetData.IndexTableName) > 0 {
		table := index.GetTable(assetManager.stub, assetData.IndexTableName)
		table.DeleteRow(assetId)
	}

	return nil
}

// GetAsset documentation can be found in asset_mgmt_interfaces.go
func (assetManager assetManagerImpl) GetAsset(assetId string, assetKey data_model.Key) (*data_model.Asset, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("assetId: \"%v\", assetKeyId: \"%v\"", assetId, assetKey.ID)
	if !IsValidAssetId(assetId) {
		errMsg := "Invalid AssetID: Use asset_mgmt.GetAssetId to generate AssetID"
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}
	return getAssetByKey(assetManager.stub, assetId, assetKey.KeyBytes)
}

// GetAssetSymKey documentation can be found in asset_mgmt_interfaces.go
func (assetManager assetManagerImpl) GetAssetKey(assetId string, keyPath []string) (data_model.Key, error) {
	if len(keyPath) == 0 {
		logger.Warning("Empty keyPath: keyPath is required")
		//return data_model.Key{}, errors.New("Empty keyPath: keyPath is required")
		assetKeyId, err := GetAssetKeyId(assetManager.stub, assetId)
		if err != nil {
			logger.Errorf("Failed to get assetKeyId: %v", err)
			return data_model.Key{}, errors.Wrap(err, "Failed to get assetKeyId")
		}
		keyPath = []string{assetManager.caller.GetPubPrivKeyId(), assetKeyId}
	}
	assetKeyId := keyPath[len(keyPath)-1]

	startKey := []byte{}
	var err error
	if keyPath[0] == assetManager.caller.GetPubPrivKeyId() {
		// retrieve caller's private key
		startKey, err = crypto.DecodeStringB64(assetManager.caller.PrivateKeyB64)
		if err != nil || startKey == nil {
			custom_err := &custom_errors.InvalidPrivateKeyError{}
			logger.Errorf("%v: %v", custom_err, err)
			return data_model.Key{}, errors.WithStack(custom_err)
		}
	} else if keyPath[0] == assetManager.caller.GetSymKeyId() {
		// retrieve caller's sym key
		startKey = assetManager.caller.SymKey
	} else {
		logger.Error("First key is not caller's key")
		return data_model.Key{}, errors.New("First key is not caller's key")
	}

	assetKeyBytes := []byte{}
	if len(keyPath) == 1 {
		assetKeyBytes = startKey
	} else {
		// convert asset key bytes to data_modle.Key object
		assetKeyBytes, err = GetAssetKey(assetManager.stub, assetId, keyPath, startKey)
		if err != nil {
			logger.Errorf("Failed to get asset key:%v %v", keyPath, err)
			return data_model.Key{}, errors.Wrap(err, "Failed to get asset key")
		}
	}
	assetKey, err := key_mgmt_i.ConvertKeyBytesToKey(assetKeyId, assetKeyBytes)
	return *assetKey, err
}

// AddAccessToAsset documentation can be found in asset_mgmt_interfaces.go.
func (assetManager assetManagerImpl) AddAccessToAsset(accessControl data_model.AccessControl, allowAddAccessBeforeAssetIsCreated ...bool) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("userId: %v, assetId: %v, access: %v", accessControl.UserId, accessControl.AssetId, accessControl.Access)
	assetExist := true
	allowAdd := false
	if len(allowAddAccessBeforeAssetIsCreated) > 0 {
		allowAdd = allowAddAccessBeforeAssetIsCreated[0]
	}

	// caller same as user
	if assetManager.caller.ID == accessControl.UserId {
		logger.Error("Caller can't add access for self")
		return errors.New("Caller can't add access for self")
	}

	if !accessControl.IsValid() {
		logger.Error("Invalid accessControl")
		return errors.New("Invalid accessControl")
	}

	// get asset
	asset, err := getAssetByKey(assetManager.stub, accessControl.AssetId, nil)
	if err != nil {
		logger.Errorf("Failed to get asset: %v", err)
		return errors.Wrap(err, "Failed to get asset")
	}
	if utils.IsStringEmpty(asset.AssetId) {
		assetExist = false
	}
	if !assetExist && !allowAdd {
		// asset doesn't exist
		err := errors.WithStack(&custom_errors.GetAssetDataError{AssetId: accessControl.AssetId})
		logger.Error(err)
		return err
	}

	// only owner can add access
	// or if allowAddAccessBeforeAssetIsCreated is true, don't need to check.
	// In that case, assume caller will be the owner of the asset
	if !asset.IsOwner(assetManager.caller.ID) && !allowAdd {
		logger.Errorf("Caller %v does not have write access to asset %v", assetManager.caller.ID, asset.AssetId)
		return errors.New("Caller does not have write access to the asset")
	}

	// edge data: set AccessType
	edgeData := make(map[string]string)
	edgeData[global.EDGEDATA_ACCESS_TYPE] = accessControl.Access
	if accessControl.Access == global.ACCESS_READ_ONLY {
		edgeData[global.EDGEDATA_ACCESS_TYPE] = global.ACCESS_READ
	}

	// get start key
	if accessControl.UserKey == nil || accessControl.UserKey.IsEmpty() {
		startKey := data_model.Key{}
		if accessControl.UserId == assetManager.caller.ID {
			startKey = assetManager.caller.GetPrivateKey()
		} else {
			// start key is public key of user
			startKey, err = getUserPublicKey(assetManager.stub, accessControl.UserId)
			if err != nil || utils.IsStringEmpty(startKey.ID) {
				logger.Errorf("Failed to get public key of user \"%v\": %v", accessControl.UserId, err)
				return errors.Errorf("Failed to get public key of user \"%v\"", accessControl.UserId)
			}
		}
		accessControl.UserKey = &startKey
	}

	// get target key
	if accessControl.AssetKey == nil || accessControl.AssetKey.IsEmpty() {
		assetKey, err := assetManager.GetAssetKey(accessControl.AssetId, []string{})
		if err != nil || assetKey.IsEmpty() {
			logger.Errorf("Failed to get assetKey: %v", err)
			return errors.Wrap(err, "Failed to get assetKey")
		}
		accessControl.AssetKey = &assetKey
	}

	// write only access
	if accessControl.Access == global.ACCESS_WRITE_ONLY {
		// revoke read access first
		accessControl.Access = global.ACCESS_READ
		err := key_mgmt_i.RevokeAccess(assetManager.stub, accessControl.UserKey.ID, accessControl.AssetKey.ID)
		if err != nil {
			logger.Errorf("Failed to revoke read access before adding write only access: %v", err)
			return errors.Wrap(err, "Failed to revoke read access before adding write only access")
		}
		newKey := data_model.Key{}
		newKey.ID = key_mgmt_i.GetKeyIdForWriteOnlyAccess(accessControl.AssetId, accessControl.AssetKey.ID, assetManager.caller.ID)
		newKey.KeyBytes = accessControl.AssetKey.KeyBytes
		newKey.Type = accessControl.AssetKey.Type
		accessControl.Access = global.ACCESS_WRITE_ONLY
		accessControl.AssetKey = &newKey
	} else {
		// remove write only access
		targetKeyID := key_mgmt_i.GetKeyIdForWriteOnlyAccess(accessControl.AssetId, accessControl.AssetKey.ID, assetManager.caller.ID)
		err := key_mgmt_i.RevokeAccess(assetManager.stub, accessControl.UserKey.ID, targetKeyID)
		if err != nil {
			logger.Errorf("Failed to revoke write only access: %v", err)
			return errors.Wrap(err, "Failed to revoke write only access")
		}
	}

	// Add access
	return key_mgmt_i.AddAccess(assetManager.stub, *accessControl.UserKey, *accessControl.AssetKey, edgeData)
}

// RemoveAccessFromAsset documentation can be found in asset_mgmt_interfaces.go.
func (assetManager assetManagerImpl) RemoveAccessFromAsset(accessControl data_model.AccessControl) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("userId: %v, assetId: %v, access: %v", accessControl.UserId, accessControl.AssetId, accessControl.Access)

	// caller same as user
	if assetManager.caller.ID == accessControl.UserId {
		logger.Error("Caller can't remove access for self")
		return errors.New("Caller can't remove access for self")
	}

	// get asset
	asset, err := getAssetByKey(assetManager.stub, accessControl.AssetId, nil)
	if err != nil {
		logger.Errorf("Failed to get asset: %v", err)
		return errors.Wrap(err, "Failed to get asset")
	}

	// only owner can remove access
	if !asset.IsOwner(assetManager.caller.ID) {
		logger.Errorf("Caller %v is not owner of asset %v", assetManager.caller.ID, asset.AssetId)
		return errors.New("Caller is not owner of asset")
	}

	// get start key ID
	startKeyID := ""
	if accessControl.UserKey != nil && len(accessControl.UserKey.ID) > 0 {
		startKeyID = accessControl.UserKey.ID
	} else {
		// start key is public key of user
		user := data_model.User{ID: accessControl.UserId}
		startKeyID = user.GetPubPrivKeyId()
	}

	// get target key ID
	targetKeyID := ""
	if accessControl.AssetKey != nil && len(accessControl.AssetKey.ID) > 0 {
		targetKeyID = accessControl.AssetKey.ID
	} else {
		targetKeyID, err = GetAssetKeyId(assetManager.stub, accessControl.AssetId)
		if err != nil {
			logger.Errorf("Failed to get assetKeyId: %v", err)
			return errors.Wrap(err, "Failed to get assetKeyId")
		}
	}

	// remove write only access edge if removing write only access
	if accessControl.Access == global.ACCESS_WRITE_ONLY {
		targetKeyID2 := key_mgmt_i.GetKeyIdForWriteOnlyAccess(accessControl.AssetId, targetKeyID, assetManager.caller.ID)
		return key_mgmt_i.RevokeAccess(assetManager.stub, startKeyID, targetKeyID2)
	}

	// delete access edge if removing read access
	if accessControl.Access == global.ACCESS_READ {
		return key_mgmt_i.RevokeAccess(assetManager.stub, startKeyID, targetKeyID)
	}

	// changing edge data access type to read access
	edgeValue, edgeData, err := key_mgmt_i.GetAccessEdge(assetManager.stub, startKeyID, targetKeyID)
	if err != nil || len(edgeValue) == 0 {
		logger.Errorf("Failed to get access edge: %v", err)
		return errors.Wrap(err, "Failed to get access edge")
	}

	edgeData[global.EDGEDATA_ACCESS_TYPE] = global.ACCESS_READ
	return key_mgmt_i.UpdateAccessEdge(assetManager.stub, startKeyID, targetKeyID, edgeValue, edgeData)
}

// CheckAccessToAsset documentation can be found in asset_mgmt_interfaces.go
func (assetManager assetManagerImpl) CheckAccessToAsset(accessControl data_model.AccessControl) (bool, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	if !accessControl.IsValid() {
		logger.Error("Invalid accessControl")
		return false, errors.New("Invalid accessControl")
	}

	// Get the asset
	asset, err := GetEncryptedAssetData(assetManager.stub, accessControl.AssetId)
	if err != nil {
		logger.Errorf("GetAsset failed: %v", err)
		return false, errors.Wrap(err, "GetAsset failed")
	}

	if !asset.IsOwner(assetManager.caller.ID) && accessControl.UserId != assetManager.caller.ID {
		logger.Error("You can only check your own permissions if you are not owner of the asset")
		return false, errors.New("You can only check your own permissions if you are not owner of the asset")
	}

	// Check asset key id
	if accessControl.AssetKey != nil && len(accessControl.AssetKey.ID) > 0 && accessControl.AssetKey.ID != asset.AssetKeyId {
		logger.Errorf("Invalid asset key id: %v", accessControl.AssetKey.ID)
		return false, errors.New("Invalid asset key id")
	}

	// Get user
	user := data_model.User{ID: accessControl.UserId}
	if user.ID == assetManager.caller.ID {
		user = assetManager.caller
	}

	// check access
	if accessControl.Access == global.ACCESS_WRITE {
		return hasUserWriteAccessToAsset(assetManager.stub, user, asset, true, true)
	} else if accessControl.Access == global.ACCESS_WRITE_ONLY {
		return hasUserWriteOnlyAccessToAsset(assetManager.stub, user, asset, true, true)
	} else if accessControl.Access == global.ACCESS_READ {
		return hasUserReadAccessToAsset(assetManager.stub, user, asset, true, true)
	} else if accessControl.Access == global.ACCESS_READ_ONLY {
		return hasUserReadOnlyAccessToAsset(assetManager.stub, user, asset, true, true)
	}
	return false, nil
}

// GetAssetIter documentation can be found in asset_mgmt_interfaces.go
func (assetManager assetManagerImpl) GetAssetIter(
	assetNamespace string,
	indexTableName string,
	fieldNames []string,
	startValues []string,
	endValues []string,
	decryptPrivateData bool,
	returnPrivateAssetsOnly bool,
	assetKeyPath interface{},
	previousKey string,
	limit int,
	filterRule *simple_rule.Rule,
) (asset_manager.AssetIteratorInterface, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	// Get the index table
	indexTable := index.GetTable(assetManager.GetStub(), indexTableName)

	// --------------------------------------------
	// Create the startKey
	// --------------------------------------------
	var startKey string
	if utils.IsStringEmpty(previousKey) {
		// If we weren't given a previousKey, create a startKey using the startValues (e.g. "Index-vehicleTable-color-id~blue~")
		var err error
		startKey, err = indexTable.CreateRangeKey(fieldNames, startValues)
		if err != nil {
			err = errors.Wrapf(err, "Failed to create startKey")
			logger.Error(err)
			return &assetIter{}, err
		}
	} else {
		// Append the first unicode character to previousKey so that we don't return that asset again
		startKey = previousKey + string(global.MIN_UNICODE_RUNE_VALUE)
	}

	// --------------------------------------------
	// Create the endKey
	// --------------------------------------------
	endKey, err := indexTable.CreateRangeKey(fieldNames, endValues)
	if err != nil {
		err = errors.Wrapf(err, "Failed to create endKey")
		logger.Error(err)
		return &assetIter{}, err
	}

	// There are 2 cases in which we need to append a wildcard to the endKey:
	// 1) Partial-key queries (startValues & endValues are identical) - e.g. range("index_blue_", "index_blue_*")
	// 2) Open-ended range queries (startValues has more entries than endValues) - e.g. range("index_blue_", "index_*")
	if len(startValues) > len(endValues) || reflect.DeepEqual(startValues, endValues) {
		endKey = endKey + string(global.MAX_UNICODE_RUNE_VALUE)
	}

	logger.Debugf("GetAssetIter previousKey: %v", index.GetPrettyLedgerKey(previousKey))
	logger.Debugf("GetAssetIter startKey: %v", index.GetPrettyLedgerKey(startKey))
	logger.Debugf("GetAssetIter endKey: %v", index.GetPrettyLedgerKey(endKey))

	// --------------------------------------------
	// Query the index by range
	// --------------------------------------------
	// The iterator can be used to iterate over all keys between the startKey (inclusive) and endKey (exclusive).
	iter, err := indexTable.GetRowsByRange(startKey, endKey)
	if err != nil {
		err = errors.Wrapf(err, "Failed to GetRowsByRange")
		logger.Error(err)
		return &assetIter{}, err
	}

	// --------------------------------------------
	// Return an asset iterator
	// --------------------------------------------
	returnIter := assetIter{
		LedgerIter:              iter,
		IndexTable:              indexTable,
		AssetManager:            assetManager,
		AssetNamespace:          assetNamespace,
		DecryptPrivateData:      decryptPrivateData,
		ReturnPrivateAssetsOnly: returnPrivateAssetsOnly,
		AssetKeyPath:            assetKeyPath,
		PreviousLedgerKey:       previousKey,
		Limit:                   limit,
		FilterRule:              filterRule,
		count:                   0}

	if limit != -1 && limit <= 0 {
		returnIter.Close()
	}
	return &returnIter, nil
}

// ------------------------------------------------------
// ----------------- assetIter FUNCTIONS ----------------
// ------------------------------------------------------

// HasNext documentation can be found in asset_mgmt_interfaces.go
func (assetIter *assetIter) HasNext() bool {
	if !assetIter.closed && assetIter.LedgerIter != nil && assetIter.LedgerIter.HasNext() && assetIter.count == 0 {
		//this very first time, call getNext()
		nextAsset, err := assetIter.getNext()
		if err != nil {
			assetIter.Close()
		} else {
			assetIter.nextAsset = nextAsset
		}
	}

	if assetIter.nextAsset == nil {
		return false
	} else {
		return true
	}
}

// getNext finds next item from ledgerIter
func (assetIter *assetIter) getNext() (*data_model.Asset, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	if assetIter.closed || !assetIter.LedgerIter.HasNext() || (assetIter.Limit != -1 && assetIter.count >= assetIter.Limit) {
		assetIter.Close()
		assetIter.nextAsset = nil
		return nil, errors.New("Iter closed")
	}

	for !assetIter.closed && assetIter.LedgerIter.HasNext() && (assetIter.Limit == -1 || assetIter.count < assetIter.Limit) {
		// Get the next val, which is a row in the index table
		KV, err := assetIter.LedgerIter.Next()
		if err != nil {
			logger.Errorf("Failed to get iter.Next(): %v\n", err)
			assetIter.Close()
			return nil, errors.WithStack(err)
		}
		rowBytes := KV.GetValue()
		var row map[string]string
		err = json.Unmarshal(rowBytes, &row)
		if err != nil {
			custom_err := &custom_errors.UnmarshalError{Type: "Index Row"}
			logger.Errorf("%v: %v", custom_err.Error(), err)
			assetIter.Close()
			return nil, errors.Wrap(err, custom_err.Error())
		}

		// AssetId is the value of the primary key
		assetId := GetAssetId(assetIter.AssetNamespace, row[assetIter.IndexTable.GetPrimaryKeyId()])

		// Save the previous ledger key (used for paging)
		assetIter.PreviousLedgerKey = KV.GetKey()

		// get encrypted asset data
		assetData, err := GetEncryptedAssetData(assetIter.AssetManager.GetStub(), assetId)
		if err != nil {
			logger.Errorf("Failed to get asset data: %v", err)
			assetIter.Close()
			return nil, err
		}

		logger.Debugf("got asset data encrypted; %v", assetData)

		var assetKey data_model.Key
		assetKey.ID = assetId
		assetKey.Type = global.KEY_TYPE_SYM
		// need to get keys if decryptPrivateData or returnPrivateAssetsOnly is true
		if assetIter.DecryptPrivateData || assetIter.ReturnPrivateAssetsOnly {
			if keyId, ok := assetIter.AssetKeyPath.(string); ok {
				keyPath := []string{keyId}
				if len(keyPath) == 0 || keyPath[len(keyPath)-1] != assetData.AssetKeyId {
					keyPath = append(keyPath, assetData.AssetKeyId)
				}
				assetKey, err = assetIter.AssetManager.GetAssetKey(assetId, keyPath)
			} else if keyPath, ok := assetIter.AssetKeyPath.([]string); ok {
				// append assetKeyId if the last last item on the keyPath != assetKeyId
				if len(keyPath) == 0 || keyPath[len(keyPath)-1] != assetData.AssetKeyId {
					keyPath = append(keyPath, assetData.AssetKeyId)
				}
				assetKey, err = assetIter.AssetManager.GetAssetKey(assetId, keyPath)
			} else if keyFunc, ok := assetIter.AssetKeyPath.(asset_key_func.AssetKeyPathFunc); ok {
				keyPath, _ = keyFunc(assetIter.AssetManager.GetStub(), assetIter.AssetManager.GetCaller(), assetData)
				assetKey, _ = assetIter.AssetManager.GetAssetKey(assetId, keyPath)
			} else if keyFunc, ok := assetIter.AssetKeyPath.(asset_key_func.AssetKeyByteFunc); ok {
				assetKey.KeyBytes, _ = keyFunc(assetIter.AssetManager.GetStub(), assetIter.AssetManager.GetCaller(), assetData)
			} else {
				logger.Debugf("Unknown assetKeyPath: %v", assetIter.AssetKeyPath)
			}
		}

		// check if it's private asset or not
		if assetIter.ReturnPrivateAssetsOnly {
			if assetKey.ID != assetData.AssetKeyId || assetKey.IsEmpty() {
				//not a private asset
				continue
			}
		}

		// set assetKey to empty if decryption is not needed
		if !assetIter.DecryptPrivateData {
			assetKey = data_model.Key{}
		}

		// decrypt private data
		// it will set to encryptedData if assetKey is empty
		assetData.PrivateData, err = getPrivateData(assetIter.AssetManager.GetStub(), assetData, assetKey.KeyBytes)
		if err != nil {
			logger.Errorf("Failed to get asset data: %v", err)
			assetIter.Close()
			return nil, err
		}

		// apply filter rule
		if assetIter.FilterRule != nil {
			assetJsonBytes, err := json.Marshal(assetData)
			if err != nil {
				custom_err := &custom_errors.MarshalError{Type: "data_model.Asset"}
				logger.Errorf("%v: %v", custom_err, err)
				assetIter.Close()
				return nil, errors.Wrap(err, custom_err.Error())
			}
			assetJson := make(map[string]interface{})
			err = json.Unmarshal(assetJsonBytes, &assetJson)
			if err != nil {
				custom_err := &custom_errors.UnmarshalError{Type: "data_model.Asset"}
				logger.Errorf("%v: %v", custom_err, err)
				assetIter.Close()
				return nil, errors.Wrap(err, custom_err.Error())
			}

			// parse public data
			publicDataMap := make(map[string]interface{})
			err = json.Unmarshal(assetData.PublicData, &publicDataMap)
			if err != nil {
				custom_err := &custom_errors.UnmarshalError{Type: "asset.PublicData"}
				logger.Errorf("%v: %v", custom_err, err)
				assetIter.Close()
				return nil, errors.Wrap(err, custom_err.Error())
			}
			assetJson["public_data"] = publicDataMap

			// parse private data
			privateDataMap := make(map[string]interface{})
			err = json.Unmarshal(assetData.PrivateData, &privateDataMap)
			if err != nil {
				custom_err := &custom_errors.UnmarshalError{Type: "asset.PrivateData"}
				logger.Errorf("%v: %v", custom_err, err)
				assetIter.Close()
				return nil, errors.Wrap(err, custom_err.Error())
			}
			assetJson["private_data"] = privateDataMap

			// apply rule
			result, err := assetIter.FilterRule.Apply(assetJson)
			if err != nil {
				logger.Errorf("Failed to apply rule: %v", err)
				assetIter.Close()
				return nil, errors.Wrap(err, "Failed to apply filter rule")
			}
			if result["$result"] != simple_rule.D(true) {
				logger.Debugf("Filtered out by the filter rule; %v, %v", assetData.AssetId, assetIter.FilterRule.GetExprJSON(), err)
				continue
			}
		}

		// count for pagination
		assetIter.count = assetIter.count + 1

		// return assetData
		return &assetData, nil
	}

	assetIter.Close()
	return nil, errors.New("No more item to iterate")
}

// Next documentation can be found in asset_mgmt_interfaces.go
func (assetIter *assetIter) Next() (*data_model.Asset, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	// check if this is very very first time doing this
	if !assetIter.closed && assetIter.LedgerIter.HasNext() && assetIter.count == 0 {
		//this very first time, call getNext()
		nextAsset, err := assetIter.getNext()
		if err != nil {
			assetIter.Close()
		} else {
			assetIter.nextAsset = nextAsset
		}
	}
	// now just return nextAsset and run getNext() again
	if assetIter.nextAsset != nil {
		toReturn := assetIter.nextAsset
		assetIter.nextAsset = nil
		assetIter.nextAsset, _ = assetIter.getNext()
		return toReturn, nil
	} else {
		return nil, errors.New("you can't call Next()")
	}
}

// Close documentation can be found in asset_mgmt_interfaces.go
func (assetIter *assetIter) Close() error {
	if !assetIter.closed {
		assetIter.closed = true
		if assetIter.LedgerIter != nil {
			return assetIter.LedgerIter.Close()
		}
	}
	return nil
}

// GetPreviousLedgerKey documentation can be found in asset_mgmt_interfaces.go
func (assetIter *assetIter) GetPreviousLedgerKey() string {
	return assetIter.PreviousLedgerKey
}

// GetAssetPage documentation can be found in asset_mgmt_interfaces.go
func (assetIter *assetIter) GetAssetPage() ([]data_model.Asset, string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	assetPage := []data_model.Asset{}
	defer assetIter.Close()
	for assetIter.HasNext() {
		asset, err := assetIter.Next()
		if err != nil {
			custom_err := &custom_errors.IterError{}
			logger.Errorf("%v: %v", custom_err, err)
			return assetPage, assetIter.PreviousLedgerKey, errors.Wrap(err, custom_err.Error())
		}

		assetPage = append(assetPage, *asset)
	}

	// Return the LastLedgerKey so that it can be passed back to us to get the next page
	return assetPage, assetIter.PreviousLedgerKey, nil
}

// ------------------------------------------------------
// -----------------   HELPER FUNCTIONS    --------------
// ------------------------------------------------------

// updateCustomAssetIndices is responsible for updating custom indices for a given asset.
func updateCustomAssetIndices(stub cached_stub.CachedStubInterface, asset data_model.Asset, isNewAsset bool, isPrivateDataEncrypted bool) error {

	if utils.IsStringEmpty(asset.IndexTableName) {
		return nil
	}

	// Unmarshal PublicData into map[string]interface{}
	publicDataMap := make(map[string]interface{})
	privateDataMap := make(map[string]interface{})
	err := json.Unmarshal(asset.PublicData, &publicDataMap)
	if err != nil {
		custom_err := &custom_errors.UnmarshalError{Type: "asset.PublicData"}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	// Unmarshal PrivateData if it's not encrypted
	if !isPrivateDataEncrypted {
		err = json.Unmarshal(asset.PrivateData, &privateDataMap)
		if err != nil {
			custom_err := &custom_errors.UnmarshalError{Type: "asset.PrivateData"}
			logger.Errorf("%v: %v", custom_err, err)
			return errors.Wrap(err, custom_err.Error())
		}
	}

	// Get the index table for this asset
	indexTable := index.GetTable(stub, asset.IndexTableName)
	indexedFieldsList := indexTable.GetIndexedFields()

	// Get primary key
	primaryKeyField := indexTable.GetPrimaryKeyId()
	primaryKeyId := asset.AssetId
	if val, ok := publicDataMap[primaryKeyField]; ok {
		primaryKeyId, _ = utils.ConvertToString(val)
	} else if val, ok := privateDataMap[primaryKeyField]; ok {
		primaryKeyId, _ = utils.ConvertToString(val)
	}

	// Get existing index if it's not a new Asset
	existingIndexValues := make(map[string]string)
	if !isNewAsset {
		rowBytes, err := indexTable.GetRow(primaryKeyId)
		if err != nil {
			logger.Error("Failed to get index row")
			return errors.Wrap(err, "Failed to get index row")
		}
		err = json.Unmarshal(rowBytes, &existingIndexValues)
		if err != nil {
			custom_err := &custom_errors.UnmarshalError{Type: "Index row"}
			logger.Errorf("%v: %v", custom_err, err)
			return errors.Wrap(err, custom_err.Error())
		}
	}

	// Extract the indexed values from the asset
	updatedIndexValues := make(map[string]string)
	for _, indexedField := range indexedFieldsList {
		var err error = nil
		if val, ok := publicDataMap[indexedField]; ok {
			updatedIndexValues[indexedField], err = utils.ConvertToString(val)
		} else if val, ok := privateDataMap[indexedField]; ok {
			updatedIndexValues[indexedField], err = utils.ConvertToString(val)
		} else if val, ok := existingIndexValues[indexedField]; ok {
			// keep existing index value
			updatedIndexValues[indexedField] = val
		} else if indexedField == indexTable.GetPrimaryKeyId() {
			// Handle default primary key
			updatedIndexValues[indexedField] = asset.AssetId
		} else {
			//error
			logger.Errorf("Missing indexed field: %v", indexedField)
			return errors.Errorf("Missing indexed field: %v", indexedField)
		}
		if err != nil {
			logger.Errorf("Failed to update indexField: %v", err)
			return errors.Wrap(err, "Failed to update indexField")
		}
	}

	// Update the indices
	err = indexTable.UpdateRow(updatedIndexValues)
	if err != nil {
		logger.Errorf("Failed to UpdateRow: %v", err)
		return errors.Wrap(err, "Failed to UpdateRow")
	}
	return nil
}

// putAssetByKey adds a new asset or replaces an existing asset.
// AssetKey is required if encryption is needed.
// Empty AssetKeyBytes can be passed, if encrypted private data is provided.
// If invalid AssetKeyBytes is provided, return error.
// If adding a new asset, assetKey is required even if private data is empty.
// Only owner of the asset can add or update the asset.
// Only original owner can change owners.
// The caller is responsible for first checking for write access.
// Optional parameters: yourKeyId, yourKey, and yourEncKey are used to call AddAccess.
// yourKeyId: "" OR yourKey id
// yourKey: nil OR private or symkey
// yourEncKey: nil OR public or symkey
// isUpdate is an optional (bool) parameter
// if isUpdate is true, only update is allowed.
// if isUpdate is false, only add is allowed.
// if isUpdate is not passed, both add and update are allowed.
// if isUpdate is not a bool value, returns error.
func putAssetByKey(stub cached_stub.CachedStubInterface, caller data_model.User, asset data_model.Asset, assetKeyId string, assetKeyBytes []byte, yourKeyId string, yourKey []byte, yourEncKey []byte, isUpdate ...bool) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("putAssetByKey assetId: \"%v\", assetKeyId: \"%v\", yourKeyId \"%v\"", asset.AssetId, assetKeyId, yourKeyId)
	callerId := caller.ID
	// fill missing ownerId
	if len(asset.OwnerIds) == 0 {
		asset.OwnerIds = []string{callerId}
	}
	// only one owner is allowed
	if len(asset.OwnerIds) > 1 {
		asset.OwnerIds = []string{asset.OwnerIds[0]}
	}

	// fill missing assetKeyId
	if len(asset.AssetKeyId) == 0 {
		asset.AssetKeyId = assetKeyId
	}

	// asset key
	assetKey := data_model.Key{ID: assetKeyId, KeyBytes: assetKeyBytes, Type: global.KEY_TYPE_SYM}
	assetKeyHash := []byte{}

	if len(assetKeyBytes) > 0 {
		assetKeyHash = crypto.Hash(assetKey.KeyBytes)

		// fill missing AssetKeyHash
		if len(asset.AssetKeyHash) == 0 {
			asset.AssetKeyHash = assetKeyHash
		}
	}

	// assetKeyId is always required although assetKeyBytes is not required if
	// encyrpted private date is provided for update
	if len(assetKeyId) == 0 || asset.AssetKeyId != assetKeyId {
		logger.Error("Failed to put asset: Invalid Asset Key Id")
		return errors.New("Failed to put asset: Invalid Asset Key Id")
	}

	// if assetKeyBytes is provided, check hash
	if len(assetKeyBytes) > 0 && !bytes.Equal(asset.AssetKeyHash, assetKeyHash) {
		logger.Error("Failed to put asset: Invalid Asset Key: Hash does not match")
		return errors.New("Failed to put asset: Invalid Asset Key: Hash does not match")
	}

	// Check for existing asset on the ledger under assetId
	var isNewAsset = true
	var isPrivateDataEncrypted = false
	existingAsset, err := GetEncryptedAssetData(stub, asset.AssetId)
	if err != nil {
		custom_err := &custom_errors.GetAssetDataError{AssetId: asset.AssetId}
		logger.Errorf("Failed to put asset: %v", custom_err)
		return errors.Wrap(err, "Failed to put asset: "+custom_err.Error())
	}
	if len(existingAsset.AssetId) > 0 {
		isNewAsset = false
	}

	// Check isUpdate option
	var isStrictMode = false
	var isAdd = false
	if len(isUpdate) == 1 {
		isStrictMode = true
		isAdd = !isUpdate[0]
	} else if len(isUpdate) != 0 {
		logger.Errorf("Failed to put asset: Invalid isUpdate: %v", isUpdate)
		return errors.New("Failed to put asset: Invalid isUpdate")
	}

	logger.Debugf("isStr %v isAdd %v isNew %v", isStrictMode, isAdd, isNewAsset)
	//validate update/add option
	if isStrictMode && isAdd != isNewAsset {
		if isAdd {
			logger.Errorf("Failed to put asset: Asset with same id already exist %v", asset.AssetId)
			return errors.New("Failed to put asset: Asset with same id already exist")
		} else {
			logger.Errorf("Failed to put asset: Asset with same id does not exist %v", asset.AssetId)
			return errors.New("Failed to put asset: Asset with same id does not exist")
		}
	}

	// check if encryption is required, and more validations
	var encryptionRequired = true
	if isNewAsset {
		// new asset
		// make sure that your private data is not encrypted
		if data_model.IsEncryptedData(asset.PrivateData) {
			logger.Error("Failed to put asset: Private data cannot be encrypted for a new asset")
			return errors.New("Failed to put asset: Private data cannot be encrypted for a new asset")
		}

		// check asset key is valid if it's new asset
		valid, err := key_mgmt_i.ValidateKey(stub, assetKey, false)
		if err != nil {
			custom_err := &custom_errors.ValidateKeyError{KeyId: assetKeyId}
			logger.Errorf("Failed to put asset: %v: %v", custom_err, err)
			return errors.Wrap(err, "Failed to put asset: "+custom_err.Error())
		}
		if !valid {
			custom_err := &custom_errors.ValidateKeyError{KeyId: assetKeyId}
			logger.Errorf("Failed to put asset: %v", custom_err.Error())
			return errors.New("Failed to put asset: " + custom_err.Error())
		}
	} else {
		// existing asset
		// you can't change owner unless you are the original owner
		if !utils.EqualStringArrays(asset.OwnerIds, existingAsset.OwnerIds) && (asset.OwnerIds[0] != existingAsset.OwnerIds[0] || callerId != asset.OwnerIds[0]) {
			logger.Error("Failed to put asset: Caller cannot change onwer of the asset")
			return errors.New("Failed to put asset: Caller cannot change owner of the asset")
		}

		// you can't change asset key
		if len(asset.AssetKeyHash) == 0 {
			asset.AssetKeyHash = existingAsset.AssetKeyHash
		} else if !bytes.Equal(asset.AssetKeyHash, existingAsset.AssetKeyHash) {
			custom_err := &custom_errors.ValidateKeyError{KeyId: assetKeyId}
			logger.Errorf("Failed to put asset: %v: Hash does not match", custom_err)
			return errors.New("Failed to put asset: " + custom_err.Error())
		}
		if existingAsset.AssetKeyId != asset.AssetKeyId {
			custom_err := &custom_errors.ValidateKeyError{KeyId: assetKeyId}
			logger.Errorf("Failed to put asset: %v: Id does not match", custom_err)
			return errors.New("Failed to put asset: " + custom_err.Error())
		}

		// check if privatedate is encypted or not
		existing_wrapped := data_model.GetEncryptedDataBytes(existingAsset.PrivateData)
		if bytes.Equal(existing_wrapped, asset.PrivateData) {
			encryptionRequired = false
			asset.PrivateData = existingAsset.PrivateData
		}
	}

	logger.Debugf("IsNewAsset=%v, IsStrictMode=%v, IsAdd=%v, encryptionRequired=%v", isNewAsset, isStrictMode, isAdd, encryptionRequired)

	// assetKeyBytesRequired if encryption is reqyired
	if encryptionRequired && len(assetKeyBytes) == 0 {
		logger.Error("Failed to put asset: Asset key required for encryption")
		return errors.New("Failed to put asset: Asset key required for encryption")
	}

	// Update custom indices
	// this step need to be done before encrypting private data
	err = updateCustomAssetIndices(stub, asset, isNewAsset, isPrivateDataEncrypted)
	if err != nil {
		logger.Errorf("Failed to put asset: Failed to update custom asset indices for assetId: %v", asset.AssetId)
		return errors.Wrapf(err, "Failed to put asset: Failed to update custom asset indices for assetId: %v", asset.AssetId)
	}

	// encrypt private data
	if encryptionRequired && len(asset.PrivateData) > 0 {
		// Encrypt PrivateData with asset sym key
		privateData, err := crypto.EncryptWithSymKey(assetKeyBytes, asset.PrivateData)
		if err != nil {
			custom_err := &custom_errors.EncryptionError{ToEncrypt: "PrivateData", EncryptionKey: assetKeyId}
			logger.Errorf("Failed to put asset: %v: %v", custom_err, err)
			return errors.Wrap(err, "Failed to put asset: "+custom_err.Error())
		}
		if privateData == nil {
			custom_err := &custom_errors.EncryptionError{ToEncrypt: "PrivateData", EncryptionKey: assetKeyId}
			logger.Errorf("Failed to put asset: %v", custom_err)
			return errors.New("Failed to put asset: " + custom_err.Error())
		}
		asset.PrivateData = privateData
	}

	// normalize datatype
	datatypes, err := datatype_i.NormalizeDatatypes(stub, asset.Datatypes)
	if err != nil {
		logger.Errorf("Failed to put asset: Failed to NormalizeDatatypes: %v", err)
		return errors.Wrap(err, "Failed to put asset: Failed to NormalizeDatatypes")
	}
	asset.Datatypes = datatypes

	// update datatypes
	if isNewAsset || !utils.EqualStringArrays(asset.Datatypes, existingAsset.Datatypes) {
		// upate datatypeasset.Datatypes)
		err = updateAssetToDatatype(stub, caller, asset.AssetId, assetKey, asset.OwnerIds[0], existingAsset.Datatypes, asset.Datatypes)
		if err != nil {
			logger.Errorf("Failed to put asset: Failed to updateAssetToDatatype: %v", err)
			return errors.Wrap(err, "Failed to put asset: Failed to UpdateAssetToDatatype")
		}
	}

	// If yourKey was provided, give access from yourKey -> assetKey
	if len(yourKeyId) > 0 && len(yourKey) > 0 && len(yourEncKey) > 0 {
		var edgeData = make(map[string]string)
		edgeData["asset_type"] = "yes"
		edgeData[global.EDGEDATA_ACCESS_TYPE] = global.ACCESS_WRITE
		err := key_mgmt_i.AddAccessWithKeys(stub, yourKey, yourKeyId, assetKeyBytes, assetKeyId, yourEncKey, edgeData)
		if err != nil {
			custom_err := &custom_errors.AddAccessError{Key: "assetKey"}
			logger.Errorf("Failed to put asset: %v: %v", custom_err, err)
			return errors.Wrap(err, "Failed to put asset: "+custom_err.Error())
		}
	}

	// Check if we need to store private data to datastore
	connectionID := asset.GetDatastoreConnectionID()
	if utils.IsStringEmpty(connectionID) && !utils.IsStringEmpty(defaultDatastoreConnectionID) {
		connectionID = defaultDatastoreConnectionID
	}
	if !utils.IsStringEmpty(connectionID) && encryptionRequired && len(asset.PrivateData) > 0 {
		myDatastore, err := datastore_c.GetDatastoreImpl(stub, connectionID)
		if err != nil {
			logger.Infof("Failed to instantiate datastore: %v", err)
			return errors.Wrap(err, "Failed to instantitate datastore")
		}
		// save to the datastore
		dataKey, err := myDatastore.Put(stub, asset.PrivateData)
		if err != nil {
			logger.Errorf("Failed to save data to datastore: %v", err)
			return errors.Wrap(err, "Failed to save data to datastore")
		}
		// set privatedata to the hash of the encrypted data
		asset.PrivateData = []byte(dataKey)
	}
	// Now save to the ledger
	assetBytesE, err := json.Marshal(&asset)
	if err != nil {
		custom_err := &custom_errors.MarshalError{Type: "encrypted asset data"}
		logger.Errorf("Failed to put asset: %v: %v", custom_err, err)
		return errors.Wrap(err, "Failed to put asset: "+custom_err.Error())
	}
	assetLedgerKey := asset.AssetId
	err = stub.PutState(assetLedgerKey, assetBytesE)
	if err != nil {
		custom_err := &custom_errors.PutLedgerError{LedgerKey: assetLedgerKey}
		logger.Errorf("Failed to put asset: %v: %v", custom_err, err)
		return errors.Wrap(err, "Failed to put asset: "+custom_err.Error())
	}

	logger.Infof("Successfully put asset \"%v\" with key \"%v\"", asset.AssetId, asset.AssetKeyId)
	return nil
}

// hasUserWriteAccessToAsset returns user with write access or write only access.
// If checkMyGroup is true, also checks whether my group has write access.
func hasUserWriteAccessToAsset(stub cached_stub.CachedStubInterface, user data_model.User, asset data_model.Asset, hasUserPrivKey, checkMyGroups bool) (bool, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("user: %v, asset:%v,  %v, %v", user.ID, asset.AssetKeyId, hasUserPrivKey, checkMyGroups)

	// check cache
	cachekey := fmt.Sprintf("writeaccess-%v-%v-%v", user.ID, asset.AssetId, hasUserPrivKey)
	cache, err := stub.GetCache(cachekey)
	if err == nil {
		if cachedata, ok := cache.(bool); ok {
			logger.Debugf("writeaccess return from cache")
			return cachedata, nil
		}
	}

	// if you have user priv key also check cache for not having priv key
	if hasUserPrivKey {
		cachekey2 := fmt.Sprintf("writeaccess-%v-%v-%v", user.ID, asset.AssetId, hasUserPrivKey)
		cache, err := stub.GetCache(cachekey2)
		if err == nil {
			if cachedata, ok := cache.(bool); ok {
				logger.Debugf("writeaccess return from cache")
				return cachedata, nil
			}
		}
	}

	// 1. user is owner or admin of owner of the asset
	if hasUserPrivKey && len(asset.OwnerIds) > 0 {
		if asset.IsOwner(user.ID) {
			logger.Debug("User is owner of asset")
			stub.PutCache(cachekey, true)
			return true, nil
		}
		// check this only at the top level
		if checkMyGroups {
			if isAdmin, _ := user_mgmt_c.IsUserDirectAdminOfGroup(stub, user.ID, asset.OwnerIds[0]); isAdmin {
				logger.Debug("User is admin of owner of asset")
				stub.PutCache(cachekey, true)
				return true, nil
			}
		}
	}

	// 2. user has write permission
	// check this by edge data with write access
	if hasUserPrivKey {
		_, edgeData, _ := key_mgmt_i.GetAccessEdge(stub, user.GetPubPrivKeyId(), asset.AssetKeyId)
		if val, ok := edgeData["AccessType"]; ok {
			if val == global.ACCESS_WRITE {
				//add to cache
				stub.PutCache(cachekey, true)
				return true, nil
			}
		}
	}
	_, edgeData, _ := key_mgmt_i.GetAccessEdge(stub, user.GetSymKeyId(), asset.AssetKeyId)
	if val, ok := edgeData["AccessType"]; ok {
		if val == global.ACCESS_WRITE {
			//add to cache
			stub.PutCache(cachekey, true)
			return true, nil
		}
	}

	// 3. user has write only permission
	// check this by edge data with write_only acces
	writeonly, err := hasUserWriteOnlyAccessToAsset(stub, user, asset, hasUserPrivKey, false)
	if writeonly {
		return true, nil
	}

	// 4. user has a write datatype consent
	//    it has to be an existing asset
	if hasUserPrivKey && len(asset.OwnerIds) > 0 {
		for _, datatypeID := range asset.Datatypes {

			consentID := consent_mgmt_c.GetConsentID(datatypeID, user.ID, asset.OwnerIds[0])
			_, edgeData, err = key_mgmt_i.GetAccessEdge(stub, consentID, datatype_i.GetDatatypeKeyID(datatypeID, asset.OwnerIds[0]))
			if val, ok := edgeData["AccessType"]; ok {
				if val == global.ACCESS_WRITE {
					//add to cache
					stub.PutCache(cachekey, true)
					return true, nil
				}
			}

			parent, err := datatype_i.GetParentDatatype(stub, datatypeID)
			for err == nil && len(parent) > 0 {
				currID := parent
				consentID := consent_mgmt_c.GetConsentID(currID, user.ID, asset.OwnerIds[0])
				_, edgeData, err = key_mgmt_i.GetAccessEdge(stub, consentID, datatype_i.GetDatatypeKeyID(datatypeID, asset.OwnerIds[0]))
				if val, ok := edgeData["AccessType"]; ok {
					if val == global.ACCESS_WRITE {
						//add to cache
						stub.PutCache(cachekey, true)
						return true, nil
					}
				}
				parent, err = datatype_i.GetParentDatatype(stub, currID)
			}
		}

	}

	// 5. user is a direct admin of a group that has write access
	if checkMyGroups {
		myAdminIDs, err := user_mgmt_c.GetMyDirectAdminGroupIDs(stub, user.ID)
		if err != nil {
			return false, errors.Wrap(err, "Failed to get my direct adminIDs")
		}
		for _, adminID := range myAdminIDs {
			adminGr := data_model.User{ID: adminID}
			hasAccess, _ := hasUserWriteAccessToAsset(stub, adminGr, asset, hasUserPrivKey, false)
			if hasAccess {
				//add to cache
				stub.PutCache(cachekey, true)
				return true, nil
			}
		}
	}

	//add to cache only for the top level
	if checkMyGroups {
		stub.PutCache(cachekey, false)
	}
	return false, nil
}

// hasUserWriteOnlyAccessToAsset returns user with write only access.
func hasUserWriteOnlyAccessToAsset(stub cached_stub.CachedStubInterface, user data_model.User, asset data_model.Asset, hasUserPrivKey, checkMyGroups bool) (bool, error) {

	if hasUserPrivKey {
		// 1. user has write only permission
		// check this by edge data with write_only access
		if hasUserPrivKey && len(asset.OwnerIds) > 0 {
			keyId := key_mgmt_i.GetKeyIdForWriteOnlyAccess(asset.AssetId, asset.AssetKeyId, asset.OwnerIds[0])
			_, edgeData, _ := key_mgmt_i.GetAccessEdge(stub, user.GetPubPrivKeyId(), keyId)
			if val, ok := edgeData["AccessType"]; ok {
				if val == global.ACCESS_WRITE_ONLY {
					return true, nil
				}
			}
		}

		// 6. user is a direct admin of a group that has write only access
		if checkMyGroups {
			myAdminIDs, err := user_mgmt_c.GetMyDirectAdminGroupIDs(stub, user.ID)
			if err != nil {
				return false, errors.Wrap(err, "Failed to get my direct adminIDs")
			}
			for _, adminID := range myAdminIDs {
				adminGr := data_model.User{ID: adminID}
				hasAccess, _ := hasUserWriteOnlyAccessToAsset(stub, adminGr, asset, hasUserPrivKey, false)
				if hasAccess {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// hasUserReadAccessToAsset returns user with write access or read access.
func hasUserReadAccessToAsset(stub cached_stub.CachedStubInterface, user data_model.User, asset data_model.Asset, hasUserPrivKey, checkMyGroups bool) (bool, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("user: %v, asset:%v,  %v, %v", user.ID, asset.AssetKeyId, hasUserPrivKey, checkMyGroups)

	// check cache
	cachekey := fmt.Sprintf("readaccess-%v-%v-%v", user.ID, asset.AssetId, hasUserPrivKey)
	cache, err := stub.GetCache(cachekey)
	if err == nil {
		if cachedata, ok := cache.(bool); ok {
			logger.Debugf("readaccess return from cache")
			return cachedata, nil
		}
	}
	// if it has writeaccess, the return true
	cachekey2 := fmt.Sprintf("writeaccess-%v-%v-%v", user.ID, asset.AssetId, hasUserPrivKey)
	cache, err = stub.GetCache(cachekey2)
	if err == nil {
		if cachedata, ok := cache.(bool); ok {
			if cachedata {
				return true, nil
			}
		}
	}

	if hasUserPrivKey {
		cachekey2 = fmt.Sprintf("readaccess-%v-%v-%v", user.ID, asset.AssetId, false)
		cache, err := stub.GetCache(cachekey)
		if err == nil {
			if cachedata, ok := cache.(bool); ok {
				logger.Debugf("readaccess return from cache")
				return cachedata, nil
			}
		}
		// if it has writeaccess, the return true
		cachekey2 = fmt.Sprintf("writeaccess-%v-%v-%v", user.ID, asset.AssetId, false)
		cache, err = stub.GetCache(cachekey2)
		if err == nil {
			if cachedata, ok := cache.(bool); ok {
				if cachedata {
					return true, nil
				}
			}
		}
	}
	// 1. user is owner or admin of owner of the asset
	if hasUserPrivKey && len(asset.OwnerIds) > 0 {
		if asset.IsOwner(user.ID) {
			logger.Debug("User is owner of asset")
			stub.PutCache(cachekey, true)
			return true, nil
		}
		// check only at the top level
		if checkMyGroups {
			if isAdmin, _ := user_mgmt_c.IsUserDirectAdminOfGroup(stub, user.ID, asset.OwnerIds[0]); isAdmin {
				logger.Debug("User is admin of owner of asset")
				stub.PutCache(cachekey, true)
				return true, nil

			}
		}
	}

	// 2. user has read / write permission
	// check this by edge data with read / write access
	if hasUserPrivKey {
		_, edgeData, _ := key_mgmt_i.GetAccessEdge(stub, user.GetPubPrivKeyId(), asset.AssetKeyId)
		if val, ok := edgeData["AccessType"]; ok {
			if val == global.ACCESS_WRITE || val == global.ACCESS_READ {
				stub.PutCache(cachekey, true)
				return true, nil
			}
		}
	}
	_, edgeData, _ := key_mgmt_i.GetAccessEdge(stub, user.GetSymKeyId(), asset.AssetKeyId)
	if val, ok := edgeData["AccessType"]; ok {
		if val == global.ACCESS_WRITE || val == global.ACCESS_READ {
			stub.PutCache(cachekey, true)
			return true, nil
		}
	}

	// 3. user has a read / write datatype consent
	if hasUserPrivKey && len(asset.OwnerIds) > 0 {
		for _, datatypeID := range asset.Datatypes {
			consentID := consent_mgmt_c.GetConsentID(datatypeID, user.ID, asset.OwnerIds[0])
			_, edgeData, _ := key_mgmt_i.GetAccessEdge(stub, consentID, datatype_i.GetDatatypeKeyID(datatypeID, asset.OwnerIds[0]))
			if val, ok := edgeData["AccessType"]; ok {
				if val == global.ACCESS_WRITE || val == global.ACCESS_READ {
					stub.PutCache(cachekey, true)
					return true, nil
				}
			}

			parent, err := datatype_i.GetParentDatatype(stub, datatypeID)
			for err == nil && len(parent) > 0 {
				currID := parent
				consentID := consent_mgmt_c.GetConsentID(currID, user.ID, asset.OwnerIds[0])
				_, edgeData, _ := key_mgmt_i.GetAccessEdge(stub, consentID, datatype_i.GetDatatypeKeyID(datatypeID, asset.OwnerIds[0]))
				if val, ok := edgeData["AccessType"]; ok {
					if val == global.ACCESS_WRITE || val == global.ACCESS_READ {
						stub.PutCache(cachekey, true)
						return true, nil
					}
				}
				parent, err = datatype_i.GetParentDatatype(stub, currID)
			}
		}

	}

	// 4. user is a direct member of a group that has read access
	if checkMyGroups {
		myAdminIDs, err := user_mgmt_c.GetMyDirectAdminGroupIDs(stub, user.ID)
		if err != nil {
			return false, errors.Wrap(err, "Failed to get my direct adminIDs")
		}
		for _, adminID := range myAdminIDs {
			adminGr := data_model.User{ID: adminID}
			hasAccess, _ := hasUserReadAccessToAsset(stub, adminGr, asset, true, false)
			if hasAccess {
				//add to cache
				stub.PutCache(cachekey, true)
				return true, nil
			}
		}
		myGroupIDs, err := user_mgmt_c.GetMyDirectGroupIDs(stub, user.ID)
		if err != nil {
			return false, errors.Wrap(err, "Failed to get my direct groupIDs")
		}
		for _, groupID := range myGroupIDs {
			group := data_model.User{ID: groupID}
			hasAccess, _ := hasUserReadAccessToAsset(stub, group, asset, false, false)
			if hasAccess {
				//add to cache
				stub.PutCache(cachekey, true)
				return true, nil
			}
		}
	}

	if checkMyGroups {
		stub.PutCache(cachekey, false)
	}
	return false, nil
}

// hasUserReadOnlyAccessToAsset returns user with read access (excluding users with write access).
func hasUserReadOnlyAccessToAsset(stub cached_stub.CachedStubInterface, user data_model.User, asset data_model.Asset, hasUserPrivKey, checkMyGroups bool) (bool, error) {
	// save cache only for the true case (for readaccess)
	cachekey := fmt.Sprintf("readaccess-%v-%v-%v", user.ID, asset.AssetId, hasUserPrivKey)

	// if it has writeaccess, the return false
	cachekey2 := fmt.Sprintf("writeaccess-%v-%v-%v", user.ID, asset.AssetId, hasUserPrivKey)
	cache, err := stub.GetCache(cachekey2)
	if err == nil {
		if cachedata, ok := cache.(bool); ok {
			if cachedata {
				return false, nil
			}
		}
	}

	if hasUserPrivKey {
		cachekey2 := fmt.Sprintf("writeaccess-%v-%v-%v", user.ID, asset.AssetId, false)
		cache, err := stub.GetCache(cachekey2)
		if err == nil {
			if cachedata, ok := cache.(bool); ok {
				if cachedata {
					return false, nil
				}
			}
		}
	}

	// 1. user has read  permission
	// check this by edge data with read access
	if hasUserPrivKey {
		_, edgeData, _ := key_mgmt_i.GetAccessEdge(stub, user.GetPubPrivKeyId(), asset.AssetKeyId)
		if val, ok := edgeData["AccessType"]; ok {
			if val == global.ACCESS_READ {
				stub.PutCache(cachekey, true)
				return true, nil
			}
		}
	}

	// 2. user has a read datatype consent
	if hasUserPrivKey && len(asset.OwnerIds) > 0 {
		for _, datatypeID := range asset.Datatypes {
			consentID := consent_mgmt_c.GetConsentID(datatypeID, user.ID, asset.OwnerIds[0])
			_, edgeData, _ := key_mgmt_i.GetAccessEdge(stub, consentID, datatype_i.GetDatatypeKeyID(datatypeID, asset.OwnerIds[0]))
			if val, ok := edgeData["AccessType"]; ok {
				if val == global.ACCESS_READ {
					stub.PutCache(cachekey, true)
					return true, nil
				}
			}

			parent, err := datatype_i.GetParentDatatype(stub, datatypeID)
			for err == nil && len(parent) > 0 {
				currID := parent
				consentID := consent_mgmt_c.GetConsentID(currID, user.ID, asset.OwnerIds[0])
				_, edgeData, _ := key_mgmt_i.GetAccessEdge(stub, consentID, datatype_i.GetDatatypeKeyID(datatypeID, asset.OwnerIds[0]))
				if val, ok := edgeData["AccessType"]; ok {
					if val == global.ACCESS_READ {
						stub.PutCache(cachekey, true)
						return true, nil
					}
				}
				parent, err = datatype_i.GetParentDatatype(stub, currID)
			}
		}

	}

	// 3. user is a direct member of a group that has read access
	if checkMyGroups {
		myAdminIDs, err := user_mgmt_c.GetMyDirectAdminGroupIDs(stub, user.ID)
		if err != nil {
			return false, errors.Wrap(err, "Failed to get my direct adminIDs")
		}
		for _, adminID := range myAdminIDs {
			adminGr := data_model.User{ID: adminID}
			hasAccess, _ := hasUserReadOnlyAccessToAsset(stub, adminGr, asset, true, false)
			if hasAccess {
				//add to cache
				stub.PutCache(cachekey, true)
				return true, nil
			}
		}
		myGroupIDs, err := user_mgmt_c.GetMyDirectGroupIDs(stub, user.ID)
		if err != nil {
			return false, errors.Wrap(err, "Failed to get my direct groupIDs")
		}
		for _, groupID := range myGroupIDs {
			group := data_model.User{ID: groupID}
			hasAccess, _ := hasUserReadOnlyAccessToAsset(stub, group, asset, false, false)
			if hasAccess {
				//add to cache
				stub.PutCache(cachekey, true)
				return true, nil
			}
		}
	}

	return false, nil
}

// getAssetByKey returns the asset from the ledger specified by assetId and assetKey.
// Returns the asset with decrypted private data if assetKey is valid.
// If assetKey is nil, it returns the asset with encrypted private data.
// Returns an error if assetKey is invalid.
// Returns an empty asset if the given assetId does not exist. It is the caller's responsibility to check if returned asset is empty.
func getAssetByKey(stub cached_stub.CachedStubInterface, assetId string, assetKey []byte) (*data_model.Asset, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("getAssetByKey assetId: %v, assetKey: %v", assetId, assetKey)

	assetData, err := GetEncryptedAssetData(stub, assetId)
	if err != nil {
		custom_err := &custom_errors.GetAssetDataError{AssetId: assetId}
		logger.Errorf("%v: %v", custom_err, err)
		return &assetData, errors.Wrap(err, custom_err.Error())
	}

	// empty asset
	if utils.IsStringEmpty(assetData.AssetId) {
		return &assetData, nil
	}

	// attempt to decrypt private data
	assetData.PrivateData, err = getPrivateData(stub, assetData, assetKey)

	// return asset
	return &assetData, err
}

// getPrivateDataFromCache returns a copy of the object to avoid the "by reference" side effect.
func getPrivateDataFromCache(stub cached_stub.CachedStubInterface, assetId string) ([]byte, error) {
	assetPrivateCacheKey := getAssetPrivateCacheKey(assetId)
	cachedAssetPrivate, err := stub.GetCache(assetPrivateCacheKey)
	if err != nil {
		return nil, err
	} else if cachedAssetPrivate != nil {
		privateData, ok := cachedAssetPrivate.([]byte)
		if ok {
			privateDataCopy := make([]byte, len(privateData))
			copy(privateDataCopy, privateData)
			return privateDataCopy, nil
		} else {
			return nil, errors.New("Failed to map cache to []byte")
		}
	}
	return nil, nil
}

// putPrivateDataToCache makes a copy of the asset and saves it to the cache.
func putPrivateDataToCache(stub cached_stub.CachedStubInterface, assetId string, data []byte) error {
	assetCacheKey := getAssetPrivateCacheKey(assetId)
	privateDataCopy := make([]byte, len(data))
	copy(privateDataCopy, data)
	return stub.PutCache(assetCacheKey, privateDataCopy)
}

// getPrivateData decrypts and returns an asset's private data bytes.
// Caller should provide proper assetKey for decryption.
// If decryption is successful, it returns decrypted private data bytes.
// If assetKey is nil, it returns encrypted private data bytes and nil for error.
// If assetKey is invalid, it returns encrypted private data bytes and an error.
func getPrivateData(stub cached_stub.CachedStubInterface, assetData data_model.Asset, assetKey []byte) ([]byte, error) {
	// no assetkey, return encrypted private data
	if assetKey == nil {
		logger.Debugf("No asset key: Return EncryptedDataBytes")
		return data_model.GetEncryptedDataBytes(assetData.PrivateData), nil
	}

	// verify assetKey by hash
	if !bytes.Equal(assetData.AssetKeyHash, crypto.Hash(assetKey)) {
		custom_err := &custom_errors.ReplaceAssetKeyHashError{}
		logger.Debugf("%v", custom_err)
		return data_model.GetEncryptedDataBytes(assetData.PrivateData), errors.WithStack(custom_err)
	}

	// check cache
	decryptedAsset, err := getPrivateDataFromCache(stub, assetData.AssetId)
	if err == nil && decryptedAsset != nil {
		logger.Debugf("Returning from Cache")
		return decryptedAsset, nil
	}

	datastoreConnectionID := assetData.GetDatastoreConnectionID()
	if utils.IsStringEmpty(datastoreConnectionID) && !utils.IsStringEmpty(defaultDatastoreConnectionID) {
		datastoreConnectionID = defaultDatastoreConnectionID
	}
	if utils.IsStringEmpty(datastoreConnectionID) {
		// attempt to decrypt asset bytes
		decryptedAsset, err = crypto.DecryptWithSymKey(assetKey, assetData.PrivateData)
		if err != nil {
			custom_err := &custom_errors.DecryptionError{ToDecrypt: "asset", DecryptionKey: "sym key"}
			logger.Infof("%v: %v", custom_err, err)
			return data_model.GetEncryptedDataBytes(assetData.PrivateData), errors.WithStack(custom_err)
		}
	} else {
		// get encrypted data from datastore
		myDatastore, err := datastore_c.GetDatastoreImpl(stub, datastoreConnectionID)
		if err != nil {
			logger.Infof("error instantiating datastore: %v", err)
			return data_model.GetEncryptedDataBytes(assetData.PrivateData), errors.WithStack(err)
		}
		encryptedData, err := myDatastore.Get(stub, string(assetData.PrivateData))
		if err != nil {
			logger.Infof("error getting data from : %v", err)
			return data_model.GetEncryptedDataBytes(assetData.PrivateData), errors.WithStack(err)
		}
		// attempt to decrypt data bytes
		decryptedAsset, err = crypto.DecryptWithSymKey(assetKey, encryptedData)
		if err != nil {
			custom_err := &custom_errors.DecryptionError{ToDecrypt: "asset", DecryptionKey: "sym key"}
			logger.Infof("%v: %v", custom_err, err)
			return data_model.GetEncryptedDataBytes(assetData.PrivateData), errors.WithStack(custom_err)
		}
	}

	// if asset bytes successfully decrypted, save to cache
	putPrivateDataToCache(stub, assetData.AssetId, decryptedAsset)
	return decryptedAsset, nil
}

// getEncryptedAssetFromCache returns a copy of the object to avoid the "by reference" side effect.
func getEncryptedAssetFromCache(stub cached_stub.CachedStubInterface, assetId string) (*data_model.Asset, error) {
	assetCacheKey := getAssetCacheKey(assetId)
	cachedAsset, err := stub.GetCache(assetCacheKey)
	if err != nil {
		return nil, err
	} else if cachedAsset != nil {
		asset, ok := cachedAsset.(data_model.Asset)
		if ok {
			assetCopy := asset.Copy()
			return &assetCopy, nil
		} else {
			return nil, errors.New("Failed to map cache to Asset type")
		}
	}
	return nil, nil
}

// putEncryptedAssetToCache makes a copy of the asset and saves it to the cache.
func putEncryptedAssetToCache(stub cached_stub.CachedStubInterface, asset data_model.Asset) error {
	assetCacheKey := getAssetCacheKey(asset.AssetId)
	return stub.PutCache(assetCacheKey, asset.Copy())
}

// updateAssetToDatatype is called from putAssetByKey
// updateAssetToDatatype updates access from datatypeSymKey to assetKey with edge type "DatatypeEdge"
// it also checks all newly added datatypes are in active state
// caller is responsible for ensuring that the asset already exists.
func updateAssetToDatatype(stub cached_stub.CachedStubInterface, caller data_model.User, assetId string, assetKey data_model.Key, assetOwnerID string, prevDatatypes []string, newDatatypes []string) error {
	prevMap := make(map[string]bool)
	newMap := make(map[string]bool)
	addList := []string{}
	removeList := []string{}
	for _, key := range prevDatatypes {
		prevMap[key] = true
	}
	for _, key := range newDatatypes {
		newMap[key] = true
		if !prevMap[key] {
			addList = append(addList, key)
		}
	}

	for _, key := range prevDatatypes {
		if !newMap[key] {
			removeList = append(removeList, key)
		}
	}

	// remove data type access
	for _, datatypeID := range removeList {
		datatypeKeyId := datatype_i.GetDatatypeKeyID(datatypeID, assetOwnerID)
		err := key_mgmt_i.RevokeAccess(stub, datatypeKeyId, assetKey.ID)
		if err != nil {
			logger.Errorf("Failed to REvokeAccess: %v", err)
			return errors.Wrap(err, "Failed to RevokeAccess")
		}
	}

	// add access from datatypeSymKey to assetKey with DatatypeEdge
	for _, datatypeID := range addList {
		// check if datatype is active
		datatype, err := datatype_i.GetDatatypeWithParams(stub, datatypeID)
		if err != nil {
			logger.Errorf("Failed to GetDatatype: %v", err)
			return errors.Wrap(err, "Failed to GetDatatype")
		}
		if !datatype.IsActive() {
			logger.Errorf("Inactive datatype: %v", datatypeID)
			return errors.New("Inactive datatype: " + datatypeID)
		}

		// datatype symkeys must have been added before trying to putAsset
		datatypeKey, err := datatype_i.GetDatatypeSymKey(stub, caller, datatypeID, assetOwnerID)
		if err != nil {
			logger.Errorf("Failed to GetDatatypeSymkey: %v", err)
			return errors.Wrap(err, "Failed to GetDatatypeSymKey")
		}
		edgeData := make(map[string]string)
		edgeData["edge"] = "DatatypeEdge"
		edgeData["datatype"] = datatypeID
		err = key_mgmt_i.AddAccess(stub, datatypeKey, assetKey, edgeData)
		if err != nil {
			custom_err := &custom_errors.AddAccessError{Key: "assetKey"}
			logger.Errorf("%v: %v", custom_err, err)
			return errors.Wrap(err, custom_err.Error())
		}
	}

	return nil
}

func getAssetCacheKey(assetId string) string {
	return global.ASSET_CACHE_PREFIX + assetId
}

func getAssetPrivateCacheKey(assetId string) string {
	return global.ASSET_PRIVATE_CACHE_PREFIX + assetId
}

// IsValidAssetId checks whether the given assetId has the correct asset id prefix.
func IsValidAssetId(assetId string) bool {
	return asset_mgmt_g.IsValidAssetId(assetId)
}

func getUserPublicKey(stub cached_stub.CachedStubInterface, userId string) (data_model.Key, error) {
	userData, err := getAssetByKey(stub, user_mgmt_c.GetUserAssetID(userId), nil)
	if err != nil {
		logger.Errorf("Failed to get user \"%v\"", userId)
		return data_model.Key{}, errors.Wrapf(err, "Failed to get user \"%v\"", userId)
	} else if len(userData.AssetId) == 0 {
		logger.Errorf("Failed to get user \"%v\"", userId)
		return data_model.Key{}, errors.Errorf("Failed to get user \"%v\"", userId)
	}
	user := data_model.User{ID: userId}
	user.LoadFromAsset(userData)
	pubKey := data_model.Key{
		ID:       user.GetPubPrivKeyId(),
		KeyBytes: crypto.PublicKeyToBytes(user.PublicKey),
		Type:     global.KEY_TYPE_PUBLIC,
	}
	return pubKey, nil
}

// PutAssetByKey is to be used only by test
func PutAssetByKey(mstub *test_utils.NewMockStub, caller data_model.User, asset data_model.Asset, assetKeyId string, assetKeyBytes []byte, yourKeyId string, yourKey []byte, yourEncKey []byte, isUpdate ...bool) error {
	stub := cached_stub.NewCachedStub(mstub)
	return putAssetByKey(stub, caller, asset, assetKeyId, assetKeyBytes, yourKeyId, yourKey, yourEncKey, isUpdate...)
}
