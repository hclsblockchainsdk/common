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
	"testing"
)

func ExampleCreateMisbehavingMockStub() {
	var t *testing.T // param passed into every go test function

	CreateMisbehavingMockStub(t)
}

func ExampleMisbehavingMockStub_GetState() {
	var t *testing.T // param passed into every go test function
	misbehavingStub := CreateMisbehavingMockStub(t)

	_, err := misbehavingStub.GetState("ledger_key")

	fmt.Println(err)
	// Output: Misbehaving stub error!
}

func ExampleMisbehavingMockStub_GetStateByPartialCompositeKey() {
	var t *testing.T // param passed into every go test function
	misbehavingStub := CreateMisbehavingMockStub(t)

	_, err := misbehavingStub.GetStateByPartialCompositeKey("ledger_key", []string{})

	fmt.Println(err)
	// Output: Misbehaving stub error!
}

func ExampleMisbehavingMockStub_PutState() {
	var t *testing.T // param passed into every go test function
	misbehavingStub := CreateMisbehavingMockStub(t)

	err := misbehavingStub.PutState("ledger_key", []byte{})

	fmt.Println(err)
	// Output: Misbehaving stub error!
}

func ExampleMisbehavingMockStub_DelState() {
	var t *testing.T // param passed into every go test function
	misbehavingStub := CreateMisbehavingMockStub(t)

	err := misbehavingStub.DelState("ledger_key")

	fmt.Println(err)
	// Output: Misbehaving stub error!
}

func ExampleCreateNewMockStub() {
	var t *testing.T // param passed into every go test function

	CreateNewMockStub(t)
}

func ExampleNewMockStub_GetState() {
	var t *testing.T // param passed into every go test function
	stub := CreateNewMockStub(t)

	stub.GetState("ledger_key")
}

func ExampleNewMockStub_GetStateByRange() {
	var t *testing.T // param passed into every go test function
	stub := CreateNewMockStub(t)
	startKey := "a" //start ledger key value
	endKey := "b"   //end ledger key valuye

	stub.GetStateByRange(startKey, endKey)
}

func ExampleNewFixedMockStateRangeQueryIterator() {
	var t *testing.T // param passed into every go test function
	stub := CreateNewMockStub(t)
	startKey := "a" //start ledger key value
	endKey := "b"   //end ledger key valuye

	NewFixedMockStateRangeQueryIterator(stub, startKey, endKey)
}

func ExampleCreateExampleMockStub() {
	CreateExampleMockStub()
}
