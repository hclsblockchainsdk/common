/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package history_i handles transaction history logging.
package history_i

import (
	"common/bchcls/asset_mgmt/asset_manager"
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/custom_errors"
	"common/bchcls/data_model"
	"common/bchcls/history/history_manager"
	"common/bchcls/index"
	"common/bchcls/internal/asset_mgmt_i"
	"common/bchcls/internal/common/global"
	"common/bchcls/simple_rule"
	"common/bchcls/utils"

	"encoding/json"
	"math/rand"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("history_i")

// ------------------------------------------------------
// ---------------------- INIT FUNCTIONS ----------------
// ------------------------------------------------------

// Init sets up the history package by building an index table for transaction logs.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	//History Index
	logger.Debug("Init history")
	historyTable := index.GetTable(stub, global.INDEX_HISTORY, "transaction_id")
	historyTable.AddIndex([]string{"namespace", "field_1", "timestamp", "transaction_id"}, false)
	historyTable.AddIndex([]string{"namespace", "field_2", "timestamp", "transaction_id"}, false)
	historyTable.AddIndex([]string{"namespace", "field_3", "timestamp", "transaction_id"}, false)
	historyTable.AddIndex([]string{"namespace", "field_4", "timestamp", "transaction_id"}, false)
	historyTable.AddIndex([]string{"namespace", "field_5", "timestamp", "transaction_id"}, false)
	historyTable.AddIndex([]string{"namespace", "field_6", "timestamp", "transaction_id"}, false)
	historyTable.AddIndex([]string{"namespace", "field_7", "timestamp", "transaction_id"}, false)
	historyTable.AddIndex([]string{"namespace", "field_8", "timestamp", "transaction_id"}, false)
	err := historyTable.SaveToLedger()
	return nil, err
}

// historyManagerImpl is the default implementation of the HistoryManager interface.
type historyManagerImpl struct {
	AssetManager asset_manager.AssetManager
}

// GetAssetManager documentation can be found in the interface definition of HistoryManager.
func (historyManager historyManagerImpl) GetAssetManager() asset_manager.AssetManager {
	return historyManager.AssetManager
}

// PutInvokeTransactionLog documentation can be found in the interface definition of HistoryManager.
func (historyManager historyManagerImpl) PutInvokeTransactionLog(transactionLog data_model.TransactionLog, encryptionKey data_model.Key) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	//check transactionID
	stub := historyManager.GetAssetManager().GetStub()
	if transactionLog.TransactionID != stub.GetTxID() {
		errMsg := "Cannot save transaction log for an invoke transaction with a transaction ID other than that of the current transaction"
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	return putTransactionLog(historyManager.GetAssetManager(), transactionLog, encryptionKey)
}

// GetTransactionLog documentation can be found in the interface definition of HistoryManager.
func (historyManager historyManagerImpl) GetTransactionLog(transactionID string, logKey data_model.Key) (*data_model.TransactionLog, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	asset, err := historyManager.GetAssetManager().GetAsset(getTransactionLogAssetID(transactionID), logKey)
	if err != nil {
		custom_err := &custom_errors.GetAssetDataError{AssetId: getTransactionLogAssetID(transactionID)}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}
	if utils.IsStringEmpty(asset.AssetId) {
		custom_err := &custom_errors.GetAssetDataError{AssetId: getTransactionLogAssetID(transactionID)}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	transactionLog := convertFromAsset(asset)

	return &transactionLog, nil
}

// GetTransactionLogs documentation can be found in the interface definition of HistoryManager.
func (historyManager historyManagerImpl) GetTransactionLogs(namespace, indexField, indexFieldValue string, startTimestamp, endTimestamp int64, previousKey string, limit int, filterRule *simple_rule.Rule, logSymKeyPath interface{}) ([]data_model.TransactionLog, string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	transactionLogs := []data_model.TransactionLog{}

	//check index field is valid
	validIndexFields := []string{"field_1", "field_2", "field_3", "field_4", "field_5", "field_6", "field_7", "field_8"}
	if !utils.InList(validIndexFields, indexField) {
		errMsg := "invalid indexField"
		logger.Errorf(errMsg)
		return transactionLogs, "", errors.New(errMsg)
	}

	// build query
	indexFields := []string{"namespace", indexField}
	startValues := []string{namespace, indexFieldValue}
	endValues := []string{namespace, indexFieldValue}

	// add startTimestamp or endTimestamp or both to query if provided;
	if startTimestamp > -1 || endTimestamp > -1 {
		indexFields = append(indexFields, "timestamp")
	}
	if startTimestamp > -1 {
		startTimestampStr, err := utils.ConvertToString(startTimestamp)
		if err != nil {
			errMsg := "Failed to ConvertToString for startTimestamp"
			logger.Errorf("%v: %v", errMsg, err)
			return transactionLogs, "", errors.Wrap(err, errMsg)
		}
		startValues = append(startValues, startTimestampStr)
	}
	if endTimestamp > -1 {
		endTimestampStr, err := utils.ConvertToString(endTimestamp)
		if err != nil {
			errMsg := "Failed to ConvertToString for endTimestamp"
			logger.Errorf("%v: %v", errMsg, err)
			return transactionLogs, "", errors.Wrap(err, errMsg)
		}
		endValues = append(endValues, endTimestampStr)
	}

	// Get page of assets
	assetManager := historyManager.GetAssetManager()
	assetsIter, err := assetManager.GetAssetIter(
		global.TRANSACTION_LOG_ASSET_NAMESPACE,
		global.INDEX_HISTORY,
		indexFields,
		startValues,
		endValues,
		true,
		true,
		logSymKeyPath,
		previousKey, limit, filterRule)
	if err != nil {
		errMsg := "Failed to get transaction log assets"
		logger.Errorf("%v: %v", errMsg, err)
		return transactionLogs, "", errors.Wrap(err, errMsg)
	}

	assets, previousKey, err := assetsIter.GetAssetPage()
	if err != nil {
		errMsg := "Failed to get transaction log asset page"
		logger.Errorf("%v: %v", errMsg, err)
		return transactionLogs, "", errors.Wrap(err, errMsg)
	}
	// convert assets to transaction logs
	for _, asset := range assets {
		transactionLog := convertFromAsset(&asset)
		transactionLogs = append(transactionLogs, transactionLog)
	}
	return transactionLogs, previousKey, nil
}

// GetHistoryManager constructs and returns an historyManagerImpl instance.
func GetHistoryManager(assetManager asset_manager.AssetManager) history_manager.HistoryManager {
	return historyManagerImpl{AssetManager: assetManager}
}

// PutQueryTransactionLog stores a log for a query transaction, encrypted with the provided encryptionKey.
// To log a query, first call history.GenerateExportableTransactionLog during the query to get an exportableTransactionLog.
// Return the exportableTransactionLog from the query.
// Finally create a separate transaction to invoke this function and pass in the exportableTransactionLog.
//
// args = [exportableTransactionLog]
func PutQueryTransactionLog(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	if len(args) != 1 {
		custom_err := &custom_errors.LengthCheckingError{Type: "history.PutQueryTransactionLog args"}
		logger.Errorf(custom_err.Error())
		return custom_err
	}

	// parse ExportableTransactionLog
	exportableTransactionLog := data_model.ExportableTransactionLog{}
	exportableTransactionLogBytes := []byte(args[0])
	err := json.Unmarshal(exportableTransactionLogBytes, &exportableTransactionLog)
	if err != nil {
		custom_err := &custom_errors.UnmarshalError{Type: "history.ExportableTransactionLog"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return errors.Wrap(err, custom_err.Error())
	}

	// decrypt symkey with private key
	encSymKeyString, err := crypto.DecodeStringB64(exportableTransactionLog.EncryptedSymKey)
	if err != nil {
		errMsg := "Failed to decode symkey b64 string"
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}
	symKeyBytes, err := crypto.DecryptWithPrivateKey(caller.PrivateKey, []byte(encSymKeyString))
	if err != nil {
		custom_err := &custom_errors.DecryptionError{ToDecrypt: "randomized sym key", DecryptionKey: "caller's private key"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return errors.Wrap(err, custom_err.Error())
	}

	// decrypt log with symkey
	transactionLog := data_model.TransactionLog{}
	encTransactionLogString, err := crypto.DecodeStringB64(exportableTransactionLog.EncryptedTransactionLog)
	if err != nil {
		errMsg := "Failed to decode transaction log b64 string"
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}
	transactionLogBytes, err := crypto.DecryptWithSymKey(symKeyBytes, encTransactionLogString)
	if err != nil {
		custom_err := &custom_errors.DecryptionError{ToDecrypt: "transaction log", DecryptionKey: "randomized sym key"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return errors.Wrap(err, custom_err.Error())
	}
	err = json.Unmarshal(transactionLogBytes, &transactionLog)
	if err != nil {
		custom_err := &custom_errors.UnmarshalError{Type: "history.TransactionLog"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return errors.Wrap(err, custom_err.Error())
	}

	// decrypt log encryption key with symkey
	encryptionKey := data_model.Key{}
	encEncryptionKeyString, err := crypto.DecodeStringB64(exportableTransactionLog.EncryptedLogEncryptionKey)
	if err != nil {
		errMsg := "Failed to decode transaction log encryption key b64 string"
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}
	encryptionKeyBytes, err := crypto.DecryptWithSymKey(symKeyBytes, encEncryptionKeyString)
	if err != nil {
		custom_err := &custom_errors.DecryptionError{ToDecrypt: "transaction log encryption key", DecryptionKey: "randomized sym key"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return errors.Wrap(err, custom_err.Error())
	}
	err = json.Unmarshal(encryptionKeyBytes, &encryptionKey)
	if err != nil {
		custom_err := &custom_errors.UnmarshalError{Type: "data_model.Key"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return errors.Wrap(err, custom_err.Error())
	}

	// save the log
	assetManager := asset_mgmt_i.GetAssetManager(stub, caller)
	return putTransactionLog(assetManager, transactionLog, encryptionKey)
}

// GenerateExportableTransactionLog returns an exportable transaction log.
// It is meant to be called in a query context.
// The ExportableTransactionLog this function returns is meant to be returned from the query and then passed back into the chaincode using the function history.PutQueryTransactionLog.
func GenerateExportableTransactionLog(stub cached_stub.CachedStubInterface, caller data_model.User, log data_model.TransactionLog, encryptionKey data_model.Key) (data_model.ExportableTransactionLog, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	exportableTransactionLog := data_model.ExportableTransactionLog{}

	// ensure that the transactionID is the one for the current transaction
	log.TransactionID = stub.GetTxID()

	// generate a random sym key
	symKeyBytes := make([]byte, 32)
	_, err := rand.Read(symKeyBytes)
	if err != nil {
		errMsg := "Failed to randomize sym key"
		logger.Errorf("%v: %v", errMsg, err)
		return exportableTransactionLog, errors.Wrap(err, errMsg)
	}

	// encrypt and stringify log
	logBytes, err := json.Marshal(&log)
	if err != nil {
		custom_err := &custom_errors.MarshalError{Type: "transaction log"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return exportableTransactionLog, errors.Wrap(err, custom_err.Error())
	}
	encLogBytes, err := crypto.EncryptWithSymKey(symKeyBytes, logBytes)
	if err != nil {
		custom_err := &custom_errors.EncryptionError{ToEncrypt: "transaction log", EncryptionKey: "randomized sym key"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return exportableTransactionLog, errors.Wrap(err, custom_err.Error())
	}
	encLogString := crypto.EncodeToB64String(encLogBytes)
	exportableTransactionLog.EncryptedTransactionLog = encLogString

	// encrypt and stringify encryption key
	encryptionKeyBytes, err := json.Marshal(&encryptionKey)
	if err != nil {
		custom_err := &custom_errors.MarshalError{Type: "transaction log encryption key"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return exportableTransactionLog, errors.Wrap(err, custom_err.Error())
	}
	encEncryptionKeyBytes, err := crypto.EncryptWithSymKey(symKeyBytes, encryptionKeyBytes)
	if err != nil {
		custom_err := &custom_errors.EncryptionError{ToEncrypt: "transaction log encryption key", EncryptionKey: "randomized sym key"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return exportableTransactionLog, errors.Wrap(err, custom_err.Error())
	}
	encEncryptionKeyString := crypto.EncodeToB64String(encEncryptionKeyBytes)
	exportableTransactionLog.EncryptedLogEncryptionKey = encEncryptionKeyString

	// encrypt sym key
	encSymKeyBytes, err := crypto.EncryptWithPublicKey(caller.PublicKey, symKeyBytes)
	if err != nil {
		custom_err := &custom_errors.EncryptionError{ToEncrypt: "randomized sym kye", EncryptionKey: "caller's public key"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return exportableTransactionLog, errors.Wrap(err, custom_err.Error())
	}
	encSymKeyString := crypto.EncodeToB64String(encSymKeyBytes)
	exportableTransactionLog.EncryptedSymKey = encSymKeyString

	return exportableTransactionLog, nil
}

// putTransactionLog stores a log to the ledger encrypted with the provided encryptionKey.
func putTransactionLog(assetManager asset_manager.AssetManager, transactionLog data_model.TransactionLog, encryptionKey data_model.Key) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	//check transaction log for required fields
	if utils.IsStringEmpty(transactionLog.TransactionID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "TransactionID"}
		logger.Errorf(custom_err.Error())
		return errors.WithStack(custom_err)
	}
	if utils.IsStringEmpty(transactionLog.Namespace) {
		custom_err := &custom_errors.LengthCheckingError{Type: "Namespace"}
		logger.Errorf(custom_err.Error())
		return errors.WithStack(custom_err)
	}
	if utils.IsStringEmpty(transactionLog.FunctionName) {
		custom_err := &custom_errors.LengthCheckingError{Type: "FunctionName"}
		logger.Errorf(custom_err.Error())
		return errors.WithStack(custom_err)
	}
	if utils.IsStringEmpty(transactionLog.CallerID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "CallerID"}
		logger.Errorf(custom_err.Error())
		return errors.WithStack(custom_err)
	}
	if transactionLog.Timestamp == 0 {
		custom_err := &custom_errors.LengthCheckingError{Type: "Timestamp"}
		logger.Errorf(custom_err.Error())
		return errors.WithStack(custom_err)
	}

	//add asset
	transactionLogAsset, err := convertToAsset(transactionLog, encryptionKey.ID)
	if err != nil {
		errMsg := "Failed to convert transaction log to asset"
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}
	err = assetManager.AddAsset(*transactionLogAsset, encryptionKey, false)
	if err != nil {
		errMsg := "Failed to add transaction log"
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}
	return nil
}

// getTransactionLogAssetID generates an assetID for a transaction log.
func getTransactionLogAssetID(transactionID string) string {
	return asset_mgmt_i.GetAssetId(global.TRANSACTION_LOG_ASSET_NAMESPACE, transactionID)
}

// convertToAsset converts a transactionLog to an assetData.
func convertToAsset(transactionLog data_model.TransactionLog, logSymKeyId string) (*data_model.Asset, error) {
	var err error
	// convert all fields to strings using the proper method (if this method is not used, indexing will not work properly)
	transactionLog.Field1, err = utils.ConvertToString(transactionLog.Field1)
	if err != nil {
		logger.Errorf("Failed to convert field 1 of transaction log to string: %v", err)
		return nil, errors.Wrap(err, "Failed to convert field 1 of transaction log to string")
	}
	transactionLog.Field2, err = utils.ConvertToString(transactionLog.Field2)
	if err != nil {
		logger.Errorf("Failed to convert field 2 of transaction log to string: %v", err)
		return nil, errors.Wrap(err, "Failed to convert field 2 of transaction log to string")
	}
	transactionLog.Field3, err = utils.ConvertToString(transactionLog.Field3)
	if err != nil {
		logger.Errorf("Failed to convert field 3 of transaction log to string: %v", err)
		return nil, errors.Wrap(err, "Failed to convert field 3 of transaction log to string")
	}
	transactionLog.Field4, err = utils.ConvertToString(transactionLog.Field4)
	if err != nil {
		logger.Errorf("Failed to convert field 4 of transaction log to string: %v", err)
		return nil, errors.Wrap(err, "Failed to convert field 4 of transaction log to string")
	}
	transactionLog.Field5, err = utils.ConvertToString(transactionLog.Field5)
	if err != nil {
		logger.Errorf("Failed to convert field 5 of transaction log to string: %v", err)
		return nil, errors.Wrap(err, "Failed to convert field 5 of transaction log to string")
	}
	transactionLog.Field6, err = utils.ConvertToString(transactionLog.Field6)
	if err != nil {
		logger.Errorf("Failed to convert field 6 of transaction log to string: %v", err)
		return nil, errors.Wrap(err, "Failed to convert field 6 of transaction log to string")
	}
	transactionLog.Field7, err = utils.ConvertToString(transactionLog.Field7)
	if err != nil {
		logger.Errorf("Failed to convert field 7 of transaction log to string: %v", err)
		return nil, errors.Wrap(err, "Failed to convert field 7 of transaction log to string")
	}
	transactionLog.Field8, err = utils.ConvertToString(transactionLog.Field8)
	if err != nil {
		logger.Errorf("Failed to convert field 8 of transaction log to string: %v", err)
		return nil, errors.Wrap(err, "Failed to convert field 8 of transaction log to string")
	}

	asset := data_model.Asset{}
	asset.AssetId = getTransactionLogAssetID(transactionLog.TransactionID)
	asset.AssetKeyId = logSymKeyId
	asset.Datatypes = []string{}
	metaData := make(map[string]string)
	metaData["namespace"] = global.TRANSACTION_LOG_ASSET_NAMESPACE
	asset.Metadata = metaData
	var publicData interface{}
	asset.PublicData, _ = json.Marshal(&publicData)
	asset.PrivateData, _ = json.Marshal(&transactionLog)
	asset.IndexTableName = global.INDEX_HISTORY
	// if an off-chain datastore is specified, save the id so that the log can be saved there
	if len(transactionLog.ConnectionID) != 0 {
		asset.SetDatastoreConnectionID(transactionLog.ConnectionID)
	}

	return &asset, nil
}

// convertFromAsset converts an assetData to a transactionLog.
func convertFromAsset(asset *data_model.Asset) data_model.TransactionLog {
	transactionLog := data_model.TransactionLog{}
	json.Unmarshal(asset.PrivateData, &transactionLog)
	if datastoreConnectionID, ok := asset.Metadata[global.DATASTORE_CONNECTION_ID_METADATA_KEY]; ok {
		transactionLog.ConnectionID = datastoreConnectionID
	}
	return transactionLog
}
