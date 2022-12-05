/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package table_interface provides an interface for index table related methods.
package table_interface

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// Table represents an index table for querying items (represented by rows).
// Note that the primary key must be unique for all assets in the table.
// For example, user_name can be an indexed field but not a primary key
// because user_name might have duplicates. But user_id can be the
// primary key.
type Table interface {

	// HasIndex returns true if table has the specified index, false otherwise.
	HasIndex(keys []string) bool

	// GetIndexedFields returns a list of fields that have been indexed for this table.
	GetIndexedFields() []string

	// GetPrimaryKeyId returns the name of the field that is treated as the primary key of this table.
	GetPrimaryKeyId() string

	// AddIndex adds the specified index to the table.
	// If updateAllRows is true, updates all rows in the table.
	AddIndex(keys []string, updateAllRows bool) error

	// UpdateAllRows updates index values for all rows in the table.
	UpdateAllRows() error

	// UpdateRow updates index values for a row in the table.
	UpdateRow(keys map[string]string) error

	// DeleteRow removes the row specified by the primary key ID from the table.
	DeleteRow(id string) error

	// GetRow returns the row specified by the primary key ID from the table.
	GetRow(id string) ([]byte, error)

	// CreateRangeKey creates a simple key for calling GetRowsByRange.
	CreateRangeKey(fieldNames []string, fieldValues []string) (string, error)

	// GetRowsByRange returns a range iterator over a set of rows in this table.
	// The iterator can be used to iterate over all rows between the startKey (inclusive) and endKey (exclusive).
	// The rows are returned by the iterator in lexical order.
	// Note that startKey and endKey can be empty strings, which implies an unbounded range query at start and end.
	// GetRowsByRange is not allowed if index is encrypted (if isEncrypted = true) when adding the index table.
	GetRowsByRange(startKey string, endKey string) (shim.StateQueryIteratorInterface, error)

	// GetRowsByPartialKey returns an iterator over a set of rows in this table.
	// The iterator can be used to iterate over all rows that satisfy the provided index values.
	// Note: sort order is disabled if index is encrypted.
	GetRowsByPartialKey(fieldNames []string, fieldValues []string) (shim.StateQueryIteratorInterface, error)

	// SaveToLedger saves the table to the ledger.
	SaveToLedger() error
}
