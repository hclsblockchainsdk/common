/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package user_access_manager is an interface for high-level user access control functions.
package user_access_manager

import (
	"common/bchcls/data_model"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// UserAccessManager is an interface for user access control functions.
type UserAccessManager interface {

	// GetStub returns the stub, which provides functions for accessing and modifying the ledger.
	GetStub() shim.ChaincodeStubInterface

	// GetCaller returns the caller.
	GetCaller() data_model.User

	// AddAccess adds read or write access from user to asset.
	// Read access is given by adding access from user's public key to asset key.
	// Write access is given by adding read access and setting its access type of access graph edge data to "write".
	// Adding write access will give both read and write access.
	// If the asset already exists, there is no need to provide accessControl.AssetKey. This function will retrieve it.
	AddAccess(accessControl data_model.AccessControl) error

	// AddAccessByKey adds read access from startKey to targetKey.
	// startKey must be a sym key, private key, or public key.
	// targetKey cannot be a public key.
	// Encrypts targetKey with startKey and creates an edge in the key graph representing this access.
	AddAccessByKey(startKey data_model.Key, targetKey data_model.Key) error

	// RemoveAccess removes access read or write access from user to asset.
	// Write access is removed by removing user as an asset owner.
	// Removing read access will remove both read and write access.
	// Removing write access will keep read access.
	// Caller must be asset owner.
	RemoveAccess(accessControl data_model.AccessControl) error

	// RemoveAccessByKey removes read access from startKey to targetKey.
	RemoveAccessByKey(startKeyID string, targetKeyID string) error

	// CheckAccess returns true if the specified access has been given from user to asset.
	// You can only check your (caller's) own access if you are not the asset owner.
	// To check access of another user, get access control manager as that user and check access.
	// It only checks direct access, for the following cases:
	//
	// For Write Access, it returns true if:
	//  1. user is owner of the asset (or direct admin of owner)
	//  2. user has write access given by the owner of the asset
	//  3. user has write only access given by the owner of the asset
	//  4. user has a write consent for the asset
	//  5. user has a write consent for a datatype of the asset
	//  6. user is a direct admin of a group that has write access
	//
	// For Write Only Access, it returns true if:
	//  1. user has write only access given by the owner of the asset
	//
	// For Read Access, it returns true if:
	//  1. user is owner of the asset (or direct admin of owner)
	//  2. user has read or write access given by the owner of the asset
	//  3. user has a read or write consent for the asset
	//  4. user has a read or write consent for a datatype of the asset
	//  5. user is a direct admin of a group that has read or write access
	//
	// For Read Only Access, it returns true if:
	//  1. user has read access given by the owner of the asset
	//  2. user has a read consent for the asset
	//  3. user has a read consent for a datatype of the asset
	//  5. user is a direct admin of a group that has read access
	CheckAccess(accessControl data_model.AccessControl) (bool, error)

	// GetAccessData returns an access control object for the userId and assetId.
	// Returns nil if the user has no access to the asset.
	GetAccessData(userId string, assetId string) (*data_model.AccessControl, error)

	// SlowCheckAccessToKey traverses the path from caller to targetKey in the key graph.
	// Returns an access path and filters.
	SlowCheckAccessToKey(targetKeyID string) ([]string, data_model.AccessControlFilters, error)

	// GetKey returns a key given keyID.
	// The first items in keyPath must be the caller's key's ID, and the last key in keyPath must be target keyID.
	// If keyPath is nil or empty, key path is assumed to be [caller's key's ID, keyID].
	GetKey(keyID string, keyPath []string) (data_model.Key, error)
}
