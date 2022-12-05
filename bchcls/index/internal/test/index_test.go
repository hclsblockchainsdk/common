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
	"common/bchcls/datastore"
	"common/bchcls/index"
	"common/bchcls/init_common"
	"common/bchcls/test_utils"
	"common/bchcls/utils"

	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func setup(t *testing.T) *test_utils.NewMockStub {
	logger.SetLevel(shim.LogDebug)
	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)
	init_common.Init(stub, shim.LogDebug)
	return mstub
}

func TestIndex(t *testing.T) {
	logger.Info("TestIndex function called")

	// create a MockStub
	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)

	//create table and add index
	table := index.GetTable(stub, "TestIndex", "id")
	//key, err = s.CreateCompositeKey("Table", []string{"TestIndex"})

	table.AddIndex([]string{"company", "dept", "id"}, false)
	table.AddIndex([]string{"dept", "company", "id"}, false)
	table.SaveToLedger()
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t2")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	row := make(map[string]string)

	row["company"] = "Com A"
	row["dept"] = "D1"
	row["id"] = "AD1"
	table.UpdateRow(row)
	row["company"] = "Com A"
	row["dept"] = "D2"
	row["id"] = "AD2"
	table.UpdateRow(row)
	row["company"] = "Com A"
	row["dept"] = "D3"
	row["id"] = "AD3"
	table.UpdateRow(row)

	row["company"] = "Com B"
	row["dept"] = "D1"
	row["id"] = "BD1"
	table.UpdateRow(row)
	row["company"] = "Com B"
	row["dept"] = "D3"
	row["id"] = "BD3"
	table.UpdateRow(row)

	row["company"] = "Com C"
	row["dept"] = "D2"
	row["id"] = "CD2"
	table.UpdateRow(row)
	row["company"] = "Com C"
	row["dept"] = "D3"
	row["id"] = "CD3"
	table.UpdateRow(row)

	row["company"] = "Com D"
	row["dept"] = "D1"
	row["id"] = "DD1"
	table.UpdateRow(row)
	row["company"] = "Com D"
	row["dept"] = "D2"
	row["id"] = "DD2"
	table.UpdateRow(row)
	row["company"] = "Com D"
	row["dept"] = "D4"
	row["id"] = "DD4"
	table.UpdateRow(row)

	row = make(map[string]string)
	mstub.MockTransactionEnd("t2")

	mstub.MockTransactionStart("t3")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	logger.Debugf("sort by company/dept")
	iter, err := table.GetRowsByPartialKey([]string{"company"}, []string{"Com D"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	defer iter.Close()
	actualList := []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		err = json.Unmarshal(rowBytes, &row)

		actualList = append(actualList, row["id"])
		logger.Infof("=== row %v", row)
	}
	test_utils.AssertTrue(t, len(actualList) == 3, "expect 3, got "+strconv.Itoa(len(actualList)))
	test_utils.AssertListsEqual(t, []string{"DD1", "DD2", "DD4"}, actualList)
	mstub.MockTransactionEnd("t3")

	mstub.MockTransactionStart("t4")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	logger.Debugf("sort by dept/company")
	iter, err = table.GetRowsByPartialKey([]string{"dept"}, []string{"D1"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	defer iter.Close()
	actualList = []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		err = json.Unmarshal(rowBytes, &row)

		actualList = append(actualList, row["id"])
		logger.Infof("=== row %v", row)
	}
	test_utils.AssertTrue(t, len(actualList) == 3, "expect 3, got "+strconv.Itoa(len(actualList)))
	test_utils.AssertListsEqual(t, []string{"AD1", "BD1", "DD1"}, actualList)
	mstub.MockTransactionEnd("t4")

	//unhappy path
	mstub.MockTransactionStart("t5")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	logger.Debugf("sort by dept/company")
	iter, err = table.GetRowsByPartialKey([]string{"dept"}, []string{"X1"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	test_utils.AssertTrue(t, err == nil, "succeed")
	defer iter.Close()
	actualList = []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		err = json.Unmarshal(rowBytes, &row)

		actualList = append(actualList, row["id"])
		logger.Infof("=== row %v", row)
	}
	test_utils.AssertTrue(t, len(actualList) == 0, "expect 0, got "+strconv.Itoa(len(actualList)))
	mstub.MockTransactionEnd("t5")
}

func TestIndexEncrypted(t *testing.T) {
	logger.Info("TestIndexEncrypted function called")

	// create a MockStub
	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)

	//create table and add index
	table := index.GetTable(stub, "TestIndex", "id", false, true)
	//key, err = s.CreateCompositeKey("Table", []string{"TestIndex"})

	table.AddIndex([]string{"company", "dept", "id"}, false)
	table.AddIndex([]string{"dept", "company", "id"}, false)
	table.SaveToLedger()
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t2")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	row := make(map[string]string)

	row["company"] = "Com A"
	row["dept"] = "D1"
	row["id"] = "AD1"
	table.UpdateRow(row)
	row["company"] = "Com A"
	row["dept"] = "D2"
	row["id"] = "AD2"
	table.UpdateRow(row)
	row["company"] = "Com A"
	row["dept"] = "D3"
	row["id"] = "AD3"
	table.UpdateRow(row)

	row["company"] = "Com B"
	row["dept"] = "D1"
	row["id"] = "BD1"
	table.UpdateRow(row)
	row["company"] = "Com B"
	row["dept"] = "D3"
	row["id"] = "BD3"
	table.UpdateRow(row)

	row["company"] = "Com C"
	row["dept"] = "D2"
	row["id"] = "CD2"
	table.UpdateRow(row)
	row["company"] = "Com C"
	row["dept"] = "D3"
	row["id"] = "CD3"
	table.UpdateRow(row)

	row["company"] = "Com D"
	row["dept"] = "D1"
	row["id"] = "DD1"
	table.UpdateRow(row)
	row["company"] = "Com D"
	row["dept"] = "D2"
	row["id"] = "DD2"
	table.UpdateRow(row)
	row["company"] = "Com D"
	row["dept"] = "D4"
	row["id"] = "DD4"
	table.UpdateRow(row)
	row = make(map[string]string)

	mstub.MockTransactionEnd("t2")

	mstub.MockTransactionStart("t3")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	logger.Debugf("sort by company/dept")
	iter, err := table.GetRowsByPartialKey([]string{"company"}, []string{"Com D"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	defer iter.Close()
	actualList := []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		logger.Debugf("rowByes: %v", string(rowBytes[:]))
		err = json.Unmarshal(rowBytes, &row)

		logger.Infof("=== row %v %v", row, err)
		actualList = append(actualList, row["id"])
	}
	test_utils.AssertTrue(t, len(actualList) == 3, "expect 3, got "+strconv.Itoa(len(actualList)))
	test_utils.AssertSetsEqual(t, []string{"DD1", "DD2", "DD4"}, actualList)
	mstub.MockTransactionEnd("t=3")

	mstub.MockTransactionStart("t4")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	logger.Debugf("sort by dept/company")
	iter, err = table.GetRowsByPartialKey([]string{"dept"}, []string{"D1"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	defer iter.Close()
	actualList = []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		err = json.Unmarshal(rowBytes, &row)

		logger.Infof("=== row %v", row)
		actualList = append(actualList, row["id"])
	}
	test_utils.AssertTrue(t, len(actualList) == 3, "expect 3, got "+strconv.Itoa(len(actualList)))
	test_utils.AssertSetsEqual(t, []string{"AD1", "BD1", "DD1"}, actualList)
	mstub.MockTransactionEnd("t4")

	//unhappy path
	mstub.MockTransactionStart("t5")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	logger.Debugf("sort by dept/company")
	iter, err = table.GetRowsByPartialKey([]string{"dept"}, []string{"X1"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	test_utils.AssertTrue(t, err == nil, "succeed")
	defer iter.Close()
	actualList = []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		err = json.Unmarshal(rowBytes, &row)

		actualList = append(actualList, row["id"])
		logger.Infof("=== row %v", row)
	}
	test_utils.AssertTrue(t, len(actualList) == 0, "expect 0, got "+strconv.Itoa(len(actualList)))
	mstub.MockTransactionEnd("t5")
}

func TestIndexDatastore(t *testing.T) {
	logger.Info("TestIndexDatastore function called")

	// create a MockStub
	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	init_common.Init(stub, shim.LogDebug)

	// change the following Cloudant config to connect to a custom Cloudant and set environment variables:
	// CLOUDANT_USERNAME
	// CLOUDANT_PASSWORD
	// CLOUDANT_DATABASE
	// CLOUDANT_HOST
	username := "admin"
	password := "pass"
	database := "test"
	host := "http://127.0.0.1:9080"
	// Get values from environment variables
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_USERNAME")) {
		username = os.Getenv("CLOUDANT_USERNAME")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_PASSWORD")) {
		password = os.Getenv("CLOUDANT_PASSWORD")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_DATABASE")) {
		database = os.Getenv("CLOUDANT_DATABASE")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_HOST")) {
		host = os.Getenv("CLOUDANT_HOST")
	}

	_, err := init_common.InitDatastore(stub, username, password, database, host)
	test_utils.AssertTrue(t, err == nil, "Error Getting Cloudant datastore: Make sure to start cloudant docker before running this test")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t2")
	stub = cached_stub.NewCachedStub(mstub)

	//create table and add index
	table := index.GetTable(stub, "TestIndex", "id", false, false, datastore.DEFAULT_CLOUDANT_DATASTORE_ID)
	//key, err = s.CreateCompositeKey("Table", []string{"TestIndex"})

	table.AddIndex([]string{"company", "dept", "id"}, false)
	table.AddIndex([]string{"dept", "company", "id"}, false)
	table.SaveToLedger()
	mstub.MockTransactionEnd("t2")

	mstub.MockTransactionStart("t3")
	stub = cached_stub.NewCachedStub(mstub, true, true)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	row := make(map[string]string)

	row["company"] = "Com A"
	row["dept"] = "D1"
	row["id"] = "AD1"
	table.UpdateRow(row)
	row["company"] = "Com A"
	row["dept"] = "D2"
	row["id"] = "AD2"
	table.UpdateRow(row)
	row["company"] = "Com A"
	row["dept"] = "D3"
	row["id"] = "AD3"
	table.UpdateRow(row)

	row["company"] = "Com B"
	row["dept"] = "D1"
	row["id"] = "BD1"
	table.UpdateRow(row)
	row["company"] = "Com B"
	row["dept"] = "D3"
	row["id"] = "BD3"
	table.UpdateRow(row)

	row["company"] = "Com C"
	row["dept"] = "D2"
	row["id"] = "CD2"
	table.UpdateRow(row)
	row["company"] = "Com C"
	row["dept"] = "D3"
	row["id"] = "CD3"
	table.UpdateRow(row)

	row["company"] = "Com D"
	row["dept"] = "D1"
	row["id"] = "DD1"
	table.UpdateRow(row)
	row["company"] = "Com D"
	row["dept"] = "D2"
	row["id"] = "DD2"
	table.UpdateRow(row)
	row["company"] = "Com D"
	row["dept"] = "D4"
	row["id"] = "DD4"
	table.UpdateRow(row)
	row = make(map[string]string)

	mstub.MockTransactionEnd("t3")

	mstub.MockTransactionStart("t4")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	logger.Debugf("sort by company/dept")
	iter, err := table.GetRowsByPartialKey([]string{"company"}, []string{"Com D"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	defer iter.Close()
	actualList := []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		logger.Debugf("rowByes: %v", string(rowBytes[:]))
		err = json.Unmarshal(rowBytes, &row)

		logger.Infof("=== row %v %v", row, err)
		actualList = append(actualList, row["id"])
	}
	test_utils.AssertTrue(t, len(actualList) == 3, "expect 3, got "+strconv.Itoa(len(actualList)))
	test_utils.AssertSetsEqual(t, []string{"DD1", "DD2", "DD4"}, actualList)
	mstub.MockTransactionEnd("t4")

	//unhappy path
	mstub.MockTransactionStart("t5")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	logger.Debugf("sort by dept/company")
	iter, err = table.GetRowsByPartialKey([]string{"dept"}, []string{"X1"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	test_utils.AssertTrue(t, err == nil, "succeed")
	defer iter.Close()
	actualList = []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		err = json.Unmarshal(rowBytes, &row)

		actualList = append(actualList, row["id"])
		logger.Infof("=== row %v", row)
	}
	test_utils.AssertTrue(t, len(actualList) == 0, "expect 0, got "+strconv.Itoa(len(actualList)))
	mstub.MockTransactionEnd("t5")
}

func TestIndexDatastoreEncrypted(t *testing.T) {
	logger.Info("TestIndexDatastoreEncrypted function called")

	// create a MockStub
	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)
	init_common.Init(stub, shim.LogDebug)

	// change the following Cloudant config to connect to a custom Cloudant and set environment variables:
	// CLOUDANT_USERNAME
	// CLOUDANT_PASSWORD
	// CLOUDANT_DATABASE
	// CLOUDANT_HOST
	username := "admin"
	password := "pass"
	database := "test"
	host := "http://127.0.0.1:9080"
	// Get values from environment variables
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_USERNAME")) {
		username = os.Getenv("CLOUDANT_USERNAME")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_PASSWORD")) {
		password = os.Getenv("CLOUDANT_PASSWORD")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_DATABASE")) {
		database = os.Getenv("CLOUDANT_DATABASE")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_HOST")) {
		host = os.Getenv("CLOUDANT_HOST")
	}

	_, err := init_common.InitDatastore(stub, username, password, database, host)
	test_utils.AssertTrue(t, err == nil, "Error Getting Cloudant datastore: Make sure to start cloudant docker before running this test")
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t2")
	stub = cached_stub.NewCachedStub(mstub)

	//create table and add index
	table := index.GetTable(stub, "TestIndex", "id", false, true, datastore.DEFAULT_CLOUDANT_DATASTORE_ID)
	//key, err = s.CreateCompositeKey("Table", []string{"TestIndex"})

	table.AddIndex([]string{"company", "dept", "id"}, false)
	table.AddIndex([]string{"dept", "company", "id"}, false)
	table.SaveToLedger()
	mstub.MockTransactionEnd("t2")

	mstub.MockTransactionStart("t3")
	stub = cached_stub.NewCachedStub(mstub, true, true)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	row := make(map[string]string)

	row["company"] = "Com A"
	row["dept"] = "D1"
	row["id"] = "AD1"
	table.UpdateRow(row)
	row["company"] = "Com A"
	row["dept"] = "D2"
	row["id"] = "AD2"
	table.UpdateRow(row)
	row["company"] = "Com A"
	row["dept"] = "D3"
	row["id"] = "AD3"
	table.UpdateRow(row)

	row["company"] = "Com B"
	row["dept"] = "D1"
	row["id"] = "BD1"
	table.UpdateRow(row)
	row["company"] = "Com B"
	row["dept"] = "D3"
	row["id"] = "BD3"
	table.UpdateRow(row)

	row["company"] = "Com C"
	row["dept"] = "D2"
	row["id"] = "CD2"
	table.UpdateRow(row)
	row["company"] = "Com C"
	row["dept"] = "D3"
	row["id"] = "CD3"
	table.UpdateRow(row)

	row["company"] = "Com D"
	row["dept"] = "D1"
	row["id"] = "DD1"
	table.UpdateRow(row)
	row["company"] = "Com D"
	row["dept"] = "D2"
	row["id"] = "DD2"
	table.UpdateRow(row)
	row["company"] = "Com D"
	row["dept"] = "D4"
	row["id"] = "DD4"
	table.UpdateRow(row)
	row = make(map[string]string)

	mstub.MockTransactionEnd("t3")

	mstub.MockTransactionStart("t4")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	logger.Debugf("sort by company/dept")
	iter, err := table.GetRowsByPartialKey([]string{"company"}, []string{"Com D"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	defer iter.Close()
	actualList := []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		logger.Debugf("rowByes: %v", string(rowBytes[:]))
		err = json.Unmarshal(rowBytes, &row)

		logger.Infof("=== row %v %v", row, err)
		actualList = append(actualList, row["id"])
	}
	test_utils.AssertTrue(t, len(actualList) == 3, "expect 3, got "+strconv.Itoa(len(actualList)))
	test_utils.AssertSetsEqual(t, []string{"DD1", "DD2", "DD4"}, actualList)
	mstub.MockTransactionEnd("t4")

	mstub.MockTransactionStart("t5")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	logger.Debugf("sort by dept/company")
	iter, err = table.GetRowsByPartialKey([]string{"dept"}, []string{"D1"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	defer iter.Close()
	actualList = []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		err = json.Unmarshal(rowBytes, &row)

		logger.Infof("=== row %v", row)
		actualList = append(actualList, row["id"])
	}
	test_utils.AssertTrue(t, len(actualList) == 3, "expect 3, got "+strconv.Itoa(len(actualList)))
	test_utils.AssertSetsEqual(t, []string{"AD1", "BD1", "DD1"}, actualList)
	mstub.MockTransactionEnd("t5")

	//unhappy path
	mstub.MockTransactionStart("t6")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	logger.Debugf("sort by dept/company")
	iter, err = table.GetRowsByPartialKey([]string{"dept"}, []string{"X1"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	test_utils.AssertTrue(t, err == nil, "succeed")
	defer iter.Close()
	actualList = []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		err = json.Unmarshal(rowBytes, &row)

		actualList = append(actualList, row["id"])
		logger.Infof("=== row %v", row)
	}
	test_utils.AssertTrue(t, len(actualList) == 0, "expect 0, got "+strconv.Itoa(len(actualList)))
	mstub.MockTransactionEnd("t6")
}

func TestIndexTree(t *testing.T) {
	logger.Info("TestIndexTree function called")

	// create a MockStub
	mstub := setup(t)

	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)

	//create table and add index
	table := index.GetTable(stub, "TestIndex", "id", true)
	//key, err = s.CreateCompositeKey("Table", []string{"TestIndex"})

	table.AddIndex([]string{"company", "dept", "id"}, false)
	table.AddIndex([]string{"dept", "company", "id"}, false)
	table.SaveToLedger()
	mstub.MockTransactionEnd("t1")

	mstub.MockTransactionStart("t2")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	row := make(map[string]string)

	row["company"] = "Com A"
	row["dept"] = "D1"
	row["id"] = "AD1"
	table.UpdateRow(row)
	row["company"] = "Com A"
	row["dept"] = "D2"
	row["id"] = "AD2"
	table.UpdateRow(row)
	row["company"] = "Com A"
	row["dept"] = "D3"
	row["id"] = "AD3"
	table.UpdateRow(row)

	row["company"] = "Com B"
	row["dept"] = "D1"
	row["id"] = "BD1"
	table.UpdateRow(row)
	row["company"] = "Com B"
	row["dept"] = "D3"
	row["id"] = "BD3"
	table.UpdateRow(row)

	row["company"] = "Com C"
	row["dept"] = "D2"
	row["id"] = "CD2"
	table.UpdateRow(row)
	row["company"] = "Com C"
	row["dept"] = "D3"
	row["id"] = "CD3"
	table.UpdateRow(row)

	row["company"] = "Com D"
	row["dept"] = "D1"
	row["id"] = "DD1"
	table.UpdateRow(row)
	row["company"] = "Com D"
	row["dept"] = "D2"
	row["id"] = "DD2"
	table.UpdateRow(row)
	row["company"] = "Com D"
	row["dept"] = "D4"
	row["id"] = "DD4"
	table.UpdateRow(row)

	row = make(map[string]string)
	mstub.MockTransactionEnd("t2")

	mstub.MockTransactionStart("t3")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id", "true")
	logger.Debugf("sort by company/dept")
	iter, err := table.GetRowsByPartialKey([]string{"company"}, []string{"Com D"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	defer iter.Close()
	actualList := []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		err = json.Unmarshal(rowBytes, &row)
		actualList = append(actualList, row["id"])

		logger.Infof("=== row %v", row)
	}
	test_utils.AssertTrue(t, len(actualList) == 3, "expect 3, got "+strconv.Itoa(len(actualList)))
	test_utils.AssertSetsEqual(t, []string{"DD1", "DD2", "DD4"}, actualList)
	mstub.MockTransactionEnd("t3")

	mstub.MockTransactionStart("t4")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id", true)
	logger.Debugf("sort by dept/company")
	iter, err = table.GetRowsByPartialKey([]string{"dept"}, []string{"D1"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	defer iter.Close()
	actualList = []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		err = json.Unmarshal(rowBytes, &row)
		actualList = append(actualList, row["id"])

		logger.Infof("=== row %v", row)
	}
	test_utils.AssertTrue(t, len(actualList) == 3, "expect 3, got "+strconv.Itoa(len(actualList)))
	test_utils.AssertListsEqual(t, []string{"AD1", "BD1", "DD1"}, actualList)
	mstub.MockTransactionEnd("t4")

	//unhappy path
	mstub.MockTransactionStart("t5")
	stub = cached_stub.NewCachedStub(mstub)
	//create table and add index
	table = index.GetTable(stub, "TestIndex", "id")
	logger.Debugf("sort by dept/company")
	iter, err = table.GetRowsByPartialKey([]string{"dept"}, []string{"X1"})
	if err != nil {
		logger.Errorf("iter error %v", err)
	}
	test_utils.AssertTrue(t, err == nil, "succeed")
	defer iter.Close()
	actualList = []string{}
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}
		rowBytes := KV.GetValue()
		err = json.Unmarshal(rowBytes, &row)

		actualList = append(actualList, row["id"])
		logger.Infof("=== row %v", row)
	}
	test_utils.AssertTrue(t, len(actualList) == 0, "should be len 0: got "+strconv.Itoa(len(actualList)))
	mstub.MockTransactionEnd("t5")
}
