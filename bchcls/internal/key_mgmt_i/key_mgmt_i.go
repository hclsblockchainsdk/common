/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package key_mgmt_i

import (
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/internal/key_mgmt_i/key_mgmt_c"
	"common/bchcls/internal/key_mgmt_i/key_mgmt_c/key_mgmt_g"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("key_mgmt_i")

// Init sets up the key_mgmt package.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	return key_mgmt_c.Init(stub, logLevel...)
}

// (Deprecated, use AddAccess() function instead)
// AddAccessWithKeys gives startKey access to targetKey.
// It does this by encrypting targetKey with encKey, which can then be decrypted using startKey.
// If startKey is a sym key, encKey must be identical. Otherwise startKey must be a private key and encKey the matching public key.
// Stores an edge in the graph from startKeyId -> targetKeyId, and a reverse edge from targetKeyId -> startKeyId.
func AddAccessWithKeys(stub cached_stub.CachedStubInterface, startKey []byte, startKeyId string, targetKey []byte, targetKeyId string, encKey []byte, edgeData ...map[string]string) error {
	if len(edgeData) > 0 {
		return key_mgmt_c.AddAccessWithKeys(stub, startKey, startKeyId, targetKey, targetKeyId, encKey, edgeData[0])
	} else {
		return key_mgmt_c.AddAccessWithKeys(stub, startKey, startKeyId, targetKey, targetKeyId, encKey)
	}
}

// AddAccess gives startKey access to targetKey.
func AddAccess(stub cached_stub.CachedStubInterface, startKey data_model.Key, targetKey data_model.Key, edgeData ...map[string]string) error {
	if len(edgeData) > 0 {
		return key_mgmt_c.AddAccess(stub, startKey, targetKey, edgeData[0])
	} else {
		return key_mgmt_c.AddAccess(stub, startKey, targetKey)
	}
}

// RevokeAccess revokes access from startKey to targetKey.
// It does this by deleting the edge from startKey -> targetKey (and the reverse edge from targetKeyId -> startKeyId).
func RevokeAccess(stub cached_stub.CachedStubInterface, startKeyId string, targetKeyId string) error {
	return key_mgmt_c.RevokeAccess(stub, startKeyId, targetKeyId)
}

// GetAccessEdge gets the Access key graph edge.
// Returns edgeValueByte, edgeDataMap, error.
func GetAccessEdge(stub cached_stub.CachedStubInterface, startKeyId string, targetKeyId string) ([]byte, map[string]string, error) {
	return key_mgmt_c.GetAccessEdge(stub, startKeyId, targetKeyId)
}

// UpdateAccessEdge updates Access key graph edge without checking error.
func UpdateAccessEdge(stub cached_stub.CachedStubInterface, startKeyId string, targetKeyId string, edge ...interface{}) error {
	if len(edge) == 0 {
		return key_mgmt_c.UpdateAccessEdge(stub, startKeyId, targetKeyId)
	} else if len(edge) == 1 {
		return key_mgmt_c.UpdateAccessEdge(stub, startKeyId, targetKeyId, edge[0])
	} else {
		return key_mgmt_c.UpdateAccessEdge(stub, startKeyId, targetKeyId, edge[0], edge[1])
	}
}

// SlowVerifyAccess checks for a path in the graph from startKeyId to targetKeyId.
// Uses recursive DFS.
// Returns the list of keyIds in the path.
// If no path is found, returns nil.
func SlowVerifyAccess(stub cached_stub.CachedStubInterface, startKeyId string, targetKeyId string, filter ...interface{}) ([]string, error) {
	var filterRule interface{} = nil
	if len(filter) > 0 {
		filterRule = filter[0]
	}
	return key_mgmt_c.SlowVerifyAccess(stub, startKeyId, targetKeyId, filterRule)
}

// VerifyAccessPath checks if all edges in the path exist.
func VerifyAccessPath(stub cached_stub.CachedStubInterface, path []string) (bool, error) {
	return key_mgmt_c.VerifyAccessPath(stub, path)
}

// GetKey follows the path of keys in keyIdList, decrypting each key along the way.
// When it reaches the end of the list, it returns that final key.
func GetKey(stub cached_stub.CachedStubInterface, keyIdList []string, startKey []byte) ([]byte, error) {
	return key_mgmt_c.GetKey(stub, keyIdList, startKey)
}

// SlowVerifyAccessAndGetKey calls FindPath and passes the result to GetKey.
// This is a convenience function for callers who want a key but don't want to make 2 calls.
// If no path is found to the targetKey, (nil, nil) is returned.
func SlowVerifyAccessAndGetKey(stub cached_stub.CachedStubInterface, startKeyId string, startKey []byte, targetKeyId string, filter ...interface{}) ([]byte, error) {
	var filterRule interface{} = nil
	if len(filter) > 0 {
		filterRule = filter[0]
	}
	return key_mgmt_c.SlowVerifyAccessAndGetKey(stub, startKeyId, startKey, targetKeyId, filterRule)
}

func VerifyAccessPathAndGetKey(stub cached_stub.CachedStubInterface, startKeyId string, startKey []byte, path []string) ([]byte, error) {
	return key_mgmt_c.VerifyAccessPathAndGetKey(stub, startKeyId, startKey, path)
}

// SlowGetMyKeys returns a list of keyIds that can be accessed starting from startKeyId (directly or indirectly).
func SlowGetMyKeys(stub cached_stub.CachedStubInterface, startKeyId string, filter ...interface{}) ([]string, error) {
	var filterRule interface{} = nil
	if len(filter) > 0 {
		filterRule = filter[0]
	}
	return key_mgmt_c.SlowGetMyKeys(stub, startKeyId, filterRule)
}

// GetUserKeys returns a list of keyIds that can be accessed by the User (directly or indirectly).
func GetUserKeys(stub cached_stub.CachedStubInterface, user data_model.User, filter ...interface{}) ([]string, error) {
	var filterRule interface{} = nil
	if len(filter) > 0 {
		filterRule = filter[0]
	}
	return key_mgmt_c.GetUserKeys(stub, user, filterRule)
}

// GetOwnerKeys returns a list of keyIds which can be used to access targetKeyId (directly or indirectly).
func GetOwnerKeys(stub cached_stub.CachedStubInterface, targetKeyId string, filter ...interface{}) ([]string, error) {
	//logger.Infof("--- get owner keys %v %v", targetKeyId, filter)
	var filterRule interface{} = nil
	if len(filter) > 0 {
		filterRule = filter[0]
	}
	return key_mgmt_c.GetOwnerKeys(stub, targetKeyId, filterRule)
}

// ValidateKey checks if key matches existing key in the graph.
// If key does not exist, returns !mustExist.
// If key does exist, returns true if valid, false otherwise.
func ValidateKey(stub cached_stub.CachedStubInterface, key data_model.Key, mustExist bool) (bool, error) {
	return key_mgmt_c.ValidateKey(stub, key, mustExist)
}

// KeyExists checks if the key already exists in the graph, It does not check the validity of the key, only its existence in the graph.
func KeyExists(stub cached_stub.CachedStubInterface, keyId string) bool {
	return key_mgmt_c.KeyExists(stub, keyId)
}

// ConvertKeyBytesToKey converts a keyId + keyBytes to a Key object
func ConvertKeyBytesToKey(keyId string, keyBytes []byte) (*data_model.Key, error) {
	return key_mgmt_c.ConvertKeyBytesToKey(keyId, keyBytes)
}

func GetStateByPartialCompositeKey(stub cached_stub.CachedStubInterface, keys []string) (shim.StateQueryIteratorInterface, error) {
	return key_mgmt_c.GetStateByPartialCompositeKey(stub, keys)
}

func GetKeyIdForWriteOnlyAccess(assetId string, assetKeyId string, ownerId string) string {
	return key_mgmt_c.GetKeyIdForWriteOnlyAccess(assetId, assetKeyId, ownerId)
}

// GetPubPrivKeyId returns the ID that should be assigned to a public or private key.
func GetPubPrivKeyId(id string) string {
	return key_mgmt_g.GetPubPrivKeyId(id)
}

// GetSymKeyId returns the ID that should be assigned to a sym key.
func GetSymKeyId(id string) string {
	return key_mgmt_g.GetSymKeyId(id)
}

// GetLogSymKeyId returns the ID that should be assigned to a log sym key.
func GetLogSymKeyId(id string) string {
	return key_mgmt_g.GetLogSymKeyId(id)
}

// GetPrivateKeyHashSymKeyId returns the ID that should be assigned to a sym key derived from the hash of a private key.
func GetPrivateKeyHashSymKeyId(id string) string {
	return key_mgmt_g.GetPrivateKeyHashSymKeyId(id)
}
