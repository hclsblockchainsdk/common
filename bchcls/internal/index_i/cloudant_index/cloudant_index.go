/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package cloudant_index

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/datastore"
	"common/bchcls/internal/datastore_i/datastore_c"
	"common/bchcls/internal/datastore_i/datastore_c/cloudant"
	"common/bchcls/utils"

	"encoding/json"
	"net/url"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("cloudant_index")

type Docs struct {
	Docs []cloudant.DbCloudantData `json:"docs"`
}

// Init sets up the cloudant index package
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	return nil, nil
}

// CachedStubInterface extends ChaincodeStubInterface
type IndexDatastoreInterface interface {
	datastore.DatastoreInterface

	GetIndex(stub cached_stub.CachedStubInterface, dataKey string, dataHash ...string) ([]byte, error)

	GetIndexByRange(stub cached_stub.CachedStubInterface, startKey string, endKey string, limit int, lastKey string) (shim.StateQueryIteratorInterface, error)

	PutIndex(stub cached_stub.CachedStubInterface, dataKey string, encryptedData []byte) (string, error)
}

//Shell of cloudant impl. To be impl in different MR
type CloudantIndexDatastoreImpl struct {
	datastore.DatastoreInterface

	datastore  datastore.DatastoreInterface
	connection datastore.DatastoreConnection
	dbType     string
}

// impl DataStoreInterface methods

// connectS?tring of cloudant datastore should have the following parameters
// username - user name
// password - password
// database - database name
// host - complete host url
// create_database - true | false : if true, it will try to create the database during Instantiate process if the database does not exist

// PutIndex writes index data to the store, and returns hash
func (ds CloudantIndexDatastoreImpl) PutIndex(stub cached_stub.CachedStubInterface, dataKey string, encryptedData []byte) (string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	hash := ds.ComputeHash(stub, encryptedData)
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
	key := ds.getDsKey(dataKey)

	enckey := url.QueryEscape(key)
	readurl := host + "/" + database + "/" + enckey
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
	dbdata := cloudant.DbCloudantDataCreate{}
	dbdata.Id = key
	dbdata.Hash = hash
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

	return hash, nil
}

func (ds CloudantIndexDatastoreImpl) GetIndex(stub cached_stub.CachedStubInterface, dataKey string, dataHash ...string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("DataKey: %v", dataKey)
	key := ds.getDsKey(dataKey)
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
	enckey := url.QueryEscape(key)
	url := host + "/" + database + "/" + enckey
	logger.Debugf("url: %v", url)
	response, body, err := utils.PerformHTTP("GET", url, nil, headers, []byte(user), []byte(pass))
	if err != nil || response == nil || body == nil {
		logger.Errorf("Cloudant DB operation failed %v %v %v", err, response, body)
		return nil, errors.New("Cloudant DB operation failed")
	} else if response.StatusCode == 404 {
		logger.Debugf("Data does not exist: %v %v", response, body)
		return nil, nil
	} else if response.StatusCode < 200 || response.StatusCode > 299 {
		logger.Errorf("Error: %v %v", response, body)
		return nil, errors.New("Data does not exist")
	} else {
		logger.Debugf("Cloudant DB got data: %v", response)
	}

	dbdata := cloudant.DbCloudantData{}
	json.Unmarshal(body, &dbdata)

	bytesData, err := crypto.DecodeStringB64(dbdata.Data)
	if err != nil {
		logger.Errorf("Invalid data or B64 format: %v", err)
		return nil, errors.Wrap(err, "Invalid data or B64 format")
	}
	// verify hash
	if len(dataHash) > 0 {
		hash := ds.ComputeHash(stub, bytesData)
		if hash != dbdata.Hash && hash != dataHash[0] {
			logger.Errorf("Invalid hash value: %v %v %v", dataHash[0], hash, dbdata.Hash)
			return nil, errors.New("Invalid hash value")
		}
	}
	return bytesData, nil
}

func (ds CloudantIndexDatastoreImpl) GetIndexByRange(stub cached_stub.CachedStubInterface, startKey string, endKey string, limit int, lastKey string) (shim.StateQueryIteratorInterface, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("StartKey: %v, endKey: %v, limit: %v, lastKey: %v", startKey, endKey, limit, lastKey)
	startDsKey := ds.getDsKey(startKey)
	endDsKey := ds.getDsKey(endKey)
	lastDsKey := ds.getDsKey(lastKey)

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
	headers["Cache-Control"] = "no-cache"

	payload, err := GetJsonForFind(startDsKey, endDsKey, limit, lastDsKey)
	if err != nil {
		logger.Errorf("Failed to get JSON for Find API: %v", err)
		return nil, errors.New("Failed to get JSON for Find API")
	}

	// read data from DB
	docs := Docs{}
	url := host + "/" + database + "/_find"
	logger.Debugf("url: %v", url)
	logger.Debugf("payload: %v", string(payload))
	response, body, err := utils.PerformHTTP("POST", url, payload, headers, []byte(user), []byte(pass))
	if err != nil || response == nil || body == nil {
		logger.Errorf("Cloudant DB operation failed %v %v %v", err, response, body)
		return nil, errors.New("Cloudant DB operation failed")
	} else if response.StatusCode == 404 {
		logger.Debugf("Data does not exist: %v %v", response, body)
	} else if response.StatusCode < 200 || response.StatusCode > 299 {
		logger.Errorf("Error: %v %v", response, body)
		return nil, errors.New("Data does not exist")
	} else {
		logger.Debugf("Cloudant DB got data: %v", response)
		json.Unmarshal(body, &docs)
	}
	/*
		{
		    "docs": [
		        {
		            "_id": "ds.Cloudant:03d9ac158d44271c6a3d0e96a404e73c7aa1ed81b081ea1503b20d4f79862270",
		            "data": "eyJjb21wYW55IjoiQ29tIEMiLCJkZXB0IjoiRDIiLCJpZCI6IkNEMiJ9"
		        },
		        {
		            "_id": "ds.Cloudant:091f78b957baaaf59c944f87d22cf7aa697cd2107f1f5da85f84f4f1ebfe5859",
		            "data": "eyJjb21wYW55IjoiQ29tIEEiLCJkZXB0IjoiRDEiLCJpZCI6IkFEMSJ9"
		        }
		    ]
		}
	*/

	return NewIndexIter(startKey, endKey, docs)
}

func (ds CloudantIndexDatastoreImpl) getDsKey(key string) string {
	prefix := ds.dbType + ":"
	if len(key) == 0 {
		return ""
	} else if strings.HasPrefix(key, prefix) {
		return key
	} else {
		return prefix + key
	}
}

// GetDatastoreImpl returns a new IndexDatastoreImpl instance that is initialized with DatastoreConnection identified by datastoreConnectionID.
// Caller should call this method once in a transaction and reuse the instance for data persistence.
func GetIndexDatastoreImpl(stub cached_stub.CachedStubInterface, datastoreConnectionID string) (IndexDatastoreInterface, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	conn, err := datastore_c.GetDatastoreConnection(stub, datastoreConnectionID)
	if err != nil {
		return nil, err
	}

	if utils.IsStringEmpty(conn.ID) {
		errMsg := "DatastoreConnection with ID " + datastoreConnectionID + " does not exist."
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	if conn.Type != datastore.DATASTORE_TYPE_DEFAULT_CLOUDANT {
		errMsg := "Datastore type of '" + conn.Type + "' is not supported for index datastore"
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	datastoreImpl, err := datastore_c.GetDatastoreImpl(stub, datastoreConnectionID)
	if err != nil {
		return nil, err
	}

	indexDatastoreImpl := CloudantIndexDatastoreImpl{}
	indexDatastoreImpl.datastore = datastoreImpl
	indexDatastoreImpl.dbType = datastore.DATASTORE_TYPE_DEFAULT_CLOUDANT
	indexDatastoreImpl.connection = conn
	indexDatastoreImpl.DatastoreInterface = datastoreImpl

	logger.Debugf("new Index Datastore Instantiated for id:" + datastoreConnectionID)
	return indexDatastoreImpl, nil

}

type IndexIter struct {
	FirstKey string //inclusive
	LastKey  string //exclusive
	Docs     []cloudant.DbCloudantData
	Current  int
	Closed   bool
}

func (iIter *IndexIter) HasNext() bool {
	if !iIter.Closed && iIter.Current < len(iIter.Docs) {
		return true
	} else {
		return false
	}
}

func (iIter *IndexIter) Next() (*queryresult.KV, error) {
	if iIter.Closed {
		return nil, errors.New("Next() called after Closed()")
	}
	if iIter.HasNext() {
		dbdata := iIter.Docs[iIter.Current]
		iIter.Current = iIter.Current + 1
		bytesData, err := crypto.DecodeStringB64(dbdata.Data)
		if err != nil {
			logger.Errorf("Invalid data or B64 format: %v", err)
			return nil, errors.Wrap(err, "Invalid data or B64 format")
		}
		return &queryresult.KV{Key: dbdata.Id, Value: bytesData}, nil
	} else {
		return nil, errors.New("No more items")
	}
}

func (iIter *IndexIter) Close() error {
	iIter.Closed = true
	return nil
}

func NewIndexIter(startKey string, endKey string, docs Docs) (*IndexIter, error) {
	tr := IndexIter{}
	tr.FirstKey = startKey
	tr.LastKey = endKey
	tr.Closed = false
	tr.Docs = docs.Docs
	return &tr, nil
}

/*
{
  "selector": {
    "$and": [
      {
        "_id": {
           "$gte": "startKey"
        }
      },
      {
        "_id": {
          "$lt": "endKey"
        }
      }
    ]
  },
  "fields": ["_id", "data"],
  "sort": [{"_id": "asc"}],
  "limit": 10,
  "skip": 0
}

*/

func GetJsonForFind(startKey string, endKey string, limit int, lastKey string) ([]byte, error) {

	data := make(map[string]interface{})

	//data["skip"] = 0

	if limit > 0 {
		data["limit"] = limit
	}
	data["fields"] = []string{"_id", "data"}

	sort_item := make(map[string]string)
	sort_item["_id"] = "asc"
	sort := []interface{}{}
	sort = append(sort, sort_item)
	data["sort"] = sort

	sel1 := make(map[string]string)
	if len(lastKey) != 0 {
		sel1["$gt"] = lastKey
	} else {
		sel1["$gte"] = startKey
	}
	selA := make(map[string]interface{})
	selA["_id"] = sel1

	sel2 := make(map[string]string)
	sel2["$lt"] = endKey
	selB := make(map[string]interface{})
	selB["_id"] = sel2

	and_item := make(map[string]interface{})
	and_item["$and"] = []interface{}{selA, selB}
	data["selector"] = and_item

	dataBytes, err := json.Marshal(&data)
	return dataBytes, err
}
