/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package datastore_i

import (
	"common/bchcls/cached_stub"
	"common/bchcls/custom_errors"
	"common/bchcls/data_model"
	"common/bchcls/datastore"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/datastore_i/datastore_c"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("datastore_i")

// Init sets up the datastore package by adding default ledger connection.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	return datastore_c.Init(stub, logLevel...)
}

// InitDefaultDatastore sets up the default off-chain Cloudant datastore.
func InitDefaultDatastore(stub cached_stub.CachedStubInterface, args ...string) ([]byte, error) {

	return datastore_c.InitDefaultDatastore(stub, args...)
}

// PutDatastoreConnection puts a DatastoreConnection on the Ledger (encrypted).
// Validates that Datastore Type must be either default ones or registered via RegisterDatastoreImpl method
func PutDatastoreConnection(stub cached_stub.CachedStubInterface,
	caller data_model.User,
	datastoreConnection datastore.DatastoreConnection) error {

	if utils.IsStringEmpty(datastoreConnection.ID) {
		errMsg := "Datastore ID is mandatory attribute"
		logger.Errorf("%v", errMsg)
		return errors.New(errMsg)
	}
	if caller.Role != global.ROLE_SYSTEM_ADMIN {
		customErr := &custom_errors.RoleAccessPrivilegeError{Role: caller.Role}
		logger.Errorf("PutDatastoreConnection: %v", customErr)
		return customErr
	}

	//validate Type exists
	if !IsRegisteredDatastoreType(datastoreConnection.Type) {
		errMsg := "Datastore Type must be registered before using it in DatastoreConnection"
		logger.Errorf("%v", errMsg)
		return errors.New(errMsg)
	}

	return datastore_c.PutDatastoreConnection(stub, datastoreConnection)

}

// DeleteDatastoreConnection deletes a DatastoreConnection from the Ledger.
// If the datastoreConnectionID does not exist on the Ledger, it does nothing.
// Caller must have an app admin role to use this method.
func DeleteDatastoreConnection(stub cached_stub.CachedStubInterface,
	caller data_model.User,
	datastoreConnectionID string) error {

	if utils.IsStringEmpty(datastoreConnectionID) {
		errMsg := "DatastoreConnection ID is mandatory argument"
		logger.Errorf("%v", errMsg)
		return errors.New(errMsg)
	}
	if caller.Role != global.ROLE_SYSTEM_ADMIN {
		customErr := &custom_errors.RoleAccessPrivilegeError{Role: caller.Role}
		logger.Errorf("DeleteDatastoreConnection: %v", customErr)
		return customErr
	}
	return datastore_c.DeleteDatastoreConnection(stub, datastoreConnectionID)
}

// GetDatastoreConnection returns DatastoreConnection from the ledger
func GetDatastoreConnection(stub cached_stub.CachedStubInterface, datastoreConnectionID string) (datastore.DatastoreConnection, error) {

	if utils.IsStringEmpty(datastoreConnectionID) {
		errMsg := "DatastoreConnection ID is mandatory argument"
		logger.Errorf("%v", errMsg)
		return datastore.DatastoreConnection{}, errors.New(errMsg)
	}
	return datastore_c.GetDatastoreConnection(stub, datastoreConnectionID)
}

// IsRegisteredDatastoreType returns true if the datastore type is one of the default ones or registered via RegisterDatastoreImpl method
func IsRegisteredDatastoreType(datastoreType string) bool {
	return datastore_c.IsRegisteredDatastoreType(datastoreType)
}

// GetDatastoreImpl returns a new DatastoreImpl instance that is initialized with DatastoreConnection identified by datastoreConnectionID. Caller should call this method once in a transaction and reuse the instance for data persistence.
func GetDatastoreImpl(stub cached_stub.CachedStubInterface, datastoreConnectionID string) (datastore.DatastoreInterface, error) {
	return datastore_c.GetDatastoreImpl(stub, datastoreConnectionID)
}

// RegisterDatastoreImpl is used by Solution to register a new Datastore Type. If default off-chain storage implementation
// is sufficient for your use case, there is no need to use this method.
func RegisterDatastoreImpl(caller data_model.User, datastoreType string, implementation datastore.DatastoreInterface) error {
	//Only appAdmin allowed to call
	if caller.Role != global.ROLE_SYSTEM_ADMIN {
		customErr := &custom_errors.RoleAccessPrivilegeError{Role: caller.Role}
		logger.Errorf("RegisterDatastoreImpl: %v", customErr)
		return customErr
	}
	return datastore_c.RegisterDatastoreImpl(datastoreType, implementation)
}
