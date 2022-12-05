/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package consent_mgmt_i provides functionality for sharing assets with other users, groups, or orgs.
package consent_mgmt_i

import (
	"common/bchcls/asset_mgmt/asset_key_func"
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/custom_errors"
	"common/bchcls/data_model"
	"common/bchcls/datatype/datatype_interface"
	"common/bchcls/index"
	"common/bchcls/internal/asset_mgmt_i"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/consent_mgmt_i/consent_mgmt_c"
	"common/bchcls/internal/datatype_i"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/internal/user_mgmt_i"
	"common/bchcls/internal/user_mgmt_i/user_mgmt_c"
	"common/bchcls/simple_rule"
	"common/bchcls/utils"

	"encoding/json"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("consent_mgmt_i")

// Public data of consent object
type consentPublic struct {
	ConsentID      string `json:"consent_id"`
	ConsentAssetID string `json:"consent_asset_id"`
	CreatorID      string `json:"creator_id"`
	OwnerID        string `json:"owner_id"`
	TargetID       string `json:"target_id"`
	DatatypeID     string `json:"datatype_id"`
	AssetKeyID     string `json:"asset_key_id"`
	ConnectionID   string `json:"connection_id"`
}

// ------------------------------------------------------
// ---------------------- INIT FUNCTIONS ----------------
// ------------------------------------------------------

// Init sets up the consent package by building an index table for consents.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	logger.Debug("Init consent_mgmt")
	//Consent Index
	consentTable := index.GetTable(stub, global.INDEX_CONSENT, "consent_asset_id")
	consentTable.AddIndex([]string{"consent_id", "consent_asset_id"}, false)
	consentTable.AddIndex([]string{"owner_id", "target_id", "datatype_id", "consent_asset_id"}, false)
	consentTable.AddIndex([]string{"target_id", "owner_id", "datatype_id", "consent_asset_id"}, false)
	consentTable.AddIndex([]string{"owner_id", "datatype_id", "consent_asset_id"}, false)
	consentTable.AddIndex([]string{"target_id", "datatype_id", "consent_asset_id"}, false)
	consentTable.AddIndex([]string{"creator_id", "consent_asset_id"}, false)
	err := consentTable.SaveToLedger()
	return nil, err
}

// ConsentKeyFunc finds the keypath in an efficient manner, if caller is owner or target of consent,
// or admin of either.
// This function does not handle users who have access through "allowAccess".
// If you have access through "allowAccess", you should instead get the user object and
// call other functions as that user.
var ConsentKeyFunc asset_key_func.AssetKeyPathFunc = func(stub cached_stub.CachedStubInterface, caller data_model.User, consentAsset data_model.Asset) ([]string, error) {

	keyPath := []string{caller.GetPubPrivKeyId()}
	publicData := consentPublic{}
	json.Unmarshal(consentAsset.PublicData, &publicData)

	// check if caller is target
	if caller.ID == publicData.TargetID {
		logger.Debug("Caller is target")
		keyPath = append(keyPath, consentAsset.AssetKeyId)
		return keyPath, nil
	}

	if consentAsset.IsOwner(caller.ID) {
		logger.Debug("Caller is owner of consent")
		//datatype consent
		keyPath = append(keyPath, consentAsset.AssetKeyId)
		return keyPath, nil
	}

	// check if caller is admin of target
	isAdmin, adminPath, _ := user_mgmt_c.IsUserAdminOfGroup(stub, caller.ID, publicData.TargetID)
	if isAdmin {
		logger.Debug("Caller is an admin of targer")
		keyPath, _ = user_mgmt_i.ConvertAdminPathToPrivateKeyPath(adminPath)
		keyPath = append(keyPath, consentAsset.AssetKeyId)
		return keyPath, nil
	}

	// check if caller is admin of consent owner
	isAdmin, adminPath, _ = user_mgmt_c.IsUserAdminOfGroup(stub, caller.ID, consentAsset.OwnerIds[0])
	if isAdmin {
		logger.Debug("Caller is an admin of consent owner: datatype consent")
		keyPath, _ = user_mgmt_i.ConvertAdminPathToPrivateKeyPath(adminPath)
		keyPath = append(keyPath, consentAsset.AssetKeyId)
		return keyPath, nil
	}

	logger.Debug("Failed to get keyPath")
	return nil, nil
}

// PutConsent updates an existing consent or adds a new consent.
// Consent can be given to a datatype (all assets of a particular datatype).
// Caller must either be the owner of the consent or have access to the owner's private key.
//
// args = [consent, consentKeyB64]
//
// consent is the consent object.
// consentKeyB64 is only required when creating a new consent. A unique consent key must be used for each new consent.
func PutConsent(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("callerID: %v, args: %v", caller.ID, args)

	if len(args) != 1 && len(args) != 2 {
		custom_err := &custom_errors.LengthCheckingError{Type: "PutConsent args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	consent := data_model.Consent{}
	consentBytes := []byte(args[0])
	err := json.Unmarshal(consentBytes, &consent)
	if err != nil {
		custom_err := &custom_errors.UnmarshalError{Type: "Consent"}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	consentKeyBytes := []byte{}
	if len(args) == 2 {
		consentKeyB64 := args[1]
		consentKeyBytes, err = crypto.ParseSymKeyB64(consentKeyB64)
		if err != nil {
			logger.Errorf("Invalid consent key: %v", err)
			return nil, errors.Wrap(err, "Invalid consent key")
		}
	}

	return nil, PutConsentWithParams(stub, caller, consent, consentKeyBytes)
}

// PutConsentWithParams updates an existing consent or adds new consent.
// It takes consent object data_model.Consent, and consentKeyBytes []byte
// as arguments instead of args in JSON format.
//
// consent is the consent object.
// consentKeyBytes is only passed when creating a new consent.
// A unique consent key must be used for each new consent.
func PutConsentWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, consent data_model.Consent, consentKeyBytes []byte) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("callerID: %v, consent: %v, consentKey len: %v", caller.ID, consent, len(consentKeyBytes))

	// ==============================================================
	// Validation of incoming consent
	// ==============================================================

	if utils.IsStringEmpty(consent.TargetID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "consent.TargetID"}
		logger.Errorf(custom_err.Error())
		return errors.WithStack(custom_err)
	}

	// consent passed in must contain DatatypeID
	if utils.IsStringEmpty(consent.DatatypeID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "DatatypeID"}
		logger.Errorf(custom_err.Error())
		return errors.WithStack(custom_err)
	}

	if consent.Access != global.ACCESS_READ && consent.Access != global.ACCESS_WRITE && consent.Access != global.ACCESS_DENY {
		logger.Errorf("Access must be write, read, or deny")
		return errors.New("Access must be write, read, or deny")
	}

	// check that consentDate is within 10 mins of current time
	currTime := time.Now().Unix()
	if currTime-consent.ConsentDate > 10*60 || currTime-consent.ConsentDate < -10*60 {
		logger.Errorf("Invalid consentDate (current time: %v)  %v", currTime, consent.ConsentDate)
		return errors.New("Invalid ConsentDate, not within possible time range")
	}

	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)

	existingDatatype, err := datatype_i.GetDatatypeWithParams(stub, consent.DatatypeID)
	if err != nil {
		custom_err := &custom_errors.GetDatatypeError{Datatype: consent.DatatypeID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	if utils.IsStringEmpty(existingDatatype.GetDatatypeID()) {
		custom_err := &custom_errors.GetDatatypeError{Datatype: consent.DatatypeID}
		logger.Errorf(custom_err.Error())
		return errors.WithStack(custom_err)
	}

	// Validate ownerID
	if utils.IsStringEmpty(consent.OwnerID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "consent.OwnerID"}
		logger.Errorf(custom_err.Error())
		return errors.WithStack(custom_err)
	}

	if caller.ID != consent.OwnerID {
		// Verify caller has access to private key of owner
		privKey, err := user_mgmt_i.GetUserPrivateKey(stub, caller, consent.OwnerID)
		if err != nil || len(privKey.KeyBytes) == 0 {
			logger.Errorf("Failed to get private key. Caller does not have access to act on behalf of owner")
			return errors.New("Failed to get private key. Caller does not have access to act on behalf of owner")
		}
	}

	// Only "deny" option can be added if the datatype is inactive
	if !existingDatatype.IsActive() && consent.Access != global.ACCESS_DENY {
		logger.Errorf("You can only add DENY to inactive datatype: %v", consent.DatatypeID)
		return errors.New("You can only add DENY to inactive datatype")
	}

	// Set ConsentID
	consent.ConsentID = GetConsentID(consent.DatatypeID, consent.TargetID, consent.OwnerID)

	// ==============================================================
	// Convert consent to assetData
	// ==============================================================

	// Set CreatorID, CreatorID is always the caller
	consent.CreatorID = caller.ID

	// Init variables to be used
	isNewConsent := true
	consentOld := data_model.Consent{}
	consentKey := data_model.Key{ID: consent.ConsentID, Type: global.KEY_TYPE_SYM}
	consentAssetOld := data_model.Asset{}

	// try to get existing consent asset
	consentAssetId, err := GetConsentAssetID(stub, consent.ConsentID)
	logger.Debugf("existing consentAssetId: %v, err: %v", consentAssetId, err)
	if err == nil && len(consentAssetId) > 0 {
		isNewConsent = false
		//you should have access to consent asset key
		consentKey.KeyBytes, err = getConsentAssetKeyByConsentAssetID(stub, caller, consentAssetId)
		if err != nil {
			logger.Errorf("Failed to getConsentAssetKey with error: %v", err)
			return errors.Wrap(err, "Failed to getConsentAssetKey with error")
		}
		if len(consentKey.KeyBytes) == 0 {
			logger.Errorf("Failed to getConsentAssetKey")
			return errors.Wrap(err, "Failed to getConsentAssetKey:")
		}

		// get existing consent asset
		consentAssetOld, err := asset_mgmt_i.GetAssetManager(stub, caller).GetAsset(consentAssetId, consentKey)
		if err != nil {
			logger.Errorf("Get old consent asset failed: %v", err)
			return errors.Wrap(err, "Get old consent asset failed")
		}
		if data_model.IsEncryptedData(consentAssetOld.PrivateData) {
			logger.Error("Failed to read consent private data")
			return errors.New("Failed to read consent private data")
		}

		consentOld = convertFromAsset(consentAssetOld)

		// consent date only changes if Access level has been changed
		if consent.Access == consentOld.Access {
			consent.ConsentDate = consentOld.ConsentDate
		}

		if consent.ExpirationDate == 0 {
			consent.ExpirationDate = consentOld.ExpirationDate
		}

		// delete old consent asset's index values if caller changed
		if consentOld.CreatorID != consent.CreatorID {
			table := index.GetTable(stub, consentAssetOld.IndexTableName)
			table.DeleteRow(consentOld.ConsentAssetID)
		}
	} else {
		isNewConsent = true
		// Get or create consent key object (CK)
		if len(consentKeyBytes) == 0 {
			logger.Errorf("Invalid consent key bytes")
			return errors.Wrap(err, "Invalid consent keybytes")
		}

		consentKey.KeyBytes = consentKeyBytes
		if !crypto.ValidateSymKey(consentKey.KeyBytes) {
			logger.Errorf("Invalid consent key")
			return errors.New("Invalid consent key")
		}
	}

	logger.Debugf("isNewConsent: %v", isNewConsent)

	// Make consent edge rule, will be same as edge data for consent edge
	edgeData := make(map[string]string)
	edgeData["edge"] = global.CONSENT_EDGE
	edgeData["target"] = consent.TargetID
	edgeData["owner"] = consent.OwnerID
	edgeData["datatypeID"] = consent.DatatypeID

	consent.ConsentAssetID = consent.ConsentID + consent.CreatorID

	// convert consent to assetData
	consentConverted := convertToAsset(consent)

	// ==============================================================
	// Add / update consent
	// ==============================================================

	// SymKey is DatatypeKey
	// Add edges from CK to SymKey, and SymKey to CK if not deny
	symKey := data_model.Key{}

	// If datatypeID is not empty, get datatypeSymKey
	symKey, err = datatype_i.GetDatatypeSymKey(stub, caller, consent.DatatypeID, consent.OwnerID)
	if err != nil {
		logger.Errorf("Failed to GetDatatypeSymKey: %v", err)
		return errors.Wrap(err, "Failed to GetDatatypeSymKey")
	}

	// Check consent access level
	if consent.Access == global.ACCESS_READ || consent.Access == global.ACCESS_WRITE {
		edgeData := make(map[string]string)
		if consent.Access == global.ACCESS_WRITE {
			edgeData[global.EDGEDATA_ACCESS_TYPE] = global.ACCESS_WRITE
		} else {
			edgeData[global.EDGEDATA_ACCESS_TYPE] = global.ACCESS_READ
		}
		// If Access level is read or write, add edge from CK to SymKey
		err = key_mgmt_i.AddAccess(stub, consentKey, symKey, edgeData)
		if err != nil {
			custom_err := &custom_errors.AddAccessError{Key: "assetKey"}
			logger.Errorf("%v: %v", custom_err, err)
			return errors.Wrap(err, custom_err.Error())
		}

		// add edge from owner key to CK
		ownerKey, err := user_mgmt_i.GetUserPublicKey(stub, caller, consent.OwnerID)
		if err != nil {
			errMsg := "Failed to get public key of " + consent.OwnerID
			logger.Errorf("%v: %v", errMsg, err)
			return errors.Wrap(err, errMsg)
		}

		err = key_mgmt_i.AddAccess(stub, ownerKey, consentKey)
		if err != nil {
			custom_err := &custom_errors.AddAccessError{Key: "consentKey"}
			logger.Errorf("%v: %v", custom_err, err)
			return errors.Wrap(err, custom_err.Error())
		}

	} else { // If Access level is deny
		// remove access from CK to SymKey but keep edge from SymKey to CK
		err = key_mgmt_i.RevokeAccess(stub, consentKey.ID, symKey.ID)
		if err != nil {
			logger.Errorf("Failed to revoke access from CK to SymKey: %v", err)
			return errors.Wrap(err, "Failed to revoke access from SymKey to SymKey")
		}
	}

	// Encrypt CK with target's key, create consent edge
	targetKey, err := user_mgmt_i.GetUserPublicKey(stub, caller, consent.TargetID)
	if err != nil {
		errMsg := "Failed to get public key of " + consent.TargetID
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}

	err = key_mgmt_i.AddAccess(stub, targetKey, consentKey, edgeData)
	if err != nil {
		custom_err := &custom_errors.AddAccessError{Key: "consentKey"}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	// If this is an update and creator changed, delete old consent asset in ledger
	if !isNewConsent && consentOld.CreatorID != consent.CreatorID {
		assetLedgerKey := consentAssetOld.AssetId
		err = stub.DelState(assetLedgerKey)
		if err != nil {
			custom_err := &custom_errors.DeleteLedgerError{LedgerKey: assetLedgerKey}
			logger.Errorf("%v: %v", custom_err, err)
			return errors.Wrap(err, custom_err.Error())
		}
	}

	// Add/Update new consent asset
	if !isNewConsent && consentOld.CreatorID == consent.CreatorID {
		err = assetManager.UpdateAsset(consentConverted, consentKey, true)
	} else {
		err = assetManager.AddAsset(consentConverted, consentKey, true)
	}
	if err != nil {
		logger.Errorf("Failed to add Consent: %v", err)
		return errors.Wrap(err, "Failed to add Consent")
	}

	// all set
	return nil
}

// GetConsent returns the specified consent asset.
// Returns an error if no consent is found.
// Caller can be anyone with access to the consent key.
//
// args: [datatypeID, targetID, ownerID]
//
// datatypeID is the id of the consent datatype.
// targetID is the id of the consent target.
// ownerID is the id of the consent owner.
func GetConsent(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	if len(args) != 2 && len(args) != 3 {
		custom_err := &custom_errors.LengthCheckingError{Type: "GetConsent args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	datatypeID := args[0]
	targetID := args[1]

	if utils.IsStringEmpty(datatypeID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "datatypeID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	if utils.IsStringEmpty(targetID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "targetID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	var ownerID string
	if len(args) == 3 {
		ownerID = args[2]
		if utils.IsStringEmpty(ownerID) {
			custom_err := &custom_errors.LengthCheckingError{Type: "ownerID"}
			logger.Errorf(custom_err.Error())
			return nil, errors.WithStack(custom_err)
		}
	}

	consentID := GetConsentID(datatypeID, targetID, ownerID)
	consent, err := GetConsentWithParams(stub, caller, consentID, nil)
	if err != nil {
		custom_err := &custom_errors.GetConsentError{ConsentID: consentID}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	if utils.IsStringEmpty(consent.ConsentID) {
		custom_err := &custom_errors.GetConsentError{ConsentID: consentID}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	consentBytes, err := json.Marshal(consent)
	if err != nil {
		custom_err := &custom_errors.MarshalError{Type: "Consent"}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	return consentBytes, nil
}

// GetConsentWithParams returns the consent asset given the consentID
// ConsentKey is optional if it's not passed in, it will try get consent key using
// ConsentKeyFunc
func GetConsentWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, consentID string, consentKey ...[]byte) (data_model.Consent, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v, consentID: %v", caller.ID, consentID)

	if utils.IsStringEmpty(consentID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "consentID"}
		logger.Errorf(custom_err.Error())
		return data_model.Consent{}, errors.WithStack(custom_err)
	}

	consentKeyBytes := []byte{}
	if len(consentKey) > 0 {
		consentKeyBytes = consentKey[0]
	}
	if len(consentKeyBytes) == 0 {
		var err error
		consentKeyBytes, err = getConsentAssetKeyByConsentID(stub, caller, consentID)
		if err != nil {
			logger.Errorf("Unable to get consent key: %v", err)
			return data_model.Consent{}, errors.Wrap(err, "Unable to get consent key")
		}
	}

	consentAsset, err := getConsentAssetByConsentID(stub, caller, consentID, consentKeyBytes)
	if err != nil {
		logger.Errorf("Get consent failed: %v", err)
		return data_model.Consent{}, errors.Wrap(err, "Get consent failed")
	}

	// Here we want to return nil instead of error because in validate consent,
	// if getting consent failed then we traverse up the datatype tree
	if consentAsset == nil {
		custom_err := &custom_errors.ConsentAssetIsNilError{}
		logger.Errorf(custom_err.Error())
		return data_model.Consent{}, custom_err
	}

	return convertFromAsset(consentAsset), nil
}

// ValidateConsent gets the specified consent asset. If consent is found and if it passes the expiration date and access level checks, it returns filter rules and the consent key.
// Filter rule is a simple rule that contains consent owner ID which can be applied against an asset's owner ID, and either consent asset ID or consent
// datatype ID which can be applied against asset's datatypeID to filter out assets.
//
// args: [datatypeID, ownerID, targetID, access, currTime]
//
// targetID is the ID of the consent recipient.
// access is the desired access level that will be validated against the access recorded in the consent object.
// currTime is the current timestamp generated.
func ValidateConsent(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) (simple_rule.Rule, data_model.Key, error) {
	// Step 1. Get consent asset for given datatypeID. If datatypeID not found, traverse up datatype tree and check consent given to parent datatype, then to grandparent, etc, until reaching ROOT. Return error if ROOT has been reached and no consent has been found.
	// Step 2. Check expiration date and access.
	// Step 3. If successful, return filter rules and consent key.
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	if len(args) != 5 {
		custom_err := &custom_errors.LengthCheckingError{Type: "ValidateConsent args"}
		logger.Errorf(custom_err.Error())
		return simple_rule.NewRule(), data_model.Key{}, errors.WithStack(custom_err)
	}

	datatypeID := args[0]
	ownerID := args[1]
	targetID := args[2]
	access := args[3]

	// ==============================================================
	// Validation of parameters
	// ==============================================================

	if utils.IsStringEmpty(datatypeID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "datatypeID"}
		logger.Errorf(custom_err.Error())
		return simple_rule.NewRule(), data_model.Key{}, errors.WithStack(custom_err)
	}

	if !utils.IsStringEmpty(datatypeID) && utils.IsStringEmpty(ownerID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "ownerID"}
		logger.Errorf(custom_err.Error())
		return simple_rule.NewRule(), data_model.Key{}, errors.WithStack(custom_err)
	}

	if utils.IsStringEmpty(targetID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "targetID"}
		logger.Errorf(custom_err.Error())
		return simple_rule.NewRule(), data_model.Key{}, errors.WithStack(custom_err)
	}

	if utils.IsStringEmpty(access) {
		custom_err := &custom_errors.LengthCheckingError{Type: "access"}
		logger.Errorf(custom_err.Error())
		return simple_rule.NewRule(), data_model.Key{}, errors.WithStack(custom_err)
	}

	currTime, err := strconv.ParseInt(args[4], 10, 64)
	if err != nil {
		logger.Errorf("Error converting curr time to type int64")
		return simple_rule.NewRule(), data_model.Key{}, errors.Wrap(err, "Error converting curr time to type int64")
	}

	now := time.Now().Unix()
	if now-currTime > 10*60 || now-currTime < -10*60 {
		logger.Errorf("Invalid current time (actual current time: %v)  %v", now, now)
		return simple_rule.NewRule(), data_model.Key{}, errors.New("Invalid current time, not within possible time range")
	}

	var consentID string
	var dtype datatype_interface.DatatypeInterface
	dtype, err = datatype_i.GetDatatypeWithParams(stub, datatypeID)
	if err != nil {
		custom_err := &custom_errors.GetDatatypeError{Datatype: datatypeID}
		logger.Errorf("%v: %v", custom_err, err)
		return simple_rule.NewRule(), data_model.Key{}, errors.Wrap(err, custom_err.Error())
	}

	if utils.IsStringEmpty(dtype.GetDatatypeID()) {
		custom_err := &custom_errors.GetDatatypeError{Datatype: datatypeID}
		logger.Errorf(custom_err.Error())
		return simple_rule.NewRule(), data_model.Key{}, errors.WithStack(custom_err)
	}

	consentID, _, err = GetConsentIDForDatatype(stub, ownerID, targetID, datatypeID)
	if err != nil {
		logger.Errorf("Unable to find consent for the datatype: %v", err)
		return simple_rule.NewRule(), data_model.Key{}, errors.Wrap(err, "Unable to find consent for the datatype")
	}

	// ==============================================================
	// Get consent asset and check fields
	// ==============================================================

	// get consent key
	consentKeyBytes, err := getConsentAssetKeyByConsentID(stub, caller, consentID)
	if err != nil {
		logger.Errorf("Failed to get consent key: %v", err)
		return simple_rule.NewRule(), data_model.Key{}, errors.Wrap(err, "Failed to get consent key")
	}

	consent, err := GetConsentWithParams(stub, caller, consentID, consentKeyBytes)
	if err != nil {
		custom_err := &custom_errors.GetConsentError{ConsentID: consentID}
		logger.Errorf(custom_err.Error())
		return simple_rule.NewRule(), data_model.Key{}, errors.WithStack(custom_err)
	}
	if utils.IsStringEmpty(consent.ConsentID) {
		custom_err := &custom_errors.GetConsentError{ConsentID: consentID}
		logger.Errorf(custom_err.Error())
		return simple_rule.NewRule(), data_model.Key{}, errors.WithStack(custom_err)
	}

	// Check consent access level
	if consent.Access == global.ACCESS_DENY {
		custom_err := &custom_errors.ConsentAccessError{ConsentId: consentID}
		logger.Errorf(custom_err.Error())
		return simple_rule.NewRule(), data_model.Key{}, custom_err
	}

	if consent.Access != access && consent.Access != global.ACCESS_WRITE {
		custom_err := &custom_errors.ConsentAccessError{ConsentId: consentID}
		logger.Errorf(custom_err.Error())
		return simple_rule.NewRule(), data_model.Key{}, custom_err
	}

	// Check expiration
	if consent.ExpirationDate != 0 && consent.ExpirationDate-currTime <= 0 {
		logger.Errorf("Expiration date has passed")
		return simple_rule.NewRule(), data_model.Key{}, errors.New("Expiration date has passed")
	}

	// ==============================================================
	// Return filters and consent key
	// ==============================================================

	// get consent key
	consentKey := data_model.Key{}
	consentKey.ID = consent.ConsentID
	consentKey.KeyBytes = consentKeyBytes
	consentKey.Type = global.KEY_TYPE_SYM

	// get filter rule
	filter := simple_rule.NewRule()
	// datatype list contains all children of datatype
	datatypeList, err := dtype.GetChildDatatypes(stub)
	if err != nil {
		custom_err := &custom_errors.GetChildDatatypesError{Datatype: datatypeID}
		logger.Errorf("%v: %v", custom_err, err)
		return simple_rule.NewRule(), data_model.Key{}, errors.Wrap(err, custom_err.Error())
	}

	// Prepend datatypeID to datatype list
	datatypeList = append([]string{dtype.GetDatatypeID()}, datatypeList...)

	// Filter rule: datatypes [datatype + all subdatatypes] must have an intersection with asset's list of datatypes
	// AND consent's ownerID must be in asset's list of owners
	filter = simple_rule.NewRule(simple_rule.R("and",
		simple_rule.R("!=",
			simple_rule.R("len",
				simple_rule.R("filter",
					datatypeList,
					simple_rule.R("in",
						simple_rule.R("var", "$current"),
						simple_rule.R("var", "datatypes")))), 0),
		simple_rule.R("in", consent.OwnerID, simple_rule.R("var", "owner_ids"))))

	return filter, consentKey, nil
}

// GetConsentsWithOwnerID returns a list of consents, sorted by ownerID.
//
// args: [ownerID]
func GetConsentsWithOwnerID(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	if len(args) != 1 {
		custom_err := &custom_errors.LengthCheckingError{Type: "GetConsentsWithOwnerID args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	ownerID := args[0]
	if utils.IsStringEmpty(ownerID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "ownerID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	consents, err := getConsents(stub, caller, []string{"owner_id"}, []string{ownerID}, nil)
	if err != nil {
		custom_err := &custom_errors.GetConsentsError{SortOrder: []string{"owner_id"}, PartialKeyList: []string{ownerID}}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	return json.Marshal(&consents)
}

// GetConsentsWithTargetID returns a list of consents, sorted by targetID.
//
// args: [targetID]
func GetConsentsWithTargetID(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	if len(args) != 1 {
		custom_err := &custom_errors.LengthCheckingError{Type: "GetConsentsWithTargetID args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	targetID := args[0]
	if utils.IsStringEmpty(targetID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "targetID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	consents, err := getConsents(stub, caller, []string{"target_id"}, []string{targetID}, nil)
	if err != nil {
		custom_err := &custom_errors.GetConsentsError{SortOrder: []string{"target_id"}, PartialKeyList: []string{targetID}}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	return json.Marshal(&consents)
}

// GetConsentsWithCallerID returns a list of consents created by the caller.
//
// args: []
func GetConsentsWithCallerID(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	consents, err := getConsents(stub, caller, []string{"creator_id"}, []string{caller.ID}, nil)
	if err != nil {
		custom_err := &custom_errors.GetConsentsError{SortOrder: []string{"creator_id"}, PartialKeyList: []string{caller.ID}}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	return json.Marshal(&consents)
}

// GetConsentsWithOwnerIDAndDatatypeID returns a list of consents, sorted by ownerID and datatypeID.
//
// args: [ownerID, datatypeID]
func GetConsentsWithOwnerIDAndDatatypeID(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	if len(args) != 2 {
		custom_err := &custom_errors.LengthCheckingError{Type: "GetConsentsWithOwnerIDAndDatatypeID args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	ownerID := args[0]
	if utils.IsStringEmpty(ownerID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "ownerID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	datatypeID := args[1]
	if utils.IsStringEmpty(datatypeID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "datatypeID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	consents, err := getConsents(stub, caller, []string{"owner_id", "datatype_id"}, []string{ownerID, datatypeID}, nil)
	if err != nil {
		custom_err := &custom_errors.GetConsentsError{SortOrder: []string{"owner_id", "datatype_id"}, PartialKeyList: []string{ownerID, datatypeID}}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	return json.Marshal(&consents)
}

// GetConsentsWithTargetIDAndDatatypeID returns a list of consents, sorted by targetID and datatypeID.
//
// args: [targetID, datatypeID]
func GetConsentsWithTargetIDAndDatatypeID(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	if len(args) != 2 {
		custom_err := &custom_errors.LengthCheckingError{Type: "GetConsentsWithTargetIDAndDatatypeID args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	targetID := args[0]
	if utils.IsStringEmpty(targetID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "targetID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	datatypeID := args[1]
	if utils.IsStringEmpty(datatypeID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "datatypeID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	consents, err := getConsents(stub, caller, []string{"target_id", "datatype_id"}, []string{targetID, datatypeID}, nil)
	if err != nil {
		custom_err := &custom_errors.GetConsentsError{SortOrder: []string{"target_id", "datatype_id"}, PartialKeyList: []string{targetID, datatypeID}}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	return json.Marshal(&consents)
}

// GetConsentsWithTargetIDAndOwnerID returns a list of consents, sorted by targetID and ownerID.
//
// args: [targetID, ownerID]
func GetConsentsWithTargetIDAndOwnerID(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	if len(args) != 2 {
		custom_err := &custom_errors.LengthCheckingError{Type: "GetConsentsWithTargetIDAndOwnerID args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	targetID := args[0]
	if utils.IsStringEmpty(targetID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "targetID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	ownerID := args[1]
	if utils.IsStringEmpty(ownerID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "ownerID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	consents, err := getConsents(stub, caller, []string{"target_id", "owner_id"}, []string{targetID, ownerID}, nil)
	if err != nil {
		custom_err := &custom_errors.GetConsentsError{SortOrder: []string{"target_id", "owner_id"}, PartialKeyList: []string{targetID, ownerID}}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	return json.Marshal(&consents)
}

// GetConsentID returns the consent_id.
func GetConsentID(datatypeID string, targetID string, ownerID string) string {
	return consent_mgmt_c.GetConsentID(datatypeID, targetID, ownerID)
}

// GetConsentIDForDatatype finds consent ID by checking datatype and parents of this datatype_i.
// returns consentID, consentAssetID, err
// returns "" if no matching consent is found
func GetConsentIDForDatatype(stub cached_stub.CachedStubInterface, ownerID string, targetID string, datatypeID string) (string, string, error) {

	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("ownerID: %v, targetID: %v, datatypeID: %v", ownerID, targetID, datatypeID)
	// check cache
	cachekey := "consentID-" + ownerID + "-" + targetID + "-" + datatypeID
	cache, err := stub.GetCache(cachekey)
	if err == nil {
		if cachedata, ok := cache.([]string); ok {
			logger.Debugf("ConsentID return from cache")
			return cachedata[0], cachedata[1], nil
		}
	}

	consentID := GetConsentID(datatypeID, targetID, ownerID)
	consentAssetID, err := GetConsentAssetID(stub, consentID)
	if err == nil && len(consentAssetID) > 0 {
		return consentID, consentAssetID, nil
	}

	parent, err := datatype_i.GetParentDatatype(stub, datatypeID)
	for err == nil && len(parent) > 0 {
		currID := parent
		consentID = GetConsentID(currID, targetID, ownerID)
		consentAssetID, err = GetConsentAssetID(stub, consentID)
		if err == nil && len(consentAssetID) > 0 {
			//add to cache
			stub.PutCache(cachekey, []string{consentID, consentAssetID})
			return consentID, consentAssetID, nil
		}

		parent, err = datatype_i.GetParentDatatype(stub, currID)
	}

	//add to cache
	if err == nil {
		stub.PutCache(cachekey, []string{"", ""})
	}
	return "", "", err
}

// GetConsentAssetID returns consent asset ID from consent ID.
// The returned consent asset ID can be used to get consent asset using asset_mgmt.
func GetConsentAssetID(stub cached_stub.CachedStubInterface, consentID string) (string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("consentID: %v", consentID)
	cachekey := "concentAssetId-" + consentID
	cache, err := stub.GetCache(cachekey)
	if err == nil {
		if cachedata, ok := cache.(string); ok {
			logger.Debugf("consentAssetID return from cache")
			return cachedata, nil
		}
	}

	table := index.GetTable(stub, global.INDEX_CONSENT, "consent_asset_id")
	row := make(map[string]string)
	iter, err := table.GetRowsByPartialKey([]string{"consent_id"}, []string{consentID})
	if err != nil {
		logger.Debugf("iter error %v", err)
		return "", errors.Wrap(err, "iter error")
	}
	defer iter.Close()
	if iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("Failed to get consent: %v", err)
			return "", errors.Wrap(err, "Failed to get consent index")
		}
		rowBytes := KV.GetValue()
		err = json.Unmarshal(rowBytes, &row)
	} else {
		logger.Errorf("Consent index with consentID %v does not exist", consentID)
		return "", errors.New("Consent index with consentID " + consentID + " does not exist")
	}

	consentAssetID := asset_mgmt_i.GetAssetId(global.CONSENT_ASSET_NAMESPACE, row["consent_asset_id"])
	//save cache
	stub.PutCache(cachekey, consentAssetID)
	return consentAssetID, nil
}

// getConsentAssetKeyByConsentID gets consentAssetKey using ConsentKeyFunc.
func getConsentAssetKeyByConsentID(stub cached_stub.CachedStubInterface, caller data_model.User, consentID string) ([]byte, error) {
	consentAssetID, err := GetConsentAssetID(stub, consentID)
	if err != nil {
		logger.Debugf("Unable to get consentAssetID: %v", err)
		return nil, errors.Wrap(err, "Unable to get consentAssetID")
	}
	return getConsentAssetKeyByConsentAssetID(stub, caller, consentAssetID)
}

func getConsentAssetKeyByConsentAssetID(stub cached_stub.CachedStubInterface, caller data_model.User, consentAssetID string) ([]byte, error) {
	//check cache
	cachekey := "concentAssetKey-" + caller.ID + consentAssetID
	cache, err := stub.GetCache(cachekey)
	if err == nil {
		if cachedata, ok := cache.([]byte); ok {
			if len(cachedata) == 0 {
				logger.Debug("ConsentAssetKey error return from cached")
				return nil, errors.New("ConsentAssetKey error return from cached")
			}
			logger.Debugf("ConsentAssetKey return from cache")
			dataCopy := make([]byte, len(cachedata))
			copy(dataCopy, cachedata)
			return dataCopy, nil
		}
	}

	consentAsset, err := asset_mgmt_i.GetEncryptedAssetData(stub, consentAssetID)
	if err != nil {
		logger.Debugf("Failed to get consent asset: %v", err)
		stub.PutCache(cachekey, []byte{})
		return nil, errors.Wrap(err, "Failed to get consent asset")
	}
	// get keyPath using consent key func
	keyPath, err := ConsentKeyFunc(stub, caller, consentAsset)
	if err != nil {
		logger.Debugf("Failed to get consent key path: %v", err)
		stub.PutCache(cachekey, []byte{})
		return nil, err
	}

	startKey := caller.GetPrivateKey()
	assetKey, err := key_mgmt_i.GetKey(stub, keyPath, startKey.KeyBytes)

	if err == nil && len(assetKey) > 0 {
		datacopy := make([]byte, len(assetKey))
		copy(datacopy, assetKey)
		stub.PutCache(cachekey, datacopy)
	} else {

		stub.PutCache(cachekey, []byte{})
	}

	return assetKey, err
}

// getConsentAssetByConsentID gets consent by consent_id using index.
// If no consent is found, returns nil.
// If fails to decrypt private data, returns an error.
func getConsentAssetByConsentID(stub cached_stub.CachedStubInterface, caller data_model.User, consentID string, consentKeyBytes []byte) (*data_model.Asset, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v, consentID: %v, consentKeyBytes length:%v", caller.ID, consentID, len(consentKeyBytes))

	if len(consentKeyBytes) == 0 {
		logger.Error("ConsentKey is requried")
		return nil, errors.New("ConsentKey is required")
	}
	consentKey := data_model.Key{ID: consentID, Type: global.KEY_TYPE_SYM}
	consentKey.KeyBytes = consentKeyBytes

	// try to get existing consent asset
	consentAssetId, err := GetConsentAssetID(stub, consentID)
	logger.Debugf("existing consentAssetId: %v, err: %v", consentAssetId, err)
	if err == nil && len(consentAssetId) > 0 {
		// get existing consent asset
		consentAssetOld, err := asset_mgmt_i.GetAssetManager(stub, caller).GetAsset(consentAssetId, consentKey)
		if err != nil {
			logger.Errorf("Get old consent asset failed: %v", err)
			return nil, errors.Wrap(err, "Get old consent asset failed")
		}
		if data_model.IsEncryptedData(consentAssetOld.PrivateData) {
			logger.Error("Failed to read consent private data")
			return nil, errors.New("Failed to read consent private data")
		}

		return consentAssetOld, nil
	} else {
		logger.Debug("Consent does not exist")
	}

	return nil, nil
}

// convertToAsset converts a consent to an assetData.
func convertToAsset(consent data_model.Consent) data_model.Asset {
	asset := data_model.Asset{}
	asset.AssetId = asset_mgmt_i.GetAssetId(global.CONSENT_ASSET_NAMESPACE, consent.ConsentAssetID)
	asset.AssetKeyId = consent.ConsentID
	asset.Datatypes = []string{}

	metaData := make(map[string]string)
	metaData["namespace"] = global.CONSENT_ASSET_NAMESPACE
	asset.Metadata = metaData

	publicData := consentPublic{
		ConsentID:      consent.ConsentID,
		ConsentAssetID: consent.ConsentAssetID,
		CreatorID:      consent.CreatorID,
		OwnerID:        consent.OwnerID,
		TargetID:       consent.TargetID,
		DatatypeID:     consent.DatatypeID,
		AssetKeyID:     consent.AssetKeyID,
		ConnectionID:   consent.ConnectionID,
	}

	asset.PublicData, _ = json.Marshal(&publicData)
	asset.PrivateData, _ = json.Marshal(&consent)
	asset.IndexTableName = global.INDEX_CONSENT

	// if an off-chain datastore is specified, save the id so that the asset can be saved there
	if !utils.IsStringEmpty(consent.ConnectionID) {
		asset.SetDatastoreConnectionID(consent.ConnectionID)
	}

	return asset
}

// convertFromAsset converts an assetData to a consent.
func convertFromAsset(asset *data_model.Asset) data_model.Consent {
	consent := data_model.Consent{}
	json.Unmarshal(asset.PrivateData, &consent)
	if datastoreConnectionID, ok := asset.Metadata[global.DATASTORE_CONNECTION_ID_METADATA_KEY]; ok {
		consent.ConnectionID = datastoreConnectionID
	}

	return consent
}

// getConsents gets consents.
func getConsents(stub cached_stub.CachedStubInterface, caller data_model.User, sortOrder []string, partialKeyList []string, consentKey interface{}) ([]data_model.Consent, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v, sortOrder: %v, partialKeyList:%v, consentKey:%v", caller.ID, sortOrder, partialKeyList, consentKey)

	if consentKey == nil {
		consentKey = ConsentKeyFunc
	}

	consents := []data_model.Consent{}

	// Use index to find all consents
	iter, err := asset_mgmt_i.GetAssetManager(stub, caller).GetAssetIter(
		global.CONSENT_ASSET_NAMESPACE,
		global.INDEX_CONSENT,
		sortOrder,
		partialKeyList,
		partialKeyList,
		true,
		true,
		consentKey,
		"", -1, nil)
	if err != nil {
		logger.Errorf("GetAssets failed: %v", err)
		return nil, errors.Wrap(err, "GetAssets failed")
	}
	// Iterate over all consents
	defer iter.Close()
	for iter.HasNext() {
		consentAsset, err := iter.Next()
		if err != nil {
			custom_err := &custom_errors.IterError{}
			logger.Errorf("%v: %v", custom_err, err)
			continue
		}
		consent := convertFromAsset(consentAsset)
		consents = append(consents, consent)
	}

	return consents, nil
}
