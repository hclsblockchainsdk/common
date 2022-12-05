/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package history_i

import (
	"common/bchcls/asset_mgmt/asset_key_func"
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/datastore/datastore_manager"
	"common/bchcls/internal/asset_mgmt_i"
	"common/bchcls/internal/datastore_i/datastore_c"
	"common/bchcls/internal/datastore_i/datastore_c/cloudant/cloudant_datastore_test_utils"
	"common/bchcls/internal/datatype_i"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/simple_rule"
	"common/bchcls/test_utils"
	"common/bchcls/user_mgmt"
	"fmt"
	"strings"

	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func setup(t *testing.T) *test_utils.NewMockStub {
	mstub := test_utils.CreateNewMockStub(t)
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	Init(stub)
	user_mgmt.Init(stub)
	asset_mgmt_i.Init(stub)
	datatype_i.Init(stub)
	datastore_c.Init(stub)
	mstub.MockTransactionEnd("t1")
	logger.SetLevel(shim.LogDebug)
	return mstub
}

// test basic put and get transaction log from ledger
func TestPutAndGetTransactionLog(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutAndGetTransactionLog function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	callerBytes, _ := json.Marshal(&caller)
	logSymKey := caller.GetLogSymKey()
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	transactionLog := data_model.TransactionLog{TransactionID: "txid", Namespace: "testing1", FunctionName: "function_name", CallerID: "caller", Timestamp: 1234567890, Data: "any arbitrary stringified data object", Field1: "abc"}
	mstub.MockTransactionStart(transactionLog.TransactionID)
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)
	err = putTransactionLog(assetManager, transactionLog, logSymKey)
	mstub.MockTransactionEnd(transactionLog.TransactionID)
	test_utils.AssertTrue(t, err == nil, "Expected putTransactionLog to succeed")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager = GetHistoryManager(assetManager)

	transactionLogRetrieved, err := historyManager.GetTransactionLog(transactionLog.TransactionID, logSymKey)
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLog to succeed")
	test_utils.AssertTrue(t, transactionLogRetrieved.TransactionID == transactionLog.TransactionID, "Transaction IDs do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Namespace == transactionLog.Namespace, "Namespaces do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.FunctionName == transactionLog.FunctionName, "Function names do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.CallerID == transactionLog.CallerID, "Caller IDs do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Timestamp == transactionLog.Timestamp, "Timestamps do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Data == transactionLog.Data, "Data does not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Field1 == transactionLog.Field1, "Field1's do not match")
	mstub.MockTransactionEnd("t1")

	// invalid logs; missing a required field
	transactionLogInvalid1 := data_model.TransactionLog{Namespace: "namespace", FunctionName: "function_name", CallerID: "caller", Timestamp: 1234567890, Field1: "abc"}
	transactionLogInvalid2 := data_model.TransactionLog{TransactionID: "txid2", FunctionName: "function_name", CallerID: "caller", Timestamp: 1234567890, Field1: "abc"}
	transactionLogInvalid3 := data_model.TransactionLog{TransactionID: "txid3", Namespace: "namespace", CallerID: "caller", Timestamp: 1234567890, Field1: "abc"}
	transactionLogInvalid4 := data_model.TransactionLog{TransactionID: "txid4", Namespace: "namespace", FunctionName: "function_name", Timestamp: 1234567890, Field1: "abc"}
	transactionLogInvalid5 := data_model.TransactionLog{TransactionID: "txid5", Namespace: "namespace", FunctionName: "function_name", CallerID: "caller", Field1: "abc"}
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager = GetHistoryManager(assetManager)
	err = putTransactionLog(assetManager, transactionLogInvalid1, caller.GetSymKey())
	test_utils.AssertTrue(t, err != nil, "Expected putTransactionLog to fail")
	err = putTransactionLog(assetManager, transactionLogInvalid2, caller.GetSymKey())
	test_utils.AssertTrue(t, err != nil, "Expected putTransactionLog to fail")
	err = putTransactionLog(assetManager, transactionLogInvalid3, caller.GetSymKey())
	test_utils.AssertTrue(t, err != nil, "Expected putTransactionLog to fail")
	err = putTransactionLog(assetManager, transactionLogInvalid4, caller.GetSymKey())
	test_utils.AssertTrue(t, err != nil, "Expected putTransactionLog to fail")
	err = putTransactionLog(assetManager, transactionLogInvalid5, caller.GetSymKey())
	test_utils.AssertTrue(t, err != nil, "Expected putTransactionLog to fail")
	mstub.MockTransactionEnd("t1")
}

// test basic put invoke transaction log to ledger
func TestPutInvokeTransactionLog(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutInvokeTransactionLog function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// valid log
	transactionLog := data_model.TransactionLog{TransactionID: "txid", Namespace: "namespace", FunctionName: "function_name", CallerID: "caller", Timestamp: 1234567890, Field1: "abc"}
	mstub.MockTransactionStart(transactionLog.TransactionID)
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)
	err = historyManager.PutInvokeTransactionLog(transactionLog, caller.GetSymKey())
	mstub.MockTransactionEnd(transactionLog.TransactionID)
	test_utils.AssertTrue(t, err == nil, "Expected putTransactionLog to succeed")

	// invalid log; transactionID not matching current transaction
	transactionLog = data_model.TransactionLog{TransactionID: "txid", Namespace: "namespace", FunctionName: "function_name", CallerID: "caller", Timestamp: 1234567890, Field1: "abc"}
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager = GetHistoryManager(assetManager)
	err = historyManager.PutInvokeTransactionLog(transactionLog, caller.GetSymKey())
	mstub.MockTransactionEnd("t1")
	test_utils.AssertFalse(t, err == nil, "Expected putTransactionLog to fail")
}

// test basic put and get a query transaction log from ledger
func TestPutAndGetQueryTransactionLog(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutAndGetQueryTransactionLog function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// create transaction log
	transactionLog := data_model.TransactionLog{TransactionID: "query-txid", Namespace: "namespace", FunctionName: "function_name", CallerID: "caller", Timestamp: 1234567890, Field1: "abc"}

	// create exportable log
	mstub.MockTransactionStart(transactionLog.TransactionID)
	stub = cached_stub.NewCachedStub(mstub)
	exportableLog, err := GenerateExportableTransactionLog(stub, caller, transactionLog, caller.GetSymKey())
	test_utils.AssertTrue(t, err == nil, "Expected GenerateExportableTransactionLog to succeed")
	mstub.MockTransactionEnd(transactionLog.TransactionID)

	// stringify exportable log to simulate it coming from outside the chaincode and try to put the log to the ledger
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	exportableLogBytes, _ := json.Marshal(&exportableLog)
	err = PutQueryTransactionLog(stub, caller, []string{string(exportableLogBytes)})
	test_utils.AssertTrue(t, err == nil, "Expected putTransactionLog to succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)

	// get the log back from the ledger and check the fields are correct
	transactionLogRetrieved, err := historyManager.GetTransactionLog(transactionLog.TransactionID, caller.GetSymKey())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLog to succeed")
	test_utils.AssertTrue(t, transactionLogRetrieved.TransactionID == transactionLog.TransactionID, "Transaction IDs do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Namespace == transactionLog.Namespace, "Namespaces do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.FunctionName == transactionLog.FunctionName, "Function names do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.CallerID == transactionLog.CallerID, "Caller IDs do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Timestamp == transactionLog.Timestamp, "Timestamps do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Data == transactionLog.Data, "Data does not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Field1 == transactionLog.Field1, "Field1's do not match")
	mstub.MockTransactionEnd("t1")
}

// test getting multiple logs of the same function
func TestGetTransactionLogs(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetTransactionLogs function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	putTransactionLogForTest(t, mstub, caller, "txid1", "testing1", "function_name1", "caller", 1234567890, "", "", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid2", "testing1", "function_name1", "caller", 1234567890, "", "", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid3", "testing1", "function_name1", "caller", 1234567890, "", "", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid4", "testing2", "function_name2", "caller", 1234567890, "", "", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid5", "testing2", "function_name2", "caller", 1234567890, "", "", "", "", "", "", "", "", "", caller.GetSymKey())

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)
	logs, _, err := historyManager.GetTransactionLogs("testing1", "field_1", "", -1, -1, "", 10, nil, caller.GetSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 3, "Expected to get 3 transaction logs")

	logs, _, err = historyManager.GetTransactionLogs("testing2", "field_8", "", -1, -1, "", 10, nil, caller.GetSymKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 2, "Expected to get 2 transaction logs")
	mstub.MockTransactionEnd("t1")
}

// test getting multiple pages of transactions logs
func TestGetTransactionLogs_Page(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetTransactionLogs_Page function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	putTransactionLogForTest(t, mstub, caller, "txid1", "testing1", "function_name1", "caller", 1234567890, "", "a", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid2", "testing1", "function_name1", "caller", 1234567890, "", "a", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid3", "testing1", "function_name1", "caller", 1234567890, "", "a", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid4", "testing1", "function_name1", "caller", 1234567890, "", "a", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid5", "testing1", "function_name1", "caller", 1234567890, "", "a", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid6", "testing1", "function_name1", "caller", 1234567890, "", "a", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid7", "testing1", "function_name1", "caller", 1234567890, "", "a", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid8", "testing1", "function_name1", "caller", 1234567890, "", "b", "", "", "", "", "", "", "", caller.GetSymKey())

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)
	logs, previousKey, err := historyManager.GetTransactionLogs("testing1", "field_1", "a", -1, -1, "", 4, nil, caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 4, "Expected to get 4 transaction logs")

	logs, _, err = historyManager.GetTransactionLogs("testing1", "field_1", "a", -1, -1, previousKey, 4, nil, caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 3, "Expected to get 3 transaction logs")
	mstub.MockTransactionEnd("t1")
}

// test log indexing capability and invalid index field names
func TestGetTransactionLogs_Index(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetTransactionLogs_Index function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	putTransactionLogForTest(t, mstub, caller, "txid1", "testing1", "function_name1", "caller", 1000000001, "", "abc", "", "123", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid2", "testing1", "function_name1", "caller", 1000000002, "", "abc", "", "456", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid3", "testing1", "function_name1", "caller", 1000000003, "", "def", "", "123", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid4", "testing2", "function_name2", "caller", 1000000004, "", "abc", "", "123", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid5", "testing2", "function_name2", "caller", 1000000005, "", "abc", "", "123", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid6", "testing2", "function_name2", "caller", 1000000006, "", "abc", "", "123", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid7", "testing2", "function_name2", "caller", 1000000007, "", "abc", "", "456", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid8", "testing2", "function_name2", "caller", 1000000008, "", "def", "", "456", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid9", "testing2", "function_name2", "caller", 1000000009, "", "abc", "", "456", "", "", "", "", "", caller.GetSymKey())

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)
	// invalid field name
	_, _, err = historyManager.GetTransactionLogs("testing1", "invalid field", "", -1, -1, "", 10, nil, caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err != nil, "Expected GetTransactionLogs to fail")

	logs, _, err := historyManager.GetTransactionLogs("testing1", "field_1", "abc", -1, -1, "", 10, nil, caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 2, "Expected to get 2 transaction logs")

	logs, _, err = historyManager.GetTransactionLogs("testing2", "field_1", "abc", 1000000005, 1000000007, "", 10, nil, caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 2, "Expected to get 2 transaction logs")
	test_utils.AssertTrue(t, logs[0].TransactionID == "txid5", "Expected txid5")
	test_utils.AssertTrue(t, logs[1].TransactionID == "txid6", "Expected txid6")

	logs, _, err = historyManager.GetTransactionLogs("testing2", "field_1", "abc", 1000000007, -1, "", 10, nil, caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 2, "Expected to get 2 transaction logs")
	test_utils.AssertTrue(t, logs[0].TransactionID == "txid7", "Expected txid7")
	test_utils.AssertTrue(t, logs[1].TransactionID == "txid9", "Expected txid9")

	logs, _, err = historyManager.GetTransactionLogs("testing2", "field_1", "abc", -1, 1000000007, "", 10, nil, caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 3, "Expected to get 3 transaction logs")
	test_utils.AssertTrue(t, logs[0].TransactionID == "txid4", "Expected txid4")
	test_utils.AssertTrue(t, logs[1].TransactionID == "txid5", "Expected txid5")
	test_utils.AssertTrue(t, logs[2].TransactionID == "txid6", "Expected txid6")

	logs, _, err = historyManager.GetTransactionLogs("testing1", "field_3", "456", -1, -1, "", 10, nil, caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 1, "Expected to get 1 transaction logs")

	logs, _, err = historyManager.GetTransactionLogs("testing2", "field_3", "456", -1, -1, "", 10, nil, caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 3, "Expected to get 3 transaction logs")

	logs, _, err = historyManager.GetTransactionLogs("testing2", "field_7", "", -1, -1, "", 10, nil, caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 6, "Expected to get 6 transaction logs")

	logs, _, err = historyManager.GetTransactionLogs("testing1", "field_1", "ghi", -1, -1, "", 10, nil, caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 0, "Expected to get 0 transaction logs")
	mstub.MockTransactionEnd("t1")
}

// test log indexing capability and invalid index field names
func TestGetTransactionLogs_RuleFilter(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetTransactionLogs_RuleFilter function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	type solutionLevelObject struct {
		PatientId   string `json:"patient_id"`
		DoctorId    string `json:"doctor_id"`
		SurgeryDate int64  `json:"surgery_date"`
	}

	solutionData1 := solutionLevelObject{PatientId: "patient1", DoctorId: "doc1", SurgeryDate: 1000000001}
	solutionData2 := solutionLevelObject{PatientId: "patient2", DoctorId: "doc1", SurgeryDate: 1000000002}
	solutionData3 := solutionLevelObject{PatientId: "patient3", DoctorId: "doc1", SurgeryDate: 1000000003}
	solutionData4 := solutionLevelObject{PatientId: "patient4", DoctorId: "doc2", SurgeryDate: 1000000004}
	solutionData5 := solutionLevelObject{PatientId: "patient5", DoctorId: "doc2", SurgeryDate: 1000000005}
	solutionData6 := solutionLevelObject{PatientId: "patient6", DoctorId: "doc2", SurgeryDate: 1000000006}
	solutionData7 := solutionLevelObject{PatientId: "patient7", DoctorId: "doc1", SurgeryDate: 1000000007}
	solutionData8 := solutionLevelObject{PatientId: "patient8", DoctorId: "doc1", SurgeryDate: 1000000008}
	solutionData9 := solutionLevelObject{PatientId: "patient9", DoctorId: "doc3", SurgeryDate: 1000000009}

	putTransactionLogForTest(t, mstub, caller, "txid1", "testing1", "function_name1", "caller", 1000000001, solutionData1, "", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid2", "testing1", "function_name1", "caller", 1000000002, solutionData2, "", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid3", "testing1", "function_name1", "caller", 1000000003, solutionData3, "", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid4", "testing1", "function_name1", "caller", 1000000004, solutionData4, "", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid5", "testing1", "function_name1", "caller", 1000000005, solutionData5, "", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid6", "testing1", "function_name1", "caller", 1000000006, solutionData6, "", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid7", "testing1", "function_name1", "caller", 1000000007, solutionData7, "", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid8", "testing1", "function_name1", "caller", 1000000008, solutionData8, "", "", "", "", "", "", "", "", caller.GetSymKey())
	putTransactionLogForTest(t, mstub, caller, "txid9", "testing1", "function_name1", "caller", 1000000009, solutionData9, "", "", "", "", "", "", "", "", caller.GetSymKey())

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)
	rule := simple_rule.NewRule(simple_rule.R("==", simple_rule.R("var", "private_data.data.doctor_id"), "doc1"))
	logs, _, err := historyManager.GetTransactionLogs("testing1", "field_1", "", -1, -1, "", 10, &rule, caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 5, "Expected to get 5 transaction logs")
	test_utils.AssertTrue(t, logs[0].TransactionID == "txid1", "Expected txid1")
	test_utils.AssertTrue(t, logs[1].TransactionID == "txid2", "Expected txid2")
	test_utils.AssertTrue(t, logs[2].TransactionID == "txid3", "Expected txid3")
	test_utils.AssertTrue(t, logs[3].TransactionID == "txid7", "Expected txid7")
	test_utils.AssertTrue(t, logs[4].TransactionID == "txid8", "Expected txid8")

	rule = simple_rule.NewRule(simple_rule.R("or",
		simple_rule.R(">", simple_rule.R("var", "private_data.data.surgery_date"), 1000000007),
		simple_rule.R("<", simple_rule.R("var", "private_data.data.surgery_date"), 1000000002),
	))
	logs, _, err = historyManager.GetTransactionLogs("testing1", "field_1", "", -1, -1, "", 10, &rule, caller.GetPubPrivKeyId())
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 3, "Expected to get 3 transaction logs")
	test_utils.AssertTrue(t, logs[0].TransactionID == "txid1", "Expected txid1")
	test_utils.AssertTrue(t, logs[1].TransactionID == "txid8", "Expected txid8")
	test_utils.AssertTrue(t, logs[2].TransactionID == "txid9", "Expected txid9")
	mstub.MockTransactionEnd("t1")
}

// test access to logs that user has direct, indirect, and no access too; only logs which the user has access too should be returned
// if caller does not have access to any logs, 0 logs are returned and no err
func TestGetTransactionLogs_Access(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetTransactionLogs_Access function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	callerBytes, _ := json.Marshal(&caller)
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	key1 := test_utils.CreateSymKey("key1")
	key2 := test_utils.CreateSymKey("key2")
	key3 := test_utils.CreateSymKey("key3")

	// give caller access to key1 and give key1 access to key2
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	err = key_mgmt_i.AddAccess(stub, caller.GetSymKey(), key1)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	err = key_mgmt_i.AddAccess(stub, key1, key2)
	test_utils.AssertTrue(t, err == nil, "Expected AddAccess to succeed")
	mstub.MockTransactionEnd("t1")

	putTransactionLogForTest(t, mstub, caller, "txid1", "testing1", "function_name1", "caller", 1234567890, "", "", "", "", "", "", "", "", "", key1)
	putTransactionLogForTest(t, mstub, caller, "txid2", "testing1", "function_name1", "caller", 1234567890, "", "", "", "", "", "", "", "", "", key2)
	putTransactionLogForTest(t, mstub, caller, "txid3", "testing1", "function_name1", "caller", 1234567890, "", "", "", "", "", "", "", "", "", key3)
	putTransactionLogForTest(t, mstub, caller, "txid4", "testing2", "function_name2", "caller", 1234567890, "", "", "", "", "", "", "", "", "", key1)
	putTransactionLogForTest(t, mstub, caller, "txid5", "testing2", "function_name2", "caller", 1234567890, "", "", "", "", "", "", "", "", "", key1)
	putTransactionLogForTest(t, mstub, caller, "txid6", "testing2", "function_name2", "caller", 1234567890, "", "", "", "", "", "", "", "", "", key3)
	putTransactionLogForTest(t, mstub, caller, "txid7", "testing2", "function_name2", "caller", 1234567890, "", "", "", "", "", "", "", "", "", key2)
	putTransactionLogForTest(t, mstub, caller, "txid8", "testing2", "function_name2", "caller", 1234567890, "", "", "", "", "", "", "", "", "", key1)
	putTransactionLogForTest(t, mstub, caller, "txid9", "testing2", "function_name2", "caller", 1234567890, "", "", "", "", "", "", "", "", "", key2)
	putTransactionLogForTest(t, mstub, caller, "txid10", "testing3", "function_name3", "caller", 1234567890, "", "", "", "", "", "", "", "", "", key3)
	putTransactionLogForTest(t, mstub, caller, "txid11", "testing3", "function_name3", "caller", 1234567890, "", "", "", "", "", "", "", "", "", key3)
	putTransactionLogForTest(t, mstub, caller, "txid12", "testing3", "function_name3", "caller", 1234567890, "", "", "", "", "", "", "", "", "", key3)

	var keyFunc asset_key_func.AssetKeyPathFunc = func(stub cached_stub.CachedStubInterface, caller data_model.User, asset data_model.Asset) ([]string, error) {
		keyPath := []string{caller.GetSymKeyId()}
		if asset.AssetKeyId == "key1" {
			keyPath = append(keyPath, "key1")
		} else if asset.AssetKeyId == "key2" {
			keyPath = append(keyPath, "key1")
			keyPath = append(keyPath, "key2")
		} else if asset.AssetKeyId == "key3" {
			keyPath = append(keyPath, "key3")
		}
		return keyPath, nil
	}

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)
	logs, _, err := historyManager.GetTransactionLogs("testing1", "field_1", "", -1, -1, "", 10, nil, keyFunc)
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 2, "Expected to get 2 transaction logs")

	logs, _, err = historyManager.GetTransactionLogs("testing2", "field_8", "", -1, -1, "", 10, nil, keyFunc)
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 5, "Expected to get 5 transaction logs")

	logs, _, err = historyManager.GetTransactionLogs("testing3", "field_4", "", -1, -1, "", 10, nil, keyFunc)
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLogs to succeed")
	test_utils.AssertTrue(t, len(logs) == 0, "Expected to get 0 transaction logs")
	mstub.MockTransactionEnd("t1")
}

// test put and get transaction log from offchain DB
func TestPutAndGetTransactionLog_Offchain(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutAndGetTransactionLog_Offchain function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	logSymKey := caller.GetLogSymKey()
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Setup cloudant datastore
	datastoreConnectionID := "cloudant1"
	err = cloudant_datastore_test_utils.SetupDatastore(mstub, caller, datastoreConnectionID)
	test_utils.AssertTrue(t, err == nil, "SetupDatastore should be successful")

	transactionLog := data_model.TransactionLog{TransactionID: "txid", Namespace: "testing1", FunctionName: "function_name", CallerID: "caller", Timestamp: 1234567890, Data: "any arbitrary stringified data object", Field1: "abc"}
	transactionLog.ConnectionID = datastoreConnectionID
	mstub.MockTransactionStart(transactionLog.TransactionID)
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)
	err = putTransactionLog(assetManager, transactionLog, logSymKey)
	mstub.MockTransactionEnd(transactionLog.TransactionID)
	test_utils.AssertTrue(t, err == nil, "Expected putTransactionLog to succeed")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager = GetHistoryManager(assetManager)

	transactionLogRetrieved, err := historyManager.GetTransactionLog(transactionLog.TransactionID, logSymKey)
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLog to succeed")
	test_utils.AssertTrue(t, transactionLogRetrieved.TransactionID == transactionLog.TransactionID, "Transaction IDs do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Namespace == transactionLog.Namespace, "Namespaces do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.FunctionName == transactionLog.FunctionName, "Function names do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.CallerID == transactionLog.CallerID, "Caller IDs do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Timestamp == transactionLog.Timestamp, "Timestamps do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Data == transactionLog.Data, "Data does not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Field1 == transactionLog.Field1, "Field1's do not match")
	mstub.MockTransactionEnd("t1")
}

func TestPutAndGetTransactionLog_OffChain_NoDatastore(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutAndGetTransactionLog_OffChain_NoDatastore function called")
	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	logSymKey := caller.GetLogSymKey()
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Attempt to add log
	datastoreConnectionID := "cloudant1"
	transactionLog := data_model.TransactionLog{TransactionID: "txid", Namespace: "testing1", FunctionName: "function_name", CallerID: "caller", Timestamp: 1234567890, Data: "any arbitrary stringified data object", Field1: "abc"}
	transactionLog.ConnectionID = datastoreConnectionID // not set up
	mstub.MockTransactionStart(transactionLog.TransactionID)
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)
	err = putTransactionLog(assetManager, transactionLog, logSymKey)
	test_utils.AssertTrue(t, err != nil, "Expected putTransactionLog to fail")
	expectedErrorMsg := fmt.Sprintf("DatastoreConnection with ID %v does not exist.", datastoreConnectionID)
	test_utils.AssertTrue(t, strings.Contains(err.Error(), expectedErrorMsg), fmt.Sprintf("Expected error message: %v", expectedErrorMsg))
	mstub.MockTransactionEnd(transactionLog.TransactionID)

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager = GetHistoryManager(assetManager)

	_, err = historyManager.GetTransactionLog(transactionLog.TransactionID, logSymKey)
	test_utils.AssertTrue(t, err != nil, "Expected GetTransactionLog to fail")
	mstub.MockTransactionEnd("t1")
}

// test put and get transaction log from offchain DB after deleting DB connection
func TestPutAndGetTransactionLog_Deleted_Offchain(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutAndGetTransactionLog_Deleted_Offchain function called")

	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	caller := test_utils.CreateTestUser("caller")
	caller.Role = "system"
	callerBytes, _ := json.Marshal(&caller)
	logSymKey := caller.GetLogSymKey()
	_, err := user_mgmt.RegisterUser(stub, caller, []string{string(callerBytes), "false"})
	test_utils.AssertTrue(t, err == nil, "Expected RegisterUser to succeed")
	mstub.MockTransactionEnd("t1")

	// Setup cloudant datastore
	datastoreConnectionID := "cloudant1"
	err = cloudant_datastore_test_utils.SetupDatastore(mstub, caller, datastoreConnectionID)
	test_utils.AssertTrue(t, err == nil, "SetupDatastore should be successful")

	transactionLog := data_model.TransactionLog{TransactionID: "txid", Namespace: "testing1", FunctionName: "function_name", CallerID: "caller", Timestamp: 1234567890, Data: "any arbitrary stringified data object", Field1: "abc"}
	transactionLog.ConnectionID = datastoreConnectionID
	mstub.MockTransactionStart(transactionLog.TransactionID)
	stub = cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager := GetHistoryManager(assetManager)
	err = putTransactionLog(assetManager, transactionLog, logSymKey)
	mstub.MockTransactionEnd(transactionLog.TransactionID)
	test_utils.AssertTrue(t, err == nil, "Expected putTransactionLog to succeed")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager = GetHistoryManager(assetManager)

	transactionLogRetrieved, err := historyManager.GetTransactionLog(transactionLog.TransactionID, logSymKey)
	test_utils.AssertTrue(t, err == nil, "Expected GetTransactionLog to succeed")
	test_utils.AssertTrue(t, transactionLogRetrieved.TransactionID == transactionLog.TransactionID, "Transaction IDs do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Namespace == transactionLog.Namespace, "Namespaces do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.FunctionName == transactionLog.FunctionName, "Function names do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.CallerID == transactionLog.CallerID, "Caller IDs do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Timestamp == transactionLog.Timestamp, "Timestamps do not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Data == transactionLog.Data, "Data does not match")
	test_utils.AssertTrue(t, transactionLogRetrieved.Field1 == transactionLog.Field1, "Field1's do not match")
	mstub.MockTransactionEnd("t1")

	// remove datastore connection
	mstub.MockTransactionStart("t1")
	err = datastore_manager.DeleteDatastoreConnection(stub, caller, datastoreConnectionID)
	test_utils.AssertTrue(t, err == nil, "Expected DeleteDatastoreConnection to succeed")
	mstub.MockTransactionEnd("t1")

	// attempt to get logs again, should fail
	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	assetManager = asset_mgmt_i.GetAssetManager(stub, caller)
	historyManager = GetHistoryManager(assetManager)

	transactionLogRetrieved, err = historyManager.GetTransactionLog(transactionLog.TransactionID, logSymKey)
	test_utils.AssertTrue(t, err != nil, "Expected GetTransactionLog to fail")
	mstub.MockTransactionEnd("t1")
}

func putTransactionLogForTest(t *testing.T, mstub *test_utils.NewMockStub, caller data_model.User, transactionID, namespace, functionName, callerID string, timestamp int64, data interface{}, field1, field2, field3, field4, field5, field6, field7, field8 interface{}, encKey data_model.Key) data_model.TransactionLog {
	transactionLog := data_model.TransactionLog{TransactionID: transactionID, Namespace: namespace, FunctionName: functionName, CallerID: callerID, Timestamp: timestamp, Data: data, Field1: field1, Field2: field2, Field3: field3, Field4: field4, Field5: field5, Field6: field6, Field7: field7, Field8: field8}
	mstub.MockTransactionStart(transactionID)
	stub := cached_stub.NewCachedStub(mstub)
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	err := putTransactionLog(assetManager, transactionLog, encKey)
	mstub.MockTransactionEnd(transactionID)
	test_utils.AssertTrue(t, err == nil, "Expected putTransactionLog to succeed")
	return transactionLog
}
