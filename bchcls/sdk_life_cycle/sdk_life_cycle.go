/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package sdk_life_cycle contains wrappers needed for SDK functionalities.
package sdk_life_cycle

import (
	"common/bchcls/cached_stub"
	"common/bchcls/internal/metering_i"
	"common/bchcls/utils"

	pb "github.com/hyperledger/fabric/protos/peer"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("sdk_life_cycle")

// SuccessWrapper is a wrapper for shim.Success
func SuccessWrapper(stub cached_stub.CachedStubInterface, payload []byte) pb.Response {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.UpdateRow(stub, payload, "success")
	return shim.Success(payload)
}

// ErrorWrapper is a wrapper for shim.Error
func ErrorWrapper(stub cached_stub.CachedStubInterface, err string) pb.Response {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.UpdateRow(stub, err, "failure")
	return shim.Error(err)
}

// GetResult is a convenience function for SuccessWrapper and ErrorWrapper
func GetResult(stub cached_stub.CachedStubInterface, function string, returnBytes []byte, returnError error) pb.Response {
	if returnError != nil {
		logger.Errorf("Invoke %v Error: %v", function, returnError)
		return ErrorWrapper(stub, returnError.Error())
	}
	logger.Debugf("Invoke %v Success", function)
	return SuccessWrapper(stub, returnBytes)
}
