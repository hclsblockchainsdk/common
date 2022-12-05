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
	"common/bchcls/cached_stub"
	"common/bchcls/test_utils"

	"bytes"
	"strconv"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func setup(t *testing.T) *test_utils.NewMockStub {
	logger.SetLevel(shim.LogDebug)
	stub := test_utils.CreateNewMockStub(t)
	return stub
}

func putData(stub shim.ChaincodeStubInterface) {
	logger.Debug("adding test data")
	//composit key
	for _, k1 := range []string{"tom", "jane", "alex"} {
		for _, k2 := range []string{"1", "2", "3", "4"} {
			key, _ := stub.CreateCompositeKey("Test", []string{k1, k2})
			val := []byte("value for " + k1 + k2)
			stub.PutState(key, val)
			//logger.Debugf("PutState %v %v %v", key, string(val), err)
		}
	}

	// range key
	for i := 1; i < 30; i = i + 1 {
		s := strconv.Itoa(i)
		val := []byte("value for " + s)
		stub.PutState(s, val)
		//logger.Debugf("PutState %v %v %v", s, string(val), err)
	}

}

func TestGetState(t *testing.T) {
	logger.Info("TestGetState function called")
	stub := setup(t)
	stub.MockTransactionStart("t123")
	cstub := cached_stub.NewCachedStub(stub)
	putData(stub)
	stub.MockTransactionEnd("t123")

	stub.MockTransactionStart("t123")
	b, err := cstub.GetState("11")
	logger.Debugf("GetState %v %v", "11", string(b))
	test_utils.AssertTrue(t, string(b) == "value for 11", "succeed")

	err = cstub.PutState("11", []byte("new 11 val"))
	b, err = cstub.GetState("11")
	logger.Debugf("GetState %v %v", "11", string(b))
	test_utils.AssertTrue(t, string(b) == "value for 11", "succeed")

	b, err = cstub.GetState("12")
	logger.Debugf("GetState %v %v", "12", string(b))
	test_utils.AssertTrue(t, string(b) == "value for 12", "succeed")

	b, err = cstub.GetState("13")
	logger.Debugf("GetState %v %v", "13", string(b))
	test_utils.AssertTrue(t, string(b) == "value for 13", "succeed")

	test_utils.AssertTrue(t, err == nil, "succeed")
	stub.MockTransactionEnd("t123")

	//after commited
	cstub = cached_stub.NewCachedStub(stub)
	b, err = cstub.GetState("11")
	logger.Debugf("GetState %v %v", "11", string(b))
	test_utils.AssertTrue(t, string(b) == "new 11 val", "succeed")

}

func TestGetStateByRange(t *testing.T) {
	logger.Info("TestGetStateByRange function called")
	stub := setup(t)
	stub.MockTransactionStart("t123")
	cstub := cached_stub.NewCachedStub(stub)
	putData(stub)
	stub.MockTransactionEnd("t123")

	stub.MockTransactionStart("t123")
	cstub = cached_stub.NewCachedStub(stub)

	b, err := cstub.GetState("24")
	logger.Debugf("GetState %v %v", "24", string(b))
	test_utils.AssertTrue(t, string(b) == "value for 24", "succeed")

	k1 := "18"
	k2 := "22"
	expected := []string{"18", "19", "2", "20", "21"}
	result := []string{}
	iter, err := cstub.GetStateByRange(k1, k2)
	test_utils.AssertTrue(t, err == nil, "succeed")
	defer iter.Close()
	for iter.HasNext() {
		KV, err := iter.Next()
		k := KV.GetKey()
		b := KV.GetValue()
		logger.Debugf("GetStateByRange %v %v %v", k, string(b), err)
		result = append(result, k)
	}

	test_utils.AssertListsEqual(t, expected, result)

}

func TestGetStateByRange_advanced(t *testing.T) {
	logger.Info("TestGetStateByRange3 function called")
	stub := setup(t)
	stub.MockTransactionStart("t123")
	cstub := cached_stub.NewCachedStub(stub)
	putData(stub)
	stub.MockTransactionEnd("t123")

	stub.MockTransactionStart("t123")
	cstub = cached_stub.NewCachedStub(stub)

	b, err := cstub.GetState("19")
	logger.Debugf("GetState %v %v", "19", string(b))
	test_utils.AssertTrue(t, string(b) == "value for 19", "succeed")

	b, err = cstub.GetState("24")
	logger.Debugf("GetState %v %v", "24", string(b))
	test_utils.AssertTrue(t, string(b) == "value for 24", "succeed")

	logger.Debug("-------- case 1")
	k1 := "18"
	k2 := "26"
	expected := []string{"18", "19", "2", "20", "21", "22", "23", "24", "25"}
	result := []string{}
	iter, err := cstub.GetStateByRange(k1, k2)
	test_utils.AssertTrue(t, err == nil, "succeed")
	defer iter.Close()
	for iter.HasNext() {
		KV, err := iter.Next()
		k := KV.GetKey()
		b := KV.GetValue()
		logger.Debugf("+++> GetStateByRange %v %v %v", k, string(b), err)
		result = append(result, k)
	}

	test_utils.AssertListsEqual(t, expected, result)

	logger.Debug("-------- case 2")

	k3 := "20"
	k4 := "29"
	expected = []string{"20", "21", "22", "23", "24", "25", "26", "27", "28"}
	result = []string{}
	iter, err = cstub.GetStateByRange(k3, k4)
	test_utils.AssertTrue(t, err == nil, "succeed")
	defer iter.Close()
	for iter.HasNext() {
		KV, err := iter.Next()
		k := KV.GetKey()
		b := KV.GetValue()
		logger.Debugf("+++> GetStateByRange %v %v %v", k, string(b), err)
		result = append(result, k)
	}

	test_utils.AssertListsEqual(t, expected, result)

	logger.Debug("-------- case 3")

	k5 := "19"
	k6 := "27"
	expected = []string{"19", "2", "20", "21", "22", "23", "24", "25", "26"}
	result = []string{}
	iter, err = cstub.GetStateByRange(k5, k6)
	test_utils.AssertTrue(t, err == nil, "succeed")
	defer iter.Close()
	for iter.HasNext() {
		KV, err := iter.Next()
		k := KV.GetKey()
		b := KV.GetValue()
		logger.Debugf("+++> GetStateByRange %v %v %v", k, string(b), err)
		result = append(result, k)
	}

	test_utils.AssertListsEqual(t, expected, result)

	stub.MockTransactionEnd("t123")
}
func TestGetStateByPartialCompositeKey(t *testing.T) {
	logger.Info("TestGetStateByPartialCompositeKey function called")
	stub := setup(t)
	stub.MockTransactionStart("t123")
	cstub := cached_stub.NewCachedStub(stub)
	putData(cstub)
	stub.MockTransactionEnd("t123")

	stub.MockTransactionStart("t123")
	expected := []string{}
	key, _ := cstub.CreateCompositeKey("Test", []string{"jane", "1"})
	expected = append(expected, key)
	key, _ = cstub.CreateCompositeKey("Test", []string{"jane", "2"})
	expected = append(expected, key)
	key, _ = cstub.CreateCompositeKey("Test", []string{"jane", "3"})
	expected = append(expected, key)
	key, _ = cstub.CreateCompositeKey("Test", []string{"jane", "4"})
	expected = append(expected, key)

	result := []string{}
	iter, err := cstub.GetStateByPartialCompositeKey("Test", []string{"jane"})
	test_utils.AssertTrue(t, err == nil, "succeed")
	defer iter.Close()
	for iter.HasNext() {
		KV, err := iter.Next()
		k := KV.GetKey()
		b := KV.GetValue()

		logger.Debugf("GetStateByPartialCompositKey %v %v %v", k, string(b), err)
		result = append(result, k)
	}

	test_utils.AssertListsEqual(t, expected, result)

	stub.MockTransactionEnd("t123")
}

func TestGetStateByPartialCompositeKey_advanced(t *testing.T) {
	logger.Info("TestGetStateByPartialCompositeKey function called")
	stub := setup(t)
	stub.MockTransactionStart("t123")
	cstub := cached_stub.NewCachedStub(stub)
	putData(cstub)
	stub.MockTransactionEnd("t123")

	stub.MockTransactionStart("t123")
	expected := []string{}
	key, _ := cstub.CreateCompositeKey("Test", []string{"jane", "1"})
	expected = append(expected, key)
	key, _ = cstub.CreateCompositeKey("Test", []string{"jane", "2"})
	expected = append(expected, key)
	key, _ = cstub.CreateCompositeKey("Test", []string{"jane", "3"})
	expected = append(expected, key)
	key, _ = cstub.CreateCompositeKey("Test", []string{"jane", "4"})
	expected = append(expected, key)

	result := []string{}
	iter, err := cstub.GetStateByPartialCompositeKey("Test", []string{"jane"})
	test_utils.AssertTrue(t, err == nil, "succeed")
	defer iter.Close()
	for iter.HasNext() {
		KV, err := iter.Next()
		k := KV.GetKey()
		b := KV.GetValue()

		logger.Debugf("GetStateByPartialCompositKey %v %v %v", k, string(b), err)
		result = append(result, k)
	}

	test_utils.AssertListsEqual(t, expected, result)

	logger.Debug("---- second time; should get it from cache")

	result = []string{}
	iter, err = cstub.GetStateByPartialCompositeKey("Test", []string{"jane"})
	test_utils.AssertTrue(t, err == nil, "succeed")
	defer iter.Close()
	for iter.HasNext() {
		KV, err := iter.Next()
		k := KV.GetKey()
		b := KV.GetValue()

		logger.Debugf("GetStateByPartialCompositKey %v %v %v", k, string(b), err)
		result = append(result, k)
	}

	test_utils.AssertListsEqual(t, expected, result)

	stub.MockTransactionEnd("t123")
}

func TestCacheWithStruct(t *testing.T) {
	logger.Info("TestCache function called")
	stub := setup(t)

	type Data struct {
		Field1 []byte
		Field2 string
		Field3 string
	}

	stub.MockTransactionStart("t123")
	cstub := cached_stub.NewCachedStub(stub)
	data1 := Data{Field1: []byte{1}, Field2: "b", Field3: "c"}
	logger.Debugf("PutCache cache1 - original data: %v", data1)
	err := cstub.PutCache("cache1", data1)
	test_utils.AssertTrue(t, err == nil, "succeed")

	cachedData1, err := cstub.GetCache("cache1")
	test_utils.AssertTrue(t, err == nil, "succeed")
	data2, ok := cachedData1.(Data)
	test_utils.AssertTrue(t, ok, "succeed")
	logger.Debugf("GetCache %v as data2", "cache1")
	logger.Debugf("from cached data2: %v", data2)
	test_utils.AssertTrue(t, (data1.Field1[0] == data2.Field1[0] && data1.Field2 == data2.Field2 && data1.Field3 == data2.Field3), "data from cache should be same as original data")

	// this is to show that cache value can be effected if you change the internal
	// value of the object
	data2.Field1[0] = 10
	logger.Debugf("Changin Field1 value of data2")
	logger.Debugf("original data1: %v", data1)
	logger.Debugf("changed data2: %v", data2)
	test_utils.AssertTrue(t, (data1.Field1[0] == data2.Field1[0] && data1.Field2 == data2.Field2 && data1.Field3 == data2.Field3), "chainging data from cache should change original data")

	// this is to show that if you are not changing the internal value of the object,
	// cache is not affected
	data1.Field2 = "new b"
	logger.Debugf("Changin Field1 value of original data1")
	logger.Debugf("original data1: %v", data1)

	cachedData1, err = cstub.GetCache("cache1")
	test_utils.AssertTrue(t, err == nil, "succeed")
	data3, ok := cachedData1.(Data)
	test_utils.AssertTrue(t, ok, "succeed")
	logger.Debugf("GetCache %v as data3", "cache1")
	logger.Debugf("original data1: %v", data1)
	logger.Debugf("from cached data3: %v", data3)
	test_utils.AssertTrue(t, (data3.Field1[0] == 10 && data3.Field2 == "b" && data3.Field3 == "c"), "chaining original data after PutCache, should not change value of GetCache")

	// this is to show that if you are chainging both internal value []byte
	// and non-internal value of the object, cache is affected only for
	// the internal value affected
	data1.Field2 = "new b"
	data1.Field1[0] = 20
	logger.Debugf("Changin Field1 value of original data1")
	logger.Debugf("original data1: %v", data1)

	cachedData1, err = cstub.GetCache("cache1")
	test_utils.AssertTrue(t, err == nil, "succeed")
	data3, ok = cachedData1.(Data)
	test_utils.AssertTrue(t, ok, "succeed")
	logger.Debugf("GetCache %v as data3", "cache1")
	logger.Debugf("original data1: %v", data1)
	logger.Debugf("from cached data3: %v", data3)
	test_utils.AssertTrue(t, (data3.Field1[0] == 20 && data3.Field2 == "b" && data3.Field3 == "c"), "chaining original data after PutCache, should change value of GetCache")
	// delete cache
	logger.Debug("DelCache cache1")
	err = cstub.DelCache("cache1")
	test_utils.AssertTrue(t, err == nil, "succeed")

	cachedData1, err = cstub.GetCache("cache1")
	logger.Debugf("GetCache cache1 %v", cachedData1)
	test_utils.AssertTrue(t, err == nil, "succeed")
	test_utils.AssertTrue(t, cachedData1 == nil, "shuld be nil")
	data2, ok = cachedData1.(Data)
	test_utils.AssertTrue(t, !ok, "should fail")

	stub.MockTransactionEnd("t123")

}

func TestCacheWithByteArray(t *testing.T) {
	logger.Info("TestCache function called")
	stub := setup(t)

	stub.MockTransactionStart("t123")
	cstub := cached_stub.NewCachedStub(stub)
	data1 := []byte{1, 2, 3}
	logger.Debugf("PutCache cache1 - original data: %v", data1)
	err := cstub.PutCache("cache1", data1)
	test_utils.AssertTrue(t, err == nil, "succeed")

	cachedData1, err := cstub.GetCache("cache1")
	test_utils.AssertTrue(t, err == nil, "succeed")
	data2, ok := cachedData1.([]byte)
	test_utils.AssertTrue(t, ok, "succeed")
	logger.Debugf("GetCache %v as data2", "cache1")
	logger.Debugf("from cached data2: %v", data2)
	test_utils.AssertTrue(t, bytes.Equal(data1, data2), "data from cache should be same as original data")

	// changing internal value of []byte, can affect cache, and original data
	data2[0] = 10
	logger.Debugf("Changin Field1 value of data2")
	logger.Debugf("original data1: %v", data1)
	logger.Debugf("changed data2: %v", data2)
	test_utils.AssertTrue(t, (data1[0] == data2[0] && data1[1] == data2[1] && data1[2] == data2[2]), "chainging data from cache should change original data")

	// changing internal value of []byte, changes the cache value too
	data1[0] = 11
	logger.Debugf("Changin Field1 value of original data1")
	logger.Debugf("original data1: %v", data1)

	cachedData1, err = cstub.GetCache("cache1")
	test_utils.AssertTrue(t, err == nil, "succeed")
	data3, ok := cachedData1.([]byte)
	test_utils.AssertTrue(t, ok, "succeed")
	logger.Debugf("GetCache %v as data3", "cache1")
	logger.Debugf("original data1: %v", data1)
	logger.Debugf("from cached data3: %v", data3)
	test_utils.AssertTrue(t, (data3[0] == 11 && data3[1] == 2 && data3[2] == 3), "chaining original data after PutCache, should not change value of GetCache")

	// delete cache
	logger.Debug("DelCache cache1")
	err = cstub.DelCache("cache1")
	test_utils.AssertTrue(t, err == nil, "succeed")

	cachedData1, err = cstub.GetCache("cache1")
	logger.Debugf("GetCache cache1 %v", cachedData1)
	test_utils.AssertTrue(t, err == nil, "succeed")
	test_utils.AssertTrue(t, cachedData1 == nil, "shuld be nil")
	data2, ok = cachedData1.([]byte)
	test_utils.AssertTrue(t, !ok, "should fail")

	stub.MockTransactionEnd("t123")

}
