/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package history_manager is an interface for high-level history management functions.
package history_manager

import (
	"common/bchcls/asset_mgmt/asset_manager"
	"common/bchcls/data_model"
	"common/bchcls/simple_rule"
)

// HistoryManager is an interface for high-level history management functions.
type HistoryManager interface {

	// GetAssetManager returns the assetManager.
	GetAssetManager() asset_manager.AssetManager

	// PutInvokeTransactionLog stores a log for an invoke transaction, encrypted with the provided encryptionKey.
	// If you are going to log a transaction for a query, reference the GoDoc for PutQueryTransactionLog.
	PutInvokeTransactionLog(transactionLog data_model.TransactionLog, encryptionKey data_model.Key) error

	// GetTransactionLog returns a log from the ledger decrypted by the given log sym key.
	GetTransactionLog(transactionID string, logKey data_model.Key) (*data_model.TransactionLog, error)

	// GetTransactionLogs returns a set of logs from the ledger.
	// function name         - Name of the function being logged.
	// indexField            - Indexed transaction log fields (must be one of field_1-field_8).
	// indexFieldValue       - Value of each indexed field.
	// startTimestamp        - Filters out transaction logs prior to startTimestamp (inclusive); pass -1 to not filter by startTimestamp.
	// endTimestamp          - Filters out transaction logs after to endTimestamp (exclusive); pass -1 to not filter by endTimestamp.
	// previousKey           - The ledger key from the previous call to GetTransactionLogs (second return value).
	//                         Used for paging; pass "" if this will be the first page.
	// limit                 - Max number of logs to return for each call.
	// filterRule            - This rule is applied to each log asset. Only logs which evaluate to true against this rule will be returned.
	//                         The rule must be contrived such that a boolean is stored in a key called "$result" in the result.
	//                         When the rule is applied, only the "$result" key of the result will be checked.
	//                         To access fields from the transaction log to filter by, use a "var" operator and an operand value starting with "private_data"
	//                         and followed by the name of any field from history.TransactionLog.
	//                         You can access arbitrary fields by indexing through the data field using the dot notation.
	//                         Examples:  {"var", "private_data.timestamp"}   {"var", "private_data.field_1"}   {"var", "private_data.data.any_arbitrary_field"}
	// logSymKeyPath         - logSymKeyPath can be asset_key_func.AssetKeyPathFunc type, asset_key_func.AssetKeyByteFunc, string, or []string type.
	//                         If logSymKeyPath is string type, it's converted to []string, and processed as []string input
	//                         If logSymKeyPath is []string type, asset key path for an asset will be append(assetKeys, asset.AssetId)
	//                         and logSymKeyPath will be GetAssetKey(assetId, assetKeyPath).
	//                         If logSymKeyPath is asset_key_func.AssetKeyPathFunc, asset key path will be assetKeyPath(caller,asest)
	//                         and assetKeyByte will be GetAssetKey(assetId, assetKeyPath).
	//                         If logSymKeyPath is asset_key_func.AssetKeyByteFunc, asset key byte will be assetKeyPath(caller,asest).
	GetTransactionLogs(namespace, indexField, indexFieldValue string, startTimestamp, endTimestamp int64, previousKey string, limit int, filterRule *simple_rule.Rule, logSymKeyPath interface{}) ([]data_model.TransactionLog, string, error)
}
