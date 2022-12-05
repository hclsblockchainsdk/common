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

package key_mgmt_g

import (
	"common/bchcls/internal/common/global"
)

// GetPubPrivKeyId returns the ID that should be assigned to a public or private key.
func GetPubPrivKeyId(id string) string {
	return global.KEY_PREFIX_PUB_PRIV + "-" + id
}

// GetSymKeyId returns the ID that should be assigned to a sym key.
func GetSymKeyId(id string) string {
	return global.KEY_PREFIX_SYM_KEY + "-" + id
}

// GetLogSymKeyId returns the ID that should be assigned to a log sym key.
func GetLogSymKeyId(id string) string {
	return global.KEY_PREFIX_LOG_SYM_KEY + "-" + id
}

// GetPrivateKeyHashSymKeyId returns the ID that should be assigned to a sym key derived from the hash of a private key.
func GetPrivateKeyHashSymKeyId(id string) string {
	return global.KEY_PREFIX_PRIV_HASH + "-" + id
}
