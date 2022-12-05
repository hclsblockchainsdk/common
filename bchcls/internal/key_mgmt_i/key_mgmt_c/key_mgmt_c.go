/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package key_mgmt_c

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/custom_errors"
	"common/bchcls/data_model"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/common/graph"
	"common/bchcls/internal/key_mgmt_i/key_mgmt_c/key_mgmt_g"
	"common/bchcls/utils"

	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("key_mgmt_c")

// Init sets up the key_mgmt package.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	return nil, nil
}

// keyGraphEdge is a value stored for each edge of the KeyGraph and ReverseKeyGraph.
type keyGraphEdge struct {
	StartKeyId         string `json:"start_key_id"`
	TargetKeyId        string `json:"target_key_id"`
	EncryptedTargetKey []byte `json:"encrypted_target_key"`
}

// keyGraphNode represents a key stored in the KeyGraph or ReverseKeyGraph.
// KeyId should be the symKeyID or pubPrivKeyId.
// IsSymKey tells us whether this is a sym key node or an RSA key node.
// SymKeyHash is the hash of the sym key (empty for RSA key nodes).
// PublicKey is the public key for this RSA key pair (empty for sym keys).
type keyGraphNode struct {
	KeyId      string `json:"key_id"`
	IsSymKey   bool   `json:"is_sym_key"`
	SymKeyHash []byte `json:"sym_key_hash"`
	PublicKey  []byte `json:"public_key"`
}

// Equal returns whether the two keys are equal.
func (knode *keyGraphNode) Equal(node *keyGraphNode) bool {
	if knode.KeyId != node.KeyId {
		return false
	}
	if knode.IsSymKey != node.IsSymKey {
		return false
	}
	if !bytes.Equal(knode.SymKeyHash, node.SymKeyHash) {
		return false
	}
	if !bytes.Equal(knode.PublicKey, node.PublicKey) {
		return false
	}
	return true
}

// (Deprecated, use AddAccess() function instead)
// AddAccessWithKeys gives startKey access to targetKey.
// It does this by encrypting targetKey with encKey, which can then be decrypted using startKey.
// If startKey is a sym key, encKey must be identical. Otherwise startKey must be a private key and encKey the matching public key.
// Stores an edge in the graph from startKeyId -> targetKeyId, and a reverse edge from targetKeyId -> startKeyId.
func AddAccessWithKeys(stub cached_stub.CachedStubInterface, startKey []byte, startKeyId string, targetKey []byte, targetKeyId string, encKey []byte, edgeData ...map[string]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	//if encKey is nil, assume it's a sym key and use startKey
	if encKey == nil {
		encKey = startKey
	}

	// Create a Key object from startKey
	startKeyObject, err := ConvertKeyBytesToKey(startKeyId, encKey)
	if err != nil {
		logger.Errorf("Failed to ConvertKeyBytesToKey for keyId \"%v\"", startKeyId)
		return errors.Wrapf(err, "Failed to ConvertKeyBytesToKey for keyId \"%v\"", startKeyId)
	}

	// Create a Key object from targetKey
	targetKeyObject, err := ConvertKeyBytesToKey(targetKeyId, targetKey)
	if err != nil {
		logger.Errorf("Failed to ConvertKeyBytesToKey for keyId \"%v\"", targetKeyId)
		return errors.Wrapf(err, "Failed to ConvertKeyBytesToKey for keyId \"%v\"", targetKeyId)
	}

	var edgeDataMap map[string]string = make(map[string]string)
	if len(edgeData) > 0 {
		edgeDataMap = edgeData[0]
	}

	return AddAccess(stub, *startKeyObject, *targetKeyObject, edgeDataMap)
}

// AddAccess gives startKey access to targetKey.
func AddAccess(stub cached_stub.CachedStubInterface, startKey data_model.Key, targetKey data_model.Key, edgeData ...map[string]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("startKeyId: %v, targetKeyId: %v, edgeData:%v", startKey.ID, targetKey.ID, edgeData)

	var edgeValueBytes []byte
	var edgeDataMap map[string]string = make(map[string]string)
	if len(edgeData) > 0 {
		edgeDataMap = edgeData[0]
	}

	// Public key is not allowed as target
	if targetKey.Type == global.KEY_TYPE_PUBLIC {
		logger.Errorf("Public key cannot be added as a targetKey in the key graph")
		return errors.New("Public key cannot be added as a targetKey in the key graph")
	}

	needToCheckExistingEdge := true
	if KeyExists(stub, startKey.ID) {
		// Checks for an existing startKey node and makes sure they match
		isStartKeyValid, err := ValidateKey(stub, startKey, false)
		if !isStartKeyValid || err != nil {
			invalidKeyError := &custom_errors.InvalidKeyError{KeyId: startKey.ID}
			logger.Errorf("%v: %v", invalidKeyError, err)
			return errors.WithStack(invalidKeyError)
		}
	} else {
		// it's a new edge
		needToCheckExistingEdge = false
		// Create a node in the graph for startKey (if one doesn't already exist)
		startKeyNode, err := convertKeyToKeyGraphNode(startKey)
		if err != nil {
			logger.Errorf("Failed to convertKeyToKeyGraphNode for startKey.ID \"%v\"", startKey.ID)
			return errors.Wrapf(err, "Failed to convertKeyToKeyGraphNode for startKey.ID \"%v\"", startKey.ID)
		}
		err = putKeyGraphNode(stub, *startKeyNode)
		if err != nil {
			logger.Errorf("Failed to putKeyGraphNode for startKey.ID \"%v\"", startKey.ID)
			return errors.Wrapf(err, "Failed to putKeyGraphNode for startKey.ID \"%v\"", startKey.ID)
		}
	}

	if KeyExists(stub, targetKey.ID) {
		// Checks for an existing targetKey node and makes sure they match
		isTargetKeyValid, err := ValidateKey(stub, targetKey, false)
		if !isTargetKeyValid || err != nil {
			invalidKeyError := &custom_errors.InvalidKeyError{KeyId: targetKey.ID}
			logger.Errorf("%v: %v", invalidKeyError, err)
			return errors.WithStack(invalidKeyError)
		}
	} else {
		// it's a new edge
		needToCheckExistingEdge = false
		// Create a node in the graph for targetKey (if one doesn't already exist)
		targetKeyNode, err := convertKeyToKeyGraphNode(targetKey)
		if err != nil {
			logger.Errorf("Failed to convertKeyToKeyGraphNode for targetKey.ID \"%v\"", targetKey.ID)
			return errors.Wrapf(err, "Failed to convertKeyToKeyGraphNode for targetKey.ID \"%v\"", targetKey.ID)
		}
		err = putKeyGraphNode(stub, *targetKeyNode)
		if err != nil {
			logger.Errorf("Failed to putKeyGraphNode for targetKey.ID \"%v\"", targetKey.ID)
			return errors.Wrapf(err, "Failed to putKeyGraphNode for targetKey.ID \"%v\"", targetKey.ID)
		}
	}

	// Check existing edge
	// You don't need to check existing edge if any of startKey or targetkey
	// is a new node.
	needToEncrypt := true
	if needToCheckExistingEdge {
		edgeValueBytes2, edgeDataMap2, err := graph.GetEdge(stub, global.KEY_GRAPH_PREFIX, startKey.ID, targetKey.ID)
		if err == nil && len(edgeValueBytes2) > 0 {
			if reflect.DeepEqual(edgeDataMap2, edgeDataMap) {
				logger.Infof("Existing edge from \"%v\" to \"%v\"", startKey.ID, targetKey.ID)
				return nil
			}
			needToEncrypt = false
			edgeValueBytes = edgeValueBytes2
		}
	}

	// Encrypt targetKey with startKey
	if needToEncrypt {
		encryptedTargetKey := []byte{}
		var err error
		switch startKey.Type {
		case global.KEY_TYPE_SYM:
			// Encrypt targetKey with sym key
			encryptedTargetKey, err = crypto.EncryptWithSymKey(startKey.KeyBytes, targetKey.KeyBytes)
			if err != nil {
				custom_err := &custom_errors.EncryptionError{ToEncrypt: targetKey.ID, EncryptionKey: startKey.ID}
				logger.Errorf("%v: %v", custom_err, err)
				return errors.Wrap(err, custom_err.Error())
			}

		case global.KEY_TYPE_PRIVATE:
			// Parse startKey into an RSA PrivateKey
			privateKey, err := crypto.ParsePrivateKey(startKey.KeyBytes)
			if err != nil {
				custom_err := &custom_errors.ParseKeyError{Type: "private key"}
				logger.Errorf("%v: %v", custom_err, err)
				return errors.Wrap(err, custom_err.Error())
			}
			if privateKey == nil {
				custom_err := &custom_errors.ParseKeyError{Type: "private key"}
				logger.Errorf("%v", custom_err)
				return errors.WithStack(custom_err)
			}
			// Extract publicKey from privateKey
			publicKey := privateKey.Public().(*rsa.PublicKey)
			// Encrypt targetKey with publicKey
			encryptedTargetKey, err = crypto.EncryptWithPublicKey(publicKey, targetKey.KeyBytes)
			if err != nil {
				custom_err := &custom_errors.EncryptionError{ToEncrypt: targetKey.ID, EncryptionKey: startKey.ID}
				logger.Errorf("%v: %v", custom_err, err)
				return errors.Wrap(err, custom_err.Error())
			}

		case global.KEY_TYPE_PUBLIC:
			// Parse startKey into an RSA PublicKey
			publicKey, err := crypto.ParsePublicKey(startKey.KeyBytes)
			if err != nil {
				custom_err := &custom_errors.ParseKeyError{Type: "public key"}
				logger.Errorf("%v: %v", custom_err, err)
				return errors.Wrap(err, custom_err.Error())
			}
			if publicKey == nil {
				custom_err := &custom_errors.ParseKeyError{Type: "public key"}
				logger.Errorf("%v", custom_err)
				return errors.WithStack(custom_err)
			}
			// Encrypt targetKey with publicKey
			encryptedTargetKey, err = crypto.EncryptWithPublicKey(publicKey, targetKey.KeyBytes)
			if err != nil {
				custom_err := &custom_errors.EncryptionError{ToEncrypt: targetKey.ID, EncryptionKey: startKey.ID}
				logger.Errorf("%v: %v", custom_err, err)
				return errors.Wrap(err, custom_err.Error())
			}

		default:
			logger.Errorf("Unsupported key type \"%v\"", startKey.Type)
			return errors.Errorf("Unsupported key type \"%v\"", startKey.Type)
		}

		// Make sure that the encryption worked
		if len(encryptedTargetKey) == 0 {
			custom_err := &custom_errors.EncryptionError{ToEncrypt: targetKey.ID, EncryptionKey: startKey.ID}
			logger.Errorf("%v", custom_err)
			return errors.WithStack(custom_err)
		}

		// Create the new edge in the graph
		edgeValue := keyGraphEdge{
			StartKeyId:         startKey.ID,
			TargetKeyId:        targetKey.ID,
			EncryptedTargetKey: encryptedTargetKey}

		edgeValueBytes, err = json.Marshal(edgeValue)
		if err != nil {
			custom_err := &custom_errors.MarshalError{Type: "edge data"}
			logger.Errorf("%v: %v", custom_err, err)
			return errors.Wrap(err, custom_err.Error())
		}
	}

	// if access type is not specified default it to read access
	if val, ok := edgeDataMap[global.EDGEDATA_ACCESS_TYPE]; ok {
		// read only cannot be used for add access; change it to read access
		if val == global.ACCESS_READ_ONLY {
			edgeDataMap[global.EDGEDATA_ACCESS_TYPE] = global.ACCESS_READ
		}
	} else {
		edgeDataMap[global.EDGEDATA_ACCESS_TYPE] = global.ACCESS_READ
	}

	logger.Infof("Adding access edge from \"%v\" to \"%v\" %v", startKey.ID, targetKey.ID, edgeDataMap)
	// Store edge in KeyGraph
	return graph.PutEdge(stub, global.KEY_GRAPH_PREFIX, startKey.ID, targetKey.ID, edgeValueBytes, edgeDataMap)
}

// RevokeAccess revokes access from startKey to targetKey.
// It does this by deleting the edge from startKey -> targetKey (and the reverse edge from targetKeyId -> startKeyId).
func RevokeAccess(stub cached_stub.CachedStubInterface, startKeyId string, targetKeyId string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Infof("Revoking access edge from \"%v\" to \"%v\"", startKeyId, targetKeyId)
	return graph.DeleteEdge(stub, global.KEY_GRAPH_PREFIX, startKeyId, targetKeyId)
}

// GetAccessEdge gets the Access key graph edge.
// Returns edgeValueByte, edgeDataMap, error.
func GetAccessEdge(stub cached_stub.CachedStubInterface, startKeyId string, targetKeyId string) ([]byte, map[string]string, error) {
	return graph.GetEdge(stub, global.KEY_GRAPH_PREFIX, startKeyId, targetKeyId)
}

// UpdateAccessEdge updates Access key graph edge without checking error.
func UpdateAccessEdge(stub cached_stub.CachedStubInterface, startKeyId string, targetKeyId string, edge ...interface{}) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	if len(edge) == 0 {
		custom_err := errors.WithStack(&custom_errors.LengthCheckingError{Type: "edge"})
		logger.Errorf("%v", custom_err)
		return custom_err
	} else if len(edge) == 1 {
		return graph.PutEdge(stub, global.KEY_GRAPH_PREFIX, startKeyId, targetKeyId, edge[0])
	} else {
		return graph.PutEdge(stub, global.KEY_GRAPH_PREFIX, startKeyId, targetKeyId, edge[0], edge[1])
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

	return graph.SlowFindPath(stub, global.KEY_GRAPH_PREFIX, startKeyId, targetKeyId, filterRule)
}

func getPathCacheKey(path []string) string {
	return global.KEY_GRAPH_PREFIX + "_path_" + strings.Join(path, ",")
}

// VerifyAccessPath checks if all edges in the path exist.
func VerifyAccessPath(stub cached_stub.CachedStubInterface, path []string) (bool, error) {
	logger.Debugf("VerifyAccessPath %v", path)
	// ckeck cache
	cachekey := getPathCacheKey(path)
	verifiedCache, err := stub.GetCache(cachekey)
	if err == nil {
		verified, ok := verifiedCache.(bool)
		if ok {
			logger.Debugf("Getting from cache %v", cachekey)
			return verified, nil
		}
	}

	// verify path from graph
	verified, err := graph.HasPath(stub, global.KEY_GRAPH_PREFIX, path)
	if err == nil {
		//save to cache
		//logger.Debugf("Save to cache %v", cachekey)
		stub.PutCache(cachekey, verified)
	}
	return verified, err
}

func getCacheKey(keyId string) string {
	return global.KEY_GRAPH_PREFIX + "-" + keyId
}

// getKeyFromCache returns a copy of the cache value to avoid side effect.
func getKeyFromCache(stub cached_stub.CachedStubInterface, keyId string) ([]byte, error) {
	cachekey := getCacheKey(keyId)
	keyCache, err := stub.GetCache(cachekey)
	if err != nil {
		return nil, err
	}
	if targetKey, ok := keyCache.([]byte); ok {
		targetKeyCopy := make([]byte, len(targetKey))
		copy(targetKeyCopy, targetKey)
		logger.Debugf("Get key from cache: %v", keyId)
		return targetKeyCopy, nil
	} else {
		return nil, errors.New("Invalid cache value")
	}
}

// putKeyToCache saves a copy of keyByte to cache to avoid an unintended change of value.
func putKeyToCache(stub cached_stub.CachedStubInterface, keyId string, keyByte []byte) error {
	cachekey := getCacheKey(keyId)
	keyCopy := make([]byte, len(keyByte))
	copy(keyCopy, keyByte)
	//logger.Debugf("Save key to cache: %v", keyId)
	return stub.PutCache(cachekey, keyCopy)
}

// GetKey follows the path of keys in keyIdList, decrypting each key along the way.
// When it reaches the end of the list, it returns that final key.
func GetKey(stub cached_stub.CachedStubInterface, keyIdList []string, startKey []byte) ([]byte, error) {
	// don't need decrypt anything
	if len(keyIdList) == 1 {
		return startKey, nil
	}

	// invalid input
	if len(keyIdList) < 2 {
		logger.Errorf("Invalid Input: kyeIdList need to have at least two elements")
		return nil, errors.New("Invalid Input: kyeIdList need to have at least two elements")
	}

	// verify path(keyIdList) first
	verified, err := VerifyAccessPath(stub, keyIdList)
	if err != nil || verified == false {
		logger.Errorf("Access through keyIdList denied: %v", keyIdList)
		return nil, errors.New("Access through keyIdList denied")
	}

	// verify first key
	firstNode, err := getKeyGraphNode(stub, keyIdList[0])
	if err != nil {
		return nil, err
	}
	firstNode2Key, err := ConvertKeyBytesToKey(keyIdList[0], startKey)
	if err != nil {
		return nil, err
	}
	firstNode2, err := convertKeyToKeyGraphNode(*firstNode2Key)
	if err != nil {
		return nil, err
	}
	if !firstNode.Equal(firstNode2) {
		logger.Errorf("Invalid startKey")
		return nil, errors.New("Invalid startKey")
	}

	//get target key
	targetKeyId := keyIdList[len(keyIdList)-1]

	//check cache
	decryptionKey, err := getKeyFromCache(stub, targetKeyId)
	if err == nil {
		logger.Debug("Return decrypted keys from cache")
		return decryptionKey, nil
	}

	// traverse the keyIdList
	// for each key, get edge
	// once edge is retrieved, get edge's target key for decryption the next edge's start key
	decryptionKey = startKey
	for i := 0; i < len(keyIdList)-1; i++ {
		//if startkey and targetkey is same skip it
		if keyIdList[i] == keyIdList[i+1] {
			continue
		}

		//edgeKey, _ := stub.CreateCompositeKey(KEY_GRAPH_PREFIX, []string{keyIdList[i], keyIdList[i+1]})
		//keyGraphEdgeBytes, err := stub.GetState(edgeKey)
		keyGraphEdgeBytes, _, err := graph.GetEdge(stub, global.KEY_GRAPH_PREFIX, keyIdList[i], keyIdList[i+1])

		if err != nil {
			custom_err := &custom_errors.GetEdgeError{ParentNode: keyIdList[i], ChildNode: keyIdList[i+1]}
			logger.Errorf("%v: %v", custom_err, err)
			return nil, errors.Wrap(err, custom_err.Error())
		}
		if keyGraphEdgeBytes == nil {
			custom_err := &custom_errors.GetEdgeError{ParentNode: keyIdList[i], ChildNode: keyIdList[i+1]}
			logger.Errorf("%v: %v", custom_err, err)
			return nil, errors.Wrap(err, custom_err.Error())
		}

		edge := keyGraphEdge{}
		err = json.Unmarshal(keyGraphEdgeBytes, &edge)
		if err != nil {
			custom_err := &custom_errors.UnmarshalError{Type: "keyGraphEdgeBytes"}
			logger.Errorf("%v: %v", custom_err, err)
			return nil, errors.Wrap(err, custom_err.Error())
		}

		decryptedTargetKey := []byte{}

		//check cache
		decryptedTargetKey, err = getKeyFromCache(stub, edge.TargetKeyId)
		if err != nil {
			// sym key decryption
			if crypto.ValidateSymKey(decryptionKey) {
				decryptedTargetKey, err = crypto.DecryptWithSymKey(decryptionKey, edge.EncryptedTargetKey)
				if err != nil {
					custom_err := &custom_errors.DecryptionError{ToDecrypt: "targetKey", DecryptionKey: "sym key"}
					logger.Errorf("%v: %v", custom_err, err)
					return nil, errors.Wrap(err, custom_err.Error())
				}
			} else {
				// pub key decryption
				priv, err := crypto.ParsePrivateKey(decryptionKey)
				if err != nil {
					custom_err := &custom_errors.ParseKeyError{Type: "private key"}
					logger.Errorf("%v: %v", custom_err, err)
					return nil, errors.Wrap(err, custom_err.Error())
				}
				if priv == nil {
					custom_err := &custom_errors.ParseKeyError{Type: "private key"}
					logger.Errorf("%v", custom_err)
					return nil, custom_err
				}

				decryptedTargetKey, err = crypto.DecryptWithPrivateKey(priv, edge.EncryptedTargetKey)
				if err != nil {
					custom_err := &custom_errors.DecryptionError{ToDecrypt: "targetKey", DecryptionKey: "private key"}
					logger.Errorf("%v: %v", custom_err, err)
					return nil, errors.Wrap(err, custom_err.Error())
				}
			}

			// save cache
			if decryptedTargetKey != nil {
				putKeyToCache(stub, edge.TargetKeyId, decryptedTargetKey)
			}
		}

		if decryptedTargetKey == nil {
			custom_err := &custom_errors.DecryptionError{ToDecrypt: "targetKey", DecryptionKey: edge.TargetKeyId}
			logger.Errorf("empty key: %v", custom_err)
			return nil, custom_err
		}

		// set decryptionKey to this edge's targetKey
		decryptionKey = decryptedTargetKey
	}

	// save cache
	putKeyToCache(stub, targetKeyId, decryptionKey)
	return decryptionKey, nil
}

// SlowVerifyAccessAndGetKey calls FindPath and passes the result to GetKey.
// This is a convenience function for callers who want a key but don't want to make 2 calls.
// If no path is found to the targetKey, (nil, nil) is returned.
func SlowVerifyAccessAndGetKey(stub cached_stub.CachedStubInterface, startKeyId string, startKey []byte, targetKeyId string, filter ...interface{}) ([]byte, error) {
	var filterRule interface{} = nil
	if len(filter) > 0 {
		filterRule = filter[0]
	}

	// Create a Key object for startKey
	startKeyObject, err := ConvertKeyBytesToKey(startKeyId, startKey)
	if err != nil {
		logger.Errorf("Failed to ConvertKeyBytesToKey for keyId \"%v\"", startKeyId)
		return nil, errors.Wrapf(err, "Failed to ConvertKeyBytesToKey for keyId \"%v\"", startKeyId)
	}

	// Check that startKeyId & startKey match
	isValid, err := ValidateKey(stub, *startKeyObject, true)
	if !isValid || err != nil {
		logger.Errorf("startKeyId \"%v\" & startKey do not match!", startKeyId)
		return nil, errors.Errorf("startKeyId \"%v\" & startKey do not match!", startKeyId)
	}

	// First call FindPath to get the path to the targetKey
	path, err := SlowVerifyAccess(stub, startKeyId, targetKeyId, filterRule)
	if err != nil {
		custom_err := &custom_errors.VerifyAccessError{StartKeyId: startKeyId, TargetKeyId: targetKeyId}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	} else if len(path) == 0 {
		logger.Debugf("No path found from key \"%v\" to key \"%v\"", startKeyId, targetKeyId)
		return nil, nil
	}
	// Now call GetKey
	return GetKey(stub, path, startKey)
}

func VerifyAccessPathAndGetKey(stub cached_stub.CachedStubInterface, startKeyId string, startKey []byte, path []string) ([]byte, error) {

	// Create a Key object for startKey
	startKeyObject, err := ConvertKeyBytesToKey(startKeyId, startKey)
	if err != nil {
		logger.Errorf("Failed to ConvertKeyBytesToKey for keyId \"%v\"", startKeyId)
		return nil, errors.Wrapf(err, "Failed to ConvertKeyBytesToKey for keyId \"%v\"", startKeyId)
	}

	// Check that startKeyId & startKey match
	isValid, err := ValidateKey(stub, *startKeyObject, true)
	if !isValid || err != nil {
		logger.Errorf("startKeyId \"%v\" & startKey do not match!", startKeyId)
		return nil, errors.Errorf("startKeyId \"%v\" & startKey do not match!", startKeyId)
	}

	verified, err := VerifyAccessPath(stub, path)
	if err != nil {
		return nil, err
	}
	if verified == false {
		return nil, errors.New("Invalid Path")
	}

	// Now call GetKey
	return GetKey(stub, path, startKey)
}

// SlowGetMyKeys returns a list of keyIds that can be accessed starting from startKeyId (directly or indirectly).
func SlowGetMyKeys(stub cached_stub.CachedStubInterface, startKeyId string, filter ...interface{}) ([]string, error) {
	var filterRule interface{} = nil
	if len(filter) > 0 {
		filterRule = filter[0]
	}
	return graph.SlowGetChildren(stub, global.KEY_GRAPH_PREFIX, startKeyId, filterRule)
}

// GetUserKeys returns a list of keyIds that can be accessed by the User (directly or indirectly).
func GetUserKeys(stub cached_stub.CachedStubInterface, user data_model.User, filter ...interface{}) ([]string, error) {
	var filterRule interface{} = nil
	if len(filter) > 0 {
		filterRule = filter[0]
	}
	return graph.SlowGetChildren(stub, global.KEY_GRAPH_PREFIX, user.GetPubPrivKeyId(), filterRule)
}

// GetOwnerKeys returns a list of keyIds which can be used to access targetKeyId (directly or indirectly).
func GetOwnerKeys(stub cached_stub.CachedStubInterface, targetKeyId string, filter ...interface{}) ([]string, error) {
	//logger.Infof("--- get owner keys %v %v", targetKeyId, filter)
	var filterRule interface{} = nil
	if len(filter) > 0 {
		filterRule = filter[0]
	}
	return graph.SlowGetParents(stub, global.KEY_GRAPH_PREFIX, targetKeyId, filterRule)
}

// ValidateKey checks if key matches existing key in the graph.
// If key does not exist, returns !mustExist.
// If key does exist, returns true if valid, false otherwise.
func ValidateKey(stub cached_stub.CachedStubInterface, key data_model.Key, mustExist bool) (bool, error) {

	if len(key.ID) == 0 || len(key.KeyBytes) == 0 {
		//not a valid key
		return false, nil
	}

	// Look for existing keyGraphNode
	existingKeyGraphNode, err := getKeyGraphNode(stub, key.ID)
	if err != nil {
		logger.Errorf("Failed to get getKeyGraphNode with key.ID \"%v\"", key.ID)
		return false, errors.Wrapf(err, "Failed to get getKeyGraphNode with key.ID \"%v\"", key.ID)
	}

	// Key does not exist
	if len(existingKeyGraphNode.KeyId) == 0 {
		logger.Debug("key does not exist")
		return !mustExist, nil
	}

	// Compare key to existing key in graph
	switch key.Type {
	case global.KEY_TYPE_PUBLIC:
		return bytes.Equal(existingKeyGraphNode.PublicKey, key.KeyBytes), nil
	case global.KEY_TYPE_PRIVATE:
		// Extract the public key and compare
		currKeyNode, err := convertKeyToKeyGraphNode(key)
		if err != nil {
			logger.Errorf("Failed to getKeyGraphNodeFromKey with id \"%v\"", key.ID)
			return false, errors.Wrapf(err, "Failed to getKeyGraphNodeFromKey with id \"%v\"", key.ID)
		}
		return bytes.Equal(existingKeyGraphNode.PublicKey, currKeyNode.PublicKey), nil
	case global.KEY_TYPE_SYM:
		return bytes.Equal(existingKeyGraphNode.SymKeyHash, crypto.Hash(key.KeyBytes)), nil
	default:
		logger.Errorf("Unsupported key type \"%v\"", key.Type)
		return false, errors.Errorf("Unsupported key type \"%v\"", key.Type)
	}
}

// KeyExists checks if the key already exists in the graph, It does not check the validity of the key, only its existence in the graph.
func KeyExists(stub cached_stub.CachedStubInterface, keyId string) bool {
	// check if keyId exists in graph
	existingKeyGraphNode, err := getKeyGraphNode(stub, keyId)
	if err != nil || len(existingKeyGraphNode.KeyId) == 0 {
		return false
	} else {
		return true
	}
}

// convertKeyToKeyGraphNode converts a Key object to a keyGraphNode.
func convertKeyToKeyGraphNode(key data_model.Key) (*keyGraphNode, error) {

	node := keyGraphNode{KeyId: key.ID, IsSymKey: false}

	switch key.Type {
	case global.KEY_TYPE_SYM:
		node.IsSymKey = true
		node.SymKeyHash = crypto.Hash(key.KeyBytes)
	case global.KEY_TYPE_PUBLIC:
		node.PublicKey = key.KeyBytes
	case global.KEY_TYPE_PRIVATE:
		// get publicKey from privateKey
		privateKey, err := crypto.ParsePrivateKey(key.KeyBytes)
		if err != nil {
			custom_err := &custom_errors.ParseKeyError{Type: "private key"}
			logger.Errorf("%v: %v", custom_err, err)
			return nil, errors.Wrap(err, custom_err.Error())
		}
		if privateKey == nil {
			custom_err := &custom_errors.ParseKeyError{Type: "private key"}
			logger.Errorf("%v", custom_err)
			return nil, errors.WithStack(custom_err)
		}
		publicKeyBytes, _ := x509.MarshalPKIXPublicKey(privateKey.Public())
		node.PublicKey = publicKeyBytes
	default:
		logger.Errorf("Unsupported key type \"%v\"", key.Type)
		return nil, errors.Errorf("Unsupported key type \"%v\"", key.Type)
	}

	return &node, nil
}

// ConvertKeyBytesToKey converts a keyId + keyBytes to a Key object
func ConvertKeyBytesToKey(keyId string, keyBytes []byte) (*data_model.Key, error) {
	// Create a new Key object
	key := data_model.Key{ID: keyId, KeyBytes: keyBytes}

	// Set Key.Type
	if crypto.ValidateSymKey(keyBytes) {
		key.Type = global.KEY_TYPE_SYM
	} else if crypto.ValidatePublicKey(keyBytes) {
		key.Type = global.KEY_TYPE_PUBLIC
	} else if crypto.ValidatePrivateKey(keyBytes) {
		key.Type = global.KEY_TYPE_PRIVATE
	} else {
		logger.Errorf("Unknown key type with keyId \"%v\"", keyId)
		return nil, errors.Errorf("Unknown key type with keyId \"%v\"", keyId)
	}
	return &key, nil
}

// getKeyGraphNode retrieves a keyGraphNode from the ledger.
func getKeyGraphNode(stub cached_stub.CachedStubInterface, keyId string) (*keyGraphNode, error) {
	node := keyGraphNode{}
	keyLedgerKey, _ := stub.CreateCompositeKey(global.KEY_NODE_PREFIX, []string{keyId})
	nodeBytes, err := stub.GetState(keyLedgerKey)
	if err != nil {
		getLedgerError := &custom_errors.GetLedgerError{LedgerKey: keyLedgerKey, LedgerItem: global.KEY_NODE_PREFIX}
		logger.Errorf("%v: %v", getLedgerError, err)
		return nil, errors.Wrap(err, getLedgerError.Error())
	}
	json.Unmarshal(nodeBytes, &node)
	return &node, nil
}

// putKeyGraphNode stores a keyGraphNode on the ledger.
func putKeyGraphNode(stub cached_stub.CachedStubInterface, keyGraphNode keyGraphNode) error {
	keyLedgerKey, _ := stub.CreateCompositeKey(global.KEY_NODE_PREFIX, []string{keyGraphNode.KeyId})
	keyNodeBytes, _ := json.Marshal(&keyGraphNode)
	return stub.PutState(keyLedgerKey, keyNodeBytes)
}

func GetStateByPartialCompositeKey(stub cached_stub.CachedStubInterface, keys []string) (shim.StateQueryIteratorInterface, error) {
	return stub.GetStateByPartialCompositeKey(global.KEY_GRAPH_PREFIX, keys)
}

func GetKeyIdForWriteOnlyAccess(assetId string, assetKeyId string, ownerId string) string {
	return global.ACCESS_WRITE_ONLY + "::" + assetKeyId + "::" + assetId + "::" + ownerId
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
