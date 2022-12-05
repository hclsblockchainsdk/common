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

package user_mgmt_g

import (
	"common/bchcls/internal/asset_mgmt_i/asset_mgmt_c/asset_mgmt_g"
	"common/bchcls/internal/common/global"
)

// GetUserAssetID returns the asset ID for the stored user object identified by the given userID.
func GetUserAssetID(userID string) string {
	return asset_mgmt_g.GetAssetId(global.USER_ASSET_NAMESPACE, userID)
}
