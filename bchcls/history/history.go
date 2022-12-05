/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package history handles transaction history logging.
package history

import (
	"common/bchcls/asset_mgmt/asset_manager"
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/history/history_manager"
	"common/bchcls/internal/history_i"
	"common/bchcls/internal/metering_i"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("history")

// ------------------------------------------------------
// ---------------------- INIT FUNCTIONS ----------------
// ------------------------------------------------------
// Init sets up the history package by building the history index table.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return history_i.Init(stub, logLevel...)
}

// GetHistoryManager constructs and returns a HistoryManagerImpl instance.
func GetHistoryManager(assetManager asset_manager.AssetManager) history_manager.HistoryManager {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(assetManager.GetStub())

	return history_i.GetHistoryManager(assetManager)
}

// PutQueryTransactionLog stores a log for a query transaction, encrypted with the provided encryptionKey.
// To log a query, first call history.GenerateExportableTransactionLog during the query to get an exportableTransactionLog.
// Then return the exportableTransactionLog from the query. Finally, create a separate transaction to invoke
// this function and pass in the exportableTransactionLog.
//
// args = [exportableTransactionLog]
func PutQueryTransactionLog(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return history_i.PutQueryTransactionLog(stub, caller, args)
}

// GenerateExportableTransactionLog returns an exportable transaction log.
// It is meant to be called in a query context.
// The ExportableTransactionLog this function returns should be returned from the query and then passed back into the
// chaincode using the function history.PutQueryTransactionLog.
func GenerateExportableTransactionLog(stub cached_stub.CachedStubInterface, caller data_model.User, log data_model.TransactionLog, encryptionKey data_model.Key) (data_model.ExportableTransactionLog, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return history_i.GenerateExportableTransactionLog(stub, caller, log, encryptionKey)
}
