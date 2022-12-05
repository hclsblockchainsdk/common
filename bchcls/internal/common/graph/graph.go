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
	"common/bchcls/custom_errors"
	"common/bchcls/internal/common/global"
	"common/bchcls/simple_rule"

	"encoding/json"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("graph")

type Edge struct {
	EdgeValue []byte            `json:"edge_value"`
	EdgeData  map[string]string `json:"edge_data"`
}

// PutEdge updates an existing edge or adds an edge if it does not exist.
// edge is an edgeValue ([]byte), and edgeData is a map[string]string (optional)
func PutEdge(stub cached_stub.CachedStubInterface, graphName string, startNodeId string, targetNodeId string, edge ...interface{}) error {
	logger.Debugf("PutEdge graphName: %v, startNodeId: %v, targetNodeId: %v, edge: %v", graphName, startNodeId, targetNodeId, edge)

	// Make sure input parameters are valid
	if len(graphName) == 0 || len(startNodeId) == 0 || len(targetNodeId) == 0 {
		custom_err := errors.WithStack(&custom_errors.LengthCheckingError{Type: "edge"})
		logger.Errorf("%v", custom_err)
		return custom_err
	}

	edgeData := Edge{}
	if len(edge) > 0 {
		if edgeValue, ok := edge[0].([]byte); ok {
			edgeData.EdgeValue = edgeValue
		} else {
			custom_err := &custom_errors.TypeAssertionError{Item: "edgeValue", Type: "[]byte"}
			logger.Errorf("%v", custom_err)
			return errors.WithStack(custom_err)
		}
	}

	if len(edge) >= 2 && edge[1] != nil {
		if edgeMetaData, ok := edge[1].(map[string]string); ok {
			edgeData.EdgeData = edgeMetaData
		} else {
			custom_err := &custom_errors.TypeAssertionError{Item: "edgeData", Type: "map[string]string"}
			logger.Errorf("%v", custom_err)
			return errors.WithStack(custom_err)
		}
	}

	edgeValue, _ := json.Marshal(&edgeData)

	// Create the new edge and store edge in graph
	edgeKey, _ := stub.CreateCompositeKey(graphName, []string{startNodeId, targetNodeId})
	err := stub.PutState(edgeKey, edgeValue)
	if err != nil {
		custom_err := errors.WithStack(&custom_errors.PutLedgerError{LedgerKey: edgeKey})
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	// Reverse graph
	// Store edge in reverse graph
	if len(edgeData.EdgeData) == 0 {
		edgeData.EdgeData = make(map[string]string)
		edgeData.EdgeData["reverse_graph"] = "yes"
	} else {
		edgeData.EdgeData["reverse_graph"] = "yes"
	}
	edgeValue, _ = json.Marshal(&edgeData)
	reverseEdgeKey, _ := stub.CreateCompositeKey(global.REVERSE_GRAPH_PREFIX+graphName, []string{targetNodeId, startNodeId})
	err = stub.PutState(reverseEdgeKey, edgeValue)
	if err != nil {
		custom_err := errors.WithStack(&custom_errors.PutLedgerError{LedgerKey: reverseEdgeKey})
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	return nil
}

// DeleteEdge deletes an edge value.
func DeleteEdge(stub cached_stub.CachedStubInterface, graphName string, startNodeId string, targetNodeId string) error {
	// Make sure input parameters are valid
	if len(graphName) == 0 || len(startNodeId) == 0 || len(targetNodeId) == 0 {
		custom_err := errors.WithStack(&custom_errors.LengthCheckingError{})
		logger.Errorf("%v", custom_err)
		return custom_err
	}

	// Delete the edge from graph
	edgeKey, _ := stub.CreateCompositeKey(graphName, []string{startNodeId, targetNodeId})
	err := stub.DelState(edgeKey)
	if err != nil {
		custom_err := errors.WithStack(&custom_errors.DeleteLedgerError{LedgerKey: edgeKey})
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	// Reverse graph
	// Delete the edge from reverse graph
	reverseEdgeKey, _ := stub.CreateCompositeKey(global.REVERSE_GRAPH_PREFIX+graphName, []string{targetNodeId, startNodeId})
	err = stub.DelState(reverseEdgeKey)
	if err != nil {
		custom_err := errors.WithStack(&custom_errors.DeleteLedgerError{LedgerKey: reverseEdgeKey})
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	return nil
}

// GetEdge gets an edge value and edge data given node ids.
// Returns edge value, edge data, error.
func GetEdge(stub cached_stub.CachedStubInterface, graphName string, startNodeId string, targetNodeId string) ([]byte, map[string]string, error) {
	// Make sure input parameters are valid
	if len(graphName) == 0 || len(startNodeId) == 0 || len(targetNodeId) == 0 {
		custom_err := errors.WithStack(&custom_errors.LengthCheckingError{})
		logger.Errorf("%v", custom_err)
		return nil, nil, custom_err
	}

	// Gets edge value
	edgeKey, _ := stub.CreateCompositeKey(graphName, []string{startNodeId, targetNodeId})
	edgeValueBytes, err := stub.GetState(edgeKey)
	if err != nil {
		custom_err := errors.WithStack(&custom_errors.GetLedgerError{LedgerKey: edgeKey, LedgerItem: "edgeValueBytes"})
		logger.Errorf("%v: %v", custom_err, err)
		return nil, nil, errors.Wrap(err, custom_err.Error())
	}

	var edgeData = Edge{}
	if edgeValueBytes != nil {
		err = json.Unmarshal(edgeValueBytes, &edgeData)
		if err != nil {
			custom_err := errors.WithStack(&custom_errors.UnmarshalError{Type: "Edge"})
			logger.Errorf("%v: %v", custom_err, err)
			return nil, nil, errors.Wrap(err, custom_err.Error())
		}
	}

	// return encoded edgeValueBytes
	return edgeData.EdgeValue, edgeData.EdgeData, nil
}

// HasEdge checks if the edge exists in the graph.
func HasEdge(stub cached_stub.CachedStubInterface, graphName string, startNodeId string, targetNodeId string) (bool, error) {
	// Make sure input parameters are valid
	if len(graphName) == 0 || len(startNodeId) == 0 || len(targetNodeId) == 0 {
		custom_err := errors.WithStack(&custom_errors.LengthCheckingError{})
		logger.Errorf("%v", custom_err)
		return false, custom_err
	}

	// Gets edge value
	edgeKey, _ := stub.CreateCompositeKey(graphName, []string{startNodeId, targetNodeId})
	edgeValueBytes, err := stub.GetState(edgeKey)
	if err != nil {
		custom_err := errors.WithStack(&custom_errors.GetLedgerError{LedgerKey: edgeKey, LedgerItem: "edgeValueBytes"})
		logger.Errorf("%v: %v", custom_err, err)
		return false, errors.Wrap(err, custom_err.Error())
	}
	if edgeValueBytes == nil {
		return false, nil
	}

	return true, nil
}

func getPathCacheKey(graphName string, path []string) string {
	return graphName + "path_" + strings.Join(path, ",")
}

// HasPath checks if the path exists in the graph.
// The existence of a path from an id to the same id will return true.
func HasPath(stub cached_stub.CachedStubInterface, graphName string, path []string) (bool, error) {
	if len(path) < 2 {
		return false, nil
	}

	// ckeck cache
	cachekey := getPathCacheKey(graphName, path)
	verifiedCache, err := stub.GetCache(cachekey)
	if err == nil {
		verified, ok := verifiedCache.(bool)
		if ok {
			logger.Debugf("Getting from cache %v", cachekey)

			return verified, nil
		}
	}

	startId := path[0]
	targetId := path[0]
	for _, id := range path[1:] {
		startId = targetId
		targetId = id
		// skip if targetId is same as startId
		if startId != targetId {
			hasEdge, err := HasEdge(stub, graphName, startId, targetId)
			//logger.Debugf("hasEdge %v %v %v %v", startId, targetId, hasEdge, err)
			if !hasEdge || err != nil {
				if err == nil {
					stub.PutCache(cachekey, false)
				}
				return hasEdge, err
			}
		}
	}
	//save to cache
	stub.PutCache(cachekey, true)
	return true, nil
}

// GetDirectChildren returns the immediate children of the given nodeId
func GetDirectChildren(stub cached_stub.CachedStubInterface, graphName string, nodeId string) ([]string, error) {
	return getNeighbors(stub, graphName, nodeId)
}

// GetDirectParents returns the immediate parents of the given nodeId
func GetDirectParents(stub cached_stub.CachedStubInterface, graphName string, nodeId string) ([]string, error) {
	return getNeighbors(stub, global.REVERSE_GRAPH_PREFIX+graphName, nodeId)
}

// SlowGetChildren gets all children in the graph
func SlowGetChildren(stub cached_stub.CachedStubInterface, graphName string, nodeId string, filter ...interface{}) ([]string, error) {
	//return bfs(stub, graphName, nodeId)
	var filterRule interface{} = nil
	if len(filter) > 0 {
		filterRule = filter[0]
	}
	var visited = make(map[string]bool)
	nodes, _, err := dfsNode(stub, graphName, nodeId, visited, filterRule)
	return nodes, err
}

// SlowGetParents gets all parents in the graph
func SlowGetParents(stub cached_stub.CachedStubInterface, graphName string, nodeId string, filter ...interface{}) ([]string, error) {
	//return bfs(stub, global.REVERSE_GRAPH_PREFIX+graphName, nodeId)
	var filterRule interface{} = nil
	if len(filter) > 0 {
		filterRule = filter[0]
	}
	var visited = make(map[string]bool)
	nodes, _, err := dfsNode(stub, global.REVERSE_GRAPH_PREFIX+graphName, nodeId, visited, filterRule)
	return nodes, err
}

func getdfsNodeCacheKey(graphName string, currNodeId string, filter interface{}) string {
	filterStr := ""
	if filter != nil {
		rule := simple_rule.NewRule(filter)
		filterStr = rule.GetExprJSON()
	}
	return graphName + "-dfsNode-" + currNodeId + "-" + filterStr
}

// dfsNode is a depth first search helper function that finds all children.
// If filterRule evaluates to true, the node is not added to the list of children.
func dfsNode(stub cached_stub.CachedStubInterface, graphName string, currNodeId string, visited map[string]bool, filterRule interface{}) ([]string, map[string]bool, error) {
	// Make sure input parameters are valid
	if len(graphName) == 0 || len(currNodeId) == 0 {
		custom_err := errors.WithStack(&custom_errors.LengthCheckingError{})
		logger.Errorf("%v", custom_err)
		return nil, visited, custom_err
	}

	// cachekey -- use cache only if this is top level call, meaing visited = empty
	cachekey := ""
	if len(visited) == 0 {
		cachekey = getdfsNodeCacheKey(graphName, currNodeId, filterRule)
		// ckeck cache
		cache, err := stub.GetCache(cachekey)
		if err == nil {
			path, ok := cache.([]string)
			if ok {
				logger.Debugf("Getting from cache %v, %v", cachekey, path)
				return path, visited, nil
			}
		}
	}

	var nodes = []string{}
	// If curr node id has been visited, skip
	if visited[currNodeId] == true {
		return nil, visited, nil
	} else {
		// If curr node id has not been visited, set visited to true
		visited[currNodeId] = true
	}

	// get child edges by paritial composite key with curr node id
	iter, err := stub.GetStateByPartialCompositeKey(graphName, []string{currNodeId})
	if err != nil {
		custom_err := errors.WithStack(&custom_errors.GetLedgerError{LedgerKey: currNodeId, LedgerItem: "child edges"})
		logger.Errorf("%v: %v", custom_err, err)
		return nil, visited, errors.Wrap(err, custom_err.Error())
	}

	defer iter.Close()
	for iter.HasNext() {
		// examine next child edge
		KV, err := iter.Next()
		if err != nil {
			custom_err := errors.WithStack(&custom_errors.IterError{})
			logger.Errorf("%v: %v", custom_err, err)
			continue
			//return nil, visited, errors.Wrap(err, custom_err.Error())
		}

		item := KV.GetKey()
		_, attributes, err := stub.SplitCompositeKey(item)
		if err != nil {
			custom_err := errors.WithStack(&custom_errors.SplitCompositeKeyError{Key: item})
			logger.Errorf("%v: %v", custom_err, err)
			continue
			//return nil, visited, errors.Wrap(err, custom_err.Error())
		}

		if len(attributes) < 2 {
			custom_err := errors.WithStack(&custom_errors.LengthCheckingError{"composite key attributes"})
			logger.Errorf("%v", custom_err)
			continue
			//return nil, visited, custom_err
		}

		nextNodeId := attributes[1]

		//skip value
		//evaluate the skip rule againsty edge data and
		//if result == true, don't add node
		skip := false
		if filterRule != nil {
			var edge Edge
			edgeBytes := KV.GetValue()
			json.Unmarshal(edgeBytes, &edge)
			edgeData, err := json.Marshal(&edge.EdgeData)
			if err == nil {
				rule := simple_rule.NewRule(filterRule)
				m, e := rule.Apply(string(edgeData))
				//logger.Infof("%v %v", m,e)
				if e == nil {
					if m["$result"] == simple_rule.D(true) {
						skip = true
					}
				}
			}
		}

		if !skip && visited[nextNodeId] != true {
			nodes = append(nodes, nextNodeId)
		}

		// call dfs recursively on child
		found, visited2, err := dfsNode(stub, graphName, nextNodeId, visited, filterRule)
		visited = visited2
		if err != nil {
			logger.Errorf("Error calling DFS with child edge targetNodeId: %v", err)
			//return nil, visited, errors.Wrap(err, "Error calling DFS with child edge targetNodeId")
			continue
		}

		// When targetNodeId has been found
		if found != nil {
			// return the result of previous call with this call to construct path
			nodes = append(nodes, found...)
		}
	}
	// return all nodes found
	if len(cachekey) > 0 {
		stub.PutCache(cachekey, nodes)
		//logger.Debugf("put cache %v, %v, %v", currNodeId, nodes, visited)
	}
	return nodes, visited, nil
}

// bfs searches for nodes --NEVER USE THIS FUNCTION!!!!!!!!!!!!!!!!!!!!!!!!!
func bfs(stub cached_stub.CachedStubInterface, graphName string, nodeId string) ([]string, error) {
	// Make sure input parameters are valid
	if len(graphName) == 0 || len(nodeId) == 0 {
		custom_err := errors.WithStack(&custom_errors.LengthCheckingError{})
		logger.Errorf("%v", custom_err)
		return nil, custom_err
	}

	// Use a queue to control which node to visit
	queue := []string{}
	// Append start node id to queue
	queue = append(queue, nodeId)
	// map to detect cycle
	visited := make(map[string]bool)
	// mark start node id as visited
	visited[nodeId] = true
	// keep track of index of queue
	index := 0

	// while queue is not empty
	for len(queue) > 0 && len(queue) > index {
		// current node id is the first item in queue
		currNodeId := queue[index]
		index++

		// get nodes of current node id
		nodes, err := getNeighbors(stub, graphName, currNodeId)
		if err != nil {
			custom_err := errors.WithStack(&custom_errors.GetNodesError{})
			logger.Errorf("%v", custom_err)
			//return nil, errors.Wrap(err, custom_err.Error())
			continue
		}

		for _, node := range nodes {
			if visited[node] == true {
				continue
			} else {
				queue = append(queue, node)

				// set node to visited
				visited[node] = true
			}
		}
	}

	// return
	return queue[1:], nil
}

// getNeighbors is a helper function that gets children of the given node id.
func getNeighbors(stub cached_stub.CachedStubInterface, graphName string, nodeId string) ([]string, error) {

	// check cache
	valueCache, err := getCacheNeighbors(stub, graphName, nodeId)
	if err == nil {
		return valueCache, nil
	}

	// This is the list of neighbors we will return
	var neighbors []string

	// create partial composite key with node id and direction
	iter, err := stub.GetStateByPartialCompositeKey(graphName, []string{nodeId})
	if err != nil {
		custom_err := errors.WithStack(&custom_errors.GetLedgerError{LedgerKey: nodeId, LedgerItem: "neighbors"})
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	defer iter.Close()
	for iter.HasNext() {
		KV, err := iter.Next()
		if err != nil {
			custom_err := errors.WithStack(&custom_errors.IterError{})
			logger.Errorf("%v: %v", custom_err, err)
			return nil, errors.Wrap(err, custom_err.Error())
		}

		item := KV.GetKey()

		_, attributes, err := stub.SplitCompositeKey(item)
		if err != nil {
			custom_err := errors.WithStack(&custom_errors.SplitCompositeKeyError{Key: item})
			logger.Errorf("%v: %v", custom_err, err)
			return nil, errors.Wrap(err, custom_err.Error())
		}

		if len(attributes) < 2 {
			custom_err := errors.WithStack(&custom_errors.LengthCheckingError{"composite key attributes"})
			logger.Errorf("%v", custom_err)
			return nil, custom_err
		}

		neighbor := attributes[1]

		// found neighbor, append to list
		neighbors = append(neighbors, neighbor)
	}
	putCacheNeighbors(stub, graphName, nodeId, neighbors)
	return neighbors, nil
}

func getCacheKeyNeighbors(graphName string, nodeId string) string {
	return graphName + "neighboer_" + nodeId
}

func putCacheNeighbors(stub cached_stub.CachedStubInterface, graphName string, nodeId string, neighbors []string) error {
	cachekey := getCacheKeyNeighbors(graphName, nodeId)
	pathCopy := make([]string, len(neighbors))
	copy(pathCopy, neighbors)
	return stub.PutCache(cachekey, pathCopy)
}

func getCacheNeighbors(stub cached_stub.CachedStubInterface, graphName string, nodeId string) ([]string, error) {

	cachekey := getCacheKeyNeighbors(graphName, nodeId)
	keyCache, err := stub.GetCache(cachekey)
	if err != nil {
		return nil, err
	}
	if value, ok := keyCache.([]string); ok {
		valueCopy := make([]string, len(value))
		copy(valueCopy, value)
		logger.Debugf("Get neighbors from cache: %v %v", graphName, nodeId)
		return valueCopy, nil
	} else {
		return nil, errors.New("Invalid cache value")
	}
}

// SlowFindPath finds the first path in the graph from startNodeId to targetNodeId.
// Uses recursive DFS.
// If no path is found, returns nil.
func SlowFindPath(stub cached_stub.CachedStubInterface, graphName string, startNodeId string, targetNodeId string, filterRule ...interface{}) ([]string, error) {
	// Make sure input parameters are valid
	if len(graphName) == 0 || len(startNodeId) == 0 || len(targetNodeId) == 0 {
		custom_err := errors.WithStack(&custom_errors.LengthCheckingError{})
		logger.Errorf("%v", custom_err)
		return nil, custom_err
	}

	var filter interface{}
	if len(filterRule) == 0 {
		filter = nil
	} else {
		filter = filterRule[0]
	}

	// ckeck cache
	// cache is saved from dfs, and don't need to save in this fucntion
	cachekey := getFindPathCacheKey(graphName, startNodeId, targetNodeId, filter)
	cache, err := stub.GetCache(cachekey)
	if err == nil {
		path, ok := cache.([]string)
		if ok {
			logger.Debugf("Getting from cache %v", cachekey)
			if len(path) == 0 {
				path = nil
			}
			return path, nil
		}
	}

	// Visited map to keep track of visited edges
	visited := make(map[string]bool)
	// Call DFS recursive search helper function
	path, _, err := dfs(stub, graphName, startNodeId, targetNodeId, visited, filter)
	if err != nil {
		logger.Errorf("Error finding path: %v", err)
		return nil, errors.Wrap(err, "Error finding path")
	}
	// backward compatible
	if len(path) == 0 {
		path = nil
	}
	return path, nil
}

func getFindPathCacheKey(graphName string, startKey string, targetKey string, filter interface{}) string {
	filterStr := ""
	if filter != nil {
		rule := simple_rule.NewRule(filter)
		filterStr = rule.GetExprJSON()
	}
	return graphName + "-verify-" + startKey + "-" + targetKey + "-" + filterStr
}

// dfs is a depth first search helper function to find path.
func dfs(stub cached_stub.CachedStubInterface, graphName string, currNodeId string, targetNodeId string, visited map[string]bool, filterRule interface{}) ([]string, map[string]bool, error) {

	// If curr node id has been visited, skip
	if visited[currNodeId] == true {
		return nil, visited, nil
	} else {
		// If curr node id has not been visited, set visited to true
		visited[currNodeId] = true
	}

	// if found, return
	if currNodeId == targetNodeId {
		return []string{currNodeId}, visited, nil
	}

	// ckeck cache
	cachekey := getFindPathCacheKey(graphName, currNodeId, targetNodeId, filterRule)
	cache, err := stub.GetCache(cachekey)
	if err == nil {
		path, ok := cache.([]string)
		if ok {
			logger.Debugf("Getting from cache %v", cachekey)
			return path, visited, nil
		}
	}

	// get child edges by paritial composite key with curr node id
	iter, err := stub.GetStateByPartialCompositeKey(graphName, []string{currNodeId})
	if err != nil {
		custom_err := errors.WithStack(&custom_errors.GetLedgerError{LedgerKey: currNodeId, LedgerItem: "child edges"})
		logger.Errorf("%v: %v", custom_err, err)
		return nil, visited, errors.Wrap(err, custom_err.Error())
	}

	defer iter.Close()
	for iter.HasNext() {
		// examine next child edge
		KV, err := iter.Next()
		if err != nil {
			custom_err := errors.WithStack(&custom_errors.IterError{})
			logger.Errorf("%v: %v", custom_err, err)
			continue
			//return nil, visited, errors.Wrap(err, custom_err.Error())
		}

		item := KV.GetKey()
		_, attributes, err := stub.SplitCompositeKey(item)
		if err != nil {
			custom_err := errors.WithStack(&custom_errors.SplitCompositeKeyError{Key: item})
			logger.Errorf("%v: %v", custom_err, err)
			continue
			//return nil, visited, errors.Wrap(err, custom_err.Error())
		}

		if len(attributes) < 2 {
			custom_err := errors.WithStack(&custom_errors.LengthCheckingError{"composite key attributes"})
			logger.Errorf("%v", custom_err)
			continue
			//return nil, visited, custom_err
		}

		//skip rule
		//evaluate the skip rule againsty edge data and
		//if result == true skip the dege
		if filterRule != nil {
			var edge Edge
			edgeBytes := KV.GetValue()
			json.Unmarshal(edgeBytes, &edge)
			edgeData, err := json.Marshal(&edge.EdgeData)
			if err == nil {
				rule := simple_rule.NewRule(filterRule)
				m, e := rule.Apply(string(edgeData))
				logger.Debugf("Filter rule: %v %v %v", filterRule, m, e)
				if e == nil {
					if m["$result"] == simple_rule.D(true) {
						//logger.Info("skip")
						continue
					}
				}
			}
		}

		nextNodeId := attributes[1]

		// call dfs recursively on child
		found, visited2, err := dfs(stub, graphName, nextNodeId, targetNodeId, visited, filterRule)
		visited = visited2
		if err != nil {
			logger.Errorf("Error calling DFS with child edge targetNodeId: %v", err)
			//return nil, visited, errors.Wrap(err, "Error calling DFS with child edge targetNodeId")
			continue
		}

		// When targetNodeId has been found
		if len(found) > 0 {
			// save to cache
			found = append([]string{currNodeId}, found...)
			stub.PutCache(cachekey, found)

			// return the result of previous call with this call to construct path
			return found, visited, nil
		}
	}

	// No child edges or no path has been found
	// save to cache
	stub.PutCache(cachekey, []string{})
	return nil, visited, nil
}

// GetFirstEdgeContainingNodeId returns the first edge value of edge that contains node id.
func GetFirstEdgeContainingNodeId(stub cached_stub.CachedStubInterface, graphName string, nodeId string) ([]byte, error) {
	var return_error error
	iter, err := stub.GetStateByPartialCompositeKey(graphName, []string{nodeId})
	if err != nil {
		custom_err := errors.WithStack(&custom_errors.GetLedgerError{LedgerKey: nodeId, LedgerItem: "child edges"})
		logger.Errorf("%v: %v", custom_err, err)
		return_error = errors.Wrap(err, custom_err.Error())
	} else {

		defer iter.Close()
		for iter.HasNext() {
			// examine first child edge
			KV, err := iter.Next()
			if err != nil {
				custom_err := errors.WithStack(&custom_errors.IterError{})
				logger.Errorf("%v: %v", custom_err, err)
				return_error = errors.Wrap(err, custom_err.Error())
			} else {
				return KV.GetValue(), nil
			}
		}
	}
	// REVERSE LOOKUP
	iter, err = stub.GetStateByPartialCompositeKey(global.REVERSE_GRAPH_PREFIX+graphName, []string{nodeId})
	if err != nil {
		custom_err := errors.WithStack(&custom_errors.GetLedgerError{LedgerKey: nodeId, LedgerItem: "parent edges"})
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	defer iter.Close()
	for iter.HasNext() {
		// examine first child edge
		KV, err := iter.Next()
		if err != nil {
			custom_err := errors.WithStack(&custom_errors.IterError{})
			logger.Errorf("%v: %v", custom_err, err)
			return_error = errors.Wrap(err, custom_err.Error())
		} else {
			return KV.GetValue(), nil
		}
	}

	return nil, return_error
}
