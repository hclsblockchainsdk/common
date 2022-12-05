/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package user_access_ctrl_i handles access to assets and keys.
// Read access is granted by adding an edge to the key graph.
// Write access is granted by adding the user as an owner of the asset.
// It is the caller's responsibility to call CheckAccess before updating an asset.
package user_access_ctrl_i

import (
	"encoding/json"

	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/custom_errors"
	"common/bchcls/data_model"
	"common/bchcls/internal/asset_mgmt_i"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/user_access_ctrl/user_access_manager"
	"common/bchcls/user_mgmt"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

// userAccessManagerImpl is the default implementation of the UserAccessManager interface.
type userAccessManagerImpl struct {
	stub   cached_stub.CachedStubInterface
	caller data_model.User
}

var logger = shim.NewLogger("user_access_ctrl")

// ------------------------------------------------------
// ----------------- EXPORTED FUNCTIONS -----------------
// ------------------------------------------------------

// GetUserAccessManager constructs and returns an userAccessManagerImpl instance.
func GetUserAccessManager(stub cached_stub.CachedStubInterface, caller data_model.User) user_access_manager.UserAccessManager {
	// need to verify you are actually whom you claim you are.
	// Do this by get caller object, and check user identity is same.
	user, err := user_mgmt.GetUserData(stub, caller, caller.ID, false, false)
	if err != nil {
		logger.Errorf("Invalid caller %v", err)
		return userAccessManagerImpl{stub: stub}
	}
	if !caller.IsSameUser(user) {
		logger.Errorf("Invalid caller")
		return userAccessManagerImpl{stub: stub}
	}

	return userAccessManagerImpl{stub: stub, caller: caller}
}

// GetStub documentation can be found in user_access_ctrl_interfaces.go.
func (userAccessManager userAccessManagerImpl) GetStub() shim.ChaincodeStubInterface {
	return userAccessManager.stub
}

// GetCaller documentation can be found in user_access_ctrl_interfaces.go.
func (userAccessManager userAccessManagerImpl) GetCaller() data_model.User {
	return userAccessManager.caller
}

// AddAccess documentation can be found in user_access_ctrl_interfaces.go.
func (userAccessManager userAccessManagerImpl) AddAccess(accessControl data_model.AccessControl) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("userID: %v, assetID: %v, access: %v", accessControl.UserId, accessControl.AssetId, accessControl.Access)
	if len(userAccessManager.caller.ID) == 0 {
		logger.Errorf("Invalid Caller")
		return errors.New("Invalid caller")
	}
	return asset_mgmt_i.GetAssetManager(userAccessManager.stub, userAccessManager.caller).AddAccessToAsset(accessControl)
}

// AddAccessByKey documentation can be found in user_access_ctrl_interfaces.go.
func (userAccessManager userAccessManagerImpl) AddAccessByKey(startKey data_model.Key, targetKey data_model.Key) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	if len(userAccessManager.caller.ID) == 0 {
		return errors.New("Invalid caller")
	}
	var edgeData = make(map[string]string)
	edgeData[global.EDGEDATA_ACCESS_TYPE] = global.ACCESS_READ
	err := key_mgmt_i.AddAccess(userAccessManager.stub, startKey, targetKey, edgeData)
	if err != nil {
		logger.Error("Unable to add access")
		return errors.Wrap(err, "Unable to add access")
	}

	return nil
}

// RemoveAccess documentation can be found in user_access_ctrl_interfaces.go.
func (userAccessManager userAccessManagerImpl) RemoveAccess(accessControl data_model.AccessControl) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("userID: %v, assetID: %v, access: %v", accessControl.UserId, accessControl.AssetId, accessControl.Access)
	if len(userAccessManager.caller.ID) == 0 {
		logger.Errorf("Invalid Caller")
		return errors.New("Invalid caller")
	}
	return asset_mgmt_i.GetAssetManager(userAccessManager.stub, userAccessManager.caller).RemoveAccessFromAsset(accessControl)
}

// RemoveAccessByKey documentation can be found in user_access_ctrl_interfaces.go.
func (userAccessManager userAccessManagerImpl) RemoveAccessByKey(startKeyID string, targetKeyID string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	if len(userAccessManager.caller.ID) == 0 {
		logger.Errorf("Invalid Caller")
		return errors.New("Invalid caller")
	}
	if utils.IsStringEmpty(startKeyID) {
		logger.Error("Empty startKeyID")
		return errors.New("Empty startKeyID")
	}
	if utils.IsStringEmpty(targetKeyID) {
		logger.Error("Empty targetKey.ID")
		return errors.New("Empty targetKey.ID")
	}
	err := key_mgmt_i.RevokeAccess(userAccessManager.stub, startKeyID, targetKeyID)
	if err != nil {
		logger.Error("Unable to remove access")
		return errors.Wrap(err, "Unable to remove access")
	}
	return nil
}

// CheckAccess documentation can be found in user_access_ctrl_interfaces.go.
func (userAccessManager userAccessManagerImpl) CheckAccess(accessControl data_model.AccessControl) (bool, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("userID: %v, assetID: %v, access: %v", accessControl.UserId, accessControl.AssetId, accessControl.Access)
	if len(userAccessManager.caller.ID) == 0 {
		return false, errors.New("Invalid caller")
	}
	return asset_mgmt_i.GetAssetManager(userAccessManager.stub, userAccessManager.caller).CheckAccessToAsset(accessControl)
}

// GetAccessData documentation can be found in user_access_ctrl_interfaces.go.
func (userAccessManager userAccessManagerImpl) GetAccessData(userId string, assetId string) (*data_model.AccessControl, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	if len(userAccessManager.caller.ID) == 0 {
		logger.Errorf("Invalid Caller")
		return nil, errors.New("Invalid caller")
	}
	accessControl := data_model.AccessControl{}
	accessControl.UserId = userId
	accessControl.AssetId = assetId
	accessControl.Access = global.ACCESS_WRITE

	// First check for write-access
	hasAccess, err := userAccessManager.CheckAccess(accessControl)
	if err != nil {
		logger.Errorf("CheckAccess failed for userId \"%v\" and assetId \"%v\": %v", userId, assetId, err)
		return nil, errors.Wrapf(err, "CheckAccess failed for userId \"%v\" and assetId \"%v\"", userId, assetId)
	}
	if hasAccess {
		return &accessControl, nil
	}

	// Now check for read-access
	accessControl.Access = global.ACCESS_READ
	hasAccess, err = userAccessManager.CheckAccess(accessControl)
	if err != nil {
		logger.Errorf("CheckAccess failed for userId \"%v\" and assetId \"%v\": %v", userId, assetId, err)
		return nil, errors.Wrapf(err, "CheckAccess failed for userId \"%v\" and assetId \"%v\"", userId, assetId)
	}
	if hasAccess {
		return &accessControl, nil
	}

	return nil, nil
}

// SlowCheckAccessToKey documentation can be found in user_access_ctrl_interfaces.go.
func (userAccessManager userAccessManagerImpl) SlowCheckAccessToKey(targetKeyID string) ([]string, data_model.AccessControlFilters, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	if len(userAccessManager.caller.ID) == 0 {
		logger.Errorf("Invalid Caller")
		return nil, data_model.AccessControlFilters{}, errors.New("Invalid caller")
	}
	callerKeyID := userAccessManager.caller.GetPubPrivKeyId()
	filters := data_model.AccessControlFilters{}
	if len(callerKeyID) == 0 || len(targetKeyID) == 0 {
		custom_err := errors.WithStack(&custom_errors.LengthCheckingError{})
		logger.Errorf("%v", custom_err)
		return nil, filters, custom_err
	}
	visited := make(map[string]bool)

	path, filters, err := dfsKeyGraph(userAccessManager.stub, userAccessManager.caller, callerKeyID, targetKeyID, visited, filters)
	if err != nil {
		logger.Errorf("Error finding path to target key: %v", err)
		return nil, filters, errors.Wrap(err, "Error finding path")
	}
	return path, filters, err
}

// dfsKeyGraph is a helper function for CheckAccessToKey.
func dfsKeyGraph(stub cached_stub.CachedStubInterface, caller data_model.User, currNodeID string, targetNodeID string, visited map[string]bool, filters data_model.AccessControlFilters) ([]string, data_model.AccessControlFilters, error) {

	visited[currNodeID] = true

	// if found, return
	if currNodeID == targetNodeID {
		return []string{currNodeID}, filters, nil
	}

	// get child edges by paritial composite key with curr node id
	iter, err := key_mgmt_i.GetStateByPartialCompositeKey(stub, []string{currNodeID})
	if err != nil {
		custom_err := errors.WithStack(&custom_errors.GetLedgerError{LedgerKey: currNodeID, LedgerItem: "child edges"})
		logger.Errorf("%v: %v", custom_err, err)
		return nil, filters, errors.Wrap(err, custom_err.Error())
	}

	defer iter.Close()
	for iter.HasNext() {
		// examine next child edge
		KV, err := iter.Next()
		if err != nil {
			custom_err := errors.WithStack(&custom_errors.IterError{})
			logger.Errorf("%v: %v", custom_err, err)
			continue
		}

		item := KV.GetKey()
		_, attributes, err := stub.SplitCompositeKey(item)
		nextNodeID := attributes[1]
		if visited[nextNodeID] == true {
			continue
		}

		// call dfs recursively on child, pass along filter rules
		path, nextFilters, err := dfsKeyGraph(stub, caller, nextNodeID, targetNodeID, visited, filters)
		if err != nil {
			logger.Errorf("Error calling DFS with child edge targetNodeID: %v", err)
			continue
		}

		// When targetNodeID has been found
		if path != nil {
			// return the result of previous call with this call to construct path
			return append([]string{currNodeID}, path...), nextFilters, nil
		}
	}

	// No child edges or no path has been found
	return nil, filters, nil
}

// parseConsentFilter gets the required data from consent dataEdge filter.
func parseConsentFilter(filter map[string]string) (string, string, string, error) {
	ownerID := parseIDFromFilterString(filter["ownerFilter"])
	if utils.IsStringEmpty(ownerID) {
		return "", "", "", errors.New("Missing ownerID.")
	}

	assetID := parseIDFromFilterString(filter["assetFilter"])
	datatypeID := parseIDFromFilterString(filter["datatypeFilter"])

	if utils.IsStringEmpty(assetID) && utils.IsStringEmpty(datatypeID) {
		return "", "", "", errors.New("Missing assetID/datatypeID.")
	} else if utils.IsStringEmpty(assetID) {
		return "DATATYPE", datatypeID, ownerID, nil
	} else {
		return "ASSET", assetID, ownerID, nil
	}
}

// parseIDFromFilterString parses ID from a single filter string, e.g. `{"==": [{"var": "assetID"}, "asset1"]}`.
func parseIDFromFilterString(filterString string) string {
	if utils.IsStringEmpty(filterString) {
		return ""
	}
	filterMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(filterString), &filterMap)
	if err != nil {
		logger.Errorf("Error calling parsingIDFromFilterString: %v", err)
		return ""
	}
	id := filterMap["=="].([]interface{})[1]
	return id.(string)
}

// appendFilters adds filters returned from ValidateConsent to current filter list.
func appendFilters(filters *data_model.AccessControlFilters, filter map[string]string) {
	filters.AssetFilters = append(filters.AssetFilters, filter["assetFilter"])
	filters.OwnerFilters = append(filters.OwnerFilters, filter["ownerFilter"])
	filters.DatatypeFilters = append(filters.DatatypeFilters, filter["datatypeFilter"])
}

// GetKey documentation can be found in user_access_ctrl_interfaces.go.
func (userAccessManager userAccessManagerImpl) GetKey(keyID string, keyPath []string) (data_model.Key, error) {
	callerKeyID := userAccessManager.caller.GetPubPrivKeyId()
	callerKeyBytes := crypto.PrivateKeyToBytes(userAccessManager.caller.PrivateKey)
	keyBytes, err := key_mgmt_i.SlowVerifyAccessAndGetKey(userAccessManager.stub, callerKeyID, callerKeyBytes, keyID)
	if err != nil {
		custom_err := &custom_errors.VerifyAccessAndGetKeyError{}
		logger.Infof("%v: %v", custom_err, err)
		return data_model.Key{}, errors.Wrap(err, custom_err.Error())
	}
	key, err := key_mgmt_i.ConvertKeyBytesToKey(keyID, keyBytes)
	if err != nil {
		logger.Errorf("Failed to ConvertKeyBytesToKey: %v", err)
		return data_model.Key{}, errors.Wrap(err, "Failed to ConvertKeyBytesToKey")
	}
	return *key, nil
}
