/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package test_utils

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

func copyData(a []byte) []byte {
	if len(a) == 0 {
		return nil
	}
	b := []byte{}
	for _, i := range a {
		b = append(b, i)
	}
	return b
}

// MockChaincode is a mock chaincode.
type MockChaincode struct {
}

// Init is mocked for MockChaincode.
func (t *MockChaincode) Init(shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

// Query is mocked for MockChaincode.
func (t *MockChaincode) Query(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Error("Unknown supported call - Query()")
}

// Invoke is mocked for MockChaincode.
func (t *MockChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

// createMockStub creates a mock stub.
func createMockStub(t *testing.T, name string, cc shim.Chaincode) *shim.MockStub {
	if len(name) == 0 {
		name = "MockStub"
	}
	if cc == nil {
		cc = new(MockChaincode)
	}
	stub := shim.NewMockStub("mockStub", new(MockChaincode))
	AssertTrue(t, stub != nil, "MockStub creation failed")
	return stub
}

// MisbehavingMockStub returns errors for GetState, PutState, and DelState.
type MisbehavingMockStub struct {
	*shim.MockStub
}

// GetState returns a value for a MisbehavingMockStub.
func (stub *MisbehavingMockStub) GetState(key string) ([]byte, error) {
	return nil, errors.New("Misbehaving stub error!")
}

// GetStateByPartialCompositeKey returns a partial composite key for a MisbehavingMockStub.
func (stub *MisbehavingMockStub) GetStateByPartialCompositeKey(key string, value []string) (shim.StateQueryIteratorInterface, error) {
	return nil, errors.New("Misbehaving stub error!")
}

// GetStateByRange returns a range query for a MisbehavingMockStub.
func (stub *MisbehavingMockStub) GetStateByRange(startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	return nil, errors.New("Misbehaving stub error!")
}

// PutState adds a value for a MisbehavingMockStub.
func (stub *MisbehavingMockStub) PutState(key string, value []byte) error {
	return errors.New("Misbehaving stub error!")
}

// DelState deletes a value for a MisbehavingMockStub.
func (stub *MisbehavingMockStub) DelState(key string) error {
	return errors.New("Misbehaving stub error!")
}

// CreateMisbehavingMockStub returns a misbehaving mock stub which returns errors for GetState, PutState, and DelState.
func CreateMisbehavingMockStub(t *testing.T) *MisbehavingMockStub {
	return &MisbehavingMockStub{MockStub: createMockStub(t, "", nil)}
}

// NewMockStub is a mock stub.
type NewMockStub struct {
	*shim.MockStub
	cache          map[string][]byte
	deleted        map[string]bool
	tmap           map[string][]byte
	args           [][]byte
	cc             shim.Chaincode
	signedProposal *peer.SignedProposal
}

// GetState returns an item stored in NewMockStub.
func (stub *NewMockStub) GetState(key string) ([]byte, error) {
	item := copyData(stub.State[key])
	return item, nil
}

// PutState adds a value for a mock stub.
func (stub *NewMockStub) PutState(key string, val []byte) error {
	stub.cache[key] = val
	stub.deleted[key] = false
	return nil
}

// DelState deletes a value for a mock stub.
func (stub *NewMockStub) DelState(key string) error {
	stub.cache[key] = nil
	stub.deleted[key] = true
	return nil
}

// MockTransactionStart returns a transaction ID.
func (stub *NewMockStub) MockTransactionStart(txid string) {
	//reset state
	stub.cache = make(map[string][]byte)
	stub.deleted = make(map[string]bool)
	stub.TxID = txid

	stub.MockStub.MockTransactionStart(txid)
	stub.signedProposal, _ = stub.MockStub.GetSignedProposal()
	logger.Infof(">>>>>>> starting transaction: %v", txid)
}

// MockTransactionEnd returns a transaction ID.
func (stub *NewMockStub) MockTransactionEnd(txid string) {
	//save to state
	for k, d := range stub.deleted {
		//logger.Debugf("Save to Ledger key:%v delete:%v", k, d)
		if d == true {
			err := stub.MockStub.DelState(k)
			//logger.Debugf("Delete ledger key: %v %v", k, err)
			if err != nil {
				logger.Errorf("%v", err)
				break
			}
		} else {
			v, _ := stub.cache[k]
			err := stub.MockStub.PutState(k, v)
			//logger.Debugf("Save Ledger key:%v %v", k, err)
			if err != nil {
				logger.Errorf("%v", err)
				break
			}
		}
	}
	stub.cache = make(map[string][]byte)
	stub.deleted = make(map[string]bool)

	stub.TxID = ""
	stub.args = [][]byte{}
	stub.tmap = make(map[string][]byte)
	stub.signedProposal = nil
	stub.MockStub.MockTransactionEnd(txid)
	logger.Infof("<<<<<<< ending transaction: %v", txid)
}

// GetStateByRange returns a NewFixedMockStateRangeQueryIterator.
func (stub *NewMockStub) GetStateByRange(startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	if err := validateSimpleKeys(startKey, endKey); err != nil {
		return nil, err
	}
	return NewFixedMockStateRangeQueryIterator(stub, startKey, endKey), nil
}

// GetTransient returns transient map of the transaction
func (stub *NewMockStub) GetTransient() (map[string][]byte, error) {
	return stub.tmap, nil
}

// GetArgs returns arguments.
func (stub *NewMockStub) GetArgs() [][]byte {
	return stub.args
}

// GetStringArgs returns a slice of arguments.
func (stub *NewMockStub) GetStringArgs() []string {
	args := stub.GetArgs()
	strargs := make([]string, 0, len(args))
	for _, barg := range args {
		strargs = append(strargs, string(barg))
	}
	return strargs
}

// GetFunctionAndParameters returns function name and parameters.
func (stub *NewMockStub) GetFunctionAndParameters() (function string, params []string) {
	allargs := stub.GetStringArgs()
	function = ""
	params = []string{}
	if len(allargs) >= 1 {
		function = allargs[0]
		params = allargs[1:]
	}
	return
}

// GetSignedProposal simulates peer proposal response.
func (stub *NewMockStub) GetSignedProposal() (*peer.SignedProposal, error) {
	return stub.signedProposal, nil
}

// MockInit initializes this chaincode,  also starts and ends a transaction.
func (stub *NewMockStub) MockInit(uuid string, args [][]byte) peer.Response {
	stub.args = args
	stub.MockTransactionStart(uuid)
	res := stub.cc.Init(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

// MockInvoke invokes this chaincode, also starts and ends a transaction.
func (stub *NewMockStub) MockInvoke(uuid string, args [][]byte, tmap map[string][]byte) peer.Response {
	stub.tmap = tmap
	stub.args = args
	stub.MockTransactionStart(uuid)
	res := stub.cc.Invoke(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

// MockInvokeWithSignedProposal invokes this chaincode, also starts and ends a transaction.
func (stub *NewMockStub) MockInvokeWithSignedProposal(uuid string, args [][]byte, tmap map[string][]byte, sp *peer.SignedProposal) peer.Response {
	stub.tmap = tmap
	stub.args = args
	stub.MockTransactionStart(uuid)
	stub.signedProposal = sp
	res := stub.cc.Invoke(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

// CreateNewMockStub returns a mock stub.
// options = [name string, cc shim.Chaincode, tmap map[string][]byte]
func CreateNewMockStub(t *testing.T, options ...interface{}) *NewMockStub {
	var name string = ""
	var cc shim.Chaincode = nil
	var tmap map[string][]byte = make(map[string][]byte)
	if len(options) >= 1 {
		if val, ok := options[0].(string); ok {
			name = val
			if len(options) >= 2 {
				if val, ok := options[1].(shim.Chaincode); ok {
					cc = val
				}
			}
		}
	}
	stub := &NewMockStub{MockStub: createMockStub(t, name, cc), cache: make(map[string][]byte), deleted: make(map[string]bool), tmap: tmap, cc: cc}
	return stub
}

// COMPOSITE_KEY_NAMESPACE is the namespace for composite keys.
// Simple keys should not enter the composite key namespace to avoid simple/composite
// key collisions.
const COMPOSITE_KEY_NAMESPACE = "\x00"

// validateSimpleKeys validates simpleKeys to check whether they start
// with 0x00 (which is the namespace for compositeKey). This is to ensure that
// simple keys do not enter the composite key namespace, avoiding simple/composite
// key collisions.
func validateSimpleKeys(simpleKeys ...string) error {
	for _, key := range simpleKeys {
		if len(key) > 0 && key[0] == COMPOSITE_KEY_NAMESPACE[0] {
			return fmt.Errorf(`First character of the key [%s] contains a null character which is not allowed`, key)
		}
	}
	return nil
}

/*****************************
 Range Query Iterator
*****************************/

// FixedMockStateRangeQueryIterator overrides the broken HasNext() and Next() methods of MockStateRangeQueryIterator.
// Fabric's current implementation doesn't handle one-sided open-ended range queries.
// Also, if fixes an issue with endKey being inclusive: startKey is inclusive and endKey should be exclusive
type FixedMockStateRangeQueryIterator struct {
	shim.MockStateRangeQueryIterator
}

// NewFixedMockStateRangeQueryIterator returns a range query iterator that supports one-sided and open-ended range queries.
func NewFixedMockStateRangeQueryIterator(stub *NewMockStub, startKey string, endKey string) *FixedMockStateRangeQueryIterator {
	logger.Debug("NewFixedMockStateRangeQueryIterator(", stub, startKey, endKey, ")")
	iter := new(FixedMockStateRangeQueryIterator)
	iter.Closed = false
	iter.Stub = stub.MockStub
	iter.StartKey = startKey
	iter.EndKey = endKey
	iter.Current = stub.Keys.Front()

	iter.Print()

	return iter
}

// HasNext returns true if the range query iterator contains additional keys and values.
func (iter *FixedMockStateRangeQueryIterator) HasNext() bool {
	if iter.Closed {
		// previously called Close()
		logger.Error("HasNext() but already closed")
		return false
	}

	if iter.Current == nil {
		logger.Debug("HasNext() couldn't get Current")
		return false
	}

	current := iter.Current
	for current != nil {
		comp1 := strings.Compare(current.Value.(string), iter.StartKey)
		comp2 := strings.Compare(current.Value.(string), iter.EndKey)
		if comp1 >= 0 || len(iter.StartKey) == 0 {
			if comp2 < 0 || len(iter.EndKey) == 0 {
				logger.Debug("HasNext() got next")
				return true
			} else {
				logger.Debug("HasNext() but no next")
				return false

			}
		}
		current = current.Next()
	}

	// we've reached the end of the underlying values
	logger.Debug("HasNext() but no next")
	return false
}

// Next returns the next key and value in the range query iterator.
func (iter *FixedMockStateRangeQueryIterator) Next() (*queryresult.KV, error) {
	if iter.Closed == true {
		logger.Error("FixedMockStateRangeQueryIterator.Next() called after Close()")
		return nil, errors.New("FixedMockStateRangeQueryIterator.Next() called after Close()")
	}

	if iter.HasNext() == false {
		logger.Error("FixedMockStateRangeQueryIterator.Next() called when it does not HaveNext()")
		return nil, errors.New("FixedMockStateRangeQueryIterator.Next() called when it does not HaveNext()")
	}

	for iter.Current != nil {
		comp1 := strings.Compare(iter.Current.Value.(string), iter.StartKey)
		comp2 := strings.Compare(iter.Current.Value.(string), iter.EndKey)
		// compare to start and end keys. or, if this is an open-ended query for
		// all keys, it should always return the key and value
		if (comp1 >= 0 || len(iter.StartKey) == 0) && (comp2 < 0 || len(iter.EndKey) == 0) {
			key := iter.Current.Value.(string)
			value, err := iter.Stub.GetState(key)
			iter.Current = iter.Current.Next()
			return &queryresult.KV{Key: key, Value: value}, err
		}
		iter.Current = iter.Current.Next()
	}
	logger.Error("FixedMockStateRangeQueryIterator.Next() went past end of range")
	return nil, errors.New("FixedMockStateRangeQueryIterator.Next() went past end of range")
}

// CreateExampleMockStub returns a mock stub without *testing.T object, for use in godoc examples.
func CreateExampleMockStub() *NewMockStub {
	stub := shim.NewMockStub("mockStub", new(MockChaincode))
	mockStub := &NewMockStub{MockStub: stub}
	return mockStub
}
