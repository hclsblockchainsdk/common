package metering_connections

import (
	"common/bchcls/internal/common/global"
	"common/bchcls/utils"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("metering_connections")

// SetMeteringEnv reads from credentials.json and sets env vars for metering
// Called by init_common.Init function
func SetMeteringEnv() error {
	// If any field is missing, assume dev environment
	if utils.IsStringEmpty(url) || utils.IsStringEmpty(username) || utils.IsStringEmpty(password) {
		global.DevelopmentEnv = global.DevelopmentEnvString
		return nil
	}

	global.Cloudant_url = url
	global.Cloudant_username = username
	global.Cloudant_password = password
	global.DevelopmentEnv = global.ProductionEnvString

	return nil
}

// InstantiateCloudant creates a database for the channel in IBM Cloud's Cloudant instance
func InstantiateCloudant(channelID string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	isDBReady := isReady(channelID)
	if !isDBReady {
		// create database and check again if the database exist
		// each channel will have its own database
		writeURL := global.Cloudant_url + "/" + channelID
		logger.Debugf("database url: %v", writeURL)

		response, body, err := utils.PerformHTTP("PUT", writeURL, nil, headers, []byte(global.Cloudant_username), []byte(global.Cloudant_password))
		if err != nil || response == nil {
			logger.Errorf("Cloudant DB creation failed %v %v %v", err, response, body)
		} else if response.StatusCode >= 200 && response.StatusCode <= 299 {
			logger.Debugf("Cloudant DB is created: %v %v", response, body)
		} else {
			logger.Debugf("Cloudant DB creation response: %v", response)
		}

		isDBReady = isReady(channelID)
		if !isDBReady {
			return errors.New("Cloudant DB is not ready")
		}
	}

	return nil
}

// CreateMeteringIndex creates all the metering indices in the Cloudant Metering DB
func CreateMeteringIndex(channelID string) error {
	// number of transaction across a date range
	fields := []string{"txTimestamp"}
	name := "tx_date_range"
	err := createIndex(fields, name, channelID)
	if err != nil {
		return errors.Wrap(err, "Failed to create index")
	}

	// payloadsize
	fields = []string{"totalPayloadSize"}
	name = "total_payload_size"
	err = createIndex(fields, name, channelID)
	if err != nil {
		return errors.Wrap(err, "Failed to create index")
	}

	// transaction duration
	fields = []string{"timeElapsed"}
	name = "tx_duration"
	err = createIndex(fields, name, channelID)
	if err != nil {
		return errors.Wrap(err, "Failed to create index")
	}

	// transaction status
	fields = []string{"status"}
	name = "tx_status"
	err = createIndex(fields, name, channelID)
	if err != nil {
		return errors.Wrap(err, "Failed to create index")
	}

	return nil
}

// CreateIndex creates a single index in the Cloudant Metering DB
// Does not set optional design document in request
func createIndex(fields []string, indexName string, channelID string) error {
	fieldsObj := make(map[string]interface{})
	fieldsObj["fields"] = fields

	indexObj := make(map[string]interface{})
	indexObj["index"] = fieldsObj
	indexObj["name"] = indexName
	indexObj["type"] = "json"
	indexObj["partitioned"] = false
	indexBytes, _ := json.Marshal(indexObj)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	URL := global.Cloudant_url + "/" + channelID + "/_index"

	response, body, err := utils.PerformHTTP("POST", URL, indexBytes, headers, []byte(global.Cloudant_username), []byte(global.Cloudant_password))
	logger.Errorf("%v, %v, %v", response, body, err)
	if err != nil || response == nil {
		logger.Errorf("Cloudant DB create index operation failed: %v", err)
		return errors.New("Cloudant DB operation failed")
	} else if response.StatusCode >= 200 && response.StatusCode <= 299 {
		logger.Infof("Cloudant DB create index operation success")
		return nil
	} else {
		logger.Errorf("Cloudant DB create index operation failed %v %v", response, body)
		return errors.New("Cloudant DB create index operation failed")
	}
}

func isReady(database string) bool {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	//check if the database exist
	readURL := global.Cloudant_url + "/" + database
	logger.Debugf("read url: %v", readURL)
	response, body, err := utils.PerformHTTP("GET", readURL, nil, headers, []byte(global.Cloudant_username), []byte(global.Cloudant_password))
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
