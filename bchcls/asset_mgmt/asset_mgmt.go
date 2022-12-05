/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package asset_mgmt is responsible for storing any type of asset on the ledger.
// It handles encryption/decryption of asset private data as well as indexing.
package asset_mgmt

import (
	"common/bchcls/asset_mgmt/asset_manager"
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/internal/asset_mgmt_i"
	"common/bchcls/internal/asset_mgmt_i/asset_mgmt_c"
	"common/bchcls/internal/metering_i"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("asset_mgmt")

// ------------------------------------------------------
// ---------------------- INIT FUNCTIONS ----------------
// ------------------------------------------------------

// Init sets up the asset_mgmt package.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return asset_mgmt_i.Init(stub, logLevel...)
}

// ------------------------------------------------------
// ----------------- TOP-LEVEL FUNCTIONS ----------------
// ------------------------------------------------------

// GetAssetManager constructs and returns an assetManagerImpl instance.
func GetAssetManager(stub cached_stub.CachedStubInterface, caller data_model.User) asset_manager.AssetManager {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return asset_mgmt_i.GetAssetManager(stub, caller)
}

// GetAssetId returns the assetId given the asset's namespace and unique identifier.
func GetAssetId(assetNamespace string, id string) string {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	return asset_mgmt_c.GetAssetId(assetNamespace, id)
}

// GetAssetKeyId returns the asset key id for the given assetId.
// Returns an error if asset with assetId does not exist.
func GetAssetKeyId(stub cached_stub.CachedStubInterface, assetId string) (string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return asset_mgmt_i.GetAssetKeyId(stub, assetId)
}

// GetAssetKey returns decrypted asset key given a key path.
// Caller must supply the key path.
// If keyPath is valid, it returns decrypted asset key.
// If keyPath is invalid, it returns nil for asset key and the error.
func GetAssetKey(stub cached_stub.CachedStubInterface, assetId string, keyPath []string, startKey []byte) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return asset_mgmt_i.GetAssetKey(stub, assetId, keyPath, startKey)
}

// GetEncryptedAssetData returns an asset object with encrypted PrivateData.
// If the AssetId passed in does not exist, it returns an empty data_model.Asset object.
// It is the caller's responsibility to check if the return object is empty.
func GetEncryptedAssetData(stub cached_stub.CachedStubInterface, assetId string) (data_model.Asset, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return asset_mgmt_i.GetEncryptedAssetData(stub, assetId)
}

// GetAssetPrivateData returns an asset's private data bytes.
// Caller should provide proper assetKey for decryption.
// If decryption is successful, it returns decrypted private data bytes.
// If assetKey is nil, it returns encrypted private data bytes and nil for error.
// If assetKey is invalid, it returns encrypted private data bytes and an error.
func GetAssetPrivateData(stub cached_stub.CachedStubInterface, assetData data_model.Asset, assetKey []byte) ([]byte, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	return asset_mgmt_i.GetAssetPrivateData(stub, assetData, assetKey)
}
