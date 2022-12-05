/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package history

import (
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/internal/asset_mgmt_i"
	"common/bchcls/simple_rule"
	"common/bchcls/test_utils"

	"encoding/json"
)

func ExampleGetHistoryManager() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)

	GetHistoryManager(assetManager)
}

func ExampleHistoryManagerImpl_GetAssetManager() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller1")
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)

	historyManager.GetAssetManager()
}

func ExampleHistoryManagerImpl_PutInvokeTransactionLog() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")

	txTimestamp, _ := stub.GetTxTimestamp()
	transactionLog := data_model.TransactionLog{
		TransactionID: stub.GetTxID(),
		Namespace:     "namespace",
		FunctionName:  "function_name",
		CallerID:      "caller",
		Timestamp:     txTimestamp.GetSeconds(),
		Data:          "any arbitrary data object",
		Field1:        "abc",
	}
	encryptionKey := caller.GetSymKey()

	mstub.MockTransactionStart(transactionLog.TransactionID)
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)
	historyManager.PutInvokeTransactionLog(transactionLog, encryptionKey)
	mstub.MockTransactionEnd(transactionLog.TransactionID)

}

func ExamplePutQueryTransactionLog() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")

	txTimestamp, _ := stub.GetTxTimestamp()
	transactionLog := data_model.TransactionLog{
		TransactionID: stub.GetTxID(),
		Namespace:     "namespace",
		FunctionName:  "function_name",
		CallerID:      "caller",
		Timestamp:     txTimestamp.GetSeconds(),
		Data:          "any arbitrary data object",
		Field1:        "abc",
	}

	mstub.MockTransactionStart(transactionLog.TransactionID)
	stub = cached_stub.NewCachedStub(mstub)
	exportableLog, _ := GenerateExportableTransactionLog(stub, caller, transactionLog, caller.GetSymKey())
	mstub.MockTransactionEnd(transactionLog.TransactionID)

	mstub.MockTransactionStart("different-tx")
	stub = cached_stub.NewCachedStub(mstub)
	exportableLogBytes, _ := json.Marshal(&exportableLog)
	PutQueryTransactionLog(stub, caller, []string{string(exportableLogBytes)})
	mstub.MockTransactionEnd("different-tx")
}

func ExampleGenerateExportableTransactionLog() {
	mstub := test_utils.CreateExampleMockStub()
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")

	txTimestamp, _ := stub.GetTxTimestamp()
	transactionLog := data_model.TransactionLog{
		TransactionID: stub.GetTxID(),
		Namespace:     "namespace",
		FunctionName:  "function_name",
		CallerID:      "caller",
		Timestamp:     txTimestamp.GetSeconds(),
		Data:          "any arbitrary data object",
		Field1:        "abc",
	}

	mstub.MockTransactionStart(transactionLog.TransactionID)
	stub = cached_stub.NewCachedStub(mstub)
	GenerateExportableTransactionLog(stub, caller, transactionLog, caller.GetSymKey())
	mstub.MockTransactionEnd(transactionLog.TransactionID)
}

func ExampleHistoryManagerImpl_GetTransactionLog() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller")
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)
	logKey := caller.GetLogSymKey()

	historyManager.GetTransactionLog("txid", logKey)
}

func ExampleHistoryManagerImpl_GetTransactionLogs() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	caller := test_utils.CreateTestUser("caller")
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)

	rule := simple_rule.NewRule(
		simple_rule.R("==",
			simple_rule.R("var", "private_data.data.doctor_id"),
			"doc1"))
	logKeyId := caller.GetLogSymKeyId()

	historyManager.GetTransactionLogs("namespace", "field_1", "abc", 1000000002, -1, "", 10, &rule, logKeyId)
}
