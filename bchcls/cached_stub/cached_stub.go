/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package cached_stub is used for caching ledger data and arbitrary data to
// improve transaction efficiency.
package cached_stub

import (
	"common/bchcls/crypto"
	"common/bchcls/custom_errors"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/common/metering_connections"
	"common/bchcls/utils"

	"sort"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("cached_stub")

const compositeKeyNamespacePrefix = "\x02"
const compositeKeyNamespace = "\x00"
const emptyKeySubstitute = "\x01"

// CachedStubInterface extends Fabric Shim's ChaincodeStubInterface, please refer to Shim's docs for more info.
type CachedStubInterface interface {
	shim.ChaincodeStubInterface

	// GetCache gets a stored object (interface{}) from the cache.
	// Returns nil and an error if key does not exist in the cache.
	// Note that the object is stored as an interface.
	// This implies that the actual value stored in the cache is a pointer to the caller's object.
	// Hence, if the caller makes changes to the object (i.e []byte type) in solution chaincode,
	// the changed value might remain when the caller gets the object from the cache.
	// It is therefore the caller's responsibility to prevent side effects in solution chaincode.
	GetCache(key string) (interface{}, error)

	// PutCache stores an object as an interface in the cache.
	PutCache(key string, value interface{}) error

	// DelCache deletes an object from the cache by setting its value in the cache to nil.
	DelCache(key string) error
}

// NewCachedStub creates a new instance of cachedStub
//
// options: enable_get_cache (default: true), enable_put_cache (default: false)
// cache for storing arbitrary data is always enabled
func NewCachedStub(stub shim.ChaincodeStubInterface, options ...bool) CachedStubInterface {
	// As long as we are not in development env, do metering
	if !utils.IsStringEmpty(global.DevelopmentEnv) && global.DevelopmentEnv != global.DevelopmentEnvString {
		channelID := stub.GetChannelID()
		_ = metering_connections.InstantiateCloudant(channelID)
		_ = metering_connections.CreateMeteringIndex(channelID)
	}

	enable_get_cache := true
	enable_put_cache := false
	if len(options) >= 1 {
		enable_get_cache = options[0]
	}
	if len(options) >= 2 {
		enable_put_cache = options[1]
	}
	return &cachedStub{stub: stub,
		cache_state:            make(map[string][]byte),
		del_state:              make(map[string]bool),
		ChaincodeStubInterface: stub,
		cache:                  make(map[string]interface{}),
		enable_get_cache:       enable_get_cache,
		enable_put_cache:       enable_put_cache,
		sorted_keys_map:        make(map[string]bool),
		sorted_keys:            []string{}}
}

// cachedStub extends ChaincodeStubInterface and implements
// caching for both ledger data and arbitrary data.
type cachedStub struct {
	shim.ChaincodeStubInterface

	stub             shim.ChaincodeStubInterface
	cache_state      map[string][]byte
	del_state        map[string]bool
	cache            map[string]interface{}
	enable_get_cache bool
	enable_put_cache bool
	sorted_keys_map  map[string]bool
	sorted_keys      []string
}

func (stub *cachedStub) normalizeKey(key string) string {
	if len(key) > 0 && key[0] == compositeKeyNamespace[0] {
		return compositeKeyNamespacePrefix + key
	}
	return key
}

func (stub *cachedStub) deNormalizeKey(key string) string {
	if len(key) > 0 && key[0] == compositeKeyNamespacePrefix[0] {
		return key[1:]
	}
	return key
}

// GetState calls chaincodeStub.GetState and returns the value of the specified `key` from either the cache
// or the ledger. If stub.enable_get_cache is true, it first checks the cache to see if it exists there.
func (stub *cachedStub) GetState(key string) ([]byte, error) {
	key = stub.normalizeKey(key)
	// TODO: If enable_put_cache option is set to true, GetState retuns data modified by PutState even
	// if that has not been committed.
	chaincodeStub, ok := stub.stub.(interface {
		GetState(key string) ([]byte, error)
	})
	if ok {
		// If enable_get_cache option is set to true, it first checks cache and returns from
		// cache if the key and value exist in cache. Otherwise, it returns value from ledger.
		// If enable_get_cache option is set to false, it always get values from ledger.
		var value []byte
		var err error
		if stub.enable_get_cache {
			value, err = stub.getStateCache(key)
			if err == nil {
				return value, nil
			}
		}

		// get it from ledger
		value, err = chaincodeStub.GetState(key)
		if err != nil {
			return value, err
		} else if value != nil {
			return stub.putStateCache(key, value, true)
		} else {
			return nil, stub.delStateCache(key)
		}
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "GetState"}))
	}
}

func (stub *cachedStub) getStateCache(key string) ([]byte, error) {
	if val, ok := stub.cache_state[key]; ok {
		// return from cache
		if val != nil {
			tmp := make([]byte, len(val))
			copy(tmp, val)
			logger.Debugf("return from cache %v", key)
			return tmp, nil
		} else {
			// If the key does not exist in the state database or cache, (nil, nil) is returned.
			return val, nil
		}
	} else {
		return nil, errors.New("Cache doesn't exist")
	}
}

// default value of isStartKey should be true
func (stub *cachedStub) putStateCache(key string, value []byte, isStartKey bool) ([]byte, error) {
	stub.del_state[key] = false
	if value != nil {
		// We make a copy of the []byte value in cache_state before returning,
		// in order to prevent the side effect of changing values that could
		// occur when returning []bytes as a reference.
		tmp := make([]byte, len(value))
		copy(tmp, value)
		//save the value to cache
		stub.cache_state[key] = value
		logger.Debugf("add to cache %v %v", key, isStartKey)

		//add key to sorted key list
		stub.putToSortedKeys(key, isStartKey)
		return tmp, nil

	} else {
		//save the value to cache
		stub.cache_state[key] = value
		logger.Debugf("add to cache %v %v", key, isStartKey)

		//add key to sorted key list
		stub.putToSortedKeys(key, isStartKey)
		return value, nil
	}
}

// default value of isStartKey should be true
func (stub *cachedStub) delStateCache(key string) error {
	//indicate it's deleted
	stub.del_state[key] = true
	//save the value to cache
	stub.cache_state[key] = nil
	logger.Debugf("del cache %v ", key)

	//don't need to keep key in sorted list
	utils.RemoveString(stub.sorted_keys, key)
	delete(stub.sorted_keys_map, key)
	return nil

}

// PutState calls chaincodeStub.PutState and saves the key value pair to the cache.
func (stub *cachedStub) PutState(key string, value []byte) error {
	key = stub.normalizeKey(key)
	chaincodeStub, ok := stub.stub.(interface {
		PutState(key string, value []byte) error
	})
	if ok {
		err := chaincodeStub.PutState(key, value)
		if err != nil {
			return err
		}
		if stub.enable_put_cache {
			//make sure to mark that it's no longer deleted
			_, err := stub.putStateCache(key, value, true)
			return err
		}
		return nil
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "putState"}))
	}
}

// DelState calls chaincodeStub.DelState to delete key from the ledger
// and remove it from the cache.
func (stub *cachedStub) DelState(key string) error {
	key = stub.normalizeKey(key)
	chaincodeStub, ok := stub.stub.(interface {
		DelState(key string) error
	})
	if ok {
		err := chaincodeStub.DelState(key)
		if err != nil {
			return err
		}
		if stub.enable_put_cache {
			err := stub.delStateCache(key)
			logger.Debugf("mark cache delted: %v", key)
			return err
		}
		return nil
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "DelState"}))
	}
}

// GetStateByRange calls chaincodeStub.GetStateByRange and stores the results in the cache.
func (stub *cachedStub) GetStateByRange(startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("startKey: %v endKey: %v", startKey, endKey)

	_, ok := stub.stub.(interface {
		GetStateByRange(startKey, endKey string) (shim.StateQueryIteratorInterface, error)
	})
	if ok {
		if len(startKey) == 0 {
			startKey = emptyKeySubstitute
		}
		if err := stub.validateSimpleKeys(startKey, endKey); err != nil {
			return nil, err
		}
		return stub.getCachedStubIter(startKey, endKey, "", "")
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "GetStateByRange"}))
	}
}

// GetStateByRange calls chaincodeStub.GetStateByRange and stores the results in the cache.
func (stub *cachedStub) GetStateByRangeWithPagination(startKey, endKey string, pageSize int32,
	bookmark string) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("startKey: %v endKey: %v", startKey, endKey)

	chaincodeStub, ok := stub.stub.(interface {
		GetStateByRangeWithPagination(startKey, endKey string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error)
	})
	if ok {
		if len(startKey) == 0 {
			startKey = emptyKeySubstitute
		}
		if err := stub.validateSimpleKeys(startKey, endKey); err != nil {
			return nil, nil, err
		}

		iter, metadata, err := chaincodeStub.GetStateByRangeWithPagination(startKey, endKey, pageSize, bookmark)
		if err != nil {
			return iter, metadata, err
		}
		return &cachedStubIterSimple{iter: iter, stub: stub, prefix: "", isStart: true}, metadata, nil

	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "GetStateByRangeWithPagination"}))
	}
}

// add key to the sorted_keys list and return whether the key is start key or not
// if value is true, this item is a start key
func (stub *cachedStub) putToSortedKeys(key string, value bool) bool {
	existingValue := value
	if value1, ok := stub.sorted_keys_map[key]; ok {
		existingValue = value1
	} else {
		// add to sorted key list
		stub.sorted_keys = utils.InsertString(stub.sorted_keys, key)
	}
	// update sorted_keys_map
	newValue := existingValue && value
	stub.sorted_keys_map[key] = newValue

	return newValue
}

func (stub *cachedStub) getIter(startKey, endKey, collection string) (shim.StateQueryIteratorInterface, error) {
	//defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	//logger.Debugf("start %v end %v", startKey, endKey)
	if len(collection) > 0 {
		chaincodeStub, ok := stub.stub.(interface {
			GetPrivateDataByRange(collection, startKey, endKey string) (shim.StateQueryIteratorInterface, error)
		})
		if ok {
			return chaincodeStub.GetPrivateDataByRange(collection, startKey, endKey)
		} else {
			panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "GetPrivateDataByRange"}))
		}
	} else {

		iter, err := stub.stub.GetStateByRange(startKey, endKey)
		return iter, err
	}
}

func (stub *cachedStub) getCachedStubIter(startKey, endKey string, prefix string, collection string) (shim.StateQueryIteratorInterface, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("getCachedStubIter startKey: %v endKey: %v prefix: %v collection: %v", startKey, endKey, prefix, collection)
	var iter shim.StateQueryIteratorInterface
	var err error
	enclosing, index, err := stub.getKeyRangeIndex(prefix+startKey, prefix+endKey)
	if index[0] < len(stub.sorted_keys) {
		sKey := stub.sorted_keys[index[0]]
		if sKey != prefix+startKey {
			// sKey must be bigger than startKey
			isStart := stub.sorted_keys_map[sKey]
			if isStart {
				if sKey < prefix+endKey {
					if len(prefix) > 0 {
						iter, err = stub.getIter(startKey, sKey[len(prefix):], collection)
						if err != nil {
							logger.Errorf("Error getting iter: %v", err)
							return nil, err
						}
					} else {
						iter, err = stub.getIter(startKey, sKey, collection)
						if err != nil {
							logger.Errorf("Error getting iter: %v", err)
							return nil, err
						}
					}
				} else {
					iter, err = stub.getIter(startKey, endKey, collection)
					if err != nil {
						logger.Errorf("Error getting iter: %v", err)
						return nil, err
					}
				}
			}
		}
	} else {
		iter, err = stub.getIter(startKey, endKey, collection)
		if err != nil {
			logger.Errorf("Error getting iter: %v", err)
			return nil, err
		}
	}
	return &cachedStubIter{
		stub:          stub,
		prefix:        prefix,
		collection:    collection,
		startKey:      startKey,
		endKey:        endKey,
		enclosingKeys: enclosing,
		rangeIndex:    index,
		closed:        false,
		index:         index[0],
		iter:          iter,
		nextKV:        nil,
		err:           err,
		isFirst:       true}, nil

}

// given start and end key, return slice within sorted_keys
// return
//  list of keys that encloses the range [before-startkey, after-endkey]
//  list of keys within the range (start -end) [from-startkey, ... , before-endkey]
func (stub *cachedStub) getKeyRangeIndex(startKey, endKey string) ([]string, []int, error) {
	//defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	//logger.Debugf("start: %v end: %v sorted: %v", startKey, endKey, stub.sorted_keys)
	i := sort.SearchStrings(stub.sorted_keys, startKey)
	j := sort.SearchStrings(stub.sorted_keys, endKey)
	s := ""
	e := ""
	if i >= 1 {
		s = stub.sorted_keys[i-1]
		if s == startKey {
			i = i - 1
			if i >= 1 {
				s = stub.sorted_keys[i-1]
			} else {
				s = ""
			}
		}
	}
	if j < len(stub.sorted_keys) {
		e = stub.sorted_keys[j]
	}

	enclosure := []string{s, e}
	within := []int{i, j}
	//logger.Debugf("index start %v end %v index %v", startKey, endKey, within)
	return enclosure, within, nil
}

//SplitCompositeKey documentation can be found in interfaces.go
func (stub *cachedStub) SplitCompositeKey(compositeKey string) (string, []string, error) {
	chaincodeStub, ok := stub.stub.(interface {
		SplitCompositeKey(compositeKey string) (string, []string, error)
	})
	if ok {
		if compositeKey[0] == compositeKeyNamespacePrefix[0] {
			compositeKey = compositeKey[1:]
		}
		return chaincodeStub.SplitCompositeKey(compositeKey)
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "SplitCompositeKey"}))
	}
}

//To ensure that simple keys do not go into composite key namespace,
//we validate simplekey to check whether the key starts with 0x00 (which
//is the namespace for compositeKey). This helps in avoding simple/composite
//key collisions.
func (stub *cachedStub) validateSimpleKeys(simpleKeys ...string) error {
	for _, key := range simpleKeys {
		if len(key) > 0 && key[0] == compositeKeyNamespacePrefix[0] {
			key = key[1:]
			if len(key) > 0 && key[0] == compositeKeyNamespace[0] {
				return errors.Errorf(`first character of the key [%s] contains a null character which is not allowed`, key)
			}
		} else if len(key) > 0 && key[0] == compositeKeyNamespace[0] {
			return errors.Errorf(`first character of the key [%s] contains a null character which is not allowed`, key)
		}
	}
	return nil
}

// GetStateByPartialCompositeKey calls chaincodeStub.GetStateByPartialCompositeKey and stores the results in the cache.
func (stub *cachedStub) GetStateByPartialCompositeKey(objectType string, keys []string) (shim.StateQueryIteratorInterface, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("objectType: %v keys: %v", objectType, keys)

	_, ok := stub.stub.(interface {
		GetStateByPartialCompositeKey(objectType string, keys []string) (shim.StateQueryIteratorInterface, error)
	})
	if ok {
		if partialCompositeKey, err := stub.CreateCompositeKey(objectType, keys); err == nil {
			partialCompositeKey = stub.normalizeKey(partialCompositeKey)
			return stub.getCachedStubIter(partialCompositeKey, partialCompositeKey+string(global.MAX_UNICODE_RUNE_VALUE), "", "")
		} else {
			return nil, err
		}
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "GetStateByPartialCompositeKey"}))
	}
}

// GetStateByPartialCompositeKeyWithPagination calls chaincodeStub.GetStateByPartialCompositeKeyWithPagination and stores the results in the cache.
func (stub *cachedStub) GetStateByPartialCompositeKeyWithPagination(objectType string, keys []string,
	pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("objectType: %v keys: %v", objectType, keys)

	chaincodeStub, ok := stub.stub.(interface {
		GetStateByRangeWithPagination(startKey, endKey string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error)
	})
	if ok {
		if partialCompositeKey, err := stub.CreateCompositeKey(objectType, keys); err == nil {
			partialCompositeKey = stub.normalizeKey(partialCompositeKey)

			iter, metadata, err := chaincodeStub.GetStateByRangeWithPagination(partialCompositeKey, partialCompositeKey+string(global.MAX_UNICODE_RUNE_VALUE), pageSize, bookmark)
			if err != nil {
				return iter, metadata, err
			}
			return &cachedStubIterSimple{iter: iter, stub: stub, prefix: "", isStart: true}, metadata, nil

		} else {
			return nil, nil, err
		}
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "GetStateByRangeWithPagination"}))
	}
}

// GetQueryResult calls chaincodeStub.GetQueryResult and stores the results in the cache.
func (stub *cachedStub) GetQueryResult(query string) (shim.StateQueryIteratorInterface, error) {

	chaincodeStub, ok := stub.stub.(interface {
		GetQueryResult(query string) (shim.StateQueryIteratorInterface, error)
	})
	if ok {
		iter, err := chaincodeStub.GetQueryResult(query)
		if err != nil {
			return iter, err
		}
		return &cachedStubIterSimple{isStart: true, iter: iter, stub: stub, prefix: ""}, nil
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "GetQueryResult"}))
	}
}

// GetQueryResultWithPagination calls chaincodeStub.GetQueryResultWithPagination and stores the results in the cache.
func (stub *cachedStub) GetQueryResultWithPagination(query string, pageSize int32,
	bookmark string) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {

	chaincodeStub, ok := stub.stub.(interface {
		GetQueryResultWithPagination(query string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error)
	})
	if ok {
		iter, metadata, err := chaincodeStub.GetQueryResultWithPagination(query, pageSize, bookmark)
		if err != nil {
			return iter, metadata, err
		}
		return &cachedStubIterSimple{isStart: true, iter: iter, stub: stub, prefix: ""}, metadata, nil
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "GetQueryResult"}))
	}
}

// GetPrivateData calls chaincodeStub.GetPrivateData and stores the results in the cache.
func (stub *cachedStub) GetPrivateData(collection, key string) ([]byte, error) {
	key = stub.normalizeKey(key)
	chaincodeStub, ok := stub.stub.(interface {
		GetPrivateData(collection, key string) ([]byte, error)
	})
	if ok {
		return chaincodeStub.GetPrivateData(collection, key)
		ckey := "P_" + collection + "_P_" + key
		if val, ok := stub.cache_state[ckey]; ok {
			// return from cache
			return val, nil
		} else {
			value, err := chaincodeStub.GetPrivateData(collection, key)
			if err != nil {
				return value, err
			} else {
				//save the value to cache
				stub.putStateCache(ckey, value, true)
				//stub.cache_state[ckey] = value
				return value, nil
			}
		}
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "GetPrivateData"}))
	}
}

// PutPrivateData calls chaincodeStub.PutPrivateData and stores the key value pair in the cache.
func (stub *cachedStub) PutPrivateData(collection, key string, value []byte) error {
	key = stub.normalizeKey(key)
	chaincodeStub, ok := stub.stub.(interface {
		PutPrivateData(collection, key string, value []byte) error
	})
	if ok {
		err := chaincodeStub.PutPrivateData(collection, key, value)
		if err != nil {
			return err
		}
		if stub.enable_put_cache {
			ckey := "P_" + collection + "_P_" + key
			//make sure to mark that it's no longer deleted
			stub.del_state[ckey] = false
			tmp := make([]byte, len(value))
			copy(tmp, value)
			//save the value to cache
			stub.putStateCache(ckey, value, true)
			logger.Debugf("add to cache %v %v", ckey, tmp)
			return nil
		}
		return nil
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "putPrivateData"}))
	}
}

// DelPrivateData calls chaincodeStub.DelPrivateData and deletes the key from the cache.
func (stub *cachedStub) DelPrivateData(collection, key string) error {
	key = stub.normalizeKey(key)
	chaincodeStub, ok := stub.stub.(interface {
		DelPrivateData(collection, key string) error
	})
	if ok {
		err := chaincodeStub.DelPrivateData(collection, key)
		if err != nil {
			return err
		}
		if stub.enable_put_cache {
			ckey := "P_" + collection + "_P_" + key
			stub.delStateCache(ckey)
			logger.Debugf("mark cache delted: %v", ckey)
			return nil
		}
		return nil
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "DelPrivateData"}))
	}
}

// GetPrivateDataByRange calls chaincodeStub.GetPrivateDataByRange and stores the result in the cache.
func (stub *cachedStub) GetPrivateDataByRange(collection, startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	_, ok := stub.stub.(interface {
		GetPrivateDataByRange(collection, startKey, endKey string) (shim.StateQueryIteratorInterface, error)
	})
	if ok {
		if len(collection) == 0 {
			return nil, errors.New("collection must not be an empty string")
		}
		if len(startKey) == 0 {
			startKey = emptyKeySubstitute
		}
		if err := stub.validateSimpleKeys(startKey, endKey); err != nil {
			return nil, err
		}
		return stub.getCachedStubIter(startKey, endKey, "P_"+collection+"_P", collection)

	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "GetPrivateDataByRange"}))
	}
}

// GetPrivateDataByPartialCompositeKey calls chaincodeStub.GetPrivateDataByPartialCompositeKey and stores the result in the cache.
func (stub *cachedStub) GetPrivateDataByPartialCompositeKey(collection, objectType string, keys []string) (shim.StateQueryIteratorInterface, error) {
	_, ok := stub.stub.(interface {
		GetPrivateDataByPartialCompositeKey(collection, objectType string, keys []string) (shim.StateQueryIteratorInterface, error)
	})
	if ok {
		if partialCompositeKey, err := stub.CreateCompositeKey(objectType, keys); err == nil {
			partialCompositeKey = stub.normalizeKey(partialCompositeKey)
			return stub.getCachedStubIter(partialCompositeKey, partialCompositeKey+string(global.MAX_UNICODE_RUNE_VALUE), "_P_"+collection+"_P", collection)
		} else {
			return nil, err
		}
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "GetPrivateDataByPartialCompositeKey"}))
	}
}

// GetPrivateDataQueryResult calls chaincodeStub.GetPrivateDataQueryResult and stores the result in the cache.
func (stub *cachedStub) GetPrivateDataQueryResult(collection, query string) (shim.StateQueryIteratorInterface, error) {
	chaincodeStub, ok := stub.stub.(interface {
		GetPrivateDataQueryResult(collection, query string) (shim.StateQueryIteratorInterface, error)
	})
	if ok {
		iter, err := chaincodeStub.GetPrivateDataQueryResult(collection, query)
		if err != nil {
			return iter, err
		}
		return &cachedStubIterSimple{isStart: true, iter: iter, stub: stub, prefix: "P_" + collection + "_P"}, nil
	} else {
		panic(errors.WithStack(&custom_errors.MethodNotImplementedError{Method: "GetPrivateDataQueryResult"}))
	}
}

// getCacheKey returns a hash of the key.
func (stub *cachedStub) getCacheKey(key string) string {
	return crypto.HashB64([]byte(key))
}

// GetCache gets a stored object (interface{}) from the cache.
// Returns nil and an error if key does not exist in the cache.
// Note that the object is stored as an interface.
// This implies that the actual value stored in the cache is a pointer to the caller's object.
// Hence, if the caller makes changes to the object (i.e []byte type) in solution chaincode,
// the changed value might remain when the caller gets the object from the cache.
// It is therefore the caller's responsibility to prevent side effects in solution chaincode.
func (stub *cachedStub) GetCache(key string) (interface{}, error) {
	hashKey := stub.getCacheKey(key)
	if val, ok := stub.cache[hashKey]; ok {
		return val, nil
	} else {
		return nil, errors.New("key does not exist")
	}
}

// PutCache stores an object as an interface in the cache.
func (stub *cachedStub) PutCache(key string, value interface{}) error {
	hashKey := stub.getCacheKey(key)
	stub.cache[hashKey] = value
	return nil
}

// DelCache deletes an object from the cache by setting its value in the cache to nil.
func (stub *cachedStub) DelCache(key string) error {
	hashKey := stub.getCacheKey(key)
	stub.cache[hashKey] = nil
	return nil
}
