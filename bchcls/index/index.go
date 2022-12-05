/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package index allows for creating indices in the state database.
package index

import (
	"common/bchcls/cached_stub"
	"common/bchcls/index/table_interface"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/index_i"
	"common/bchcls/internal/metering_i"

	"github.com/hyperledger/fabric/core/chaincode/shim"

	"strings"
)

var logger = shim.NewLogger("index")

// Init sets up the index package.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	return index_i.Init(stub, logLevel...)
}

// GetTable returns the table from the ledger or creates a new one.
// Name is the name of the index table.
// Note: If table already exists, all options are ignored. This is because caller won't be able to change the options once the table is already created.
// options[0] is the name of the index field that will be used to uniquely identify a row in the table.
// If it is not provided, the table's primaryKeyId is set to "id".
// options[1] indicates whether to use custom binary tree implementation (on-chain) for index. Default value is false.
// options[2] indicates whether to encrypt index or not. Default value is false.
// options[3] datastoreConnectionID. if specified it will store index data to off-chain datastore.
// Note that currently only Cloudant off-chain datastore is supported for indexing.
// Also, when option[3] is specified, option[1] will be ignored.
func GetTable(stub cached_stub.CachedStubInterface, name string, options ...interface{}) table_interface.Table {

	_ = metering_i.SetEnvAndAddRow(stub)

	return index_i.GetTable(stub, name, options...)
}

// GetPrettyLedgerKey is used for debugging print statements.
// Should only be used during debugging.
// Replaces global.MIN_UNICODE_RUNE_VALUE with "_" and global.MAX_UNICODE_RUNE_VALUE with "*".
// Composite keys are also prefixed with a global.MIN_UNICODE_RUNE_VALUE.
func GetPrettyLedgerKey(ledgerKey string) string {
	prettyKey := strings.Replace(ledgerKey, string(global.MIN_UNICODE_RUNE_VALUE), "_", -1)
	prettyKey = strings.Replace(prettyKey, string(global.MAX_UNICODE_RUNE_VALUE), "*", -1)
	return prettyKey
}
