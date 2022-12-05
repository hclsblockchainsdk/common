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

package consent_mgmt_c

import (
	"common/bchcls/internal/consent_mgmt_i/consent_mgmt_c/consent_mgmt_g"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("consent_mgmt_i")

// GetConsentID returns the consent_id.
func GetConsentID(objectID string, targetID string, ownerID string) string {
	return consent_mgmt_g.GetConsentID(objectID, targetID, ownerID)
}
