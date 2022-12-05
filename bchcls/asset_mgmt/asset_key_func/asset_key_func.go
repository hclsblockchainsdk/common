/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package asset_key_func provides asset key related functions.
package asset_key_func

import (
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
)

// AssetKeyPathFunc is a function that takes a caller and an asset as input,
// and returns a key path ([]string) to get the AssetKey of the asset.
type AssetKeyPathFunc func(stub cached_stub.CachedStubInterface, caller data_model.User, asset data_model.Asset) ([]string, error)

// AssetKeyByteFunc is a function that takes a caller and an asset as input,
// and returns a key byte ([]byte) to get the AssetKey of the asset.
type AssetKeyByteFunc func(stub cached_stub.CachedStubInterface, caller data_model.User, asset data_model.Asset) ([]byte, error)
