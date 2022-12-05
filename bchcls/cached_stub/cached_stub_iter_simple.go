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
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
)

// cachedStubIterSimple is an iterator of cachedStubs.
// prefix indicates whether the cachedStubIter belongs to a collection
type cachedStubIterSimple struct {
	stub    *cachedStub
	iter    shim.StateQueryIteratorInterface
	prefix  string
	isStart bool
}

// HasNext returns a boolean indicating if cachedStubIter has another element.
func (citer *cachedStubIterSimple) HasNext() bool {
	return citer.iter.HasNext()
}

// Next returns the next key value pair.
func (citer *cachedStubIterSimple) Next() (*queryresult.KV, error) {
	KV, err := citer.iter.Next()

	if err == nil {
		key := KV.GetKey()
		value := KV.GetValue()
		citer.stub.putStateCache(citer.prefix+key, value, citer.isStart)
		citer.isStart = false

		key2 := citer.stub.deNormalizeKey(key)
		return &queryresult.KV{Key: key2, Value: value}, nil
	}

	return KV, err
}

// Close closes the cachedStubIter iterator.
func (citer *cachedStubIterSimple) Close() error {
	return citer.iter.Close()
}
