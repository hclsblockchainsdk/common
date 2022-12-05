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

package consent_mgmt_g

import (
	"common/bchcls/crypto"
	"common/bchcls/internal/common/global"
)

// GetConsentID returns the consent_id
// objectID is datatypeID, and ownerID is creatorID of Consent
func GetConsentID(objectID string, targetID string, ownerID string) string {
	return global.CONSENT_PREFIX + "-" + crypto.HashB64([]byte(objectID+"-"+targetID+"-"+ownerID))
}
