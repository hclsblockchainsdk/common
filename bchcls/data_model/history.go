/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package data_model contains structs used across packages to prevent circular imports.
// For example, the User struct is needed by both asset_mgmt and user_mgmt, but user_mgmt
// depends on functions in asset_mgmt.
// They can't import each other, so the shared structs live here.
package data_model

// TransactionLog stores data about an individual invoke or query ledger transaction.
// Use the data field to store arbitrary data about your transaction.
// Fields 1-8 should be used as index fields. To index logs by a particular data field, store it in one of these fields.
// Additionally, the data field can be used to store arbitrary data. Multi-level indexing can be achieved by storing a concatenation
// of two pieces of data in the data field.
type TransactionLog struct {
	TransactionID string      `json:"transaction_id"`
	Namespace     string      `json:"namespace"`
	FunctionName  string      `json:"function_name"`
	CallerID      string      `json:"caller_id"`
	Timestamp     int64       `json:"timestamp"`
	Data          interface{} `json:"data"`
	Field1        interface{} `json:"field_1"`
	Field2        interface{} `json:"field_2"`
	Field3        interface{} `json:"field_3"`
	Field4        interface{} `json:"field_4"`
	Field5        interface{} `json:"field_5"`
	Field6        interface{} `json:"field_6"`
	Field7        interface{} `json:"field_7"`
	Field8        interface{} `json:"field_8"`
	ConnectionID  string      `json:"connection_id"`
}

// ExportableTransactionLog is designed to securely pass a transaction log for a query to outside of the chaincode and be sent
// back into the chaincode in an invoke context. This is because queries do not write to the ledger.
type ExportableTransactionLog struct {
	EncryptedTransactionLog   string `json:"encrypted_transaction_log"`
	EncryptedLogEncryptionKey string `json:"encrypted_log_encryption_key"`
	EncryptedSymKey           string `json:"encrypted_sym_key"`
}
