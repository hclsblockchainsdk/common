/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package utils

import (
	"common/bchcls/custom_errors"
	"common/bchcls/test_utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"

	"testing"
)

func TestUtils(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestUtils")

	// Create the mock stub
	stub := test_utils.CreateNewMockStub(t)
	stub.MockTransactionStart("t123")

	// your test here

	stub.MockTransactionEnd("t123")
	test_utils.AssertTrue(t, true, "ok")

}

func TestCheckParam(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestCheckParam")

	args := []string{"arg1", "arg2", "arg3"}
	param1 := "arg1"
	param2 := "X"
	inArgs := CheckParam(args, param1)
	test_utils.AssertTrue(t, inArgs, "Expected CheckParam to return true")
	inArgs = CheckParam(args, param2)
	test_utils.AssertTrue(t, !inArgs, "Expected CheckParam to return true")
}

func TestInList(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestInList")

	list := []string{"item1", "item2", "item3"}
	item1 := "item1"
	item2 := "X"
	inList := CheckParam(list, item1)
	test_utils.AssertTrue(t, inList, "Expected InList to return true")
	inList = CheckParam(list, item2)
	test_utils.AssertTrue(t, !inList, "Expected InList to return true")
}

func TestGetDataList(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetDataList")

	m := make(map[string]bool)
	m["a"] = true
	m["b"] = true
	m["x"] = true
	m["l"] = true
	list := GetDataList(m)
	expectedList := []string{"a", "b", "l", "x"}
	test_utils.AssertListsEqual(t, expectedList, list)
}

func TestFloat64ToPaddedString(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestFloat64ToPaddedString")

	// Test boundaries
	paddedStr, err := Float64ToPaddedString(-MaxFloat64ToPaddedString)
	test_utils.AssertTrue(t, err != nil, "Expected Float64ToPaddedString to return an error")
	test_utils.AssertTrue(t, len(paddedStr) == 0, "Expected Float64ToPaddedString to return an empty string")
	paddedStr, err = Float64ToPaddedString(MaxFloat64ToPaddedString)
	test_utils.AssertTrue(t, err != nil, "Expected Float64ToPaddedString to return an error")
	test_utils.AssertTrue(t, len(paddedStr) == 0, "Expected Float64ToPaddedString to return an empty string")

	// Test negative numbers
	paddedStr, err = Float64ToPaddedString(-MaxFloat64ToPaddedString + .0001)
	test_utils.AssertTrue(t, err == nil, "Expected Float64ToPaddedString to succeed")
	test_utils.AssertTrue(t, paddedStr == "0000000000000.0001", "Expected Float64ToPaddedString to return a different string")
	paddedStr, err = Float64ToPaddedString(-5.1234)
	test_utils.AssertTrue(t, err == nil, "Expected Float64ToPaddedString to succeed")
	test_utils.AssertTrue(t, paddedStr == "0999999999994.8766", "Expected Float64ToPaddedString to return a different string")
	paddedStr, err = Float64ToPaddedString(-.0001)
	test_utils.AssertTrue(t, err == nil, "Expected Float64ToPaddedString to succeed")
	test_utils.AssertTrue(t, paddedStr == "0999999999999.9999", "Expected Float64ToPaddedString to return a different string")

	// Test 0
	paddedStr, err = Float64ToPaddedString(0)
	test_utils.AssertTrue(t, err == nil, "Expected Float64ToPaddedString to succeed")
	test_utils.AssertTrue(t, paddedStr == "1000000000000.0000", "Expected Float64ToPaddedString to return a different string")

	// Test positive numbers
	paddedStr, err = Float64ToPaddedString(.0001)
	test_utils.AssertTrue(t, err == nil, "Expected Float64ToPaddedString to succeed")
	test_utils.AssertTrue(t, paddedStr == "1000000000000.0001", "Expected Float64ToPaddedString to return a different string")
	paddedStr, err = Float64ToPaddedString(5.1234)
	test_utils.AssertTrue(t, err == nil, "Expected Float64ToPaddedString to succeed")
	test_utils.AssertTrue(t, paddedStr == "1000000000005.1234", "Expected Float64ToPaddedString to return a different string")
	paddedStr, err = Float64ToPaddedString(MaxFloat64ToPaddedString - .001)
	test_utils.AssertTrue(t, err == nil, "Expected Float64ToPaddedString to succeed")
	test_utils.AssertTrue(t, paddedStr == "1999999999999.9990", "Expected Float64ToPaddedString to return a different string")
}

func TestRemoveItemFromList(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestRemoveItemFromList")

	list := []string{"item1", "item2", "item3", "item4", "item4"}

	expectedList1 := []string{"item2", "item3", "item4", "item4"}
	list = RemoveItemFromList(list, "item1")
	test_utils.AssertListsEqual(t, expectedList1, list)

	expectedList2 := []string{"item2", "item3", "item4"}
	list = RemoveItemFromList(list, "item4")
	test_utils.AssertListsEqual(t, expectedList2, list)

	list = RemoveItemFromList(list, "X")
	test_utils.AssertListsEqual(t, expectedList2, list)
}

func TestIsStringEmpty(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestIsStringEmpty")

	string1 := "string1"
	empty1 := IsStringEmpty(string1)
	test_utils.AssertFalse(t, empty1, "Expected IsStringEmpty to be false")

	string2 := ""
	empty2 := IsStringEmpty(string2)
	test_utils.AssertTrue(t, empty2, "Expected IsStringEmpty to be true")

	var string3 string
	empty3 := IsStringEmpty(string3)
	test_utils.AssertTrue(t, empty3, "Expected IsStringEmpty to be true")

	bool1 := false
	empty4 := IsStringEmpty(bool1)
	test_utils.AssertTrue(t, empty4, "Expected IsStringEmpty to be true (wrong type)")
}

func TestIsInstanceOf(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestIsInstanceOf")

	customErr := &custom_errors.MarshalError{Type: "object"}

	isInstanceOf := IsInstanceOf(customErr, &custom_errors.MarshalError{})
	test_utils.AssertTrue(t, isInstanceOf, "Expected customErr to be instance of MarshalError")

	isInstanceOf = IsInstanceOf(customErr, &custom_errors.UnmarshalError{})
	test_utils.AssertFalse(t, isInstanceOf, "Expected customErr to not be instance of UnmarshalError")
}
