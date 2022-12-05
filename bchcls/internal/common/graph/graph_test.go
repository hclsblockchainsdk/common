/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package graph

import (
	"common/bchcls/cached_stub"
	"common/bchcls/test_utils"
	"encoding/json"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"reflect"
	"testing"
)

// Tests adding + updating an edge to the graph
func TestPutEdge(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutEdge function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	startNodeId := "node1"
	targetNodeId := "node2"
	edgeValue := []byte("myValue")
	graph := "myGraph"

	// TESTING ADD
	mstub.MockTransactionStart("t123")
	err := PutEdge(stub, graph, startNodeId, targetNodeId, edgeValue)
	test_utils.AssertTrue(t, err == nil, "Expected PutEdge to succeed")
	mstub.MockTransactionEnd("t123")

	// Get the edge value
	//edgeKey, _ := stub.CreateCompositeKey(graph, []string{startNodeId, targetNodeId})
	//edgeValueBytes, _ := stub.GetState(edgeKey)
	edgeValueBytes, _, err := GetEdge(stub, graph, startNodeId, targetNodeId)
	test_utils.AssertTrue(t, err == nil, "Expected GetEdge to succeed")

	// Check that the edge was added properly and got value back
	test_utils.AssertTrue(t, reflect.DeepEqual(edgeValue, edgeValueBytes), "got edge value back")

	// TESTING PUT
	// Test update edge value
	edgeValueUpdated := []byte("myValueUpdated")

	mstub.MockTransactionStart("t123")
	err = PutEdge(stub, graph, startNodeId, targetNodeId, edgeValueUpdated)
	test_utils.AssertTrue(t, err == nil, "Expected PutEdge to succeed")
	mstub.MockTransactionEnd("t123")

	stub = cached_stub.NewCachedStub(mstub)

	// Get the updated edge value
	//edgeValueBytes, _ = stub.GetState(edgeKey)
	edgeValueBytes, _, err = GetEdge(stub, graph, startNodeId, targetNodeId)
	test_utils.AssertTrue(t, err == nil, "Expected GetEdge to succeed")

	// Check that the edge was added properly and got value back
	test_utils.AssertTrue(t, reflect.DeepEqual(edgeValueUpdated, edgeValueBytes), "got edge value back")
	// Check that edge value got updated and previous edge value is no longer valid
	test_utils.AssertFalse(t, reflect.DeepEqual(edgeValue, edgeValueBytes), "old value no longer valid")
}

// Test with invalid inputs
func TestPutEdge_InvalidInputs(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutEdge_InvalidInputs function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	startNodeId := ""
	targetNodeId := "node2"
	edgeValue := []byte("myValue")
	graph := "myGraph"

	// TESTING ADD INVALID VALUE
	err := PutEdge(stub, graph, startNodeId, targetNodeId, edgeValue)
	test_utils.AssertTrue(t, err != nil, "Expected PutEdge to fail")
}

// Test with bad stub
func TestPutEdge_BadLedger(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestPutEdge_BadLedger function called")

	startNodeId := "node1"
	targetNodeId := "node2"
	edgeValue := []byte("myValue")
	graph := "myGraph"

	badStub := test_utils.CreateMisbehavingMockStub(t)
	stub := cached_stub.NewCachedStub(badStub)

	err := PutEdge(stub, graph, startNodeId, targetNodeId, edgeValue)
	test_utils.AssertTrue(t, err != nil, "Expected PutEdge to fail")
}

// Tests deleting an edge from the graph
func TestDeleteEdge(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestDeleteEdge function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	startNodeId := "node1"
	targetNodeId := "node2"
	edgeValue := []byte("myValue")
	graph := "myGraph"

	mstub.MockTransactionStart("t123")
	err := PutEdge(stub, graph, startNodeId, targetNodeId, edgeValue)
	test_utils.AssertTrue(t, err == nil, "Expected PutEdge to succeed")
	mstub.MockTransactionEnd("t123")

	// Get the edge value
	edgeKey, _ := stub.CreateCompositeKey(graph, []string{startNodeId, targetNodeId})
	//edgeValueBytes, _ := stub.GetState(edgeKey)
	edgeValueBytes, _, err := GetEdge(stub, graph, startNodeId, targetNodeId)
	test_utils.AssertTrue(t, err == nil, "Expected GetEdge to succeed")

	// Check that the edge was added properly and got value back
	test_utils.AssertTrue(t, reflect.DeepEqual(edgeValue, edgeValueBytes), "got edge value back")

	// TESTING DELETE
	mstub.MockTransactionStart("t123")
	err = DeleteEdge(stub, graph, startNodeId, targetNodeId)
	test_utils.AssertTrue(t, err == nil, "Expected DeleteEdge to succeed")
	mstub.MockTransactionEnd("t123")

	// Verify value has been deleted
	edgeValueBytes, _ = stub.GetState(edgeKey)
	test_utils.AssertFalse(t, reflect.DeepEqual(edgeValue, edgeValueBytes), "value has been deleted, should not equal")
}

// Test with invalid inputs
func TestDeleteEdge_InvalidInputs(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestDeleteEdge_InvalidInputs function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	startNodeId := ""
	targetNodeId := "node2"
	graph := "myGraph"

	// TESTING DELETE INVALID VALUE
	err := DeleteEdge(stub, graph, startNodeId, targetNodeId)
	test_utils.AssertTrue(t, err != nil, "Expected DeleteEdge to fail")
}

// Test with bad stub
func TestDeleteEdge_BadLedger(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestDeleteEdge_BadLedger function called")

	badStub := test_utils.CreateMisbehavingMockStub(t)
	stub := cached_stub.NewCachedStub(badStub)
	startNodeId := "node1"
	targetNodeId := "node2"
	graph := "myGraph"

	err := DeleteEdge(stub, graph, startNodeId, targetNodeId)
	test_utils.AssertTrue(t, err != nil, "Expected DeleteEdge to fail")
}

// Tests get edge
func TestGetEdge(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetEdge function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	startNodeId := "node1"
	targetNodeId := "node2"
	edgeValue := []byte("myValue")
	graph := "myGraph"

	mstub.MockTransactionStart("t123")
	err := PutEdge(stub, graph, startNodeId, targetNodeId, edgeValue)
	test_utils.AssertTrue(t, err == nil, "Expected PutEdge to succeed")
	mstub.MockTransactionEnd("t123")

	// Get the edge value
	edgeValueBytes, _, err := GetEdge(stub, graph, startNodeId, targetNodeId)
	test_utils.AssertTrue(t, err == nil, "Expected GetEdge to succeed")

	// Check that edge value is retrieved properly
	test_utils.AssertTrue(t, reflect.DeepEqual(edgeValue, edgeValueBytes), "got edge value back")
}

// Test with invalid inputs
func TestGetEdge_InvalidInputs(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetEdge_InvalidInputs function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	startNodeId := ""
	targetNodeId := "node2"
	graph := "myGraph"

	_, _, err := GetEdge(stub, graph, startNodeId, targetNodeId)
	test_utils.AssertTrue(t, err != nil, "Expected GetEdge to fail")
}

// Test with bad stub
func TestGetEdge_BadLedger(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetEdge_BadLedger function called")

	badStub := test_utils.CreateMisbehavingMockStub(t)
	stub := cached_stub.NewCachedStub(badStub)
	startNodeId := "node1"
	targetNodeId := "node2"
	graph := "myGraph"

	_, _, err := GetEdge(stub, graph, startNodeId, targetNodeId)
	test_utils.AssertTrue(t, err != nil, "Expected GetEdge to fail")
}

// Tests finding path between two nodes
func TestFindPath(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestFindPath function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	graph := "myGraph"
	mstub.MockTransactionStart("t123")
	PutEdge(stub, graph, "node1", "node2", []byte("myValue1"))
	PutEdge(stub, graph, "node1", "node3", []byte("myValue2"))
	PutEdge(stub, graph, "node2", "node4", []byte("myValue3"))
	PutEdge(stub, graph, "node2", "node5", []byte("myValue4"))
	PutEdge(stub, graph, "node3", "node6", []byte("myValue5"))
	PutEdge(stub, graph, "node2", "node1", []byte("myValue6")) // tight cycle
	PutEdge(stub, graph, "node5", "node1", []byte("myValue7")) // cycle
	mstub.MockTransactionEnd("t123")

	// find path from node1 to node4
	expectedPath := []string{"node1", "node2", "node4"}
	resultPath, err := SlowFindPath(stub, graph, "node1", "node4")
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to succeed")
	test_utils.AssertListsEqual(t, expectedPath, resultPath)

	// find path from node1 to node2
	expectedPath = []string{"node1", "node2"}
	resultPath, err = SlowFindPath(stub, graph, "node1", "node2")
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to succeed")
	test_utils.AssertListsEqual(t, expectedPath, resultPath)

	// find path from node2 to node1
	expectedPath = []string{"node2", "node1"}
	resultPath, err = SlowFindPath(stub, graph, "node2", "node1")
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to succeed")
	test_utils.AssertListsEqual(t, expectedPath, resultPath)

	// find path from node1 to node5
	expectedPath = []string{"node1", "node2", "node5"}
	resultPath, err = SlowFindPath(stub, graph, "node1", "node5")
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to succeed")
	test_utils.AssertListsEqual(t, expectedPath, resultPath)

	// find path from node1 to node6
	expectedPath = []string{"node1", "node3", "node6"}
	resultPath, err = SlowFindPath(stub, graph, "node1", "node6")
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to succeed")
	test_utils.AssertListsEqual(t, expectedPath, resultPath)

	// find path from node2 to node5
	expectedPath = []string{"node2", "node5"}
	resultPath, err = SlowFindPath(stub, graph, "node2", "node5")
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to succeed")
	test_utils.AssertListsEqual(t, expectedPath, resultPath)

	// find non existent path from node4 to node5
	resultPath, err = SlowFindPath(stub, graph, "node4", "node5")
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to return nil")
	test_utils.AssertListsEqual(t, nil, resultPath)
}

// Tests finding path between two nodes
func TestFindPathFilter(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestFindPathFilter function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	graph := "myGraph"
	mstub.MockTransactionStart("t123")
	edata := make(map[string]string)
	edata["type"] = "skip"
	PutEdge(stub, graph, "node1", "node2", []byte("myValue1"), edata)
	PutEdge(stub, graph, "node1", "node3", []byte("myValue2"))
	PutEdge(stub, graph, "node2", "node4", []byte("myValue3"))
	PutEdge(stub, graph, "node2", "node5", []byte("myValue4"))
	PutEdge(stub, graph, "node3", "node6", []byte("myValue5"))
	PutEdge(stub, graph, "node2", "node1", []byte("myValue6")) // tight cycle
	PutEdge(stub, graph, "node5", "node1", []byte("myValue7")) // cycle
	mstub.MockTransactionEnd("t123")

	// find path from node1 to node4
	expectedPath := []string{"node1", "node2", "node4"}
	resultPath, err := SlowFindPath(stub, graph, "node1", "node4", `{"==": [{"var": "type"}, "skip"]}`)
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to return nil")
	logger.Debugf("resultPath %v", resultPath)
	test_utils.AssertListsEqual(t, nil, resultPath)

	expectedPath = []string{"node1", "node2", "node4"}
	resultPath, err = SlowFindPath(stub, graph, "node1", "node4")
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to succeed")
	test_utils.AssertListsEqual(t, expectedPath, resultPath)

	// find path from node1 to node2
	expectedPath = []string{"node1", "node2"}
	resultPath, err = SlowFindPath(stub, graph, "node1", "node2", `{"==": [{"var": "type"}, "skip"]}`)
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to return nil")
	test_utils.AssertListsEqual(t, nil, resultPath)

	// find path from node2 to node1
	expectedPath = []string{"node2", "node1"}
	resultPath, err = SlowFindPath(stub, graph, "node2", "node1")
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to succeed")
	test_utils.AssertListsEqual(t, expectedPath, resultPath)

	// find path from node1 to node5
	expectedPath = []string{"node1", "node2", "node5"}
	resultPath, err = SlowFindPath(stub, graph, "node1", "node5")
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to succeed")
	test_utils.AssertListsEqual(t, expectedPath, resultPath)

	expectedPath = []string{"node1", "node2", "node5"}
	resultPath, err = SlowFindPath(stub, graph, "node1", "node5", `{"==": [{"var": "type"}, "skip"]}`)
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to return nil")
	test_utils.AssertListsEqual(t, nil, resultPath)

	// find path from node1 to node6
	expectedPath = []string{"node1", "node3", "node6"}
	resultPath, err = SlowFindPath(stub, graph, "node1", "node6")
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to succeed")
	test_utils.AssertListsEqual(t, expectedPath, resultPath)

	// find path from node2 to node5
	expectedPath = []string{"node2", "node5"}
	resultPath, err = SlowFindPath(stub, graph, "node2", "node5")
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to succeed")
	test_utils.AssertListsEqual(t, expectedPath, resultPath)

	// find non existent path from node4 to node5
	resultPath, err = SlowFindPath(stub, graph, "node4", "node5")
	test_utils.AssertTrue(t, err == nil, "Expected FindPath to return nil")
	test_utils.AssertListsEqual(t, nil, resultPath)

}

// Tests finding path between two nodes
func TestHasPath(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestFindPath function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	graph := "myGraph"
	mstub.MockTransactionStart("t123")
	PutEdge(stub, graph, "node1", "node2", []byte("myValue1"))
	PutEdge(stub, graph, "node1", "node3", []byte("myValue2"))
	PutEdge(stub, graph, "node2", "node4", []byte("myValue3"))
	PutEdge(stub, graph, "node2", "node5", []byte("myValue4"))
	PutEdge(stub, graph, "node3", "node6", []byte("myValue5"))
	PutEdge(stub, graph, "node2", "node1", []byte("myValue6")) // tight cycle
	PutEdge(stub, graph, "node5", "node1", []byte("myValue7")) // cycle
	mstub.MockTransactionEnd("t123")

	// find path from node1 to node4
	expectedPath := []string{"node1", "node2", "node4"}
	hasPath, err := HasPath(stub, graph, expectedPath)
	test_utils.AssertTrue(t, hasPath && err == nil, "Expected HasPath to true")

	// wrong path from node1 to node 5
	expectedPath = []string{"node1", "node3", "node5"}
	hasPath, err = HasPath(stub, graph, expectedPath)
	test_utils.AssertTrue(t, !hasPath && err == nil, "Expected HasPath to false")

}

// Test with invalid inputs
func TestFindPath_InvalidInputs(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestFindPath_InvalidInputs function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	_, err := SlowFindPath(stub, "", "", "")
	test_utils.AssertTrue(t, err != nil, "Expected TestFindPath to fail")
}

// Tests finding all child nodes
func TestGetChildren(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetChildren function called")

	mstub := test_utils.CreateNewMockStub(t)

	graph := "myGraph"
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	edata := make(map[string]string)
	edata["type"] = "my asset"
	edata2 := make(map[string]string)
	edata2["type"] = "your asset"
	PutEdge(stub, graph, "node1", "node2", []byte("myValue1"), edata)
	PutEdge(stub, graph, "node1", "node3", []byte("myValue2"), edata2)
	PutEdge(stub, graph, "node2", "node4", []byte("myValue3"), edata)
	PutEdge(stub, graph, "node2", "node5", []byte("myValue4"), edata2)
	PutEdge(stub, graph, "node3", "node6", []byte("myValue5"), edata)
	PutEdge(stub, graph, "node2", "node1", []byte("myValue6")) // tight cycle
	PutEdge(stub, graph, "node5", "node1", []byte("myValue7")) // cycle
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// find child nodes of node1
	expectedNodes := []string{"node2", "node3", "node4", "node5", "node6"}
	childNodes, err := SlowGetChildren(stub, graph, "node1")
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find child nodes of node1 with filter for my asset
	expectedNodes = []string{"node2", "node4", "node6"}
	childNodes, err = SlowGetChildren(stub, graph, "node1", `{"!=": [{"var": "type"}, "my asset"]}`)
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find child nodes of node2
	expectedNodes = []string{"node1", "node4", "node5", "node3", "node6"}

	childNodes, err = SlowGetChildren(stub, graph, "node2")
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// should getting from cache
	childNodes, err = SlowGetChildren(stub, graph, "node2")
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find child nodes of node2 with filter for your asset
	expectedNodes = []string{"node3", "node5"}
	childNodes, err = SlowGetChildren(stub, graph, "node2", `{"!=": [{"var": "type"}, "your asset"]}`)
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find child nodes of node3
	expectedNodes = []string{"node6"}
	childNodes, err = SlowGetChildren(stub, graph, "node3")
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find child nodes of node6
	expectedNodes = []string{}
	childNodes, err = SlowGetChildren(stub, graph, "node6")
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find child nodes of node5
	expectedNodes = []string{"node1", "node2", "node3", "node4", "node6"}
	childNodes, err = SlowGetChildren(stub, graph, "node5")
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find child nodes of non existent node
	expectedNodes = []string{}
	childNodes, err = SlowGetChildren(stub, graph, "node7")
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to return empty")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	mstub.MockTransactionEnd("t123")
}

// Test with invalid inputs
func TestGetChildren_InvalidInputs(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetChildren_InvalidInputs function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	_, err := SlowGetChildren(stub, "", "")
	test_utils.AssertTrue(t, err != nil, "Expected GetChildren to fail")
}

// Tests finding all parent nodes
func TestGetParents(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetParents function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	graph := "myGraph"
	mstub.MockTransactionStart("t123")
	PutEdge(stub, graph, "node1", "node2", []byte("myValue1"))
	PutEdge(stub, graph, "node1", "node3", []byte("myValue2"))
	PutEdge(stub, graph, "node2", "node4", []byte("myValue3"))
	PutEdge(stub, graph, "node2", "node5", []byte("myValue4"))
	PutEdge(stub, graph, "node3", "node6", []byte("myValue5"))
	PutEdge(stub, graph, "node2", "node1", []byte("myValue6")) // tight cycle
	PutEdge(stub, graph, "node5", "node1", []byte("myValue7")) // cycle
	mstub.MockTransactionEnd("t123")

	// find parent nodes of node1
	expectedNodes := []string{"node2", "node5"}
	parentNodes, err := SlowGetParents(stub, graph, "node1")
	test_utils.AssertTrue(t, err == nil, "Expected GetParents to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, parentNodes)

	// find parent nodes of node2
	expectedNodes = []string{"node1", "node5"}
	parentNodes, err = SlowGetParents(stub, graph, "node2")
	test_utils.AssertTrue(t, err == nil, "Expected GetParents to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, parentNodes)

	// find parent nodes of node3
	expectedNodes = []string{"node1", "node2", "node5"}
	parentNodes, err = SlowGetParents(stub, graph, "node3")
	test_utils.AssertTrue(t, err == nil, "Expected GetParents to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, parentNodes)

	// find parent nodes of non existent node
	expectedNodes = []string{}
	parentNodes, err = SlowGetParents(stub, graph, "node7")
	test_utils.AssertTrue(t, err == nil, "Expected GetParents to return empty")
	test_utils.AssertSetsEqual(t, expectedNodes, parentNodes)
}

// Test finding immediate children
func TestGetDirectChildren(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetDirectChildren function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	graph := "myGraph"
	mstub.MockTransactionStart("t123")
	PutEdge(stub, graph, "node1", "node2", []byte("myValue1"))
	PutEdge(stub, graph, "node1", "node3", []byte("myValue2"))
	PutEdge(stub, graph, "node2", "node4", []byte("myValue3"))
	PutEdge(stub, graph, "node2", "node5", []byte("myValue4"))
	PutEdge(stub, graph, "node3", "node6", []byte("myValue5"))
	PutEdge(stub, graph, "node2", "node1", []byte("myValue6")) // tight cycle
	PutEdge(stub, graph, "node5", "node1", []byte("myValue7")) // cycle
	mstub.MockTransactionEnd("t123")

	// find direct child nodes of node1
	expectedNodes := []string{"node2", "node3"}
	childNodes, err := GetDirectChildren(stub, graph, "node1")
	test_utils.AssertTrue(t, err == nil, "Expected GetDirectChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find direct child nodes of node2
	expectedNodes = []string{"node1", "node4", "node5"}
	childNodes, err = GetDirectChildren(stub, graph, "node2")
	test_utils.AssertTrue(t, err == nil, "Expected GetDirectChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find direct child nodes of node3
	expectedNodes = []string{"node6"}
	childNodes, err = GetDirectChildren(stub, graph, "node3")
	test_utils.AssertTrue(t, err == nil, "Expected GetDirectChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find direct child nodes of node4
	expectedNodes = []string{}
	childNodes, err = GetDirectChildren(stub, graph, "node4")
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find direct child nodes of node5
	expectedNodes = []string{"node1"}
	childNodes, err = GetDirectChildren(stub, graph, "node5")
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find direct child nodes of node6
	expectedNodes = []string{}
	childNodes, err = GetDirectChildren(stub, graph, "node6")
	test_utils.AssertTrue(t, err == nil, "Expected GetDirectChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find direct child nodes of non existent node
	expectedNodes = []string{}
	childNodes, err = GetDirectChildren(stub, graph, "node7")
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to return empty")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)
}

// Test finding immediate parents
func TestGetDirectParents(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetDirectParents function called")

	mstub := test_utils.CreateNewMockStub(t)

	graph := "myGraph"
	mstub.MockTransactionStart("t123")
	stub := cached_stub.NewCachedStub(mstub)
	PutEdge(stub, graph, "node1", "node2", []byte("myValue1"))
	PutEdge(stub, graph, "node1", "node3", []byte("myValue2"))
	PutEdge(stub, graph, "node2", "node4", []byte("myValue3"))
	PutEdge(stub, graph, "node2", "node5", []byte("myValue4"))
	PutEdge(stub, graph, "node3", "node6", []byte("myValue5"))
	PutEdge(stub, graph, "node2", "node1", []byte("myValue6")) // tight cycle
	PutEdge(stub, graph, "node5", "node1", []byte("myValue7")) // cycle
	mstub.MockTransactionEnd("t123")

	mstub.MockTransactionStart("t123")
	stub = cached_stub.NewCachedStub(mstub)
	// find direct parent nodes of node1
	expectedNodes := []string{"node2", "node5"}
	childNodes, err := GetDirectParents(stub, graph, "node1")
	test_utils.AssertTrue(t, err == nil, "Expected GetDirectChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find direct parent nodes of node2
	expectedNodes = []string{"node1"}
	childNodes, err = GetDirectParents(stub, graph, "node2")
	test_utils.AssertTrue(t, err == nil, "Expected GetDirectChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find direct parent nodes of node3
	expectedNodes = []string{"node1"}
	childNodes, err = GetDirectParents(stub, graph, "node3")
	test_utils.AssertTrue(t, err == nil, "Expected GetDirectChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find direct parent nodes of node4
	expectedNodes = []string{"node2"}
	childNodes, err = GetDirectParents(stub, graph, "node4")
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find direct parent nodes of node5
	expectedNodes = []string{"node2"}
	childNodes, err = GetDirectParents(stub, graph, "node5")
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find direct parent nodes of node6
	expectedNodes = []string{"node3"}
	childNodes, err = GetDirectParents(stub, graph, "node6")
	logger.Debugf("child nodes: %v", childNodes)
	test_utils.AssertTrue(t, err == nil, "Expected GetDirectChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find direct parent nodes of non existent node
	expectedNodes = []string{}
	childNodes, err = GetDirectParents(stub, graph, "node7")
	test_utils.AssertTrue(t, err == nil, "Expected GetChildren to return empty")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	// find direct parent nodes of node6 should get from cache
	expectedNodes = []string{"node3"}
	childNodes, err = GetDirectParents(stub, graph, "node6")
	logger.Debugf("child nodes: %v", childNodes)
	test_utils.AssertTrue(t, err == nil, "Expected GetDirectChildren to succeed")
	test_utils.AssertSetsEqual(t, expectedNodes, childNodes)

	mstub.MockTransactionEnd("t123")
}

// Test finding immediate parents
func TestGetFirstEdge(t *testing.T) {
	logger.SetLevel(shim.LogDebug)
	logger.Info("TestGetFirstEdge function called")

	mstub := test_utils.CreateNewMockStub(t)
	stub := cached_stub.NewCachedStub(mstub)

	graph := "myGraph"
	mstub.MockTransactionStart("t123")
	PutEdge(stub, graph, "node1", "node2", []byte("myValue12"))
	PutEdge(stub, graph, "node1", "node3", []byte("myValue13"))
	PutEdge(stub, graph, "node2", "node4", []byte("myValue24"))
	PutEdge(stub, graph, "node2", "node5", []byte("myValue25"))
	PutEdge(stub, graph, "node3", "node6", []byte("myValue36"))
	PutEdge(stub, graph, "node6", "node7", []byte("myValue76"))
	PutEdge(stub, graph, "node2", "node1", []byte("myValue21")) // tight cycle
	PutEdge(stub, graph, "node5", "node1", []byte("myValue51")) // cycle
	mstub.MockTransactionEnd("t123")

	// GetFirstEdgeContainingNodeId
	edgeValueBytes, err := GetFirstEdgeContainingNodeId(stub, graph, "node2")
	var edgeData = Edge{}
	err = json.Unmarshal(edgeValueBytes, &edgeData)
	test_utils.AssertTrue(t, err == nil, "Expected no error")
	test_utils.AssertInLists(t, string(edgeData.EdgeValue), []string{"myValue12", "myValue24", "myValue25", "myValue21"}, "got edge value back")

	edgeValueBytes, err = GetFirstEdgeContainingNodeId(stub, graph, "node5")
	edgeData = Edge{}
	err = json.Unmarshal(edgeValueBytes, &edgeData)
	test_utils.AssertTrue(t, err == nil, "Expected no error")
	test_utils.AssertInLists(t, string(edgeData.EdgeValue), []string{"myValue25", "myValue51"}, "got edge value back")

	edgeValueBytes, err = GetFirstEdgeContainingNodeId(stub, graph, "node7")
	edgeData = Edge{}
	err = json.Unmarshal(edgeValueBytes, &edgeData)
	test_utils.AssertTrue(t, err == nil, "Expected no error")
	test_utils.AssertTrue(t, edgeData.EdgeData["reverse_graph"] == "yes", "reverse graph")
	test_utils.AssertInLists(t, string(edgeData.EdgeValue), []string{"myValue76"}, "got edge value back")

}
