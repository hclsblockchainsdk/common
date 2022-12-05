/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package user_access_ctrl handles access to assets and keys.
// Read access is granted by adding an edge to the key graph.
// Write access is granted by adding read access and setting its access type edge data to "write".
package user_access_ctrl

import (
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/metering_i"
	"common/bchcls/internal/user_access_ctrl_i"
	"common/bchcls/user_access_ctrl/user_access_manager"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("user_access_ctrl")

///////////////////////////////////////////////////////
// Access control constants

// ACCESS_READ is an AccessControl.Access option that specifies read access.
const ACCESS_READ = global.ACCESS_READ

// ACCESS_READ_ONLY is an AccessControl.Access option that specifies read access without write access.
// It is only used when verifing access. If used in AddAccess, it's treated as ACCESS_READ.
const ACCESS_READ_ONLY = global.ACCESS_READ_ONLY

// ACCESS_WRITE is an AccessControl.Access option that specifies write access.
const ACCESS_WRITE = global.ACCESS_WRITE

// ACCESS_WRITE_ONLY is an AccessControl.Access option that specifies write access without read access.
const ACCESS_WRITE_ONLY = global.ACCESS_WRITE_ONLY

// EDGEDATA_ACCESS_TYPE is a key used in edgeData (of type map[string]string).
const EDGEDATA_ACCESS_TYPE = global.EDGEDATA_ACCESS_TYPE

// ------------------------------------------------------
// ----------------- EXPORTED FUNCTIONS -----------------
// ------------------------------------------------------

// GetUserAccessManager constructs and returns an userAccessManagerImpl instance.
func GetUserAccessManager(stub cached_stub.CachedStubInterface, caller data_model.User) user_access_manager.UserAccessManager {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_access_ctrl_i.GetUserAccessManager(stub, caller)
}
