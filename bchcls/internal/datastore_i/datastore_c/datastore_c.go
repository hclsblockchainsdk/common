/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package datastore_c

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/custom_errors"
	"common/bchcls/datastore"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/datastore_i/datastore_c/cloudant"
	"common/bchcls/internal/datastore_i/datastore_c/ledger"
	"common/bchcls/utils"

	"encoding/json"
	"net/url"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

const DATASTORE_CONNECTION_PREFIX = "Datastore"
const DATASTORE_CONNECTION_KEY_PREFIX = "DatastoreHashStringPrefix"

var logger = shim.NewLogger("datastore_c")

// datastoreImplementationMap Maps datastoreType -> DatastoreInterface Impl
// This map is always initialized with default implementations
var datastoreImplementationMap = defaultDatastoreImplMap()

// Init sets up the datastore package by adding default ledger connection.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	ledger.Init(stub, logLevel...)
	cloudant.Init(stub, logLevel...)

	logger.Debug("Init datastore")
	//register default datastore connection
	connection := datastore.DatastoreConnection{
		ID:   global.DEFAULT_LEDGER_DATASTORE_ID,
		Type: datastore.DATASTORE_TYPE_DEFAULT_LEDGER,
	}
	err := PutDatastoreConnection(stub, connection)
	if err != nil {
		return nil, err
	}

	return nil, err
}

// InitDefaultDatastore sets up the datastore package by adding default cloudant datastore.
// connectString will be passed in by init_common
func InitDefaultDatastore(stub cached_stub.CachedStubInterface,
	args ...string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	if len(args) != 4 && len(args) != 5 {
		logger.Error("Wrong number of args: expecting 4 or 5, but got %v", len(args))
		return nil, errors.New("Wrong number of args")
	}
	params := url.Values{}
	params.Add("username", args[0])
	params.Add("password", args[1])
	params.Add("database", args[2])
	params.Add("host", args[3])
	params.Add("create_database", "true")

	connection := datastore.DatastoreConnection{
		ID:         global.DEFAULT_CLOUDANT_DATASTORE_ID,
		Type:       datastore.DATASTORE_TYPE_DEFAULT_CLOUDANT,
		ConnectStr: params.Encode(),
	}
	err := PutDatastoreConnection(stub, connection)

	// drop database
	if len(args) >= 5 && args[4] == "true" {
		logger.Debug("Attempt to delete database before instantiating")
		// attemp to drop database
		args, _ := url.ParseQuery(strings.TrimSpace(connection.ConnectStr))

		database := "DatastoreCloudant"
		if _, ok := args["database"]; ok {
			database = args["database"][0]
		}

		var user = ""
		if _, ok := args["username"]; ok {
			user = args["username"][0]
		} else {
			logger.Error("Invalid connect string: username is missing")
			return nil, errors.New("Invalid connect string: username is missing")
		}

		var pass = ""
		if _, ok := args["password"]; ok {
			pass = args["password"][0]
		} else {
			logger.Error("Invalid connect string: password is missing")
			return nil, errors.New("Invalid connect string: password is missing")
		}

		var host = "https://" + user + ".cloudantnosqldb.appdomain.cloud"
		if _, ok := args["host"]; ok {
			host = args["host"][0]
		}

		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"

		//drop database befor create
		url := host + "/" + database
		logger.Debugf("delete url: %v", url)
		response, body, err := utils.PerformHTTP("DELETE", url, nil, headers, []byte(user), []byte(pass))
		if err != nil || response == nil {
			logger.Errorf("Cloudant DB deletion failed %v %v %v", err, response, body)
			return nil, errors.WithMessage(err, "Cloudant DB deletion failed")
		} else if response.StatusCode >= 200 && response.StatusCode <= 299 {
			logger.Debugf("Cloudant DB is deleted: %v %v", response, body)
		} else {
			logger.Debugf("Cloudant DB is not deleted, assuming it's okay: %v %v", response, body)
		}

	}
	return nil, err
}

// PutDatastoreConnection encrypts using dsSymKey and stores the connection on the ledger.
func PutDatastoreConnection(stub cached_stub.CachedStubInterface,
	datastoreConnection datastore.DatastoreConnection) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("ID: %v", datastoreConnection.ID)
	myObjectLedgerKeyHash := crypto.HashB64([]byte(DATASTORE_CONNECTION_KEY_PREFIX + datastoreConnection.ID))
	myObjectLedgerKey, err := stub.CreateCompositeKey(DATASTORE_CONNECTION_PREFIX, []string{myObjectLedgerKeyHash})
	if err != nil {
		customErr := &custom_errors.CreateCompositeKeyError{Type: DATASTORE_CONNECTION_PREFIX}
		logger.Errorf("%v: %v", customErr, err)
		return errors.Wrap(err, customErr.Error())
	}

	myObjectBytes, err := json.Marshal(&datastoreConnection)
	if err != nil {
		customErr := &custom_errors.MarshalError{Type: DATASTORE_CONNECTION_PREFIX}
		logger.Errorf("%v: %v", customErr, err)
		return errors.Wrap(err, customErr.Error())
	}

	//get dsSymKey
	dsSymKey := crypto.GetSymKeyFromHash([]byte(datastoreConnection.ID))

	// Encrypt the data with sym key
	privateData, err := crypto.EncryptWithSymKey(dsSymKey, myObjectBytes)
	if err != nil {
		customErr := &custom_errors.EncryptionError{ToEncrypt: "PrivateData", EncryptionKey: datastoreConnection.ID}
		return errors.Wrap(err, "Failed to encrypt DatastoreConnection: "+customErr.Error())
	}
	if privateData == nil {
		errMsg := "PrivateData found to be empty while encrypting DatastoreConnection details"
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}
	logger.Infof("PutState for DatastoreConnection id:" + datastoreConnection.ID)
	err = stub.PutState(myObjectLedgerKey, privateData)
	if err != nil {
		customErr := &custom_errors.PutLedgerError{LedgerKey: myObjectLedgerKey}
		logger.Errorf("%v: %v", customErr, err)
		return errors.Wrap(err, customErr.Error())
	}

	return nil
}

// DeleteDatastoreConnection deletes the connection from the ledger
func DeleteDatastoreConnection(stub cached_stub.CachedStubInterface,
	connectionID string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("ID: %v", connectionID)
	myObjectLedgerKeyHash := crypto.HashB64([]byte(DATASTORE_CONNECTION_KEY_PREFIX + connectionID))
	myObjectLedgerKey, err := stub.CreateCompositeKey(DATASTORE_CONNECTION_PREFIX, []string{myObjectLedgerKeyHash})
	if err != nil {
		customErr := &custom_errors.CreateCompositeKeyError{Type: DATASTORE_CONNECTION_PREFIX}
		logger.Errorf("%v: %v", customErr, err)
		return errors.Wrap(err, customErr.Error())
	}

	logger.Infof("DeleteState for DatastoreConnection id:" + connectionID)
	err = stub.DelState(myObjectLedgerKey)
	if err != nil {
		customErr := &custom_errors.DeleteLedgerError{LedgerKey: myObjectLedgerKey}
		logger.Errorf("%v: %v", customErr, err)
		return errors.Wrap(err, customErr.Error())
	}

	return nil
}

// GetDatastoreConnection returns DatastoreConnection from the ledger
func GetDatastoreConnection(stub cached_stub.CachedStubInterface, connectionID string) (datastore.DatastoreConnection, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	logger.Debugf("ID: %v", connectionID)
	myObject := datastore.DatastoreConnection{}
	datastoreConnLedgerKeyHash := crypto.HashB64([]byte(DATASTORE_CONNECTION_KEY_PREFIX + connectionID))
	datastoreConnLedgerKey, err := stub.CreateCompositeKey(DATASTORE_CONNECTION_PREFIX, []string{datastoreConnLedgerKeyHash})
	if err != nil {
		customErr := &custom_errors.CreateCompositeKeyError{Type: DATASTORE_CONNECTION_PREFIX}
		logger.Errorf("%v: %v", customErr, err)
		return myObject, errors.Wrap(err, customErr.Error())
	}
	logger.Debugf("GetDatastoreConnection for ID: %v ", connectionID)
	datastoreConnectionBytes, err := stub.GetState(datastoreConnLedgerKey)
	if err != nil {
		customErr := &custom_errors.GetLedgerError{LedgerKey: datastoreConnLedgerKey, LedgerItem: DATASTORE_CONNECTION_PREFIX}
		logger.Errorf("During datastore GetState: %v: %v", customErr, err)
		return myObject, errors.Wrap(err, customErr.Error())
	} else if datastoreConnectionBytes == nil {
		logger.Infof("DatastoreConnection not found with ledger key: \"%v\"", datastoreConnLedgerKey)
		return myObject, nil

	}
	//get dsSymKey
	dataStoreKey := crypto.GetSymKeyFromHash([]byte(connectionID))

	// attempt to decrypt using dataStoreKey
	decryptedBytes, err := crypto.DecryptWithSymKey(dataStoreKey, datastoreConnectionBytes)
	if err != nil {
		customErr := &custom_errors.DecryptionError{ToDecrypt: "DatastoreConnection", DecryptionKey: "sym key"}
		logger.Infof("%v: %v", customErr, err)
		return myObject, errors.Wrap(err, customErr.Error())
	}

	err = json.Unmarshal(decryptedBytes, &myObject)
	if err != nil {
		customErr := &custom_errors.UnmarshalError{Type: DATASTORE_CONNECTION_PREFIX}
		logger.Errorf("%v: %v", customErr.Error(), err)
		return myObject, errors.Wrap(err, customErr.Error())
	}
	return myObject, nil

}

// IsRegisteredDatastoreType returns true if the datastore type is one of the default ones or registered via RegisterDatastoreImpl method
func IsRegisteredDatastoreType(datastoreType string) bool {
	_, ok := datastoreImplementationMap[datastoreType]
	return ok
}

// GetDatastoreImpl returns a new DatastoreImpl instance that is initialized with DatastoreConnection identified by datastoreConnectionID. Caller should call this method once in a transaction and reuse the instance for data persistence.
func GetDatastoreImpl(stub cached_stub.CachedStubInterface, datastoreConnectionID string) (datastore.DatastoreInterface, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	conn, err := GetDatastoreConnection(stub, datastoreConnectionID)
	if err != nil {
		return nil, err
	}

	if utils.IsStringEmpty(conn.ID) {
		errMsg := "DatastoreConnection with ID " + datastoreConnectionID + " does not exist."
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	datastoreImplTemplate, ok := datastoreImplementationMap[conn.Type]
	if !ok {
		return nil, errors.New(conn.Type + " DatastoreType not found")
	}

	datastoreImpl, err := datastoreImplTemplate.Instantiate(conn)
	if err != nil {
		errMsg := " DatastoreInstantiate(DatastoreConnection) call failed"
		logger.Errorf(errMsg+" %v", err)
		return nil, errors.Wrap(err, errMsg)
	}

	logger.Debugf("new Datastore Instantiated for id:" + datastoreConnectionID)
	return datastoreImpl, nil
}

// RegisterDatastoreImpl is used by Solution to register a new Datastore Type. If default off-chain storage implementation
// is sufficient for your use case, there is no need to use this method.
func RegisterDatastoreImpl(datastoreType string, implementation datastore.DatastoreInterface) error {
	if _, ok := datastoreImplementationMap[datastoreType]; ok {
		return errors.New(datastoreType + " datastore type is already added")
	}

	datastoreImplementationMap[datastoreType] = implementation
	return nil
}

func defaultDatastoreImplMap() map[string]datastore.DatastoreInterface {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	datastoreTypes := make(map[string]datastore.DatastoreInterface)

	datastoreTypes[global.DATASTORE_TYPE_DEFAULT_CLOUDANT] = cloudant.CloudantDatastoreImpl{}
	datastoreTypes[global.DATASTORE_TYPE_DEFAULT_LEDGER] = ledger.LedgerDatastoreImpl{}
	return datastoreTypes
}
