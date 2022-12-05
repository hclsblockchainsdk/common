/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package ledger

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/datastore"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/metering_i"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("data_store/ledger")

//Shell of cloudant impl. To be impl in different MR
type LedgerDatastoreImpl struct {
	connection datastore.DatastoreConnection
	dbType     string
}

// Init sets up the ledger package
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	return nil, nil
}

// impl DataStoreInterface methods

func (ds LedgerDatastoreImpl) Instantiate(conn datastore.DatastoreConnection) (datastore.DatastoreInterface, error) {
	//todo
	return &LedgerDatastoreImpl{connection: conn, dbType: global.DEFAULT_LEDGER_DATASTORE_ID}, nil
}

func (ds LedgerDatastoreImpl) GetDatastoreConnection() datastore.DatastoreConnection {
	return ds.connection
}

func (ds LedgerDatastoreImpl) IsReady() bool {
	return true
}

func (ds LedgerDatastoreImpl) Put(stub cached_stub.CachedStubInterface, encryptedData []byte) (string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	dataKey := ds.ComputeHash(stub, encryptedData)
	key := ds.dbType + "-" + dataKey
	err := stub.PutState(key, encryptedData)
	return dataKey, err
}

func (ds LedgerDatastoreImpl) Get(stub cached_stub.CachedStubInterface, dataKey string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	key := ds.dbType + "-" + dataKey
	data, err := stub.GetState(key)
	if err != nil {
		return data, err
	}
	hash := ds.ComputeHash(stub, data)
	if hash != dataKey {
		logger.Errorf("Invalid hash %v: it should match with %v", hash, dataKey)
		return data, errors.New("Invalid hash")
	}
	return data, nil
}

func (ds LedgerDatastoreImpl) Delete(stub cached_stub.CachedStubInterface, dataKey string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	key := ds.dbType + "-" + dataKey
	return stub.DelState(key)
}

func (ds LedgerDatastoreImpl) ComputeHash(stub cached_stub.CachedStubInterface, encryptedData []byte) string {
	return crypto.HashB64(encryptedData)
}
