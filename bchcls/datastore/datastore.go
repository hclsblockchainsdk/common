/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package datastore

import (
	"common/bchcls/cached_stub"
	"common/bchcls/internal/common/global"
)

//DATASTORE_TYPE_DEFAULT_CLOUDANT is the type for Cloudant DB. This is the default off-chain datastore type.
const DATASTORE_TYPE_DEFAULT_CLOUDANT = global.DATASTORE_TYPE_DEFAULT_CLOUDANT

//DATASTORE_TYPE_DEFAULT_LEDGER is the type for default Hyperledger on-chain storage.
const DATASTORE_TYPE_DEFAULT_LEDGER = global.DATASTORE_TYPE_DEFAULT_LEDGER

// DEFAULT_LEDGER_DATASTORE_ID is the default datastore ID for on-chain datastore.
const DEFAULT_LEDGER_DATASTORE_ID = global.DEFAULT_LEDGER_DATASTORE_ID

// DEFAULT_LEDGER_DATASTORE_ID is the default datastore ID for off-chain datastore.
const DEFAULT_CLOUDANT_DATASTORE_ID = global.DEFAULT_CLOUDANT_DATASTORE_ID

// DatastoreInterface need to be implemented for a specific datastore/DB to support off-chain data storage.
// The datastore will store encrypted private data of an asset.
type DatastoreInterface interface {
	// IsReady is a lightweight test method to see if a datastore is ready for use.
	// It can be called before calling Put/Get/Delete on this datastore
	IsReady() bool

	// Instantiate instantiates a datastore implementation given the registered DatastoreConnection.
	// This method returns a new instance of the implementation and does all necessary initialization steps for the datastore.
	Instantiate(connection DatastoreConnection) (DatastoreInterface, error)

	// GetDatastoreConnection returns the DatastoreConnection supplied during Instantiate method call.
	GetDatastoreConnection() DatastoreConnection

	// Put saves the encryptedData to the DataStore and returns the data key used to store this data.
	// If the data key already exists, then this means the data has not changed. In this case, success is returned.
	// Implementer should save the data key. This is the same key as the hash returned by the ComputeHash method.
	Put(stub cached_stub.CachedStubInterface, encryptedData []byte) (string, error)

	// Get returns the data corresponding to the key. This method should also
	// verify that the key matches the ComputeHash(retrieved data) for data integrity check. Returns error if data integrity check fails.
	// If the dataKey does not exist in the datastore, it returns empty []byte
	Get(stub cached_stub.CachedStubInterface, dataKey string) ([]byte, error)

	// Delete deletes data from the datastore given the dataKey.
	// Implementation of this Delete is up to the implementer. The implementer can choose whether to just mark a key as stale
	// for garbage collection or permanently remove the data.
	Delete(stub cached_stub.CachedStubInterface, dataKey string) error

	// ComputeHash returns the hash computed from the input data. The Put function uses this hash as key to determine if the data got changed.
	ComputeHash(stub cached_stub.CachedStubInterface, encryptedData []byte) string
}

// DatastoreConnection encapsulates datastore connection details.
// ID is the unique ID for the off-chain datastore.
// Type specifies the Datastore type.
// ConnectStr comprises of connection details such as host, account, database name and etc.
// The DatastoreConnection should be added using Datastore_Manager, before a Datastore can be used.
type DatastoreConnection struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	ConnectStr string `json:"connect_str"`
}
