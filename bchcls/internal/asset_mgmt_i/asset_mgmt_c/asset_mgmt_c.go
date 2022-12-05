/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// common package contain structs and functions to be used across all
// bchcls packages.
//
// In common package, only the following bchcls packages are allowed to be imported:
// 	"common/bchcls/cached_stub"
//	"common/bchcls/crypto"
//	"common/bchcls/custom_errors"
//	"common/bchcls/data_model"
//	"common/bchcls/index"
//	"common/bchcls/internal/common/global"
//	"common/bchcls/internal/common/graph"
//	"common/bchcls/internal/key_mgmt_i"
//	"common/bchcls/internal/common/rb_tree"
//
package asset_mgmt_c

import (
	"common/bchcls/cached_stub"
	"common/bchcls/custom_errors"
	"common/bchcls/data_model"
	"common/bchcls/internal/asset_mgmt_i/asset_mgmt_c/asset_mgmt_g"
	"common/bchcls/internal/common/global"

	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("asset_mgmt")

// GetEncryptedAssetData is a helper function.
// It returns the data_model.Asset with encrypted PrivateData given the AssetId.
// If the AssetId passed in does not exist, this function returns empty data_model.Asset.
func GetEncryptedAssetData(stub cached_stub.CachedStubInterface, assetId string) (data_model.Asset, error) {

	// check cache with assetId
	assetCache, err := getEncryptedAssetFromCache(stub, assetId)
	if err == nil && assetCache != nil {
		return *assetCache, nil
	}

	assetData := data_model.Asset{}
	// get assetLedgerKey using assetId
	assetLedgerKey := assetId

	// get assetData using assetLedgerKey
	assetBytes, err := stub.GetState(assetLedgerKey)
	if err != nil {
		custom_err := &custom_errors.GetLedgerError{LedgerKey: assetLedgerKey, LedgerItem: "assetBytes"}
		logger.Errorf("%v: %v", custom_err, err)
		return assetData, errors.Wrap(err, custom_err.Error())
	}

	if assetBytes == nil {
		logger.Debugf("Asset not found with ledger key: \"%v\"", assetLedgerKey)
		return assetData, nil
	}

	err = json.Unmarshal(assetBytes, &assetData)
	if err != nil {
		custom_err := &custom_errors.UnmarshalError{Type: "assetData"}
		logger.Errorf("%v: %v", custom_err, err)
		return assetData, errors.Wrap(err, custom_err.Error())
	}

	//save to cache
	putEncryptedAssetToCache(stub, assetData)
	// return asset data
	return assetData, nil
}

// getEncryptedAssetFromCache returns a copy of the object to avoid the "by reference" side effect.
func getEncryptedAssetFromCache(stub cached_stub.CachedStubInterface, assetId string) (*data_model.Asset, error) {
	assetCacheKey := getAssetCacheKey(assetId)
	cachedAsset, err := stub.GetCache(assetCacheKey)
	if err != nil {
		return nil, err
	} else if cachedAsset != nil {
		asset, ok := cachedAsset.(data_model.Asset)
		if ok {
			assetCopy := asset.Copy()
			return &assetCopy, nil
		} else {
			return nil, errors.New("Failed to map cache to Asset type")
		}
	}
	return nil, nil
}

// putEncryptedAssetToCache makes a copy of the asset and saves it to the cache.
func putEncryptedAssetToCache(stub cached_stub.CachedStubInterface, asset data_model.Asset) error {
	assetCacheKey := getAssetCacheKey(asset.AssetId)
	return stub.PutCache(assetCacheKey, asset.Copy())
}

func getAssetCacheKey(assetId string) string {
	return global.ASSET_CACHE_PREFIX + assetId
}

func getAssetPrivateCacheKey(assetId string) string {
	return global.ASSET_PRIVATE_CACHE_PREFIX + assetId
}

// GetAssetId returns the assetId given the object's type and unique identifier.
// assetNamespace is the type of object being saved as an asset.
// The convention for assetNamespace is packagename.ObjectType (e.g. "data_model.User").
// id is the unique identifier for the given object type.
func GetAssetId(assetNamespace string, id string) string {
	return asset_mgmt_g.GetAssetId(assetNamespace, id)
}

// IsValidAssetId checks whether the given assetId has the correct asset id prefix.
func IsValidAssetId(assetId string) bool {
	return asset_mgmt_g.IsValidAssetId(assetId)
}
