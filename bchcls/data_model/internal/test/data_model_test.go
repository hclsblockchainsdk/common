/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package test

import (
	"common/bchcls/data_model"
	"common/bchcls/test_utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"

	"encoding/json"
	"testing"
)

func TestIsEncryptedData_False(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	// Test with unencrypted data
	unencryptedDataBytes := []byte(`{"Data":"my data"}`)
	isEnc := data_model.IsEncryptedData(unencryptedDataBytes)
	logger.Debugf("isEnc:%v", isEnc)
	test_utils.AssertFalse(t, isEnc, "Expected IsEncryptedData to return false")
}

func TestIsEncryptedData_True(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	// Test with encrypted data
	encryptedData := data_model.EncryptedData{Encrypted: []byte(`{"Data":"my data"}`)}
	encryptedDataBytes, _ := json.Marshal(encryptedData)
	isEnc := data_model.IsEncryptedData(encryptedDataBytes)
	logger.Debugf("isEnc:%v", isEnc)
	test_utils.AssertTrue(t, isEnc, "Expected IsEncryptedData to return true")
}
