/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package datatype manages datatypes and their relationships.
// Each asset has a list of datatypes. For example, this design allows all assets of the datatype "Medical Records,"
// to be shared through a single consent.
package datatype

import (
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/datatype/datatype_interface"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/datatype_i"
	"common/bchcls/internal/metering_i"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("datatype")

// ROOT_DATATYPE_ID is the ID of default ROOT datatype. All other datatypes are children of ROOT.
const ROOT_DATATYPE_ID = global.ROOT_DATATYPE_ID

// ------------------------------------------------------
// ---------------------- INIT FUNCTIONS ----------------
// ------------------------------------------------------

// Init sets up the datatype package by registering a default ROOT datatype. Called by init_common.Init function.
// All solutions must call init_common.Init during solution set up time.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	return datatype_i.Init(stub, logLevel...)
}

// RegisterDatatype registers a new datatype to the ledger.
// Maintains datatype tree structure.
// Assumes a ROOT datatype exists.
// Creates datatypeSymKey and maintains key relationship with parent datatypes.
// If parentDatatypeID is not provided or does not exist, the datatype will be automatically added as a child of ROOT.
//
// args = [ datatype, parentDatatypeID ]
func RegisterDatatype(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return datatype_i.RegisterDatatype(stub, caller, args)
}

// RegisterDatatypeWithParams saves a new datatype to the ledger and maintains datatype tree structure.
// "WithParams" functions should only be called from within the chaincode.
//
// Caller must pass in datatype and can optionally pass in parentDatatypeID.
// If parentDatatypeID is not provided, the datatype will be added as a child of ROOT.
// If the parentDatatypeID passed in does not exist, an error will be thrown.
// It returns a registered DatatypeInterface instance.
func RegisterDatatypeWithParams(stub cached_stub.CachedStubInterface, datatypeID, description string, isActive bool, parentDatatypeID string) (datatype_interface.DatatypeInterface, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	return datatype_i.RegisterDatatypeWithParams(stub, datatypeID, description, isActive, parentDatatypeID)
}

// GetDatatypeKeyID returns the datatype key ID associated with a datatype owner pair.
func GetDatatypeKeyID(datatypeID string, ownerID string) string {
	return datatype_i.GetDatatypeKeyID(datatypeID, ownerID)
}

// UpdateDatatype updates an existing datatype's description.
// Caller must be a system admin.
//
// args = [ datatype ]
func UpdateDatatype(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return datatype_i.UpdateDatatype(stub, caller, args)
}

// GetDatatype returns a datatype with the given datatypeID.
// Returns an empty datatype if the passed in datatypeID does not match an existing datatype's ID.
//
// args = [ datatypeID ]
func GetDatatype(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return datatype_i.GetDatatype(stub, caller, args)
}

// GetDatatypeWithParams function gets a datatype from the ledger.
// "WithParams" functions should only be called from within the chaincode.
//
// Returns an empty datatype if the passed in datatypeID does not match an existing datatype's ID.
func GetDatatypeWithParams(stub cached_stub.CachedStubInterface, datatypeID string) (datatype_interface.DatatypeInterface, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	return datatype_i.GetDatatypeWithParams(stub, datatypeID)
}

// GetAllDatatypes returns all datatypes, not including the ROOT datatype.
func GetAllDatatypes(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	return datatype_i.GetAllDatatypes(stub, caller, args)
}

// AddDatatypeSymKey adds a sym key for the given datatypeID and ownerID and returns the DatatypeSymKey.
// It will also make sure that all its parent datatypes will get new sym key for the given owner (if it does not exist already).
// If the datatype sym key already exists, it will return success.
func AddDatatypeSymKey(stub cached_stub.CachedStubInterface, caller data_model.User, datatypeID, ownerID string, keyPathForOwnerSymkey ...[]string) (data_model.Key, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	return datatype_i.AddDatatypeSymKey(stub, caller, datatypeID, ownerID, keyPathForOwnerSymkey...)
}

// GetDatatypeSymKey composes and returns a sym key for the given datatypeID and ownerID.
// Returns an empty key if the datatype is not found or if there is an error.
func GetDatatypeSymKey(stub cached_stub.CachedStubInterface, caller data_model.User, datatypeID string, ownerID string, keyPath ...[]string) (data_model.Key, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	return datatype_i.GetDatatypeSymKey(stub, caller, datatypeID, ownerID, keyPath...)
}

// GetParentDatatype returns the datatypeID of a given datatypeID's direct parent.
func GetParentDatatype(stub cached_stub.CachedStubInterface, datatypeID string) (string, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	return datatype_i.GetParentDatatype(stub, datatypeID)
}

// NormalizeDatatypes returns a list of normalized child datatype IDs.
func NormalizeDatatypes(stub cached_stub.CachedStubInterface, datatypeIDs []string) ([]string, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	return datatype_i.NormalizeDatatypes(stub, datatypeIDs)
}
