/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package data_model contains structs used across packages to prevent circular imports.
// For example, the User struct is needed by both asset_mgmt and user_mgmt, but user_mgmt
// depends on functions in asset_mgmt.
// They can't import each other, so the shared structs live here.
package data_model

import (
	"common/bchcls/internal/common/global"
)

// Asset represents an item on the ledger.
// Datatypes are the datatypes associated with this asset.
// PublicData is accessible by any caller.
// PrivateData is encrypted by the asset key and is only accessible by those with access to asset key.
// OwnerIds represent asset owners, who have write access to the asset by default. Currently an asset can
// only have a single owner, so any element after the first one is automatically ignored.
// Metadata is used to store any data that describes the asset but is not part of the asset itself, e.g. data base name, connect string
// IndexTableName is the index table for an asset to save custom indices for querying.
type Asset struct {
	AssetId        string            `json:"asset_id"`
	Datatypes      []string          `json:"datatypes"`
	PublicData     []byte            `json:"public_data"`
	PrivateData    []byte            `json:"private_data"`
	OwnerIds       []string          `json:"owner_ids"`
	Metadata       map[string]string `json:"metadata"`
	AssetKeyId     string            `json:"asset_key_id"`
	AssetKeyHash   []byte            `json:"asset_key_hash"`
	IndexTableName string            `json:"index_table_name"`
}

// IsOwner returns true if the given userId is an owner of the asset.
func (asset *Asset) IsOwner(userId string) bool {
	for _, ownerId := range asset.OwnerIds {
		if ownerId == userId {
			return true
		}
	}
	return false
}

// Copy returns a copy of the asset as a new object.
// Callers can use this function to copy an object to avoid using reference pointers.
func (asset *Asset) Copy() Asset {
	newAsset := Asset{}
	newAsset.AssetId = asset.AssetId
	if asset.Datatypes != nil {
		newAsset.Datatypes = make([]string, len(asset.Datatypes))
		copy(newAsset.Datatypes, asset.Datatypes)
	}
	if asset.PublicData != nil {
		newAsset.PublicData = make([]byte, len(asset.PublicData))
		copy(newAsset.PublicData, asset.PublicData)
	}
	if asset.PrivateData != nil {
		newAsset.PrivateData = make([]byte, len(asset.PrivateData))
		copy(newAsset.PrivateData, asset.PrivateData)
	}
	if asset.OwnerIds != nil {
		newAsset.OwnerIds = make([]string, len(asset.OwnerIds))
		copy(newAsset.OwnerIds, asset.OwnerIds)
	}
	if asset.Metadata != nil {
		newAsset.Metadata = make(map[string]string)
		for key, value := range asset.Metadata {
			newAsset.Metadata[key] = value
		}
	}
	newAsset.AssetKeyId = asset.AssetKeyId
	if asset.AssetKeyHash != nil {
		newAsset.AssetKeyHash = make([]byte, len(asset.AssetKeyHash))
		copy(newAsset.AssetKeyHash, asset.AssetKeyHash)
	}
	newAsset.IndexTableName = asset.IndexTableName
	return newAsset
}

// SetDatastoreConnectionID sets the datastore connection ID for an asset.
// If an asset has a DatastoreConnectionID set then it will be saved to that datastore.
func (asset *Asset) SetDatastoreConnectionID(DatastoreConnectionId string) {
	if asset.Metadata == nil {
		asset.Metadata = make(map[string]string)
	}
	if len(DatastoreConnectionId) == 0 {
		delete(asset.Metadata, global.DATASTORE_CONNECTION_ID_METADATA_KEY)
	} else {
		asset.Metadata[global.DATASTORE_CONNECTION_ID_METADATA_KEY] = DatastoreConnectionId
	}
}

// GetDatastoreConnectionID returns the datastore connection ID of an asset if one is set.
func (asset *Asset) GetDatastoreConnectionID() string {
	datastoreConnID, ok := asset.Metadata[global.DATASTORE_CONNECTION_ID_METADATA_KEY]
	if ok {
		return datastoreConnID
	} else {
		return ""
	}
}
