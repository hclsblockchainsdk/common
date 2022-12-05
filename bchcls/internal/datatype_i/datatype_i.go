/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package datatype manages datatypes and their relationships.
// Each asset has a list of datatypes. This allows all assets of datatype "Medical Records," for
// example, to be shared through a single consent.
package datatype_i

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/datatype/datatype_interface"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/datatype_i/datatype_c"
	"common/bchcls/internal/key_mgmt_i/key_mgmt_c"
	"common/bchcls/internal/user_mgmt_i/user_mgmt_c"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("datatype_i")

// global.ROOT_DATATYPE_ID is id of ROOT datatype. All other datatypes are children of ROOT.
const ROOT_DATATYPE_ID = global.ROOT_DATATYPE_ID

// ------------------------------------------------------
// ---------------------- INIT FUNCTIONS ----------------
// ------------------------------------------------------

// Init sets up the datatype package by building an index table for datatypes.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	return datatype_c.Init(stub, logLevel...)
}

// RegisterDatatype registers a new datatype to the ledger.
// Maintains datatype tree structure.
// Assumes a ROOT datatype exists. Every solution must call init_common.Init, which saves the ROOT datatype to the ledger.
// Creates datatypeSymKey and maintains key relationship with parent datatypes.
// If parentDatatypeID is not provided or does not exist, the datatype will be added as a child of ROOT.
//
// args = [ datatype, parentDatatypeID ]
func RegisterDatatype(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return datatype_c.RegisterDatatype(stub, caller, args)
}

// RegisterDatatypeWithParams saves a new datatype to the ledger and maintains datatype tree structure.
// Caller must pass in datatype and can optionally pass in parentDatatypeID.
// If parentDatatypeID is not provided, the datatype will be added as a child of ROOT.
// If parentDatatypeID does not exist, an error is thrown
// It returns a new DatatypeInterface instance that has been registered
func RegisterDatatypeWithParams(stub cached_stub.CachedStubInterface, datatypeID, description string, isActive bool, parentDatatypeID string) (datatype_interface.DatatypeInterface, error) {
	return datatype_c.RegisterDatatypeWithParams(stub, datatypeID, description, isActive, parentDatatypeID)
}

// GetDatatypeKeyID returns the datatype key id associated with a datatype.
func GetDatatypeKeyID(datatypeID string, ownerID string) string {
	return datatype_c.GetDatatypeKeyID(datatypeID, ownerID)
}

// TODO: remove this after full migration:
// temporarily save the old version of GetDatatypeKeyID for compatibility of
// other package for now
/*
func GetDatatypeKeyID_OLD(datatypeID string) string {
	return GetDatatypeKeyID(datatypeID, "")
}
*/

// UpdateDatatype updates existing datatype's description.
// Caller's role must be "system".
//
// args = [ datatype ]
func UpdateDatatype(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return datatype_c.UpdateDatatype(stub, caller, args)
}

// GetDatatype returns a datatype with the given datatypeID.
// Returns an empty datatype if the passed in datatypeID does not match an existing datatype's ID.
//
// args = [ datatypeID ]
func GetDatatype(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return datatype_c.GetDatatype(stub, caller, args)
}

// GetDatatypeWithParams function gets a datatype from the ledger.
// Returns an empty datatype if the passed in datatypeID does not match an existing datatype's ID.
func GetDatatypeWithParams(stub cached_stub.CachedStubInterface, datatypeID string) (datatype_interface.DatatypeInterface, error) {
	return datatype_c.GetDatatypeWithParams(stub, datatypeID)
}

// GetAllDatatypes returns all datatypes, not including the ROOT datatype.
func GetAllDatatypes(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	return datatype_c.GetAllDatatypes(stub, caller, args)
}

// AddDatatypeSymKey adds a sym key for the given datatypeID and ownerID and returns the DatatypeSymKey.
// It will also make sure that all its parent datatypes will get new sym key for the given owner (if it does not exist already).
// If the datatype sym key already exists, it will return success.
func AddDatatypeSymKey(stub cached_stub.CachedStubInterface, caller data_model.User, datatypeID, ownerID string, keyPathForOwnerSymkey ...[]string) (data_model.Key, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("AddDatatypeSymKey DatatypeID: %v Owner: %v", datatypeID, ownerID)
	// verify datatypeID
	datatype, err := GetDatatypeWithParams(stub, datatypeID)
	if err != nil {
		logger.Errorf("Error validating datatypeId %v", err)
		return data_model.Key{}, err
	} else if utils.IsStringEmpty(datatype.GetDatatypeID()) {
		logger.Errorf("Invalid datatypeID %v", datatypeID)
		return data_model.Key{}, errors.New("Invalid datatypeID")
	}
	// get datatype symkey ID
	datatypeSymkeyID := GetDatatypeKeyID(datatypeID, ownerID)
	// check if datatype sym key already exists, just return existing key
	if key_mgmt_c.KeyExists(stub, datatypeSymkeyID) {
		logger.Debugf("Datatype symkey already exist: %v", datatypeSymkeyID)
		var keyPath []string = nil
		if len(keyPathForOwnerSymkey) > 0 && keyPathForOwnerSymkey[0] != nil {
			keyPath = append(keyPathForOwnerSymkey[0], datatypeSymkeyID)
		}
		return GetDatatypeSymKey(stub, caller, datatypeID, ownerID, keyPath)
	}

	// get owner symkey
	ownerSymkey, err := user_mgmt_c.GetUserSymKey(stub, caller, ownerID, keyPathForOwnerSymkey...)
	if err != nil {
		logger.Errorf("Failed to get owner's symkey: %v", err)
		return data_model.Key{}, err
	}
	// create datatype symkey
	ownerSymkeyBytes := ownerSymkey.KeyBytes
	ownerIdBytes := []byte(ownerID)
	datatypeSymKey := crypto.GetSymKeyFromHash(crypto.Hash(append(ownerSymkeyBytes, ownerIdBytes...)))

	// add key and access (read)
	var edgeData = make(map[string]string)
	edgeData[global.EDGEDATA_ACCESS_TYPE] = global.ACCESS_READ
	datatypeKey, _ := key_mgmt_c.ConvertKeyBytesToKey(datatypeSymkeyID, datatypeSymKey)
	err = key_mgmt_c.AddAccess(stub, ownerSymkey, *datatypeKey, edgeData)
	if err != nil {
		logger.Errorf("Failed to add datatype key: %v", err)
		return data_model.Key{}, err
	}

	// if parent exists, call AddDatatypeSymKey() function for the parent datatype
	parents, err := datatype.GetParentDatatypes(stub)
	if err != nil {
		logger.Errorf("Failed to get parent datatypes: %v", err)
		return data_model.Key{}, err
	}
	// there should be only one parent for the current datatype design
	if len(parents) > 0 {
		parentDatatypeSymKey, err := AddDatatypeSymKey(stub, caller, parents[0], ownerID, keyPathForOwnerSymkey...)
		if err != nil {
			logger.Errorf("Failed to add datatype key for parent: %v", err)
			return data_model.Key{}, err
		}
		// check path from parent key to the datatype symkey
		parentSymKeyID := GetDatatypeKeyID(parents[0], ownerID)
		ok, err := key_mgmt_c.VerifyAccessPath(stub, []string{parentSymKeyID, datatypeSymkeyID})
		if err != nil {
			logger.Errorf("Faild to verify access from parent key to the datatype key: %v", err)
			return data_model.Key{}, err
		}
		if !ok {
			// add access from parent datatype to the datatype
			err = key_mgmt_c.AddAccessWithKeys(stub, parentDatatypeSymKey.KeyBytes, parentSymKeyID, datatypeSymKey, datatypeSymkeyID, nil)
			if err != nil {
				logger.Errorf("Failed to add access from parent key to the datatype key: %v", err)
				return data_model.Key{}, err
			}
		}
	}

	return *datatypeKey, nil
}

/*
// GetDatatypeSymKey composes and returns a sym key for the given datatypeID and ownerID.
// Returns an empty key if the datatype is not found or if there is an error.
func GetDatatypeSymKey(stub cached_stub.CachedStubInterface, datatypeID string) (data_model.Key, error) {
	return datatype_c.GetDatatypeSymKey(stub, datatypeID)
}
*/

// GetDatatypeSymKey composes and returns a sym key for the given datatypeID and OwnerID.
// Returns an empty key if the datatype is not found or if there is an error.
func GetDatatypeSymKey(stub cached_stub.CachedStubInterface, caller data_model.User, datatypeID string, ownerID string, keyPath ...[]string) (data_model.Key, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("GetDatatypeSymKey DatatypeID: %v Owner: %v KeyPath: %v", datatypeID, ownerID, keyPath)

	var datatypeSymkey []byte = nil
	var startKey []byte = nil
	var keyIdList []string = []string{}
	var datatypeSymkeyID = GetDatatypeKeyID(datatypeID, ownerID)

	// first check if key exists in cache already. If so, return from cache
	cachekey := getDatatypeCacheKey(caller.ID, datatypeID, ownerID)
	cachedDatatypeKey, err := stub.GetCache(cachekey)
	if err == nil {
		if datatypeKey, ok := cachedDatatypeKey.(data_model.Key); ok {
			if !datatypeKey.IsEmpty() {
				logger.Debugf("Got datatype sym key from cache")
				return datatypeKey, nil
			}
		}
	}

	if len(keyPath) > 0 && keyPath[0] != nil {
		startKey = caller.GetPrivateKey().KeyBytes
		keyIdList = keyPath[0]
	} else {
		// startkey is owner sym key

		ownerSymKey, err := user_mgmt_c.GetUserSymKey(stub, caller, ownerID)
		if err != nil {
			logger.Errorf("Failed to get owner's key: %v", err)
			return data_model.Key{}, err
		}
		if ownerSymKey.IsEmpty() {
			logger.Error("Failed to get owner's key")
			return data_model.Key{}, err
		}
		startKey = ownerSymKey.KeyBytes
		keyIdList = []string{ownerSymKey.ID, datatypeSymkeyID}
	}
	datatypeSymkey, err = key_mgmt_c.GetKey(stub, keyIdList, startKey)
	if err != nil {
		logger.Errorf("failed to get datatype sym key: %v", err)
		return data_model.Key{}, err
	}

	datatypeKey := data_model.Key{ID: datatypeSymkeyID, KeyBytes: datatypeSymkey, Type: global.KEY_TYPE_SYM}

	// put datatypeSymKey into cache
	err = stub.PutCache(cachekey, datatypeKey)
	if err != nil {
		logger.Errorf("Failed to put datatypeKey in cache: %v", err)
		return data_model.Key{}, errors.Wrap(err, "Failed to put datatypeKey in cache")
	}

	return datatypeKey, nil
}

// TODO: remove this after full migration:
// temporarily save the old version of GetDatatypeKeyID for compatibility of
// other package for now
/*
func GetDatatypeSymKey_OLD(stub cached_stub.CachedStubInterface, datatypeID string) (data_model.Key, error) {
	datatype, err := GetDatatypeWithParams(stub, datatypeID)
	if err != nil {
		custom_err := &custom_errors.GetDatatypeError{Datatype: datatypeID}
		logger.Errorf("%v: %v", custom_err, err)
		return data_model.Key{}, errors.Wrap(err, custom_err.Error())
	}
	if utils.IsStringEmpty(datatype.GetDatatypeID()) {
		custom_err := &custom_errors.GetDatatypeError{Datatype: datatypeID}
		logger.Errorf(custom_err.Error())
		return data_model.Key{}, errors.WithStack(custom_err)
	}

	datatypeKey := data_model.Key{ID: GetDatatypeKeyID(datatypeID, ""), Type: global.KEY_TYPE_SYM}
	return datatypeKey, nil
}
*/

// GetParentDatatype returns the datatypeID of a given datatypeID's direct parent.
func GetParentDatatype(stub cached_stub.CachedStubInterface, datatypeID string) (string, error) {
	return datatype_c.GetParentDatatype(stub, datatypeID)
}

// GetParentDatatypes returns a list of parents of the given datatypeID, excluding ROOT datatype.
func GetParentDatatypes(stub cached_stub.CachedStubInterface, datatypeID string) ([]string, error) {
	return datatype_c.GetParentDatatypes(stub, datatypeID)
}

// NormalizeDatatypes returns a list of datatype IDs with only the most specific children.
func NormalizeDatatypes(stub cached_stub.CachedStubInterface, datatypeIDs []string) ([]string, error) {
	return datatype_c.NormalizeDatatypes(stub, datatypeIDs)
}

// getDatatypeCacheKey returns a datatype cache key with given parameters
func getDatatypeCacheKey(callerID string, datatypeID string, ownerID string) string {
	return datatype_c.DATATYPE_CACHE_PREFIX + callerID + datatypeID + ownerID
}
