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

import (
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("data_model")

// EncryptedData stores data in order to identify it as encrypted.
type EncryptedData struct {
	Encrypted []byte `json:"encrypted"`
}

// Load unmarshals an encryptedDataByte into an EncryptedData object
func (e *EncryptedData) Load(encryptedDataByte []byte) error {
	e1 := EncryptedData{}
	err := json.Unmarshal(encryptedDataByte, &e1)
	if err != nil {
		return err
	}
	e.Encrypted = e1.Encrypted
	return nil
}

// GetEncryptedDataBytes returns data wrapped in an EncryptedData struct.
// Use this function to set or return data that needs to be identified as encrypted.
func GetEncryptedDataBytes(dataBytes []byte) []byte {
	if dataBytes == nil {
		return nil
	}
	e := EncryptedData{Encrypted: dataBytes}
	encryptedDataBytes, _ := json.Marshal(&e)
	return encryptedDataBytes
}

// IsEncryptedData returns true if data is a json instance of the EncryptedData struct.
func IsEncryptedData(data []byte) bool {
	encryptedData := EncryptedData{}
	err := json.Unmarshal(data, &encryptedData)
	if err != nil || encryptedData.Encrypted == nil {
		logger.Debug(err)
		return false
	}
	return true
}
