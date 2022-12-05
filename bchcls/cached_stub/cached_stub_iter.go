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
	"common/bchcls/internal/common/global"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/pkg/errors"
)

// cachedStubIter is an iterator of cachedStubs.
// prefix indicates whether the cachedStubIter belongs to a collection
// cachedStubIter uses sorted_keys list to get next items
type cachedStubIter struct {
	stub          *cachedStub
	prefix        string
	collection    string
	startKey      string
	endKey        string
	enclosingKeys []string
	rangeIndex    []int
	closed        bool
	index         int
	iter          shim.StateQueryIteratorInterface
	nextKV        *queryresult.KV
	err           error
	isFirst       bool
}

// HasNext returns a boolean indicating if cachedStubIter has another element.
func (citer *cachedStubIter) HasNext() bool {
	if citer.isFirst {
		// get next item
		citer.nextKV, citer.err = citer.getNext()
		citer.isFirst = false
	}
	if citer.nextKV != nil {
		return true
	} else {
		return false
	}
}

// Next returns the next key value pair.
func (citer *cachedStubIter) Next() (*queryresult.KV, error) {

	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	if citer.closed {
		return nil, errors.New("iter closed")
	}
	if citer.err != nil {
		citer.closed = true
	}
	currKV := citer.nextKV
	currErr := citer.err
	// get next item
	citer.nextKV, citer.err = citer.getNext()

	if currKV != nil {
		key := citer.stub.deNormalizeKey(currKV.GetKey())
		value := currKV.GetValue()
		//logger.Debugf("++++> return Next %v %v %v", key, value, currErr)
		return &queryresult.KV{Key: key, Value: value}, currErr
	}

	//logger.Debugf("++++> return Next %v %v", currKV, currErr)
	return currKV, currErr
}

func (citer *cachedStubIter) getNext() (*queryresult.KV, error) {
	//defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	currKV := citer.nextKV

	if citer.iter != nil {
		if citer.iter.HasNext() {
			KV, err := citer.iter.Next()
			if err != nil {
				citer.closed = true
				citer.iter.Close()
				citer.iter = nil
			} else {
				//saving cache
				key := KV.GetKey()
				value := KV.GetValue()
				citer.stub.putStateCache(citer.prefix+key, value, citer.isFirst)

				//update index
				startKey := citer.prefix + key + string(global.MIN_UNICODE_RUNE_VALUE)
				_, index, _ := citer.stub.getKeyRangeIndex(startKey, citer.endKey)
				citer.index = index[0]
				citer.rangeIndex[1] = index[1]
			}
			//logger.Debugf("++++ return from iter %v %v", KV, err)
			return KV, err
		} else {
			// no more next
			citer.iter.Close()
			citer.iter = nil
		}
	}
	// fallback to next item in the slide
	citer.iter = nil
	citer.nextKV = nil
	//logger.Debugf("++++ fallback %v %v", citer.index, citer.stub.sorted_keys)

	//update index; skip for the first time since we already have initial index
	if !citer.isFirst {
		startKey := citer.prefix + currKV.GetKey() + string(global.MIN_UNICODE_RUNE_VALUE)
		_, index, _ := citer.stub.getKeyRangeIndex(startKey, citer.endKey)
		citer.index = index[0]
		citer.rangeIndex[1] = index[1]
	}

	if citer.index < citer.rangeIndex[1] {
		key := citer.stub.sorted_keys[citer.index]
		value, _ := citer.stub.getStateCache(key)
		key2 := key
		// make sure to remove prefix before return
		if len(citer.prefix) > 0 {
			key2 = key[len(citer.prefix):]
		}
		KV := queryresult.KV{Key: key2, Value: value}

		citer.index = citer.index + 1

		// check if we need to set iter
		if citer.index < citer.rangeIndex[1] {
			nextKey := citer.stub.sorted_keys[citer.index]
			isFirst := citer.stub.sorted_keys_map[nextKey]
			if isFirst {
				//next one is iter
				startKey := key + string(global.MIN_UNICODE_RUNE_VALUE)
				if len(citer.prefix) > 0 {
					startKey = startKey[len(citer.prefix):]
					nextKey = nextKey[len(citer.prefix):]
				}
				var err error
				citer.iter, err = citer.stub.getIter(startKey, nextKey, citer.collection)
				if err != nil {
					logger.Errorf("Error getting iter: %v", err)
					return nil, err
				}
			}
		} else if citer.index == citer.rangeIndex[1] && key < citer.prefix+citer.endKey {
			//last index check if the last iter exist
			if citer.index < len(citer.stub.sorted_keys) {
				// if last index exists, next one is iter only if if last index is a startKey
				lastKey := citer.stub.sorted_keys[citer.index]
				isStartKey := citer.stub.sorted_keys_map[lastKey]
				if isStartKey {
					startKey := key + string(global.MIN_UNICODE_RUNE_VALUE)
					if len(citer.prefix) > 0 {
						startKey = startKey[len(citer.prefix):]
					}
					var err error
					citer.iter, err = citer.stub.getIter(startKey, citer.endKey, citer.collection)
					if err != nil {
						logger.Errorf("Error getting iter: %v", err)
						return nil, err
					}
				}
			} else {
				startKey := key + string(global.MIN_UNICODE_RUNE_VALUE)
				if len(citer.prefix) > 0 {
					startKey = startKey[len(citer.prefix):]
				}
				var err error
				citer.iter, err = citer.stub.getIter(startKey, citer.endKey, citer.collection)
				if err != nil {
					logger.Errorf("Error getting iter: %v", err)
					return nil, err
				}
			}
		}

		//logger.Debugf("++++ return from list %v", KV)
		return &KV, nil
	}

	citer.closed = true
	return nil, nil
}

// Close closes the cachedStubIter iterator.
func (citer *cachedStubIter) Close() error {
	citer.closed = true
	citer.nextKV = nil
	citer.err = nil
	var err error
	if citer.iter != nil {
		err = citer.iter.Close()
		citer.iter = nil
	}
	return err
}
