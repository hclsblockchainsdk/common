/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package asset_manager is an interface for high-level asset management functions.
package asset_manager

import (
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/simple_rule"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// AssetManager is an interface for high-level asset management functions.
type AssetManager interface {

	// GetStub returns the stub, which provides functions for accessing and modifying the ledger.
	GetStub() cached_stub.CachedStubInterface

	// GetCaller returns the caller.
	GetCaller() data_model.User

	// AddAsset adds an asset to the ledger.
	// asset                 - asset data
	//                       - asset.AssetId must be generated using asset_mgmt.GetAssetId().
	// assetKey              - a sym key used to encrypt the asset's PrivateData field.
	// giveAccessToCaller    - if true, the caller will be given access to the assetKey. This access can be revoked later.
	//                       - if false, the caller will only be adding the asset and not given access to the assetKey.
	AddAsset(asset data_model.Asset, assetKey data_model.Key, giveAccessToCaller bool) error

	// UpdateAsset updates an existing asset on the ledger.
	// The asset owner is responsible for giving other users access to the assetKey.
	// asset                 - asset data
	//                       - asset.AssetId must be generated using asset_mgmt.GetAssetId().
	// assetKey              - a sym key used to encrypt the asset's PrivateData field
	// strictUpdate          - (optional) default = true
	//                       - if true, it returns an error if the asset does not exist.
	//                       - if false, it adds a new asset if it does not exist.
	//
	// Caller must have access to the asset to update.
	UpdateAsset(asset data_model.Asset, assetKey data_model.Key, strictUpdate ...bool) error

	// DeleteAsset deletes the asset for the given assetId, as long as the caller has write access.
	// Also updates any existing indices for this asset.
	DeleteAsset(assetId string, assetKey data_model.Key) error

	// GetAsset finds and decrypts the asset for the given assetId.
	// Caller must have access to the asset.
	// Returns empty asset if the passed assetId does not match any existing assets. It is the caller's responsibility to check if returned asset is empty.
	// If assetKey is an empty key, there will be no attempt to get the PrivateData of the asset. This can be a good speed optimization if private data is not needed.
	// If assetKey does not belong to the passed in assetID, it returns an error.
	GetAsset(assetId string, assetKey data_model.Key) (*data_model.Asset, error)

	// GetAssetKey finds an asset key using the key path passed in.
	// The first key ID in the key path should be the caller's private key ID,
	// and the last key ID should be the assetKey ID.
	// Returns an empty asset key if the assetId passed in does not exist.
	// If keyPath is invalid, it returns an error.
	GetAssetKey(assetId string, keyPath []string) (data_model.Key, error)

	// AddAccessToAsset adds read or write access from user to asset.
	// Read access is given by adding access from user's public key to asset key.
	// Write access is given by adding read access and setting its access type of access graph edge data to "write".
	// Adding write access will give both read and write access.
	// allowAddAccessBeforeAssetIsCreated is an optional bool flag (default = false).
	// If it's set to true, access is processed even if the asset is not yet created.
	// Caller must be asset owner.
	AddAccessToAsset(accessControl data_model.AccessControl, allowAddAccessBeforeAssetIsCreated ...bool) error

	// RemoveAccessFromAsset removes read or write access from user to asset.
	// Write access is removed by updating access type of access graph edge data to "read".
	// Removing read access will remove both read and write access and will delete access graph edge.
	// Removing write access will keep read access.
	// Caller must be asset owner.
	RemoveAccessFromAsset(accessControl data_model.AccessControl) error

	// CheckAccessToAsset returns true if the specified access has been given from user to asset.
	// Caller can only check caller's own access.
	// To check the access of another user, first get access control manager with that user as the caller. This requires caller to have access to that user's keys.
	CheckAccessToAsset(accessControl data_model.AccessControl) (bool, error)

	// GetAssetIter performs an index query and returns an asset iterator on the result.
	// assetNamespace        - type of asset being queried
	//                       - the convention for assetNamespace is packagename.ObjectType (e.g. "data_model.User")
	// indexTableName        - the name of the table of indices for this asset type.
	// fieldNames            - the list of field Names to search on, which MUST match the prefix of an existing index in the table
	//                       - for example, ["make", "color", "year"]
	//                       - Search query is performed on fieldNames with value range specified in startValues/endValues array. If you know the exact value, specify that value in both startValues & endValues
	// startValues           - the values to start on, e.g. ["toyota", "blue", "2001"]
	// endValues             - the values to end on, e.g. ["toyota", "blue", "2018"]
	// decryptPrivateData    - if decryptPrivateData is false, there will be no attempt to get private portion of the assets
	//                       - this can be a good speed optimization, if private data is not needed since it skips decryption step
	// returnPrivateAssetsOnly - if returnPrivateAssetsOnly is true, only assets that caller has access to, will be returned
	//                       - If returnPrivateAssetsOnly is false, it will return all assets that matched the search query.
	//                       - returnPrivateAssetsOnly is enforced even if you set decryptPrivateData to false
	// assetKeyPath          - This param is used to specify key path for access to the asset. This is used internally to call GetAssetKey func to access/decrypt the private portion of the asset.
	//											 - It can be one of the following types: asset_key_func.AssetKeyPathFunc, asset_key_func.AssetKeyByteFunc, string, or []string
	//                       - if assetKeyPath is string type, it's converted to []string, and processed as []string input
	//                       - if assetKeyPath is []string type, key path used internally will be [assetKeys, asset.AssetId]
	//                       - if assetKeyPath is func of type asset_key_func.AssetKeyPathFunc, asset key path used internally will be the return value of a call to AssetKeyPathFunc
	//                       - if assetKeyPath is func of type asset_key_func.AssetKeyByteFunc, that func will be called to get the key byte ([]byte) to get the AssetKey

	// previousKey           - the ledger key returned by the previous call to assetIter.Next()
	//                       - used for paging; pass empty string "" if you are not paging
	// limit                 - the max page size to be returned; if limit = -1, all assets are returned
	// filterRule            - this rule is applied to each asset and only assets which evaluate true against this rule will be returned
	//                       - the rule must be contrived such that a boolean is stored in a key called "$result" in the result
	//                       - when the rule is applied only the "$result" key of the result will be checked
	//                       - to access fields from the asset to filter by, use a "var" operator and an operand value of equal to the name of any field of data_model.Asset
	//                       - you can access arbitrary fields by indexing through the private_data or public_data field and using dot notation
	//                       - examples:  {"var", "owner_ids"}   {"var", "public_data.name"}   {"var", "private_data.data.arbitrary_field"}
	//                       - examples to return only public assets (when decrypPrivateData is set to true):
	//                         {"not": [{"bool": [{"var":"private_data.encrypted"}]}]}
	//                         This filter rule returns true if private_data.encryped field does not exist.
	// NOTE 1:    startValues & endValues should be identical except for the last entry.
	//            The first n-1 entries will be used for filtering, the last entry can be used for a range query.
	//            Range queries are between the startKey (inclusive) and endKey (exclusive).
	// NOTE 2:    If numbers/timestamps are being queried on, be sure to call utils.ConvertToString() to format the values properly.
	//
	// When calling GetAssetIter, caller should utilize start and end values to avoid starting from the beginning
	// of the ledger. Supplying start and end values will trigger a range query, which is more efficient.
	//
	GetAssetIter(
		assetNamespace string,
		indexTableName string,
		fieldNames []string,
		startValues []string,
		endValues []string,
		decryptPrivateData bool,
		returnOnlyPrivateAssets bool,
		assetKeyPath interface{},
		previousKey string,
		limit int,
		filterRule *simple_rule.Rule,
	) (AssetIteratorInterface, error)
}

// AssetIteratorInterface allows a chaincode to iterate over a set of assets.
type AssetIteratorInterface interface {
	// Inherits HasNext() and Close().
	shim.CommonIteratorInterface

	// Next returns the next asset in the iterator.
	Next() (*data_model.Asset, error)

	// GetPreviousLedgerKey returns the index ledger key of the previous item that was retrieved by Next().
	GetPreviousLedgerKey() string

	// GetAssetPage converts an asset iter to an array of assets.
	// Returns array of assets and previous ledger key.
	GetAssetPage() ([]data_model.Asset, string, error)
}
