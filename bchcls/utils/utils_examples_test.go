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

	"fmt"
	"strings"
)

func ExampleEnterFnLog() {
	functionName := EnterFnLog()

	fmt.Println(functionName)
	// Output: ExampleEnterFnLog
}

func ExampleExitFnLog() {
	defer ExitFnLog(EnterFnLog())
}

func ExampleCheckParam() {
	args := []string{"arg1", "arg2", "arg3"}

	param1 := "arg1"
	includesParam1 := CheckParam(args, param1)
	fmt.Println(includesParam1)

	param2 := "X"
	includesParam2 := CheckParam(args, param2)
	fmt.Println(includesParam2)

	// Output: true
	// false
}

func ExampleInList() {
	list := []string{"item1", "item2", "item3"}

	item1 := "item1"
	inList1 := InList(list, item1)
	fmt.Println(inList1)

	item2 := "X"
	inList2 := InList(list, item2)
	fmt.Println(inList2)

	// Output: true
	// false
}

func ExampleGetDataList() {
	m := make(map[string]bool)
	m["a"] = true
	m["b"] = true
	m["x"] = true
	m["l"] = true

	list := GetDataList(m)

	fmt.Println(list)
	// Output: [a b l x]
}

func ExampleFloat64ToPaddedString() {
	paddedStringPos, _ := Float64ToPaddedString(5.1234)
	fmt.Println(paddedStringPos)

	paddedStringNeg, _ := Float64ToPaddedString(-5.1234)
	fmt.Println(paddedStringNeg)

	// Output: 1000000000005.1234
	// 0999999999994.8766
}

func ExampleRemoveItemFromList() {
	list := []string{"item1", "item2", "item3"}

	newList1 := RemoveItemFromList(list, "item1")

	fmt.Println(newList1)
	// Output: [item2 item3]
}

func ExampleIsStringEmpty() {
	string1 := "string1"
	isEmpty1 := IsStringEmpty(string1)
	fmt.Println(isEmpty1)

	string2 := ""
	isEmpty2 := IsStringEmpty(string2)
	fmt.Println(isEmpty2)

	var string3 string
	isEmpty3 := IsStringEmpty(string3)
	fmt.Println(isEmpty3)

	bool1 := false
	isEmpty4 := IsStringEmpty(bool1)
	fmt.Println(isEmpty4)

	// Output: false
	// true
	// true
	// true
}

func ExampleIsInstanceOf() {
	customErr := &custom_errors.MarshalError{Type: "object"}

	isInstanceOf1 := IsInstanceOf(customErr, &custom_errors.MarshalError{})
	fmt.Println(isInstanceOf1)

	isInstanceOf2 := IsInstanceOf(customErr, &custom_errors.UnmarshalError{})
	fmt.Println(isInstanceOf2)

	// Output: true
	// false
}

func ExampleMap() {
	list := []string{"item1", "item2", "item3"}

	listNew := Map(list, strings.Title)

	fmt.Println(listNew)
	// Output: [Item1 Item2 Item3]
}

func ExampleGetSetFromList() {
	list := []string{"item1", "item3", "item2", "item1"}

	set := GetSetFromList(list)

	fmt.Println(set)
	// Output: [item1 item2 item3]
}

func ExampleGetSet() {
	listStr := "item1, item2, item3, item3"

	set := GetSet(listStr)

	fmt.Println(set)
	// Output: [item1 item2 item3]
}

func ExampleAddToSet() {
	set := []string{"item1", "item2", "item3"}

	setNew := AddToSet(set, "item4")

	fmt.Println(setNew)
	// Output: [item1 item2 item3 item4]
}

func ExampleRemoveFromSet() {
	set := []string{"item1", "item2", "item3", "item4"}

	setNew := RemoveFromSet(set, "item4")

	fmt.Println(setNew)
	// Output: [item1 item2 item3]
}

func ExampleFilterSet() {
	list := []string{"item1", "item2", "item3", "item4"}
	filterList := []string{"item1", "item2", "item3"}

	set := FilterSet(list, filterList)

	fmt.Println(set)
	// Output: [item1 item2 item3]
}

func ExampleFilterOutFromSet() {
	list := []string{"item1", "item2", "item3", "item4"}
	filterList := []string{"item1", "item2", "item3"}

	set := FilterOutFromSet(list, filterList)

	fmt.Println(set)
	// Output: [item4]
}

func ExampleGetDataMap() {
	data1 := []string{"item1", "item2", "item3"}
	GetDataMap(data1, nil)
	// map[item1:true item2:true item3:true]

	data2 := []string{"item1:read", "item1:read", "item2:read"}
	GetDataMap(data2, nil)
	// map[item1:read:true item1:write:true item2:read:true]

	access := []string{"read", "write"}
	GetDataMap(data1, access)
	// map[item1:read:true item1:write:true item2:read:true item2:write:true item3:read:true item3:write:true ]
}

func ExampleNormalizeDataList() {
	list := []string{"item1", "item2", "item3", "item3"}

	normalizedList := NormalizeDataList(list)

	fmt.Println(normalizedList)
	// Output: [item1 item2 item3]
}

func ExampleFilterDataList() {
	list := []string{"key1:val1", "key2:val2", "key3:val3", "key4:val4"}
	filterList := []string{"val1", "val2", "val3"}

	listNew := FilterDataList(list, filterList)

	fmt.Println(listNew)
	// Output: [key1:val1 key2:val2 key3:val3]
}

func ExampleInsertInt() {
	list := []int{1, 2, 4}

	listNew := InsertInt(list, 3)

	fmt.Println(listNew)
	// Output: [1 2 3 4]
}

func ExampleRemoveInt() {
	list := []int{1, 2, 3, 4}

	listNew := RemoveInt(list, 4)

	fmt.Println(listNew)
	// Output: [1 2 3]
}

func ExampleInsertString() {
	list := []string{"a", "c", "d"}

	listNew := InsertString(list, "b")

	fmt.Println(listNew)
	// Output: [a b c d]
}

func ExampleRemoveString() {
	list := []string{"a", "b", "c", "d"}

	listNew := RemoveString(list, "d")

	fmt.Println(listNew)
	// Output: [a b c]
}

func ExamplePerformHTTP() {
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	data := `{"id": "myID", "data": "my data"}`
	PerformHTTP("GET", "https://www.example.com/", []byte(data), headers, []byte("user"), []byte("password"))
}

func ExampleTimestampToDateString() {
	timestamp := int64(1234567890)

	dateString := TimestampToDateString(timestamp)

	fmt.Println(dateString)
	// Output: 2009-02-13 18:31:30 -0500 EST
}

func ExampleDateStringToTimestamp() {
	dateString := "2009-02-13 18:31:30 -0500 EST"

	timestamp, _ := DateStringToTimestamp(dateString)

	fmt.Println(timestamp)
	// Output: 1234567890
}

func ExampleToQueryString() {
	queryMap := make(map[string][]string)
	queryMap["provider"] = []string{"provider1"}
	queryMap["patient"] = []string{"patient1"}

	queryString := ToQueryString(queryMap)

	fmt.Println(queryString)
	// Output: patient=patient1&provider=provider1
}
