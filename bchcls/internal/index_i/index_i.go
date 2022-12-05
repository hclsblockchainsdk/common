/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package index allows for creating indices in the state database.
package index_i

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/index/table_interface"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/common/rb_tree"
	"common/bchcls/internal/index_i/cloudant_index"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"

	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

var logger = shim.NewLogger("index_i")

// Init sets up the index package.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	rb_tree.Init(stub, logLevel...)
	return cloudant_index.Init(stub, logLevel...)
}

// Table represents an index table for querying items (represented by rows).
//
// Note that the primary key must be unique for all assets in the table.
// For example, user_name can be an indexed field but not a primary key
// because user_name might have duplicates. But user_id can be the
// primary key.
type Table struct {
	// name is the name of the index table.
	name string
	// keyFields are the indexable fields. Each row must have a value for each of these fields.
	keyFields map[string]int
	// index is the list of the table's indices. An index is composed of two or more keyFields in a particular
	// order, the last always being the table's primaryKeyId.
	index [][]string
	// primaryKeyId is the name of the field that uniquely identifies a row in the table.
	primaryKeyId string
	// tr and useTree are optional, for use of the binary tree implementation of the index table.
	tr      *rb_tree.RBTree
	useTree bool
	// isEncrypted is optional to indicate whether to encrypt the index table.
	isEncrypted bool
	// dataStore is optinal for use of off-chain datastore.
	dataStoreId string
	dataStore   cloudant_index.IndexDatastoreInterface
}

type iTable struct {
	Name         string         `json:"name"`
	KeyFields    map[string]int `json:"key_fields"`
	Index        [][]string     `json:"index"`
	PrimaryKeyId string         `json:"primary_key_id"`
	UseTree      bool           `json:"use_tree"`
	IsEncrypted  bool           `json:"is_encrypted"`
	DatastoreId  string         `json:"is_datastore_id"`
}

// GetTable returns the table from the ledger or creates a new one.
// name is the name of the index table.
// Note: All options are ignored if table already exist since you won't be able to change the options once the table is already created.
// options[0] is the name of the index field that will be used to uniquely identify a row in the table.
// If not provided, the table's primaryKeyId is set to "id".
// options[1] indicates whether to useTree or not. Default value is false.
// options[2] indicates whether to encrypt index or not. Default value is true.
// options[3] datastoreID. if specified it will store index data to off-chain datastore
func GetTable(stub cached_stub.CachedStubInterface, name string, options ...interface{}) table_interface.Table {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("Table: %v", name)
	tr := rb_tree.NewRBTree(stub, "Index-"+name)
	key, _ := stub.CreateCompositeKey("Table", []string{name})
	keyId := "id"
	useTree := false
	isEncrypted := false
	datastoreId := ""
	var mydatastore cloudant_index.IndexDatastoreInterface = nil
	table := Table{}

	tableBytes, err := stub.GetState(key)
	if tableBytes == nil || err != nil {

		if len(options) > 0 {
			if option, ok := options[0].(string); ok {
				keyId = option
			}
		}

		if len(options) > 1 {
			if option, ok := options[1].(bool); ok {
				useTree = option
			}
		}

		if len(options) > 2 {
			if option, ok := options[2].(bool); ok {
				isEncrypted = option
			}
		}

		if len(options) > 3 {
			if option, ok := options[3].(string); ok {
				datastoreId = option
			}
			mydatastore, err = cloudant_index.GetIndexDatastoreImpl(stub, datastoreId)
			if err != nil {
				logger.Warningf("Failed to get datatstore %v: %v", datastoreId, err)
			}
		}

		//by default table has "id"
		table.name = name
		table.primaryKeyId = keyId
		table.keyFields = make(map[string]int)
		table.keyFields[table.primaryKeyId] = 1
		table.index = [][]string{[]string{table.primaryKeyId}}
		table.tr = tr
		table.useTree = useTree
		table.isEncrypted = isEncrypted
		table.dataStoreId = datastoreId
		table.dataStore = mydatastore
		logger.Debugf("Index Table: %v useTree: %v isEncrypted: %v, datastoreId: %v", table.name, table.useTree, table.isEncrypted, table.dataStoreId)
		return &table

	} else {
		var itable iTable
		json.Unmarshal(tableBytes, &itable)
		table.name = itable.Name
		table.primaryKeyId = itable.PrimaryKeyId
		table.keyFields = itable.KeyFields
		table.index = itable.Index
		table.useTree = itable.UseTree
		table.isEncrypted = itable.IsEncrypted
		table.tr = tr
		table.dataStoreId = itable.DatastoreId
		if len(itable.DatastoreId) > 0 {
			mydatastore, err = cloudant_index.GetIndexDatastoreImpl(stub, itable.DatastoreId)
			if err != nil {
				logger.Errorf("Failed to get datatstore %v :%v", itable.DatastoreId, err)
			}
		}
		table.dataStore = mydatastore

		logger.Debugf("Index Table: %v useTree: %v isEncrypted: %v, datastoreId: %v", table.name, table.useTree, table.isEncrypted, table.dataStoreId)
		return &table

	}
}

//returns true if k1 is subset of k2
func (t *Table) in(k1 []string, k2 []string) bool {
	if len(k2) < len(k1) {
		return false
	}
	for i, k := range k1 {
		if k2[i] != k {
			return false
		}
	}
	return true
}

//returns true if k1 is equal to k2
func (t *Table) eq(k1 []string, k2 []string) bool {
	if len(k2) != len(k1) {
		return false
	}
	for i, k := range k1 {
		if k2[i] != k {
			return false
		}
	}
	return true
}

func (t *Table) checkKeys(keys map[string]string) bool {
	for k := range t.keyFields {
		if _, ok := keys[k]; !ok {
			logger.Warningf("key field not found:", k)
			return false
		}
	}
	return true
}

func (t *Table) missingKeys(keys map[string]string) []string {
	miss := []string{}
	for k := range t.keyFields {
		if _, ok := keys[k]; !ok {
			miss = append(miss, k)
		}
	}
	return miss
}

func (t *Table) prefix(keys []string) string {
	prefix := t.name
	for _, k := range keys {
		prefix = prefix + "-" + k
	}
	hash := crypto.HashShortB64([]byte(prefix))
	return "Index-" + hash
}

func (t *Table) findIndex(sortOrder []string) (string, error) {
	for _, k := range t.index {
		if t.in(sortOrder, k) {
			return t.prefix(k), nil
		}
	}
	return "", errors.New("index not found")
}

// HasIndex returns true if table has the specified index, false otherwise.
func (t *Table) HasIndex(keys []string) bool {
	for _, k := range t.index {
		if t.eq(keys, k) {
			return true
		}
	}
	return false
}

// GetIndexedFields returns a list of fields that have been indexed for this table.
func (t *Table) GetIndexedFields() []string {
	keyFieldsList := make([]string, 0, len(t.keyFields))
	for keyField := range t.keyFields {
		keyFieldsList = append(keyFieldsList, keyField)
	}
	return keyFieldsList
}

// GetPrimaryKeyId returns the name of the field that is treated as the primary key of this table ("id" by default).
func (t *Table) GetPrimaryKeyId() string {
	return t.primaryKeyId
}

// AddIndex adds the specified index to the table.
// If updateAllRows is true, updates all rows in the table.
func (t *Table) AddIndex(keys []string, updateAllRows bool) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("keys: %v", keys)
	if len(keys) == 0 {
		return errors.New("empty index")
	}

	if keys[len(keys)-1] != t.primaryKeyId {
		return errors.New("the last field must be id")
	}

	//check if index already exists
	if t.HasIndex(keys) {
		logger.Warningf("Index already exists for table %v : %v", t.name, keys)
		return nil
	}

	//add index
	t.index = append(t.index, keys)
	for _, f := range keys {
		if _, ok := t.keyFields[f]; ok {
			t.keyFields[f] = t.keyFields[f] + 1
		} else {
			t.keyFields[f] = 1
		}
	}

	if updateAllRows {
		return t.UpdateAllRows()
	}
	return nil
}

// UpdateAllRows updates index values for all rows in the table.
func (t *Table) UpdateAllRows() error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	// Get each row for this index
	iter, err := t.GetRowsByPartialKey([]string{t.primaryKeyId}, []string{})
	if err != nil {
		logger.Errorf("Error fetching rows: %v", err)
		return errors.WithStack(err)
	}

	defer iter.Close()
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("Error reading row: %v", err)
			continue
		}
		rowBytes := KV.GetValue()
		var row map[string]string
		err = json.Unmarshal(rowBytes, &row)
		if err != nil {
			logger.Errorf("Error Umnarshal row: %v", err)
			continue
		}
		t.UpdateRow(row)
	}
	return nil
}

// get full data
func (t *Table) getSymKey(primaryKey string) []byte {
	//get symkey for encryption
	symkey := []byte{}
	if t.isEncrypted {
		symkey = crypto.GetSymKeyFromHash([]byte("key" + t.name + primaryKey))
	}
	return symkey
}

// GetFullRowData returns returns a row data with all index field values populated
func (t *Table) getFullRowData(primaryKey string) (map[string]string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	var keys map[string]string = make(map[string]string)
	var err error = nil

	//ledger key to save full data
	prefix := t.prefix([]string{"data_", t.primaryKeyId})
	key, err := t.createSimpleKey(prefix, []string{primaryKey})
	if err != nil {
		return keys, err
	}

	//get full data
	var keyBytes []byte
	if len(t.dataStoreId) > 0 {
		//let's make sure dataStore is intialized
		if t.dataStore == nil {
			t.dataStore, err = cloudant_index.GetIndexDatastoreImpl(t.tr.Stub, t.dataStoreId)
			if err != nil {
				logger.Debugf("Failed to initialize datastore: %v", err)
				return keys, err
			}
		}
		keyBytes, err = t.dataStore.GetIndex(t.tr.Stub, key)
	} else if t.useTree {
		keyBytes, err = t.tr.Get(key)
	} else {
		keyBytes, err = t.tr.Stub.GetState(key)
	}

	// decrypt if encrypted
	if t.isEncrypted && err == nil && keyBytes != nil {
		symkey := t.getSymKey(primaryKey)
		keyBytes, err = crypto.DecryptWithSymKey(symkey, keyBytes)
	}

	if keyBytes != nil && err == nil {
		err = json.Unmarshal(keyBytes, &keys)
	}

	return keys, err
}

// save full data
func (t *Table) saveFullRowData(keys map[string]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	var err error = nil

	//ledger key to save full data
	prefix := t.prefix([]string{"data_", t.primaryKeyId})
	key, err := t.createSimpleKey(prefix, []string{keys[t.primaryKeyId]})
	if err != nil {
		return err
	}

	// save keybytes
	keyBytes, err := json.Marshal(&keys)
	if err != nil {
		return err
	}
	//encrypt
	if t.isEncrypted {
		symkey := t.getSymKey(keys[t.primaryKeyId])
		keyBytes, err = crypto.EncryptWithSymKey(symkey, keyBytes)
		if err != nil {
			return err
		}
	}

	// save keybytes (full data)
	if len(t.dataStoreId) > 0 {
		//let's make sure dataStore is intialized
		if t.dataStore == nil {
			t.dataStore, err = cloudant_index.GetIndexDatastoreImpl(t.tr.Stub, t.dataStoreId)
			if err != nil {
				logger.Debugf("Failed to initialize datastore: %v", err)
				return err
			}
		}
		_, err = t.dataStore.PutIndex(t.tr.Stub, key, keyBytes)
	} else if t.useTree {
		err = t.tr.Insert(key, keyBytes)
	} else {
		err = t.tr.Stub.PutState(key, keyBytes)
	}
	return err
}

// UpdateRow updates index values for a row in the table.
func (t *Table) UpdateRow(keys map[string]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	if miss := t.missingKeys(keys); len(miss) > 0 {
		return errors.Errorf("key fields missing: %v", miss)
	}
	//logger.Debugf("keys: %v", keys)

	//get symkey for encryption
	primaryKey := keys[t.primaryKeyId]

	// Save only primaryKeyID to each index entry
	savekeys := make(map[string]string)
	savekeys[t.primaryKeyId] = keys[t.primaryKeyId]
	//logger.Debugf("savekeys: %v", savekeys)
	savekeyBytes, err := json.Marshal(&savekeys)
	if err != nil {
		return errors.WithStack(err)
	}

	//check if index needs update or not by comparing old data with new data
	oldKeys, err := t.getFullRowData(primaryKey)
	if err != nil {
		return errors.WithStack(err)
	}
	//logger.Debugf("oldKeys: %v", oldKeys)

	var needFullUpdate bool = !reflect.DeepEqual(keys, oldKeys)
	logger.Debugf("Need full update: %v", needFullUpdate)

	// if full update is needed (data element value has been changed), then
	// save new data value, and then delete old index first before adding new index
	if needFullUpdate {
		// save fulldata
		err = t.saveFullRowData(keys)
		if err != nil {
			return errors.WithStack(err)
		}

		//update full index
		for _, k := range t.index {
			prefix := t.prefix(k)
			var oldKey = []string{}
			var newKey = []string{}
			for _, f := range k {
				oldKey = append(oldKey, oldKeys[f])
				newKey = append(newKey, keys[f])
			}

			//delete old index
			if len(oldKeys[t.primaryKeyId]) > 0 {
				key1, err := t.createSimpleKey(prefix, oldKey)
				if err != nil {
					return err
				}

				if len(t.dataStoreId) > 0 {
					t.dataStore.Delete(t.tr.Stub, key1)
				} else if t.useTree {
					err = t.tr.Remove(key1)
				} else {
					err = t.tr.Stub.DelState(key1)
				}
				if err != nil {
					return errors.WithStack(err)
				}
			}

			//add new index
			key2, err := t.createSimpleKey(prefix, newKey)
			if err != nil {
				return err
			}
			if len(t.dataStoreId) > 0 {
				t.dataStore.PutIndex(t.tr.Stub, key2, savekeyBytes)
			} else if t.useTree {
				err = t.tr.Insert(key2, savekeyBytes)
			} else {
				err = t.tr.Stub.PutState(key2, savekeyBytes)
			}
			if err != nil {
				return errors.WithStack(err)
			}
		}
	} else {
		for _, k := range t.index {
			prefix := t.prefix(k)
			var newKey = []string{}
			for _, f := range k {
				newKey = append(newKey, keys[f])
			}

			//add new index
			key2, err := t.createSimpleKey(prefix, newKey)
			if err != nil {
				return err
			}
			if len(t.dataStoreId) > 0 {
				t.dataStore.PutIndex(t.tr.Stub, key2, savekeyBytes)
			} else if t.useTree {
				err = t.tr.Insert(key2, savekeyBytes)
			} else {
				err = t.tr.Stub.PutState(key2, savekeyBytes)
			}
			if err != nil {
				return errors.WithStack(err)
			}
		}

	}
	return nil
}

// DeleteRow removes the row specified by id from the table.
func (t *Table) DeleteRow(id string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	// get existing index
	oldKeys, err := t.getFullRowData(id)
	if err != nil {
		return err
	}

	for _, k := range t.index {
		prefix := t.prefix(k)
		var oldKey = []string{}
		for _, f := range k {
			oldKey = append(oldKey, oldKeys[f])
		}
		//delete old index
		key1, err := t.createSimpleKey(prefix, oldKey)
		if err != nil {
			return err
		}
		if len(t.dataStoreId) > 0 {
			t.dataStore.Delete(t.tr.Stub, key1)
		} else if t.useTree {
			err = t.tr.Remove(key1)
		} else {
			err = t.tr.Stub.DelState(key1)
		}
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// GetRow returns the row specified by id from the table.
// Note that only field you will get in a row is the primaryID field.
func (t *Table) GetRow(id string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("id: %v", id)
	prefix := t.prefix([]string{t.primaryKeyId})
	key, err := t.createSimpleKey(prefix, []string{id})
	if err != nil {
		return []byte{}, err
	}
	keyBytes := []byte{}
	if len(t.dataStoreId) > 0 {
		t.dataStore.GetIndex(t.tr.Stub, key)
	} else if t.useTree {
		keyBytes, err = t.tr.Get(key)
	} else {
		keyBytes, err = t.tr.Stub.GetState(key)
	}
	return keyBytes, err
}

// createSimpleKey is just like CreateCompositeKey, but the null character is stripped from the front so that it goes in the
// simple key namespace. Then we can perform range queries on it.
func (t *Table) createSimpleKey(objectType string, attributes []string) (string, error) {
	keys := []string{}
	if t.isEncrypted {
		keys = encryptKeys(objectType, attributes)
	} else {
		keys = attributes
	}
	compositeKey, err := t.tr.Stub.CreateCompositeKey(objectType, keys)
	simpleKey := ""
	if len(compositeKey) > 0 {
		simpleKey = compositeKey[1:]
	}
	return simpleKey, errors.WithStack(err)
}

// CreateRangeKey creates a simple key for calling GetRowsByRange.
func (t *Table) CreateRangeKey(fieldNames []string, fieldValues []string) (string, error) {
	index, err := t.findIndex(fieldNames)
	if err != nil {
		return "", err
	}
	return t.createSimpleKey(index, fieldValues)
}

// GetRowsByRange returns a range iterator over a set of rows in this table.
// The iterator can be used to iterate over all rows between the startKey (inclusive) and endKey (exclusive).
// The rows are returned by the iterator in lexical order.
// Note that startKey and endKey can be empty strings, which implies an unbounded range query at start or end.
// GetRowsByRange is not allowed if index is encrypted (if table's isEncrypted = true).
func (t *Table) GetRowsByRange(startKey string, endKey string) (shim.StateQueryIteratorInterface, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("GetRowsByRange startKey: %v", GetPrettyLedgerKey(startKey))
	logger.Debugf("GetRowsByRange endKey: %v", GetPrettyLedgerKey(endKey))

	if len(t.dataStoreId) > 0 {
		return t.dataStore.GetIndexByRange(t.tr.Stub, startKey, endKey, 0, "")
	} else if t.useTree {
		return t.tr.GetKeyByRange(startKey, endKey)
	} else {
		return t.tr.Stub.GetStateByRange(startKey, endKey)
	}
}

// GetRowsByPartialKey returns an iterator over a set of rows in this table.
// The iterator can be used to iterate over all rows that satisfy the provided index values.
// Note: sort order is disabled if index is encrypted
func (t *Table) GetRowsByPartialKey(fieldNames []string, fieldValues []string) (shim.StateQueryIteratorInterface, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	index, err := t.findIndex(fieldNames)
	if err != nil {
		return nil, err
	}
	rangeKey, err := t.createSimpleKey(index, fieldValues)
	if err != nil {
		return nil, err
	}
	//logger.Debugf("RangeKey: %v", rangeKey)

	// GetStateByRange doesn't work on composite keys so we had to abandon them in favor of simple keys.
	if len(t.dataStoreId) > 0 {
		return t.dataStore.GetIndexByRange(t.tr.Stub, rangeKey, rangeKey+string(global.MAX_UNICODE_RUNE_VALUE), 0, "")
	} else if t.useTree {
		return t.tr.GetKeyByRange(rangeKey, rangeKey+string(global.MAX_UNICODE_RUNE_VALUE))
	} else {
		return t.tr.Stub.GetStateByRange(rangeKey, rangeKey+string(global.MAX_UNICODE_RUNE_VALUE))
	}
}

// SaveToLedger saves the table to the ledger.
func (t *Table) SaveToLedger() error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	var itable iTable = iTable{}
	itable.Name = t.name
	itable.Index = t.index
	itable.KeyFields = t.keyFields
	itable.PrimaryKeyId = t.primaryKeyId
	itable.UseTree = t.useTree
	itable.IsEncrypted = t.isEncrypted
	itable.DatastoreId = t.dataStoreId
	tableBytes, err := json.Marshal(itable)
	if err != nil {
		return errors.WithStack(err)
	}
	logger.Debugf("table: %v", string(tableBytes))
	key, err := t.tr.Stub.CreateCompositeKey("Table", []string{t.name})
	if err != nil {
		logger.Errorf("what error: %v", err)
		return errors.WithStack(err)
	}
	return t.tr.Stub.PutState(key, tableBytes)
}

// GetPrettyLedgerKey is to be used for debug print statements only!
// Replaces global.MIN_UNICODE_RUNE_VALUE with "_" and global.MAX_UNICODE_RUNE_VALUE with "*".
// Composite keys are also prefixed with a global.MIN_UNICODE_RUNE_VALUE.
func GetPrettyLedgerKey(ledgerKey string) string {
	prettyKey := strings.Replace(ledgerKey, string(global.MIN_UNICODE_RUNE_VALUE), "_", -1)
	prettyKey = strings.Replace(prettyKey, string(global.MAX_UNICODE_RUNE_VALUE), "*", -1)
	return prettyKey
}

// Encrypt keys to hide text based on randomized hash
func encryptKeys(prefix string, keys []string) []string {
	newkeys := []string{}
	hash := crypto.HashShort([]byte(prefix))
	for _, k := range keys {
		enck := encryptKey(hash, k)
		newkeys = append(newkeys, enck)
		hash = crypto.HashShort(hash)
	}
	return newkeys
}

// randomize index keys with our custome logic
func encryptKey(h []byte, key string) string {
	newKey := ""
	for i := 0; i < len(key); i++ {
		j := i % len(h)
		n := int(key[i])*int(h[len(h)-1-j]) + int(h[j])
		x := fmt.Sprintf("%04X", n)
		r := crypto.EncodeToB64String([]byte(x[3:4] + "8"))
		newKey = newKey + x[0:3] + r[0:1]
	}
	return newKey
}
