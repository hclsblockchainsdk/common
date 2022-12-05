/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package cloudant

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/datastore"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/metering_i"
	"common/bchcls/utils"

	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("data_store/cloudant")

var datastoreImplementationMap = make(map[string]CloudantDatastoreImpl)

//Shell of cloudant impl. To be impl in different MR
type CloudantDatastoreImpl struct {
	connection datastore.DatastoreConnection
	dbType     string
}

type DbCloudantDataCreate struct {
	Id     string `json:"_id"`
	Hash   string `json:"hash"`
	Data   string `json:"data"`
	TxID   string `json:"txid"`
	TxTime int64  `json:"txtime"`
}

type DbCloudantData struct {
	Id     string `json:"_id"`
	Rev    string `json:"_rev"`
	Hash   string `json:"hash"`
	Data   string `json:"data"`
	TxID   string `json:"txid"`
	TxTime int64  `json:"txtime"`
}

// Init sets up the cloudant package
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	return nil, nil
}

// impl DataStoreInterface methods

// Instantiate initializes the database and connection
// connectS?tring of cloudant datastore should have the following parameters
// username - user name
// password - password
// database - database name
// host - complete host url
// create_database - true | false : if true, it will try to create the database during Instantiate process if the database does not exist
func (ds CloudantDatastoreImpl) Instantiate(conn datastore.DatastoreConnection) (datastore.DatastoreInterface, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("ID: %v", conn.ID)
	connBytes, _ := json.Marshal(&conn)
	connHash := crypto.HashB64(connBytes)
	datastore, ok := datastoreImplementationMap[connHash]
	if ok {
		logger.Debugf("CloudDatastoreImple found in datastoreImplementationMap: %v", conn.ID)
		return &datastore, nil
	}

	datastore = CloudantDatastoreImpl{connection: conn, dbType: global.DATASTORE_TYPE_DEFAULT_CLOUDANT}
	if !datastore.IsReady() {
		// attemp to create database
		args, _ := url.ParseQuery(strings.TrimSpace(conn.ConnectStr))
		var create_db = false
		if _, ok := args["create_database"]; ok {
			create_db = ("true" == args["create_database"][0])
		}

		if create_db {
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

			//create database and check again if the database exist
			url := host + "/" + database + "?partitioned=true"
			logger.Debugf("database url: %v", url)
			response, body, err := utils.PerformHTTP("PUT", url, nil, headers, []byte(user), []byte(pass))
			if err != nil || response == nil {
				logger.Errorf("Cloudant DB creation failed %v %v %v", err, response, body)
				//return nil, errors.WithMessage(err, "Cloudant DB creation failed")
			} else if response.StatusCode >= 200 && response.StatusCode <= 299 {
				logger.Debugf("Cloudant DB is created: %v %v", response, body)
			} else {
				logger.Debugf("Cloudant DB creation response: %v", response)
			}

			if !datastore.IsReady() {
				return nil, errors.New("Cloudant DB is not ready")
			}
		} else {
			return nil, errors.New("Cloudant DB is not ready")
		}
	}

	datastoreImplementationMap[connHash] = datastore
	return &datastore, nil
}

func (ds CloudantDatastoreImpl) GetDatastoreConnection() datastore.DatastoreConnection {
	return ds.connection
}

func (ds CloudantDatastoreImpl) IsReady() bool {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	connection := ds.GetDatastoreConnection()
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
		return false
	}

	var pass = ""
	if _, ok := args["password"]; ok {
		pass = args["password"][0]
	} else {
		logger.Error("Invalid connect string: password is missing")
		return false
	}

	var host = "https://" + user + ".cloudantnosqldb.appdomain.cloud"
	if _, ok := args["host"]; ok {
		host = args["host"][0]
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	//check if the database exist
	url := host + "/" + database
	logger.Debugf("read url: %v", url)
	response, body, err := utils.PerformHTTP("GET", url, nil, headers, []byte(user), []byte(pass))
	if err != nil || response == nil {
		logger.Errorf("Cloudant DB operation failed %v %v %v", err, response, body)
		return false
	} else if response.StatusCode >= 200 && response.StatusCode <= 299 {
		logger.Debugf("Cloudant DB is ready: %v %v", response, body)
		return true
	}
	logger.Errorf("Something is wrong with Cloudant DB operation %v %v %v", err, response, body)
	return false
}

func (ds CloudantDatastoreImpl) Put(stub cached_stub.CachedStubInterface, encryptedData []byte) (string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	dataKey := ds.ComputeHash(stub, encryptedData)
	logger.Debugf("DataKey: %v", dataKey)
	//before writing to the db, first check if the data with same key (hash, dataType, timestamp, etc) alreay exist
	//if the data is alredy in db, skip and do not try to write again.

	//get connect string from connection
	//connect string shoul have all info needed for a given DB type.
	//database=DBNAME&username=USERID&password=PASSWORD&host=HOSTNAMEORIPADDR

	connection := ds.GetDatastoreConnection()
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
		return "", errors.New("Invalid connect string: username is missing")
	}

	var pass = ""
	if _, ok := args["password"]; ok {
		pass = args["password"][0]
	} else {
		logger.Error("Invalid connect string: password is missing")
		return "", errors.New("Invalid connect string: password is missing")
	}

	var host = "https://" + user + ".cloudantnosqldb.appdomain.cloud"
	if _, ok := args["host"]; ok {
		host = args["host"][0]
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	//check if the data is already there, and skip if it's already there
	key := ds.dbType + ":" + dataKey
	//key := dataKey
	readurl := host + "/" + database + "/" + key
	logger.Debugf("read url: %v", readurl)
	response, body, err := utils.PerformHTTP("GET", readurl, nil, headers, []byte(user), []byte(pass))
	if err != nil || response == nil {
		logger.Errorf("Cloudant DB operation failed %v %v %v", err, response, body)
		return dataKey, errors.New("Cloudant DB operation failed")
	} else if response.StatusCode >= 200 && response.StatusCode <= 299 {
		logger.Infof("Skip:Data with same key already exist: %v", dataKey)
		// TODO: verify hash
		return dataKey, nil
	}

	//write the actual data
	url := host + "/" + database
	txTimestamp, _ := stub.GetTxTimestamp()
	dbdata := DbCloudantDataCreate{}
	dbdata.Id = key
	dbdata.Hash = dataKey
	dbdata.Data = crypto.EncodeToB64String(encryptedData)
	dbdata.TxID = stub.GetTxID()
	dbdata.TxTime = txTimestamp.GetSeconds()

	dbdataBytes, _ := json.Marshal(dbdata)

	logger.Debugf("write url: %v", url)
	response, body, err = utils.PerformHTTP("POST", url, dbdataBytes, headers, []byte(user), []byte(pass))
	logger.Errorf("%v, %v, %v", response, body, err)
	if err != nil || response == nil {
		logger.Errorf("Cloudant DB create document operation failed: %v", err)

		//check again if the data is written by other peers
		logger.Debugf("read url: %v", readurl)
		response, body, err := utils.PerformHTTP("GET", readurl, nil, headers, []byte(user), []byte(pass))
		if err != nil || response == nil {
			logger.Errorf("Cloudant DB operation failed %v %v %v", err, response, body)
			return dataKey, errors.New("Cloudant DB operation failed")
		} else if response.StatusCode >= 200 && response.StatusCode <= 299 {
			logger.Infof("Skip:Data with same key already exist: %v", dataKey)
			// TODO: verify hash
			return dataKey, nil
		}

		return dataKey, errors.New("Cloudant DB operation failed")
	} else if response.StatusCode >= 200 && response.StatusCode <= 299 {

		logger.Infof("Cloudant DB create document operation success: %v %v", response, body)
	} else if response.StatusCode == 409 {

		logger.Infof("Skip:Data with same key written by other peer: %v %V", dataKey, response)
	} else {

		logger.Errorf("Cloudant DB create document operation failed %v %v", response, body)

		//check again if the data is written by other peers
		logger.Debugf("read url: %v", readurl)
		response, body, err := utils.PerformHTTP("GET", readurl, nil, headers, []byte(user), []byte(pass))
		if err != nil || response == nil {
			logger.Errorf("Cloudant DB operation failed %v %v %v", err, response, body)
			return dataKey, errors.New("Cloudant DB operation failed")
		} else if response.StatusCode >= 200 && response.StatusCode <= 299 {
			logger.Infof("Skip:Data with same key already exist: %v", dataKey)
			// TODO: verify hash
			return dataKey, nil
		}

		return dataKey, errors.New("Cloudant DB create document operation failed")
	}

	return dataKey, nil
}

func (ds CloudantDatastoreImpl) Get(stub cached_stub.CachedStubInterface, dataKey string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	logger.Debugf("DataKey: %v", dataKey)
	key := ds.dbType + ":" + dataKey
	//key := dataKey
	connection := ds.GetDatastoreConnection()
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

	// read data from DB
	url := host + "/" + database + "/" + key
	logger.Debugf("url: %v", url)
	response, body, err := utils.PerformHTTP("GET", url, nil, headers, []byte(user), []byte(pass))
	if err != nil || response == nil || body == nil {
		logger.Errorf("Cloudant DB operation failed %v %v %v", err, response, body)
		return nil, errors.New("Cloudant DB operation failed")
	} else if response.StatusCode < 200 || response.StatusCode > 299 {
		logger.Errorf("Data does not exist: %v %v", response, body)
		return nil, errors.New("Data does not exist")
	} else {
		logger.Debugf("Cloudant DB got data: %v", response)
	}

	dbdata := DbCloudantData{}
	json.Unmarshal(body, &dbdata)

	// verify hash
	bytesData, err := crypto.DecodeStringB64(dbdata.Data)
	if err != nil {
		logger.Errorf("Invalid data or B64 format: %v", err)
		return nil, errors.Wrap(err, "Invalid data or B64 format")
	}

	hash := ds.ComputeHash(stub, bytesData)
	if hash != dbdata.Hash || hash != dataKey {
		logger.Errorf("Invalid hash value: %v %v %v", dataKey, hash, dbdata.Hash)
		return nil, errors.New("Invalid hash value")
	}

	return bytesData, nil
}

func (ds CloudantDatastoreImpl) Delete(stub cached_stub.CachedStubInterface, dataKey string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	logger.Debugf("DataKey: %v", dataKey)
	key := ds.dbType + ":" + dataKey
	//key := dataKey

	connection := ds.GetDatastoreConnection()
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
		return errors.New("Invalid connect string: username is missing")
	}

	var pass = ""
	if _, ok := args["password"]; ok {
		pass = args["password"][0]
	} else {
		logger.Error("Invalid connect string: password is missing")
		return errors.New("Invalid connect string: password is missing")
	}

	var host = "https://" + user + ".cloudantnosqldb.appdomain.cloud"
	if _, ok := args["host"]; ok {
		host = args["host"][0]
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	// read data from DB
	url := host + "/" + database + "/" + key
	logger.Debugf("url: %v", url)
	response, body, err := utils.PerformHTTP("GET", url, nil, headers, []byte(user), []byte(pass))
	if err != nil || response == nil || body == nil {
		logger.Errorf("Cloudant DB operation failed %v %v %v", err, response, body)
		return errors.New("Cloudant DB operation failed")
	} else if response.StatusCode == 404 || response.StatusCode == 409 {
		logger.Errorf("Data does not exist: %v %v", response, body)
		return nil
	} else if response.StatusCode < 200 || response.StatusCode > 299 {
		logger.Errorf("Error to access data: %v %v", response, body)
		return errors.New("Error to access data")
	} else {
		logger.Debugf("Cloudant DB got data: %v", response)
	}

	dbdata := DbCloudantData{}
	json.Unmarshal(body, &dbdata)

	//delete from DB
	rev := dbdata.Rev

	url = host + "/" + database + "/" + key + "?rev=" + rev
	logger.Debugf("url: %v", url)
	response, body, err = utils.PerformHTTP("DELETE", url, nil, headers, []byte(user), []byte(pass))
	if err != nil || response == nil || body == nil {
		logger.Errorf("Cloudant DB operation failed %v %v %v", err, response, body)
		return errors.New("Cloudant DB operation failed")
	} else if response.StatusCode == 404 || response.StatusCode == 409 {
		logger.Errorf("Data deleted by other peers: %v %v", response, body)
		return nil
	} else if response.StatusCode < 200 || response.StatusCode > 299 {
		logger.Infof("Cloudant delete failed: %v %v", response, body)
		return errors.New("Cloudant delete failed")
	} else {
		logger.Debugf("Cloudant DB deleted data: %v", response)
	}
	return nil
}

func (ds CloudantDatastoreImpl) ComputeHash(stub cached_stub.CachedStubInterface, encryptedData []byte) string {
	hash := crypto.Hash(encryptedData)
	return hex.EncodeToString(hash)
}
