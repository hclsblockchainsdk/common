/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package metering_i is responsible for metering transactions through the SDK.
package metering_i

import (
	"common/bchcls/cached_stub"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/common/metering_connections"
	"common/bchcls/utils"
	"encoding/json"
	"runtime"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/chaincode/shim/ext/cid"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("metering_i")

// secret is the secret string we append to txID
// When secret string is added to a txID, solution developers will not be able to
// call GetCache() on txID and read the row data we store in cache
const secret = "sdkMeteringSecretKey"

// SetEnvAndAddRow sets the environment and adds a row to metering DB
func SetEnvAndAddRow(stub cached_stub.CachedStubInterface) error {
	// if setting env errors out, no need to add row
	err := metering_connections.SetMeteringEnv()
	if err != nil {
		return errors.Wrap(err, "Failed to set env")
	}

	err = addRow(stub)
	if err != nil {
		return errors.Wrap(err, "Failed to add row")
	}

	return nil
}

// addRow adds transaction data to the internal Metering DB.
func addRow(stub cached_stub.CachedStubInterface) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	// No need for metering if we are in development env
	if global.DevelopmentEnv == global.DevelopmentEnvString {
		return nil
	}

	txID := stub.GetTxID()

	// Check if TX has been recorded
	cachedVal, err := stub.GetCache(getMeteringCacheKey(txID))
	if err == nil && cachedVal != nil {
		if rowData, ok := cachedVal.(map[string]interface{}); ok {
			if len(rowData) != 0 {
				return nil
			}
		}
	}

	// function name
	functionName := getFunctionName(3, 10)
	// ignore if called by InvokeSetup as its payload is not needed
	if strings.Contains(functionName, "init_common.InvokeSetup") {
		return nil
	}

	// Generate random string for row key
	rowKey, _ := generateRandomRowKey(8)
	channelID := stub.GetChannelID()
	timestampResult, _ := stub.GetTxTimestamp()
	txTimestamp := timestampResult.Seconds
	mspID, _ := cid.GetMSPID(stub)
	txEntryTime := time.Now().Unix()
	signedProposal, _ := stub.GetSignedProposal()
	inputPayloadSize := len(signedProposal.ProposalBytes)

	// row data
	rowData := make(map[string]interface{})
	rowData["_id"] = rowKey
	rowData["txID"] = txID
	rowData["channelID"] = channelID
	rowData["mspID"] = mspID
	rowData["functionName"] = functionName
	rowData["inputPayloadSize"] = inputPayloadSize
	rowData["txTimestamp"] = txTimestamp
	rowData["txEntryTime"] = txEntryTime
	rowData["txExitTime"] = "unknown"
	rowData["timeElapsed"] = "unknown"
	rowData["outputPayloadSize"] = "unknown"
	rowData["totalPayloadSize"] = "unknown"
	rowData["status"] = "unknown"
	rowData["error"] = "unknown"

	rev, err := writeToCloudant(rowData)
	if err != nil {
		return errors.Wrap(err, "Failed to write row data to Cloudant")
	}

	rowData["_rev"] = rev

	// Record TX in cache
	return stub.PutCache(getMeteringCacheKey(txID), rowData)
}

// UpdateRow updates an existing row in the internal Metering DB.
func UpdateRow(stub cached_stub.CachedStubInterface, arg interface{}, status string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	// No need for metering if we are in development env
	if global.DevelopmentEnv == global.DevelopmentEnvString {
		return nil
	}

	txID := stub.GetTxID()

	// Check if TX has been recorded
	cachedVal, err := stub.GetCache(getMeteringCacheKey(txID))
	if err != nil || cachedVal == nil {
		// If row data is not found in cache, exit
		return errors.New("row data not found in cache")
	}

	rowData, ok := cachedVal.(map[string]interface{})
	if !ok || len(rowData) == 0 {
		return errors.New("Failed to get row data from cache")
	}

	outputPayloadSize := 0
	if payload, ok := arg.([]byte); ok {
		if payload != nil {
			outputPayloadSize = len(payload)
		}
	}

	errStr := "N/A"
	if str, ok := arg.(string); ok {
		errStr = str
	}

	txExitTime := time.Now().Unix()
	txEntryTime := rowData["txEntryTime"].(int64)
	timeElapsed := txExitTime - txEntryTime
	inputPayloadSize := rowData["inputPayloadSize"].(int)
	totalPayloadSize := inputPayloadSize + outputPayloadSize
	rowData["totalPayloadSize"] = totalPayloadSize
	rowData["timeElapsed"] = timeElapsed
	rowData["outputPayloadSize"] = outputPayloadSize
	rowData["txExitTime"] = txExitTime
	rowData["error"] = errStr
	rowData["status"] = status

	return updateToCloudant(rowData)
}

// generateRandomRowKey returns a securely generated random string.
func generateRandomRowKey(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	bytes, err := utils.GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes), nil
}

// getMeteringCacheKey returns a metering cache key with secret appended to it
func getMeteringCacheKey(originalKey string) string {
	return originalKey + secret
}

// writeToCloudant adds a row in IBM Cloud's Cloudant instance
func writeToCloudant(rowData map[string]interface{}) (string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	channelID, ok := rowData["channelID"].(string)
	if !ok {
		return "", errors.New("Failed to get channelID from map")
	}

	// each channel will have its own database
	writeURL := global.Cloudant_url + "/" + channelID

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	rowDataBytes, _ := json.Marshal(rowData)
	response, body, err := utils.PerformHTTP("POST", writeURL, rowDataBytes, headers, []byte(global.Cloudant_username), []byte(global.Cloudant_password))
	logger.Errorf("%v, %v, %v", response, body, err)
	if err != nil || response == nil {
		logger.Errorf("Cloudant DB create document operation failed: %v", err)
		return "", errors.New("Cloudant DB operation failed")
	} else if response.StatusCode >= 200 && response.StatusCode <= 299 {
		logger.Infof("Cloudant DB create document operation success")

		responseObj := make(map[string]interface{})
		err = json.Unmarshal(body, &responseObj)
		if err != nil {
			return "", err
		}
		rev, ok := responseObj["rev"].(string)
		if !ok {
			return "", errors.New("Failed to get rev from add row response")
		}

		return rev, nil

	} else if response.StatusCode == 409 {
		logger.Infof("Skip:Data with same key written")
	} else {
		logger.Errorf("Cloudant DB create document operation failed %v %v", response, body)
		return "", errors.New("Cloudant DB create document operation failed")
	}
	return "", nil
}

// updateToCloudant updates a row in IBM Cloud's Cloudant instance
func updateToCloudant(rowData map[string]interface{}) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	// each channel will have its own database
	channelID, ok := rowData["channelID"].(string)
	if !ok {
		return errors.New("Failed to get channelID from map")
	}

	rowKey, ok := rowData["_id"].(string)
	if !ok {
		return errors.New("Failed to get rowKey from map")
	}

	URL := global.Cloudant_url + "/" + channelID + "/" + rowKey

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	headers["Accept-Charset"] = "utf-8"

	rowDataBytes, _ := json.Marshal(rowData)

	response, body, err := utils.PerformHTTP("PUT", URL, rowDataBytes, headers, []byte(global.Cloudant_username), []byte(global.Cloudant_password))
	logger.Errorf("%v, %v, %v", response, body, err)
	if err != nil || response == nil {
		logger.Errorf("Cloudant DB update document operation failed: %v", err)
		return errors.New("Cloudant DB update document operation failed")
	} else if response.StatusCode >= 200 && response.StatusCode <= 299 {
		logger.Infof("Cloudant DB update document operation success: %v %v", response, body)
	} else if response.StatusCode == 409 {
		logger.Infof("Skip:Data with same key written %v %V", rowKey, response)
	} else {
		logger.Errorf("Cloudant DB update document operation failed %v %v", response, body)
		return errors.New("Cloudant DB update document operation failed")
	}
	return nil
}

// getFunctionName returns a sequence of function calls.
// skip is how many levels to skip
// maxDepth is the max level depth
func getFunctionName(skip int, maxDepth int) string {
	functionNameStr := ""
	fpcs := make([]uintptr, 1)

	// defaults
	if skip < 0 {
		// we are starting at 3 because we are skipping this function, addRow, SetEnvAndAddRow
		// we already know these 3 functions are being called everytime
		skip = 3
	}
	if maxDepth < 1 {
		maxDepth = 10
	}

	var n int
	for i := skip; i <= maxDepth; i++ {
		// Skip i levels
		// the following block gives you the function call chain
		n = runtime.Callers(i, fpcs)
		if n == 0 {
			return functionNameStr
		}
		// caller is name of the function one level above
		caller := runtime.FuncForPC(fpcs[0] - 1)
		if caller == nil {
			return functionNameStr
		}
		functionNameStr = shortenFunctionName(caller.Name()) + " -> " + functionNameStr
	}

	return functionNameStr
}

// shortenFunctionName returns a shortened function name
func shortenFunctionName(name string) string {
	return strings.TrimPrefix(name, "github.com/chaincodes/solution_chaincode/vendor/common/bchcls/")
}
