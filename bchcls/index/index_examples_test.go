/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package index

import (
	"common/bchcls/cached_stub"
	"common/bchcls/internal/common/global"
	"common/bchcls/test_utils"
)

func ExampleGetTable() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	GetTable(stub, "CustomAssetIndex", "objectId")
}

func ExampleTable_HasIndex() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	indexTable := GetTable(stub, "CustomAssetIndex", "objectId")

	indexTable.HasIndex([]string{"color", "objectId"})
}

func ExampleTable_GetIndexedFields() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	indexTable := GetTable(stub, "CustomAssetIndex", "objectId")

	indexTable.GetIndexedFields()
}

func ExampleTable_GetPrimaryKeyId() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	indexTable := GetTable(stub, "CustomAssetIndex", "objectId")

	// returns "objectId"
	indexTable.GetPrimaryKeyId()
}

func ExampleTable_AddIndex() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	indexTable := GetTable(stub, "CustomAssetIndex", "objectId")

	indexTable.AddIndex([]string{"color", "objectId"}, false)
}

func ExampleTable_UpdateAllRows() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	indexTable := GetTable(stub, "CustomAssetIndex", "objectId")

	indexTable.UpdateAllRows()
}

func ExampleTable_UpdateRow() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	indexTable := GetTable(stub, "CustomAssetIndex", "objectId")

	row := make(map[string]string)
	row["color"] = "blue"
	row["objectId"] = "object1"

	indexTable.UpdateRow(row)
}

func ExampleTable_DeleteRow() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	indexTable := GetTable(stub, "CustomAssetIndex", "objectId")

	indexTable.DeleteRow("object1")
}

func ExampleTable_GetRow() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	indexTable := GetTable(stub, "CustomAssetIndex", "objectId")

	indexTable.GetRow("object1")
}

func ExampleTable_CreateRangeKey() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	indexTable := GetTable(stub, "CustomAssetIndex", "objectId")

	indexTable.CreateRangeKey([]string{"color", "objectId"}, []string{"blue"})
}

func ExampleTable_GetRowsByRange() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	indexTable := GetTable(stub, "CustomAssetIndex")

	// Given the following ledger keys:
	// 	Index-marbles-color-id_blue_marble1_
	//	Index-marbles-color-id_green_marble2_
	//	Index-marbles-color-id_red_marble3_

	// Filter on "blue"
	// range("Index-marbles-color-id_blue_", "Index-marbles-color-id_blue_*") -> blue
	startKey, _ := indexTable.CreateRangeKey([]string{"color", "id"}, []string{"blue"})
	endKey, _ := indexTable.CreateRangeKey([]string{"color", "id"}, []string{"blue"})
	endKey = endKey + string(global.MAX_UNICODE_RUNE_VALUE)
	indexTable.GetRowsByRange(startKey, endKey)

	// Range starting with "blue"
	// range("Index-marbles-color-id_blue_", "Index-marbles-color-id_*") -> blue, green, red
	startKey, _ = indexTable.CreateRangeKey([]string{"color", "id"}, []string{"blue"})
	endKey, _ = indexTable.CreateRangeKey([]string{"color", "id"}, []string{})
	endKey = endKey + string(global.MAX_UNICODE_RUNE_VALUE)
	indexTable.GetRowsByRange(startKey, endKey)

	// Range starting with "green" (inclusive) and ending with "red" (exclusive)
	// range("Index-marbles-color-id_green_", "Index-marbles-color-id_red_") -> green
	startKey, _ = indexTable.CreateRangeKey([]string{"color", "id"}, []string{"green"})
	endKey, _ = indexTable.CreateRangeKey([]string{"color", "id"}, []string{"red"})
	indexTable.GetRowsByRange(startKey, endKey)

	// Range ending with "red" (exclusive)
	// range("Index-marbles-color-id_", "Index-marbles-color-id_red_") -> blue, green
	startKey, _ = indexTable.CreateRangeKey([]string{"color", "id"}, []string{})
	endKey, _ = indexTable.CreateRangeKey([]string{"color", "id"}, []string{"red"})
	indexTable.GetRowsByRange(startKey, endKey)
}

func ExampleTable_GetRowsByPartialKey() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	indexTable := GetTable(stub, "CustomAssetIndex", "objectId")

	indexTable.GetRowsByPartialKey([]string{"color"}, []string{"blue"})
}

func ExampleTable_SaveToLedger() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	indexTable := GetTable(stub, "CustomAssetIndex", "objectId")

	indexTable.SaveToLedger()
}

func ExampleGetPrettyLedgerKey() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())
	indexTable := GetTable(stub, "CustomAssetIndex", "objectId")

	ledgerKey, _ := indexTable.CreateRangeKey([]string{"color", "objectId"}, []string{"blue"})

	GetPrettyLedgerKey(ledgerKey)
}
