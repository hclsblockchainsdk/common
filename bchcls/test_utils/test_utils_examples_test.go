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
	"common/bchcls/data_model"
	"common/bchcls/internal/asset_mgmt_i/asset_mgmt_c/asset_mgmt_g"

	"encoding/json"
	"fmt"
	"testing"
)

func ExampleAssertTrue() {
	var t *testing.T // param passed into every go test function
	a := 1 + 1
	b := 2

	AssertTrue(t, a == b, "Expected a == b")
}

func ExampleAssertFalse() {
	var t *testing.T // param passed into every go test function
	a := 1
	b := 2

	AssertFalse(t, a == b, "Expected a != b")
}

func ExampleAssertInLists() {
	var t *testing.T // param passed into every go test function
	item := "item1"
	list := []string{"item1", "item2", "item3"}

	AssertInLists(t, item, list, "Expected item1 in list")
}

func ExampleAssertListsEqual() {
	var t *testing.T // param passed into every go test function
	list1 := []string{"item1", "item2", "item3"}
	list2 := []string{"item1", "item2", "item3"}

	AssertListsEqual(t, list1, list2)
}

func ExampleAssertSetsEqual() {
	var t *testing.T // param passed into every go test function
	set1 := []string{"item1", "item2", "item3"}
	set2 := []string{"item2", "item1", "item3"}

	AssertListsEqual(t, set1, set2)
}

func ExampleAssertMapsEqual() {
	var t *testing.T // param passed into every go test function
	map1 := make(map[string]string)
	map1["age"] = "40"
	map1["name"] = "Jo"
	map2 := make(map[string]string)
	map2["name"] = "Jo"
	map2["age"] = "40"

	AssertMapsEqual(t, map1, map2, "Expected map1 and map2 to be equal")
}

func ExampleAssertStringInArray() {
	var t *testing.T // param passed into every go test function
	item := "item1"
	array := []string{"item1", "item2", "item3"}

	AssertStringInArray(t, item, array)
}

func ExampleAssertNil() {
	var t *testing.T // param passed into every go test function
	_, err := json.Marshal("data")
	AssertNil(t, err)
}

func ExampleGenerateSymKey() {
	symKey := GenerateSymKey()

	fmt.Println(len(symKey))
	// Output: 32
}

func ExampleCreateSymKey() {
	CreateSymKey("key1")
}

func ExampleGeneratePrivateKey() {
	GeneratePrivateKey()
}

func ExampleGenerateRandomTxID() {
	txID := GenerateRandomTxID()

	fmt.Println(txID)
}

func ExampleCreateTestAssetData() {
	asset := data_model.Asset{}

	asset.PublicData = CreateTestAssetData("test_public_data")
}

func ExampleCreateTestAsset() {
	CreateTestAsset(asset_mgmt_g.GetAssetId("data_model.Assert", "asset1"))
}

func ExampleCreateTestUser() {
	user := CreateTestUser("user1")

	fmt.Println(user.IsGroup)
	fmt.Println(user.Role)
	// Output: false
	// user
}

func ExampleCreateTestGroup() {
	org := CreateTestGroup("org1")

	fmt.Println(org.IsGroup)
	fmt.Println(org.Role)
	// Output: true
	// org
}
