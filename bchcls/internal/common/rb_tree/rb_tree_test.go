/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package rb_tree

import (
	"common/bchcls/cached_stub"
	"common/bchcls/test_utils"

	"strconv"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func setup(t *testing.T) *test_utils.NewMockStub {
	mstub := test_utils.CreateNewMockStub(t)
	return mstub
}

func TestTree(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	//logger.SetLevel(shim.LogInfo)
	logger.Info("TestTree function called")
	DEBUG_TREE = true

	// create a MockStub
	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	//create a tree
	tr := NewRBTree(stub, "test")
	var err error

	//add composit keys
	logger.Debug("adding composit keys")
	for _, k1 := range []string{"tom", "jane", "alex"} {
		for _, k2 := range []string{"1", "2", "3", "4", "5", "6"} {
			key, _ := stub.CreateCompositeKey("Test", []string{k1, k2})
			v := []byte("value for " + key)
			err := tr.Insert(key, v)
			logger.Debugf("Insert %v %v", key, err)
			tr.PrintTree("", "")
		}
	}

	logger.Debug("adding range keys")
	for i := 1; i < 40; i = i + 1 {
		s := strconv.Itoa(i)
		v := []byte("value for " + s)
		err := tr.Insert(s, v)
		logger.Debugf("Insert %v %v", s, err)
		tr.PrintTree("", "")
	}

	tr.PrintTree("", "")

	//err = tr.SaveToLedger()
	test_utils.AssertTrue(t, err == nil, "succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	logger.Debug("-------------new transaction ")

	tr = NewRBTree(stub, "test")
	tr.PrintTree("", "")

	logger.Debug("jane by partial compositkey")
	iter, err := tr.GetKeyByPartialCompositeKey("Test", []string{"jane"})
	defer iter.Close()
	for iter.HasNext() {
		KV, err := iter.Next()
		k := KV.GetKey()
		b := KV.GetValue()

		logger.Debugf("GetStateByPartialCompositKey %v %v %v", k, string(b), err)
	}

	k1 := "11"
	k2 := "30"
	logger.Debugf("by range %v %v", k1, k2)
	iter2, err := tr.GetKeyByRange(k1, k2)
	defer iter2.Close()
	for iter2.HasNext() {
		KV, err := iter2.Next()
		k := KV.GetKey()
		b := KV.GetValue()

		logger.Debugf("GetStateByRange %v %v %v", k, string(b), err)
	}

	test_utils.AssertTrue(t, err == nil, "succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	logger.Debug("-------------new transaction delete keys")

	tr = NewRBTree(stub, "test")

	logger.Debug("adding range keys")
	for i := 22; i < 27; i = i + 1 {
		s := strconv.Itoa(i)
		err := tr.Remove(s)
		logger.Debugf("Remove %v %v", s, err)
	}

	tr.PrintTree("", "")
	//err = tr.SaveToLedger()
	test_utils.AssertTrue(t, err == nil, "succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	logger.Debug("-------------new transaction ")

	tr = NewRBTree(stub, "test")

	k1 = "11"
	k2 = "30"
	logger.Debugf("by range %v %v", k1, k2)
	iter2, err = tr.GetKeyByRange(k1, k2)
	defer iter2.Close()
	for iter2.HasNext() {
		KV, err := iter2.Next()
		k := KV.GetKey()
		b := KV.GetValue()

		logger.Debugf("GetStateByRange %v %v %v", k, string(b), err)
	}

	test_utils.AssertTrue(t, err == nil, "succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	logger.Debug("-------------new transaction remove and insert")

	tr = NewRBTree(stub, "test")
	tr.PrintTree("", "")

	logger.Debug("adding range keys")
	for i := 2; i < 7; i = i + 1 {
		s1 := strconv.Itoa(20 + i - 2)
		s2 := strconv.Itoa(20 + i)
		s3 := strconv.Itoa(20 + i + 7)
		err := tr.Remove(s1)
		logger.Debugf("Remove %v %v", s1, err)
		err = tr.Remove(s3)
		logger.Debugf("Remove %v %v", s3, err)
		//tr.PrintTree("", "")
		v := []byte("value for " + s2)
		err = tr.Insert(s2, v)
		logger.Debugf("Insert %v %v", s2, err)
	}

	tr.PrintTree("", "")
	//err = tr.SaveToLedger()
	test_utils.AssertTrue(t, err == nil, "succeed")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t1")
	stub = cached_stub.NewCachedStub(mstub)
	logger.Debug("-------------new transaction ")

	tr = NewRBTree(stub, "test")

	k1 = "11"
	k2 = "40"
	logger.Debugf("by range %v %v", k1, k2)
	iter2, err = tr.GetKeyByRange(k1, k2)
	defer iter2.Close()
	for iter2.HasNext() {
		KV, err := iter2.Next()
		k := KV.GetKey()
		b := KV.GetValue()

		logger.Debugf("GetStateByRange %v %v %v", k, string(b), err)
	}

	test_utils.AssertTrue(t, err == nil, "succeed")
	mstub.MockTransactionEnd("t1")
}
