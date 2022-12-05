/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// package common contains structs and functions to be used across all
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
//	"common/bchcls/internal/key_mgmt_c"
//	"common/bchcls/internal/common/rb_tree"
//
package common

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("common")
