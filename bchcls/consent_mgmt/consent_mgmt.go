/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package consent_mgmt provides functionality for sharing assets with other users, groups, or orgs.
package consent_mgmt

import (
	"common/bchcls/asset_mgmt/asset_key_func"
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/consent_mgmt_i"
	"common/bchcls/internal/metering_i"
	"common/bchcls/simple_rule"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("consent_mgmt")

///////////////////////////////////////////////////////
// Access control constants

// ACCESS_READ is a Consent.Access option that specifies read access.
const ACCESS_READ = global.ACCESS_READ

// ACCESS_WRITE is a Consent.Access option that specifies write access.
const ACCESS_WRITE = global.ACCESS_WRITE

// ACCESS_DENY is a Consent.Access option that specifies deny access.
const ACCESS_DENY = global.ACCESS_DENY

// ------------------------------------------------------
// ---------------------- INIT FUNCTIONS ----------------
// ------------------------------------------------------

// Init sets up the consent package by building an index table for consents.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	logger.Debug("Init consent")
	return consent_mgmt_i.Init(stub, logLevel...)
}

// ConsentKeyFunc helps find a keypath to an asset if the caller is consent owner, consent
// target, an admin of consent owner, or an admin of consent target.
// It does not handle users who have been given access via user_access_ctrl package.
var ConsentKeyFunc asset_key_func.AssetKeyPathFunc = consent_mgmt_i.ConsentKeyFunc

// PutConsent updates an existing consent or adds a new consent.
// Consent can be given to a datatype (all assets of a particular datatype).
// Caller must either be consent owner or have access to the owner's private key.
//
// args = [consent, consentKeyB64]
//
// consent is the consent object.
// consentKeyB64 is only passed when creating a new consent. A unique consent key must be used for each new consent.
func PutConsent(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("callerID: %v, args: %v", caller.ID, args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return consent_mgmt_i.PutConsent(stub, caller, args)
}

// PutConsentWithParams updates an existing consent or adds new consent.
// It takes consent object data_model.Consent and consentKeyBytes as arguments instead of an args string slice.
// "WithParams" functions should only be called from within the chaincode.
//
// consent is the consent object.
// consentKeyBytes is only passed when creating a new consent. A unique consent key must be used for each new consent.
func PutConsentWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, consent data_model.Consent, consentKeyBytes []byte) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("callerID: %v, consent: %v, consentKey len: %v", caller.ID, consent, len(consentKeyBytes))

	_ = metering_i.SetEnvAndAddRow(stub)

	return consent_mgmt_i.PutConsentWithParams(stub, caller, consent, consentKeyBytes)
}

// GetConsent returns the specified consent asset.
// Returns an error if no consent is found.
// Caller can be anyone with access to the consent key.
//
// args: [datatypeID, targetID, ownerID]
//
// datatypeID is the id of the consent datatypeID.
// targetID is the id of the consent target.
// ownerID is the id of the owner of a datatype consent.
func GetConsent(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return consent_mgmt_i.GetConsent(stub, caller, args)
}

// GetConsentWithParams returns the consent asset given the consentID.
// consentKey is optional. If it is not provided, GetConsentWithParams will try get the consent key using ConsentKeyFunc.
// "WithParams" functions should only be called from within the chaincode.
func GetConsentWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, consentID string, consentKey ...[]byte) (data_model.Consent, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v, consentID: %v", caller.ID, consentID)

	_ = metering_i.SetEnvAndAddRow(stub)

	return consent_mgmt_i.GetConsentWithParams(stub, caller, consentID, consentKey...)
}

// ValidateConsent gets the specified consent asset. If consent is found and if it passes the expiration date and access level checks, it returns filter rules and the consent key.
// Filter rule is a simple rule that contains consent owner ID, which can be applied against an asset's owner ID, and consent
// datatype ID, which can be applied against asset's datatypeID, to filter out assets.
//
// args: [datatypeID, ownerID, targetID, access, currTime]
//
// targetID is the ID of the consent recipient.
// access is the desired access level that will be validated against the access recorded in the consent object.
// currTime is the current timestamp generated.
func ValidateConsent(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) (simple_rule.Rule, data_model.Key, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return consent_mgmt_i.ValidateConsent(stub, caller, args)
}

// GetConsentsWithOwnerID returns a list of consents, sorted by ownerID. OwnerID is the original owner of a datatype consent.
//
// args: [ownerID]
func GetConsentsWithOwnerID(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return consent_mgmt_i.GetConsentsWithOwnerID(stub, caller, args)
}

// GetConsentsWithTargetID returns a list of consents, sorted by targetID. TargetID is the ID of the consent recipient.
//
// args: [targetID]
func GetConsentsWithTargetID(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return consent_mgmt_i.GetConsentsWithTargetID(stub, caller, args)
}

// GetConsentsWithCallerID returns a list of consents created by the caller.
//
// args: []
func GetConsentsWithCallerID(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return consent_mgmt_i.GetConsentsWithCallerID(stub, caller, args)
}

// GetConsentsWithOwnerIDAndDatatypeID returns a list of consents, sorted by ownerID and datatypeID.
// ownerID is the owner of a datatype consent.
//
// args: [ownerID, datatypeID]
func GetConsentsWithOwnerIDAndDatatypeID(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return consent_mgmt_i.GetConsentsWithOwnerIDAndDatatypeID(stub, caller, args)
}

// GetConsentsWithTargetIDAndDatatypeID returns a list of consents, sorted by targetID and datatypeID.
//
// args: [targetID, datatypeID]
func GetConsentsWithTargetIDAndDatatypeID(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return consent_mgmt_i.GetConsentsWithTargetIDAndDatatypeID(stub, caller, args)
}

// GetConsentsWithTargetIDAndOwnerID returns a list of consents, sorted by targetID and ownerID.
//
// args: [targetID, ownerID]
func GetConsentsWithTargetIDAndOwnerID(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return consent_mgmt_i.GetConsentsWithTargetIDAndOwnerID(stub, caller, args)
}

// GetConsentID returns the consent_id.
func GetConsentID(datatypeID string, targetID string, ownerID string) string {
	return consent_mgmt_i.GetConsentID(datatypeID, targetID, ownerID)
}

// GetConsentIDForDatatype finds consent ID by checking datatype and parents of this datatype.
// returns consentID, consentAssetID, err
// returns "" if no matching consent is found
func GetConsentIDForDatatype(stub cached_stub.CachedStubInterface, ownerID string, targetID string, datatypeID string) (string, string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("ownerID: %v, targetID: %v, datatypeID: %v", ownerID, targetID, datatypeID)

	_ = metering_i.SetEnvAndAddRow(stub)

	return consent_mgmt_i.GetConsentIDForDatatype(stub, ownerID, targetID, datatypeID)
}

// GetConsentAssetID returns consent asset ID from consent ID.
// The returned consent asset ID can be used to get consent asset using asset_mgmt.
func GetConsentAssetID(stub cached_stub.CachedStubInterface, consentID string) (string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("consentID: %v", consentID)

	_ = metering_i.SetEnvAndAddRow(stub)

	return consent_mgmt_i.GetConsentAssetID(stub, consentID)
}
