/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package datastore_manager

import (
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/datastore"
	"common/bchcls/internal/datastore_i"
	"common/bchcls/internal/metering_i"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("datastore")

// ------------------------------------------------------
// -------- DatastoreConnection Methods ----------------
// ------------------------------------------------------

// Init sets up the datastore package by adding default ledger connection.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	return datastore_i.Init(stub, logLevel...)
}

// InitDefaultDatastore sets up the default off-chain Cloudant datastore.
func InitDefaultDatastore(stub cached_stub.CachedStubInterface, args ...string) ([]byte, error) {
	return datastore_i.InitDefaultDatastore(stub, args...)
}

// PutDatastoreConnection adds or updates an encrypted DatastoreConnection on the Ledger as reference to an off-chain datastore.
// DatastoreConnection contains the connection details encrypted by a sym key.
// Note that the datastoreConnection.Type should be a valid default Type or registered Type via RegisterDatastoreImpl method.
// Caller must have a system admin role to use this method.
//
// The DatastoreConnection's ID is saved in an asset's Metadata, and the asset's private data is stored in the datastore.
func PutDatastoreConnection(stub cached_stub.CachedStubInterface,
	caller data_model.User,
	datastoreConnection datastore.DatastoreConnection) error {

	_ = metering_i.SetEnvAndAddRow(stub)

	return datastore_i.PutDatastoreConnection(stub,
		caller,
		datastoreConnection)
}

// DeleteDatastoreConnection deletes the DatastoreConnection identified by datastoreConnectionID from the Ledger.
// This will make the private data of the Asset stored off-chain inaccessible through this DatastoreConnection.
// Caller must have a system admin role to use this method.
func DeleteDatastoreConnection(stub cached_stub.CachedStubInterface,
	caller data_model.User,
	datastoreConnectionID string) error {

	_ = metering_i.SetEnvAndAddRow(stub)

	return datastore_i.DeleteDatastoreConnection(stub,
		caller,
		datastoreConnectionID)
}

// GetDatastoreConnection returns the DatastoreConnection details for given datastoreConnectionID.
func GetDatastoreConnection(stub cached_stub.CachedStubInterface, datastoreConnectionID string) (datastore.DatastoreConnection, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	return datastore_i.GetDatastoreConnection(stub, datastoreConnectionID)
}

//------------------------------------------------------
//-------- Managing Datastore Implementations ----------
//------------------------------------------------------

// GetDatastoreImpl returns a datastore implementation instance initialized with a DatastoreConnection identified by
// the datastoreConnectionID, to be used for off-chain data storage.
//
// Solutions should call this function once per transaction and re-use the DatastoreImpl to store/retrieve data during each transaction.
// Make sure that the specific datastoreConnectionID is registered prior to this method call, using RegisterDatastoreConnection method.
// For example, datastoreConnectionID can be registered during Solution Init time.
func GetDatastoreImpl(stub cached_stub.CachedStubInterface, datastoreConnectionID string) (datastore.DatastoreInterface, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	return datastore_i.GetDatastoreImpl(stub, datastoreConnectionID)
}

// RegisterDatastoreImpl allows a solution to register a new datastore Type whenever a new off-chain datastore is to be used.
// Must provide corresponding DatastoreInterface implementation.
// This method must be called as the first step in Solution transaction call (e.g. in TransactionInit before Invoke).
func RegisterDatastoreImpl(caller data_model.User, datastoreType string, implementation datastore.DatastoreInterface) error {
	return datastore_i.RegisterDatastoreImpl(caller, datastoreType, implementation)
}
