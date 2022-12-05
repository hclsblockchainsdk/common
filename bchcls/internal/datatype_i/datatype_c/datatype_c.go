/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package datatype manages datatypes and their relationships.
// Each asset has a list of datatypes. This allows all assets of datatype "Medical Records," for
// example, to be shared through a single consent.
package datatype_c

import (
	"common/bchcls/cached_stub"
	"common/bchcls/custom_errors"
	"common/bchcls/data_model"
	"common/bchcls/datatype/datatype_interface"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/common/graph"
	"common/bchcls/internal/metering_i"
	"common/bchcls/utils"
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("datatype_c")

// global.ROOT_DATATYPE_ID is id of ROOT datatype. All other datatypes are children of ROOT.
const ROOT_DATATYPE_ID = global.ROOT_DATATYPE_ID

// Prefix for Asset Cache
const DATATYPE_CACHE_PREFIX = "datatypeCache_"

// Datatype represents a type that can be used to classify assets.
// Datatypes are stored in a tree structure. Datatypes can have sub-datatypes.
type datatypeImpl struct {
	DatatypeID  string `json:"datatype_id"`
	Description string `json:"description"`
	Active      bool   `json:"acive"`
	deactivated bool
}

// ------------------------------------------------------
// ---------------------- INIT FUNCTIONS ----------------
// ------------------------------------------------------
// Init sets up the datatype package by registering a default ROOT datatype.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	logger.Debug("Init datatype")
	//register ROOT datatpe
	rootDatatype, err := GetDatatypeWithParams(stub, ROOT_DATATYPE_ID)
	if err != nil || rootDatatype.GetDatatypeID() != ROOT_DATATYPE_ID {
		_, err := RegisterDatatypeWithParams(stub, ROOT_DATATYPE_ID, "ROOT", true, "")
		return nil, err
	}
	return nil, nil
}

// ------------------------------------------------------
// -------- Datatype Interface Functions ----------------
// ------------------------------------------------------

// GetJSON returns JSON representation of datatype
func (datatype *datatypeImpl) GetDatatypeStruct() data_model.Datatype {
	json := data_model.Datatype{}
	json.DatatypeID = datatype.DatatypeID
	json.Description = datatype.Description
	json.IsActive = datatype.Active
	return json
}

// GetDatatypeID returns DatatypeID
func (datatype *datatypeImpl) GetDatatypeID() string {
	return datatype.DatatypeID
}

// GetDescription returns Description
func (datatype *datatypeImpl) GetDescription() string {
	return datatype.Description
}

// SetDescription sets Description
func (datatype *datatypeImpl) SetDescription(description string) error {
	datatype.Description = description
	return nil
}

// IsActive returns true if the datatype is active
func (datatype *datatypeImpl) IsActive() bool {
	return datatype.Active
}

//Activate() changes state to active if it's not already in active state
//It does not return error if the state is already in ative state
func (datatype *datatypeImpl) Activate() error {
	datatype.Active = true
	return nil
}

//Deactivate() changes state to inactive if it's in active state
//It does not return error if the state is already in inative state
//Also it will deactivate all children datatypes during PutDatatype
func (datatype *datatypeImpl) Deactivate() error {
	if datatype.Active {
		datatype.deactivated = true
	}
	datatype.Active = false

	return nil
}

// [Deprecated and removed from DatatypeInterface] keeping here as a private function
// removeDatatype removes datatype from the ledger, removes its relationships to other datatypes in the graph, and adds relationships between its children and its parents.
// It will also update datatype symkey encryptions
// Caller must have access to all datatype symkeys
// This is a very slow operation, and it should only be used with caution
// Also it's callers responsibility to remove any reference to this datatype
func (datatype *datatypeImpl) removeDatatype(stub cached_stub.CachedStubInterface, caller data_model.User) error {
	// Remove from ledger
	datatypeLedgerKey, _ := stub.CreateCompositeKey(global.DATATYPE_PREFIX, []string{datatype.DatatypeID})
	err := stub.DelState(datatypeLedgerKey)
	if err != nil {
		custom_err := &custom_errors.DeleteLedgerError{LedgerKey: datatypeLedgerKey}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	// Remove relationships from graph
	// Get parent
	var parent datatype_interface.DatatypeInterface
	parents, err := graph.GetDirectParents(stub, global.DATATYPE_GRAPH, datatype.DatatypeID)
	// since we do not allow multiple parents, return error if length of result is > 1
	if err != nil || len(parents) > 1 {
		custom_err := &custom_errors.GetDirectParentsError{Child: datatype.DatatypeID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.WithStack(custom_err)
	}

	// this datatype has a parent
	if len(parents) == 1 {
		parent, err = GetDatatypeWithParams(stub, parents[0])
		if err != nil {
			custom_err := &custom_errors.GetDatatypeError{Datatype: parents[0]}
			logger.Errorf("%v: %v", custom_err, err)
			return errors.Wrap(err, custom_err.Error())
		}
		if utils.IsStringEmpty(parent.GetDatatypeID()) {
			custom_err := &custom_errors.GetDatatypeError{Datatype: parent.GetDatatypeID()}
			logger.Errorf("%v: %v", custom_err, err)
			return errors.WithStack(custom_err)
		}
		// remove relationship with parent
		err = graph.DeleteEdge(stub, global.DATATYPE_GRAPH, parent.GetDatatypeID(), datatype.DatatypeID)
		if err != nil {
			custom_err := &custom_errors.RemoveRelationshipError{Parent: parent.GetDatatypeID(), Child: datatype.DatatypeID}
			logger.Errorf("%v: %v", custom_err, err)
			return errors.Wrap(err, custom_err.Error())
		}
	}

	// Get its direct children
	childrenDatatypeIDs, err := graph.GetDirectChildren(stub, global.DATATYPE_GRAPH, datatype.DatatypeID)
	if err != nil {
		custom_err := &custom_errors.GetDirectChildrenError{Parent: datatype.DatatypeID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	// if has children
	if len(childrenDatatypeIDs) > 0 {
		for _, childDatatypeId := range childrenDatatypeIDs {
			child, err := GetDatatypeWithParams(stub, childDatatypeId)
			if err != nil {
				custom_err := &custom_errors.GetDatatypeError{Datatype: childDatatypeId}
				logger.Errorf("%v: %v", custom_err, err)
				return errors.Wrap(err, custom_err.Error())
			}
			if utils.IsStringEmpty(child.GetDatatypeID()) {
				custom_err := &custom_errors.GetDatatypeError{Datatype: parent.GetDatatypeID()}
				logger.Errorf("%v: %v", custom_err, err)
				return errors.WithStack(custom_err)
			}

			// remove child relationships
			err = graph.DeleteEdge(stub, global.DATATYPE_GRAPH, datatype.DatatypeID, child.GetDatatypeID())

			if err != nil {
				custom_err := &custom_errors.RemoveRelationshipError{Parent: datatype.GetDatatypeID(), Child: child.GetDatatypeID()}
				logger.Errorf("%v: %v", custom_err, err)
				return errors.Wrap(err, custom_err.Error())
			}

			if len(parents) == 1 {
				// add relationship from each child to parent
				err = graph.PutEdge(stub, global.DATATYPE_GRAPH, parent.GetDatatypeID(), child.GetDatatypeID())
				if err != nil {
					custom_err := &custom_errors.AddRelationshipError{Parent: parent.GetDatatypeID(), Child: child.GetDatatypeID()}
					logger.Errorf("%v: %v", custom_err, err)
					return errors.Wrap(err, custom_err.Error())
				}
			}
		}
	}

	return nil
}

// PutDatatype saves updated datatype to the ledger
func (datatype *datatypeImpl) PutDatatype(stub cached_stub.CachedStubInterface) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("PutDatatype %v", datatype)

	_ = metering_i.SetEnvAndAddRow(stub)

	datatypeLedgerKey, err := stub.CreateCompositeKey(global.DATATYPE_PREFIX, []string{datatype.DatatypeID})
	if err != nil {
		custom_err := &custom_errors.CreateCompositeKeyError{Type: global.DATATYPE_PREFIX}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	datatypeBytes, err := json.Marshal(&datatype)
	if err != nil {
		custom_err := &custom_errors.MarshalError{Type: "datatype"}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	err = stub.PutState(datatypeLedgerKey, datatypeBytes)
	if err != nil {
		custom_err := &custom_errors.PutLedgerError{LedgerKey: datatypeLedgerKey}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	// deactivate child datatypes
	if !datatype.IsActive() && datatype.deactivated {
		childDatatypes, err := datatype.GetChildDatatypes(stub)
		if err != nil {
			logger.Errorf("Getting child datatypes failed: %v", err)
			return err
		}
		for _, childDatatypeID := range childDatatypes {
			childDatatype, err := GetDatatypeWithParams(stub, childDatatypeID)
			if err != nil {
				logger.Errorf("Getting child datatype failed: %v", err)
				return err
			}
			if childDatatype.IsActive() {
				childDatatype.Deactivate()
				err := childDatatype.PutDatatype(stub)
				if err != nil {
					logger.Errorf("Decativating child datatype failed: %v", err)
					return err
				}
			}
		}
	}
	return nil
}

// GetChildDatatypes returns a list of ids for child datatypes of datatype.
func (datatype *datatypeImpl) GetChildDatatypes(stub cached_stub.CachedStubInterface) ([]string, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	return graph.SlowGetChildren(stub, global.DATATYPE_GRAPH, datatype.DatatypeID)
}

// GetParentDatatypeID returns ID for parent datatypes of datatype.
func (datatype *datatypeImpl) GetParentDatatypeID(stub cached_stub.CachedStubInterface) (string, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	parents, err := graph.GetDirectParents(stub, global.DATATYPE_GRAPH, datatype.DatatypeID)
	if err != nil {
		return "", err
	}
	if len(parents) == 0 || utils.IsStringEmpty(parents[0]) {
		return "", nil
	}
	return parents[0], nil
}

// GetParentDatatypes returns a list of parents of the given datatypeID, excluding ROOT datatype.
func (datatype *datatypeImpl) GetParentDatatypes(stub cached_stub.CachedStubInterface) ([]string, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	parents, err := graph.SlowGetParents(stub, global.DATATYPE_GRAPH, datatype.DatatypeID)
	if err != nil {
		return []string{}, err
	}

	return utils.RemoveItemFromList(parents, ROOT_DATATYPE_ID), nil
}

// IsParentOf returns true or false based on whether child is in the child chain of datatype.
func (datatype *datatypeImpl) IsParentOf(stub cached_stub.CachedStubInterface, childDatatypeID string) (bool, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	// Calls Graph.FindPath from parent to child
	// if length of result is greater than 0, return true; else return false
	path, err := graph.SlowFindPath(stub, global.DATATYPE_GRAPH, datatype.DatatypeID, childDatatypeID)
	if err != nil {
		var errMsg = "Error calling FindPath()"
		logger.Errorf("%v: %v", errMsg, err)
		return false, errors.Wrap(err, errMsg)
	}
	if len(path) > 0 {
		return true, nil
	}
	return false, nil
}

// IsChildOf returns true or false based on whether parent is in the parent chain of datatype.
func (datatype *datatypeImpl) IsChildOf(stub cached_stub.CachedStubInterface, parentDatatypeID string) (bool, error) {

	_ = metering_i.SetEnvAndAddRow(stub)

	// Calls Graph.FindPath from child to parent, user REVERSE_DatatypeGraph
	// if length of result is greater than 0, return true; else return false
	path, err := graph.SlowFindPath(stub, global.DATATYPE_GRAPH, parentDatatypeID, datatype.DatatypeID)
	if err != nil {
		var errMsg = "Error calling FindPath()"
		logger.Errorf("%v: %v", errMsg, err)
		return false, errors.Wrap(err, errMsg)
	}
	if len(path) > 0 {
		return true, nil
	}
	return false, nil
}

// GetDatatypeKeyID returns the datatype symkey ID for the owner
func (datatype *datatypeImpl) GetDatatypeKeyID(ownerID string) string {
	return GetDatatypeKeyID(datatype.DatatypeID, ownerID)
}

// ------------------------------------------------------
// -------------------- Helper Functions ----------------
// ------------------------------------------------------

// RegisterDatatype registers a new datatype to the ledger.
// Maintains datatype tree structure.
// Assumes a ROOT datatype exists. Every solution must call init_common.Init, which saves the ROOT datatype to the ledger.
// Creates datatypeSymKey and maintains key relationship with parent datatypes.
// If parentDatatypeID is not provided or does not exist, the datatype will be added as a child of ROOT.
//
// args = [ datatype, parentDatatypeID ]
func RegisterDatatype(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("RegisterDatatype args: %v", args)

	// parse args
	if len(args) != 1 && len(args) != 2 {
		custom_err := &custom_errors.LengthCheckingError{Type: "RegisterDatatype args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	// check caller
	if caller.Role != global.ROLE_SYSTEM_ADMIN && caller.Role != global.ROLE_ORG && caller.Role != global.ROLE_USER {
		logger.Errorf("Caller does not have permission to RegisterDatatype")
		return nil, errors.New("Caller does not have permission to RegisterDatatype")
	}

	datatype := data_model.Datatype{}
	datatypeBytes := []byte(args[0])
	err := json.Unmarshal(datatypeBytes, &datatype)
	if err != nil {
		custom_err := &custom_errors.UnmarshalError{Type: "Datatype"}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}
	if utils.IsStringEmpty(datatype.DatatypeID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "datatypeID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	parentDatatypeID := global.ROOT_DATATYPE_ID
	if len(args) == 2 && args[1] != global.ROOT_DATATYPE_ID {
		parentDatatypeID = args[1]
	}

	existingDatatype, err := GetDatatypeWithParams(stub, datatype.DatatypeID)
	if err != nil {
		custom_err := &custom_errors.GetDatatypeError{Datatype: datatype.DatatypeID}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}
	if !utils.IsStringEmpty(existingDatatype.GetDatatypeID()) {
		logger.Errorf("Failed to RegisterDatatype because this id already exists")
		return nil, errors.New("Failed to RegisterDatatype because this id already exists")
	}

	_, err = RegisterDatatypeWithParams(stub, datatype.DatatypeID, datatype.Description, datatype.IsActive, parentDatatypeID)
	if err != nil {
		logger.Errorf("Failed to RegisterDatatypeWithParams: %v", err)
		return nil, errors.Wrap(err, "Failed to RegisterDatatypeWithParams")
	}

	return nil, nil
}

// RegisterDatatypeWithParams saves a new datatype to the ledger and maintains datatype tree structure.
// Caller must pass in datatype and can optionally pass in parentDatatypeID.
// If parentDatatypeID is not provided, the datatype will be added as a child of ROOT.
// If parentDatatypeID does not exist, an error is thrown
// It returns a new DatatypeInterface instance that has been registered
func RegisterDatatypeWithParams(stub cached_stub.CachedStubInterface, datatypeID, description string, isActive bool, parentDatatypeID string) (datatype_interface.DatatypeInterface, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("RegisterDatatypeWithParams ID:%v Active:%v ParentID:%v", datatypeID, isActive, parentDatatypeID)
	// get parent datatype
	if datatypeID == ROOT_DATATYPE_ID {
		parentDatatypeID = ""
	} else {
		if utils.IsStringEmpty(parentDatatypeID) {
			parentDatatypeID = ROOT_DATATYPE_ID
		}
		parentDatatype, err := GetDatatypeWithParams(stub, parentDatatypeID)
		if err != nil {
			custom_err := &custom_errors.GetDatatypeError{Datatype: parentDatatypeID}
			logger.Errorf("%v: %v", custom_err, err)
			return nil, errors.Wrap(err, custom_err.Error())
		}
		if utils.IsStringEmpty(parentDatatype.GetDatatypeID()) {
			custom_err := &custom_errors.GetDatatypeError{Datatype: parentDatatypeID}
			logger.Errorf("%v", custom_err)
			return nil, custom_err
		}
		//if parent datatype is inactive, you can't add an active datatype
		if !parentDatatype.IsActive() && isActive {
			logger.Error("You cannot add active datatype under an inactive parent datatype")
			return nil, errors.New("You cannot add active datatype under an inactive parent datatype")
		}
	}

	// check for existing datatype
	existingDatatype, err := GetDatatypeWithParams(stub, datatypeID)
	if err != nil {
		custom_err := &custom_errors.GetDatatypeError{Datatype: datatypeID}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}
	if !utils.IsStringEmpty(existingDatatype.GetDatatypeID()) {
		custom_err := errors.New("Datatype already exist: " + datatypeID)
		logger.Errorf("%v", custom_err)
		return nil, custom_err
	}

	// save to ledger
	datatypeLedgerKey, err := stub.CreateCompositeKey(global.DATATYPE_PREFIX, []string{datatypeID})
	if err != nil {
		custom_err := &custom_errors.CreateCompositeKeyError{Type: global.DATATYPE_PREFIX}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	datatypeInternal := datatypeImpl{}
	datatypeInternal.DatatypeID = datatypeID
	datatypeInternal.Active = isActive
	datatypeInternal.Description = description
	datatypeInternal.deactivated = false

	datatypeBytes, err := json.Marshal(&datatypeInternal)
	if err != nil {
		custom_err := &custom_errors.MarshalError{Type: "datatype"}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	err = stub.PutState(datatypeLedgerKey, datatypeBytes)
	if err != nil {
		custom_err := &custom_errors.PutLedgerError{LedgerKey: datatypeLedgerKey}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	//add relationship
	if len(parentDatatypeID) > 0 {
		err := graph.PutEdge(stub, global.DATATYPE_GRAPH, parentDatatypeID, datatypeID)
		if err != nil {
			logger.Errorf("Failed to add relationship: %v", err)
			return nil, err
		}
	}

	return &datatypeInternal, nil
}

// GetDatatypeKeyID returns the datatype key id associated with a datatype.
func GetDatatypeKeyID(datatypeID string, ownerID string) string {
	return global.KEY_TYPE_SYM + "-" + ownerID + "-" + datatypeID
}

// UpdateDatatype updates existing datatype's description.
// Caller's role must be "system".
//
// args = [ datatype ]
func UpdateDatatype(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("UpdateDatatype args: %v", args)

	// parse args
	if len(args) != 1 {
		custom_err := &custom_errors.LengthCheckingError{Type: "UpdateDatatype args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	// check caller
	if caller.Role != global.ROLE_SYSTEM_ADMIN && caller.Role != global.ROLE_ORG && caller.Role != global.ROLE_USER {
		logger.Errorf("Caller does not have permission to UpdateDatatype")
		return nil, errors.New("Caller does not have permission to UpdateDatatype")
	}

	datatype := datatypeImpl{}
	datatypeBytes := []byte(args[0])
	err := json.Unmarshal(datatypeBytes, &datatype)
	if err != nil {
		custom_err := &custom_errors.UnmarshalError{Type: "Datatype"}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}
	if utils.IsStringEmpty(datatype.DatatypeID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "datatypeID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	// get existing datatype
	existingDatatype, err := GetDatatypeWithParams(stub, datatype.DatatypeID)
	if err != nil {
		custom_err := &custom_errors.GetDatatypeError{Datatype: datatype.DatatypeID}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}
	if utils.IsStringEmpty(existingDatatype.GetDatatypeID()) {
		custom_err := &custom_errors.LengthCheckingError{Type: "datatypeID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	// update description if changed
	if existingDatatype.GetDescription() != datatype.Description {
		existingDatatype.SetDescription(datatype.Description)

		// save updated datatype to ledger
		existingDatatype.PutDatatype(stub)
	}

	return nil, nil
}

// GetDatatype returns a datatype with the given datatypeID.
// Returns an empty datatype if the datatypeID passed in does not exist.
//
// args = [ datatypeID ]
func GetDatatype(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("GetDatatype args: %v", args)

	// parse args
	if len(args) != 1 {
		custom_err := &custom_errors.LengthCheckingError{Type: "GetDatatype args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	datatypeID := args[0]
	if utils.IsStringEmpty(datatypeID) {
		custom_err := &custom_errors.LengthCheckingError{Type: "datatypeID"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}

	datatype, err := GetDatatypeWithParams(stub, datatypeID)
	if err != nil {
		logger.Errorf("Failed to GetDatatypeWithParams")
		return nil, errors.Wrap(err, "Failed to GetDatatypeWithParams")
	}

	datatypeBytes, err := json.Marshal(datatype.GetDatatypeStruct())
	if err != nil {
		custom_err := &custom_errors.MarshalError{Type: "datatype"}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}
	return datatypeBytes, nil
}

// GetDatatypeWithParams function gets a datatype from the ledger.
// Returns an empty datatype if the datatypeId passed in does not exist.
func GetDatatypeWithParams(stub cached_stub.CachedStubInterface, datatypeID string) (datatype_interface.DatatypeInterface, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("GetDatatypeWithParams ID:%v", datatypeID)
	datatype := datatypeImpl{}
	datatypeLedgerKey, _ := stub.CreateCompositeKey(global.DATATYPE_PREFIX, []string{datatypeID})
	datatypeBytes, err := stub.GetState(datatypeLedgerKey)
	if err != nil {
		custom_err := &custom_errors.GetLedgerError{LedgerKey: datatypeLedgerKey, LedgerItem: "Datatype"}
		logger.Errorf("%v: %v", custom_err, err)
		return &datatype, errors.Wrap(err, custom_err.Error())
	} else if datatypeBytes == nil {
		logger.Infof("Datatype not found with ledger key: \"%v\"", datatypeLedgerKey)
		return &datatype, nil
	}
	err = json.Unmarshal(datatypeBytes, &datatype)
	if err != nil {
		custom_err := &custom_errors.UnmarshalError{Type: "Datatype"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return &datatype, errors.Wrap(err, custom_err.Error())
	}
	return &datatype, nil
}

// GetAllDatatypes returns all datatypes, not including the ROOT datatype.
func GetAllDatatypes(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	// get root datatype
	rootDatatype, err := GetDatatypeWithParams(stub, global.ROOT_DATATYPE_ID)
	if err != nil {
		custom_err := &custom_errors.GetDatatypeError{Datatype: global.ROOT_DATATYPE_ID}
		logger.Errorf("%v: %v", custom_err, err)
		return nil, errors.Wrap(err, custom_err.Error())
	}
	// get root's child datatype ids
	datatypeIds, err := rootDatatype.GetChildDatatypes(stub)
	if err != nil {
		logger.Errorf("Failed to GetChildDatatypes: %v", err)
		return nil, errors.Wrap(err, "Failed to GetChildDatatypes")
	}
	// get root's child datatypes
	datatypes := []data_model.Datatype{}
	for _, datatypeId := range datatypeIds {
		datatype, err := GetDatatypeWithParams(stub, datatypeId)
		if err != nil {
			custom_err := &custom_errors.GetDatatypeError{Datatype: datatypeId}
			logger.Errorf("%v: %v", custom_err, err)
			return nil, errors.Wrap(err, custom_err.Error())
		}
		datatypes = append(datatypes, datatype.GetDatatypeStruct())
	}
	return json.Marshal(datatypes)
}

// GetParentDatatype returns the first direct parent of the given datatypeID.
func GetParentDatatype(stub cached_stub.CachedStubInterface, datatypeID string) (string, error) {
	parents, err := graph.GetDirectParents(stub, global.DATATYPE_GRAPH, datatypeID)
	if err != nil {
		return "", err
	}
	if len(parents) == 0 || utils.IsStringEmpty(parents[0]) {
		return "", nil
	}
	return parents[0], nil
}

// GetParentDatatypes returns a list of parents of the given datatypeID, excluding ROOT datatype.
// The first element is the direct parent, and the second is the grandparent, etc.
func GetParentDatatypes(stub cached_stub.CachedStubInterface, datatypeID string) ([]string, error) {
	parents, err := graph.SlowGetParents(stub, global.DATATYPE_GRAPH, datatypeID)
	if err != nil {
		return []string{}, err
	}

	return utils.RemoveItemFromList(parents, ROOT_DATATYPE_ID), nil
}

// NormalizeDatatypes returns a list of normalized child datatype IDs.
func NormalizeDatatypes(stub cached_stub.CachedStubInterface, datatypeIDs []string) ([]string, error) {
	if len(datatypeIDs) == 0 {
		return datatypeIDs, nil
	}

	normalizedDatatypes := []string{}
	// for each datatype,
	for _, datatypeID := range datatypeIDs {
		// get child datatypes
		childIDs, err := graph.SlowGetChildren(stub, global.DATATYPE_GRAPH, datatypeID)
		if err != nil {
			logger.Errorf("Failed to GetChildren: %v", err)
			return nil, errors.Wrap(err, "Failed to GetChildren")
		}
		// if a datatype's child is in the list, do not add datatypeID
		keepDatatype := true
		for _, childID := range childIDs {
			if utils.InList(datatypeIDs, childID) {
				keepDatatype = false
				break
			}
		}
		// add datatypeID to list
		if keepDatatype {
			normalizedDatatypes = append(normalizedDatatypes, datatypeID)
		}
	}
	return normalizedDatatypes, nil
}
