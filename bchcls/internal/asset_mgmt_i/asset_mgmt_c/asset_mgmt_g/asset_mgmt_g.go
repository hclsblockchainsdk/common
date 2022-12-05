/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// global package contains global data, variables, constants, or functions to be
// used across all bchcls packages.
// This should be the lowest level package (below data_model and common).

// In global package, only the following bchcls packages are allowed to be imported:
// 	"common/bchcls/cached_stub"
//	"common/bchcls/crypto"
//	"common/bchcls/custom_errors"
//	"common/bchcls/internal/common/graph"
//	"common/bchcls/internal/common/rb_tree"
//	"common/bchcls/internal/common/global"
//

package asset_mgmt_g

import (
	"strings"

	"common/bchcls/crypto"
	"common/bchcls/internal/common/global"
)

// GetAssetId returns the assetId given the object's type and unique identifier.
// assetNamespace is the type of object being saved as an asset.
// The convention for assetNamespace is packagename.ObjectType (e.g. "data_model.User").
// id is the unique identifier for the given object type.
func GetAssetId(assetNamespace string, id string) string {
	hash := crypto.HashB64([]byte(assetNamespace + "-" + id))
	return global.ASSET_ID_PREFIX + hash
	//return assetIdPrefix +assetNamespace + "-" + id
}

// IsValidAssetId checks whether the given assetId has the correct asset id prefix.
func IsValidAssetId(assetId string) bool {
	return strings.HasPrefix(assetId, global.ASSET_ID_PREFIX)
}
