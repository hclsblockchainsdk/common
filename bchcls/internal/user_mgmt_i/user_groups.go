/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package user_mgmt_i

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/custom_errors"
	"common/bchcls/data_model"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/common/graph"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/internal/user_mgmt_i/user_mgmt_c"
	"common/bchcls/utils"

	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/pkg/errors"
)

// ----------------- GROUP MEMBERSHIP FUNCTIONS -----------------

// PutUserInGroup adds the user as a member of the group.
// If the user is already a member of the group, then the admin status can be updated.
// Admins have read/write access to any assets that the group has read/write access to.
// Members have read access to assets that the group has read access to.
// userID must be the ID of a user, not a group.
// groupID must be the ID of a group, not a user.
// If isAdmin is true, user will be given write access to group assets.
// Caller must be an admin of the group in order to add members and admins.
// keyPaths are optional parameters. If passed in, they are used to get group's keys.
// The first keyPath is for getting the group symKey, and the second keyPath is for getting the group privateKey.
func PutUserInGroup(stub cached_stub.CachedStubInterface, caller data_model.User, userID string, groupID string, isAdmin bool, keyPaths ...[]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v userId: %v groupId: %v isAdmin: %v", caller.ID, userID, groupID, isAdmin)

	var symkeyPath []string = nil
	var privkeyPath []string = nil
	if len(keyPaths) > 0 {
		symkeyPath = keyPaths[0]
	}
	if len(keyPaths) > 1 {
		privkeyPath = keyPaths[1]
	}

	// Caller must be admin of group in order to add members/admins
	isCallerAdmin, adminPath, err := user_mgmt_c.IsUserAdminOfGroup(stub, caller.ID, groupID)
	if !isCallerAdmin {
		custom_err := &custom_errors.NotGroupAdminError{UserID: caller.ID, GroupID: groupID}
		logger.Errorf("%v", custom_err)
		return errors.WithStack(custom_err)
	}

	// get keyPath
	if len(privkeyPath) == 0 && len(symkeyPath) == 0 && len(adminPath) > 1 {
		privkeyPath, _ = ConvertAdminPathToPrivateKeyPath(adminPath)
		logger.Debugf("prikeyPath from admin path: %v", privkeyPath)

		symkeyPath, _ = ConvertAdminPathToSymKeyPath(adminPath)
		logger.Debugf("symkeyPath from admin path: %v", symkeyPath)
	}

	// Get the user & group from the ledger
	group, err := GetUserData(stub, caller, groupID, true, false, symkeyPath, privkeyPath)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: groupID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	// If you are admin you should have access to these keys
	if len(group.PrivateKeyB64) == 0 {
		logger.Errorf("Failed to get group's private key: %v", group.ID)
		return errors.New("Failed to get group's private key")
	}
	if len(group.SymKey) == 0 {
		logger.Errorf("Failed to get group's sym key: %v", group.ID)
		return errors.New("Failed to get group's sym key")
	}

	user, err := GetUserData(stub, caller, userID, false, false)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: userID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	// Group must be a group
	if !group.IsGroup {
		var errMsg = "GroupID cannot be user: " + group.ID
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	// User must NOT be a group
	if user.IsGroup {
		var errMsg = "User type cannot be group. Please use RegisterSubgroup() instead"
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	return putUserInGroup(stub, caller, user, group, isAdmin)
}

// putUserInGroup puts a user in a group once you already have the User objects.
func putUserInGroup(stub cached_stub.CachedStubInterface, caller, user, group data_model.User, isAdmin bool) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller:%v user: %v group: %v isAdmin: %v", caller.ID, user.ID, group.ID, isAdmin)
	// The following conditons must be checked by the caller of this function:
	// Caller must be admin of group in order to add members/admins
	// Group must be a group
	// User must NOT be a group

	// Give group member access to group sym key
	userPubKey := user.GetPublicKey()
	groupSymKey := group.GetSymKey()
	edgeData := make(map[string]string)
	edgeData["type"] = global.KEY_TYPE_SYM
	err := key_mgmt_i.AddAccess(stub, userPubKey, groupSymKey, edgeData)
	if err != nil {
		errMsg := "Failed saving " + group.ID + " symkey for " + user.ID
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}

	if isAdmin {
		// give user access to group private key hash
		groupPrivHash := group.GetPrivateKeyHashSymKey()
		err = key_mgmt_i.AddAccess(stub, userPubKey, groupPrivHash, edgeData)
		if err != nil {
			errMsg := "Failed saving access to " + group.ID + " private key for " + user.ID
			logger.Errorf("%v: %v", errMsg, err)
			return errors.Wrap(err, errMsg)
		}
	} else {
		// make sure there is no direct access to the group private key
		// this will not return an error or have any effect if no direct access exists
		err = key_mgmt_i.RevokeAccess(stub, user.GetPubPrivKeyId(), group.GetPrivateKeyHashSymKeyId())
		if err != nil {
			errMsg := "Failed to revoke access to " + group.ID + " private key for " + user.ID
			logger.Errorf("%v: %v", errMsg, err)
			return errors.Wrap(err, errMsg)
		}
	}

	// Determine edge value for user graph
	var edgeValue []byte
	if isAdmin {
		edgeValue = []byte(global.ADMIN_EDGE)
	} else {
		edgeValue = []byte(global.MEMBER_EDGE)
	}

	// Call PutEdge from graph package
	err = graph.PutEdge(stub, global.USER_GRAPH, group.ID, user.ID, edgeValue)
	if err != nil {
		custom_err := &custom_errors.PutEdgeError{ParentNode: group.ID, ChildNode: user.ID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}
	return nil
}

// RegisterSubgroup registers a new group as a subgroup of an existing group.
// Admins of the parent group are admins of the subgroup.
// Members of the subgroup are members of parent group.
// Auditors of the parent group are auditors of the subgroup.
// Subgroups can only have one parent group.
//
// args = [subgroup, parentGroupID]
//
// subgroup is the subgroup to be registered.
// parentGroupID is the id of the parent group.
func RegisterSubgroup(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	// Parse args
	if len(args) != 2 {
		custom_err := &custom_errors.LengthCheckingError{Type: "user_mgmt.RegisterSubgroup args"}
		logger.Errorf(custom_err.Error())
		return nil, errors.WithStack(custom_err)
	}
	// Parse subgroup object from args[0]
	subgroup := data_model.User{}
	subgroupBytes := []byte(args[0])
	err := json.Unmarshal(subgroupBytes, &subgroup)
	if err != nil {
		custom_err := &custom_errors.UnmarshalError{Type: "User object for subgroup"}
		logger.Errorf("%v: %v", custom_err.Error(), err)
		return nil, errors.Wrap(err, custom_err.Error())
	}

	// Set the subgroup's keys from B64
	// If any of these fail, they will return nil for the key. That will trigger an "invalid keys" error in RegisterSubgroupWithParams.
	subgroup.SymKey, _ = crypto.ParseSymKeyB64(subgroup.SymKeyB64)
	subgroup.PublicKey, _ = crypto.ParsePublicKeyB64(subgroup.PublicKeyB64)
	subgroup.PrivateKey, _ = crypto.ParsePrivateKeyB64(subgroup.PrivateKeyB64)

	// Parse parentGroupID from args[1]
	parentGroupID := args[1]
	if utils.IsStringEmpty(parentGroupID) {
		logger.Errorf("parentGroupID must be provided")
		return nil, errors.New("parentGroupID must be provided")
	}

	return nil, RegisterSubgroupWithParams(stub, caller, subgroup, parentGroupID)
}

// RegisterSubgroupWithParams registers a new group as a subgroup of an existing group.
// "WithParams" functions should only be called from within the chaincode.
//
// subgroup is the subgroup to register.
// parentGroupID is the id of the parent group.
// keyPaths (optional) keyPath to symkey, keyPath to privKey
func RegisterSubgroupWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, subgroup data_model.User, parentGroupID string, keyPaths ...[]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v, subgroup: %v, parentGroup: %v, keyPath: %v", caller.ID, subgroup.ID, parentGroupID, keyPaths)
	// keyPaths
	var symkeyPath []string = nil
	var privkeyPath []string = nil
	if len(keyPaths) > 0 {
		symkeyPath = keyPaths[0]
	}
	if len(keyPaths) > 1 {
		privkeyPath = keyPaths[1]
	}

	// Check if subgroup already exists. This is currently not allowed.
	existingGroup, err := GetUserData(stub, caller, subgroup.ID, false, false)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: subgroup.ID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}
	if len(existingGroup.ID) > 0 {
		logger.Errorf("Group/User with ID \"%v\" already exists", subgroup.ID)
		return errors.Errorf("Group/User with ID \"%v\" already exists", subgroup.ID)
	}

	// Check that the subgroup's keys were provided
	if subgroup.PrivateKey == nil || subgroup.PublicKey == nil || !crypto.ValidateSymKey(subgroup.SymKey) {
		logger.Errorf("Public, private, and sym keys must be provided to register a new subgroup.")
		return errors.New("Public, private, and sym keys must be provided to register a new subgroup.")
	}

	// Caller must be admin of group in order to add members/admins
	canAdd, adminPath, err := user_mgmt_c.IsUserAdminOfGroup(stub, caller.ID, parentGroupID)
	if !canAdd {
		custom_err := &custom_errors.NotGroupAdminError{UserID: caller.ID, GroupID: parentGroupID}
		logger.Errorf("%v", custom_err)
		return errors.WithStack(custom_err)
	}

	// get keyPath
	if len(privkeyPath) == 0 && len(symkeyPath) == 0 && len(adminPath) > 1 {
		privkeyPath, _ = ConvertAdminPathToPrivateKeyPath(adminPath)
		logger.Debugf("prikeyPath from admin path: %v", privkeyPath)

		symkeyPath, _ = ConvertAdminPathToSymKeyPath(adminPath)
		logger.Debugf("symkeyPath from admin path: %v", symkeyPath)
	}

	// Get the parent group
	parentGroup, err := GetUserData(stub, caller, parentGroupID, true, false, symkeyPath, privkeyPath)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: parentGroupID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}
	if parentGroup.IsGroup != true {
		var errMsg = "GroupID cannot be user: " + parentGroupID
		logger.Error(errMsg)
		return errors.New(errMsg)
	}
	if parentGroup.PrivateKey == nil {
		logger.Errorf("PrivateKey of parentGroup \"%v\" cannot be nil", parentGroup.ID)
		return errors.Errorf("PrivateKey of parentGroup \"%v\" cannot be nil", parentGroup.ID)
	}

	// ------------------------------------------
	// Add keyGraph relationships
	// ------------------------------------------
	edgeData := make(map[string]string)
	edgeData["type"] = global.KEY_TYPE_SYM

	// 1. parent group's private key -> subgroup's private key hash
	// This makes admins of parent group admins of subgroup
	err = key_mgmt_i.AddAccess(stub, parentGroup.GetPublicKey(), subgroup.GetPrivateKeyHashSymKey(), edgeData)
	if err != nil {
		logger.Errorf("Failed to give parentGroup access to subgroup's private key hash")
		return errors.Wrap(err, "Failed to give parentGroup access to subgroup's private key hash")
	}

	// 2. subgroup sym key -> parent group sym key
	// This makes members of subgroup members of parent group
	err = key_mgmt_i.AddAccess(stub, subgroup.GetSymKey(), parentGroup.GetSymKey(), edgeData)
	if err != nil {
		logger.Errorf("Failed to give access to parent group's sym key")
		return errors.Wrap(err, "Failed to give access to parent group's sym key")
	}

	// 3. parent group log sym key -> subgroup log sym key
	// This makes auditors of the parent group also auditors of the subgroup
	err = key_mgmt_i.AddAccess(stub, parentGroup.GetLogSymKey(), subgroup.GetLogSymKey(), edgeData)
	if err != nil {
		logger.Errorf("Failed to give access to subgroup's log sym key")
		return errors.Wrap(err, "Failed to give access to subgroup's log sym key")
	}

	// 4. parent group sym key -> subgroup sym key
	// This makes members of subgroup members of parent group
	err = key_mgmt_i.AddAccess(stub, parentGroup.GetSymKey(), subgroup.GetSymKey(), edgeData)
	if err != nil {
		logger.Errorf("Failed to give access to parent group's sym key")
		return errors.Wrap(err, "Failed to give access to parent group's sym key")
	}

	// Create subgroup
	err = registerOrgInternal(stub, caller, subgroup, false)
	if err != nil {
		logger.Errorf("Failed to register subgroup: %v", err)
		return errors.Wrap(err, "Failed to register subgroup")
	}

	// Add edge to user graph
	// This logic is very confusing:
	// - edgeValue is used to identify the edge type
	// - edgeData is used for filtering on "subgroup"
	// TODO: We should remove one of them in the future to avoid confusion.
	edgeValue := []byte(global.SUBGROUP_EDGE)
	edgeData["type"] = global.SUBGROUP_EDGE
	err = graph.PutEdge(stub, global.USER_GRAPH, parentGroupID, subgroup.ID, edgeValue, edgeData)
	if err != nil {
		custom_err := &custom_errors.PutEdgeError{ParentNode: parentGroupID, ChildNode: subgroup.ID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}

	return nil
}

// RemoveSubgroupFromGroup removes a subgroup from a group.
//
// subgroupID is the id of the subgroup to remove from group.
// groupID is the id of the group that the subgroup currently belongs to.
// keyPaths are optional parameters. If passed in, they are used to get the parent group's keys.
// The first keyPath is for getting the parent group symKey, and the second keyPath is for getting the parent group privateKey.
func RemoveSubgroupFromGroup(stub cached_stub.CachedStubInterface, caller data_model.User, subgroupID string, groupID string, keyPaths ...[]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v, subgroupID: %v, groupID: %v, keyPaths: %v", caller.ID, subgroupID, groupID, keyPaths)

	// keyPaths
	var symKeyPath []string = nil
	var privKeyPath []string = nil
	if len(keyPaths) > 0 {
		symKeyPath = keyPaths[0]
	}
	if len(keyPaths) > 1 {
		privKeyPath = keyPaths[1]
	}

	// Caller must be admin of group in order to remove subgroup
	canRemove, adminPath, _ := user_mgmt_c.IsUserAdminOfGroup(stub, caller.ID, groupID)
	if !canRemove {
		custom_err := &custom_errors.NotGroupAdminError{UserID: caller.ID, GroupID: groupID}
		logger.Errorf("%v", custom_err)
		return errors.WithStack(custom_err)
	}

	subgroup, err := GetUserData(stub, caller, subgroupID, true, false)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: subgroupID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}
	if utils.IsStringEmpty(subgroup.ID) {
		errMsg := "subgroup.ID cannot be empty"
		logger.Error(errMsg)
		return errors.New(errMsg)
	}
	if subgroup.IsGroup != true {
		var errMsg = "subgroup.IsGroup cannot be false"
		logger.Error(errMsg)
		return errors.New(errMsg)
	}
	if subgroup.PrivateKey == nil {
		errMsg := "subgroup.PrivateKey cannot be nil"
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	// get keyPath
	if len(privKeyPath) == 0 && len(symKeyPath) == 0 && len(adminPath) > 1 {
		privKeyPath, _ = ConvertAdminPathToPrivateKeyPath(adminPath)
		logger.Debugf("privKeyPath from adminPath: %v", privKeyPath)

		symKeyPath, _ = ConvertAdminPathToSymKeyPath(adminPath)
		logger.Debugf("symKeyPath from adminPath: %v", symKeyPath)
	}

	group, err := GetUserData(stub, caller, groupID, true, false, symKeyPath, privKeyPath)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: groupID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}
	if utils.IsStringEmpty(group.ID) {
		errMsg := "group.ID cannot be empty"
		logger.Error(errMsg)
		return errors.New(errMsg)
	}
	if group.IsGroup != true {
		var errMsg = "group.IsGroup cannot be false"
		logger.Error(errMsg)
		return errors.New(errMsg)
	}
	if group.PrivateKey == nil {
		errMsg := "group.PrivateKey cannot be nil"
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	// Revoke Access: parent group's private key -> subgroup's private key hash
	err = key_mgmt_i.RevokeAccess(stub, group.GetPubPrivKeyId(), subgroup.GetPrivateKeyHashSymKeyId())
	if err != nil {
		errMsg := "Failed to revoke access to " + subgroupID + " private key hash sym key for " + groupID
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}

	// RevokeAccess: subgroup sym key -> parent group sym key
	err = key_mgmt_i.RevokeAccess(stub, subgroup.GetSymKeyId(), group.GetSymKeyId())
	if err != nil {
		errMsg := "Failed to revoke access to " + groupID + " symkey for " + subgroupID
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}

	// Revoke Access: parent group log sym key -> subgroup log sym key
	err = key_mgmt_i.RevokeAccess(stub, group.GetLogSymKeyId(), subgroup.GetLogSymKeyId())
	if err != nil {
		errMsg := "Failed to revoke access to " + subgroupID + " log symkey for " + groupID
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}

	// RevokeAccess: parent group sym key -> subgroup sym key
	err = key_mgmt_i.RevokeAccess(stub, group.GetSymKeyId(), subgroup.GetSymKeyId())
	if err != nil {
		errMsg := "Failed to revoke access to " + subgroupID + " symkey for " + groupID
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}

	// Call DeleteEdge from graph package
	err = graph.DeleteEdge(stub, global.USER_GRAPH, groupID, subgroupID)
	if err != nil {
		var errMsg = "Failed to DeleteEdge from " + groupID + "to " + subgroupID
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}
	return nil
}

// RemoveUserFromGroup removes a user from a group.
//
// args = [userID, groupID, removeSubGroup(optional: default=false)]
// If removeFromSubGroup is true, it will also traverse the org tree, and remove the user from all
// subgroups of groupID. This operation might take a long time to process.
// Default value of removeFromSubGroup is false.
func RemoveUserFromGroup(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	// Parse parentGroupID from args[1]
	userID := args[0]
	groupID := args[1]
	if utils.IsStringEmpty(userID) || utils.IsStringEmpty(groupID) {
		logger.Errorf("User ID and Group ID must be provided")
		return nil, errors.New("User ID and Group ID must be provided")
	}

	removeFromSubGroup := false
	if len(args) == 3 {
		removeFromSubGroup, _ = strconv.ParseBool(args[2])
	}

	// THIS ROUTINE IS SLOW!!!
	////////////////////////////////////////////////////////////
	if removeFromSubGroup {
		subgroups, err := SlowGetSubgroups(stub, groupID)
		if err != err {
			var errMsg = "Failed to get subgroups of " + groupID
			logger.Errorf("%v: %v", errMsg, err)
			return nil, errors.Wrap(err, errMsg)
		}

		for _, subgroup := range subgroups {
			inGroup, err := user_mgmt_c.IsUserInGroup(stub, userID, subgroup)
			if err != nil {
				var errMsg = "IsUserInGroup Error"
				logger.Errorf("%v: %v", errMsg, err)
				return nil, errors.Wrap(err, errMsg)
			}
			if inGroup {
				err = RemoveUserFromGroupWithParams(stub, caller, userID, subgroup)
				if err != err {
					var errMsg = "Failed to remove " + userID + " from " + subgroup
					logger.Errorf("%v: %v", errMsg, err)
					return nil, errors.Wrap(err, errMsg)
				}
			}
		}
	}
	////////////////////////////////////////////////////////////

	return nil, RemoveUserFromGroupWithParams(stub, caller, userID, groupID)
}

// RemoveUserFromGroupWithParams removes a user from a group.
// "WithParams" functions should only be called from within the chaincode.
func RemoveUserFromGroupWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, userID string, groupID string, keyPaths ...[]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v, userID: %v, groupID: %v, keyPaths: %v", caller.ID, userID, groupID, keyPaths)
	// keyPaths
	/*
		var symkeyPath []string = nil
		var privkeyPath []string = nil
		if len(keyPaths) > 0 {
			symkeyPath = keyPaths[0]
		}
		if len(keyPaths) > 1 {
			privkeyPath = keyPaths[1]
		}
	*/

	// Caller must be admin of group in order to add members/admins
	canRemove, _, _ := user_mgmt_c.IsUserAdminOfGroup(stub, caller.ID, groupID)
	if !canRemove {
		custom_err := &custom_errors.NotGroupAdminError{UserID: caller.ID, GroupID: groupID}
		logger.Errorf("%v", custom_err)
		return errors.WithStack(custom_err)
	}

	user, err := GetUserData(stub, caller, userID, false, false)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: userID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}
	if user.IsGroup == true {
		var errMsg = "User type cannot be group."
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	group, err := GetUserData(stub, caller, groupID, false, false)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: groupID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	}
	if group.IsGroup != true {
		var errMsg = "GroupID cannot be user: " + groupID
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	// Remove the user's access to group sym key and group private key
	// If no direct access exists to these keys, RevokeAccess will not return error and will have no effect
	err = key_mgmt_i.RevokeAccess(stub, user.GetPubPrivKeyId(), group.GetSymKeyId())
	if err != nil {
		errMsg := "Failed to revoke access to " + group.ID + " symkey for " + userID
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}
	err = key_mgmt_i.RevokeAccess(stub, user.GetPubPrivKeyId(), group.GetPrivateKeyHashSymKeyId())
	if err != nil {
		errMsg := "Failed to revoke access to " + group.ID + " private key for " + userID
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}

	// Call DeleteEdge from graph package
	err = graph.DeleteEdge(stub, global.USER_GRAPH, groupID, userID)
	if err != nil {
		var errMsg = "Remove user from group error"
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}
	return nil
}

// SlowGetGroupMemberIDs returns a list of group member ids, including admins.
func SlowGetGroupMemberIDs(stub cached_stub.CachedStubInterface, groupID string) ([]string, error) {
	// Use graph package function to get all child members
	children, err := graph.SlowGetChildren(stub, global.USER_GRAPH, groupID)
	members := append(children, groupID)
	if err != nil {
		return []string{}, err
	}
	sort.Strings(members)
	return members, nil
}

// SlowGetGroupAdminIDs returns a list of group admin ids.
func SlowGetGroupAdminIDs(stub cached_stub.CachedStubInterface, groupID string) ([]string, error) {
	var admins = []string{}
	// Get the list of parent groups of groupID(including itself), return true if userID is admin of any of them
	var parents []string
	parents, err := graph.SlowGetParents(stub, global.USER_GRAPH, groupID)
	if err != nil {
		var errMsg = "Error calling GetGroupAdminIDs()"
		logger.Errorf("%v: %v", errMsg, err)
		return admins, errors.Wrap(err, errMsg)
	}
	parents = append(parents, groupID)

	// Merge admin ID list of group and parent groups
	for _, parent := range parents {
		adminsOfParents, err := getDirectGroupAdminIDs(stub, parent)
		if err != nil {
			var errMsg = "Error calling GetGroupAdminIDs()"
			logger.Errorf("%v: %v", errMsg, err)
			return admins, errors.Wrap(err, errMsg)
		}
		admins = append(admins, adminsOfParents...)
	}
	// The admin list should also include original parent group admins
	admins = append(admins, parents...)
	sort.Strings(admins)
	return admins, nil
}

// ----------------- HELPER FUNCTIONS -----------------

// getDirectGroupAdminIDs returns direct admin IDs of group.
func getDirectGroupAdminIDs(stub cached_stub.CachedStubInterface, groupID string) ([]string, error) {
	var admins []string
	children, err := graph.GetDirectChildren(stub, global.USER_GRAPH, groupID)
	if err != nil {
		var errMsg = "Error calling getDirectGroupAdminIDs()"
		logger.Errorf("%v: %v", errMsg, err)
		return admins, errors.Wrap(err, errMsg)
	}
	for _, child := range children {
		edgeValueBytesResult, _, err := graph.GetEdge(stub, global.USER_GRAPH, groupID, child)
		edgeString := string(edgeValueBytesResult[:])
		if err != nil {
			var errMsg = "Error calling getDirectGroupAdminIDs()"
			logger.Errorf("%v: %v", errMsg, err)
			continue
		}
		if edgeString == global.ADMIN_EDGE {
			admins = append(admins, child)
		} else {
			continue
		}
	}
	return admins, nil
}

// SlowGetMyGroupIDs returns a list of group ids of which user is a direct or indirect member.
// If adminOnly is true, only returns group ids of which user is a direct or indirect admin.
func SlowGetMyGroupIDs(stub cached_stub.CachedStubInterface, caller data_model.User, userID string, adminOnly bool) ([]string, error) {
	groupIDs := make(map[string]bool)

	if !adminOnly {
		// Get groupIDs for which I am a member or admin
		parents, err := graph.SlowGetParents(stub, global.USER_GRAPH, userID)
		if err != nil {
			logger.Errorf("Failed to get direct parents of \"%v\" in USER_GRAPH: %v", userID, err)
			return []string{}, errors.Wrapf(err, "Failed to get direct parents of \"%v\" in USER_GRAPH", userID)
		}
		for _, parent := range parents {
			groupIDs[parent] = true
		}
	}

	// If I'm an admin of a group, I'm automatically an admin of all subgroups
	adminGroupIDs, err := user_mgmt_c.GetMyDirectAdminGroupIDs(stub, userID)
	if err != nil {
		switch err := errors.Cause(err).(type) {
		case *custom_errors.CannotBeGroupError:
			// We don't mind this error type. It happens when userID is actually a group.
			break
		default:
			// This is an unexpected error
			logger.Errorf("Failed to GetMyDirectAdminGroupIDs: %v", err)
			return []string{}, errors.WithStack(err)
		}
	}

	for _, adminGroupID := range adminGroupIDs {
		groupIDs[adminGroupID] = true
		// For each group I'm an admin of, get all subgroups
		subgroups, _ := SlowGetSubgroups(stub, adminGroupID)
		for _, subgroup := range subgroups {
			groupIDs[subgroup] = true
		}
	}
	return utils.GetDataList(groupIDs), nil
}

// GetSubgroups returns a list of ids of group's child groups.
func SlowGetSubgroups(stub cached_stub.CachedStubInterface, groupID string) ([]string, error) {
	filter := fmt.Sprintf(`{"!=": [{"var": "type"}, "%v"]}`, global.SUBGROUP_EDGE)
	return graph.SlowGetChildren(stub, global.USER_GRAPH, groupID, filter)
}

// IsParentGroup returns true if parentGroup is a direct or indirect parent of childGroup, false otherwise.
func IsParentGroup(stub cached_stub.CachedStubInterface, caller data_model.User, parentGroupID string, childGroupID string) bool {

	//make sure parentGroupID and childGroupID are groups and not users
	parentGroup, err := GetUserData(stub, caller, parentGroupID, false, false)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: parentGroupID}
		logger.Errorf("%v: %v", custom_err, err)
		return false
	}
	if parentGroup.IsGroup != true {
		var errMsg = "parentGroupID cannot be user: " + parentGroupID
		logger.Error(errMsg)
		return false
	}
	childGroup, err := GetUserData(stub, caller, childGroupID, false, false)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: childGroupID}
		logger.Errorf("%v: %v", custom_err, err)
		return false
	}
	if childGroup.IsGroup != true {
		var errMsg = "childGroupID cannot be user: " + childGroupID
		logger.Error(errMsg)
		return false
	}

	isParent, err := user_mgmt_c.IsUserInGroup(stub, childGroupID, parentGroupID)
	if err != nil {
		var errMsg = "IsUserInGroup Error: " + err.Error()
		logger.Error(errMsg)
		return false
	}

	return isParent
}

// GiveAdminPermissionOfGroup gives user admin permission to group.
// Caller must be admin of group.
func GiveAdminPermissionOfGroup(stub cached_stub.CachedStubInterface, caller data_model.User, userID string, groupID string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("userID: %v, groupID: %v", userID, groupID)
	return PutUserInGroup(stub, caller, userID, groupID, true)
}

// RemoveAdminPermissionOfGroup removes admin permission from user who is a member of group.
// Caller must be admin of group.
//
// args = [userID, groupID]
func RemoveAdminPermissionOfGroup(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	// Parse parentGroupID from args[1]
	userID := args[0]
	groupID := args[1]
	if utils.IsStringEmpty(userID) || utils.IsStringEmpty(groupID) {
		logger.Errorf("User ID and Group ID must be provided")
		return nil, errors.New("User ID and Group ID must be provided")
	}

	return nil, RemoveAdminPermissionOfGroupWithParams(stub, caller, userID, groupID)
}

// RemoveAdminPermissionOfGroupWithParams removes admin permission from user who is a member of group.
// "WithParams" functions should only be called from within the chaincode.
//
// keyPaths is optional : symkeyPath, privkeyPath
func RemoveAdminPermissionOfGroupWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, userID string, groupID string, keyPaths ...[]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v userId: %v groupId: %v", caller.ID, userID, groupID)

	var symkeyPath []string = nil
	var privkeyPath []string = nil
	if len(keyPaths) > 0 {
		symkeyPath = keyPaths[0]
	}
	if len(keyPaths) > 1 {
		privkeyPath = keyPaths[1]
	}

	isAdmin, _, _ := user_mgmt_c.IsUserAdminOfGroup(stub, userID, groupID)
	if !isAdmin {
		custom_err := &custom_errors.NotGroupAdminError{UserID: caller.ID, GroupID: groupID}
		logger.Errorf("%v", custom_err)
		return errors.WithStack(custom_err)
	}

	return PutUserInGroup(stub, caller, userID, groupID, false, symkeyPath, privkeyPath)
}

// GiveAuditorPermissionOfGroupById gives audit permission to an audit group.
// Caller must be direct or indirect admin of group.
func GiveAuditorPermissionOfGroupById(stub cached_stub.CachedStubInterface, caller data_model.User, auditorID string, groupID string, keyPaths ...[]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v auditorID: %v groupId: %v", caller.ID, auditorID, groupID)

	var symkeyPath []string = nil
	var privkeyPath []string = nil
	if len(keyPaths) > 0 {
		symkeyPath = keyPaths[0]
	}
	if len(keyPaths) > 1 {
		privkeyPath = keyPaths[1]
	}

	// Caller must be admin of group in order to add members/admins
	isCallerAdmin, adminPath, err := user_mgmt_c.IsUserAdminOfGroup(stub, caller.ID, groupID)
	if !isCallerAdmin {
		custom_err := &custom_errors.NotGroupAdminError{UserID: caller.ID, GroupID: groupID}
		logger.Errorf("%v", custom_err)
		return errors.WithStack(custom_err)
	}

	// get keyPath
	if len(privkeyPath) == 0 && len(symkeyPath) == 0 && len(adminPath) > 1 {
		privkeyPath, _ = ConvertAdminPathToPrivateKeyPath(adminPath)
		logger.Debugf("prikeyPath from admin path: %v", privkeyPath)

		symkeyPath, _ = ConvertAdminPathToSymKeyPath(adminPath)
		logger.Debugf("symkeyPath from admin path: %v", symkeyPath)
	}

	auditor, err := GetUserData(stub, caller, auditorID, true, false)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: auditorID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	} else if utils.IsStringEmpty(auditor.ID) {
		logger.Errorf("Failed to get user \"%v\"", auditorID)
		return errors.New("Failed to get user " + auditorID)
	}
	group, err := GetUserData(stub, caller, groupID, true, false, symkeyPath, privkeyPath)
	if err != nil {
		custom_err := &custom_errors.GetUserError{ID: groupID}
		logger.Errorf("%v: %v", custom_err, err)
		return errors.Wrap(err, custom_err.Error())
	} else if utils.IsStringEmpty(group.ID) {
		logger.Errorf("Failed to get user \"%v\"", groupID)
		return errors.New("Failed to get user " + groupID)
	}
	return GiveAuditorPermissionOfGroup(stub, caller, auditor, group)
}

// GiveAuditorPermissionOfGroup gives audit permission to an audit group.
// Caller must be admin of group.
func GiveAuditorPermissionOfGroup(stub cached_stub.CachedStubInterface, caller, auditor, group data_model.User) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v auditorID: %v groupId: %v", caller.ID, auditor.ID, group.ID)
	//make sure caller is admin of group
	isCallerAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, caller.ID, group.ID)
	if !isCallerAdmin {
		custom_err := &custom_errors.NotGroupAdminError{UserID: caller.ID, GroupID: group.ID}
		logger.Errorf("%v", custom_err)
		return errors.WithStack(custom_err)
	}

	// must have private and sym key if caller is admin of group
	if len(group.PrivateKeyB64) == 0 || len(group.SymKey) == 0 {
		custom_err := &custom_errors.NotGroupAdminError{UserID: caller.ID, GroupID: group.ID}
		logger.Errorf("%v", custom_err)
		return errors.New(custom_err.Error())
	}

	//make sure auditorID is actually an auditor based on role in user object
	if auditor.Role != global.ROLE_AUDIT {
		errMsg := "Cannot give audit permission to a non-auditor user"
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	//give auditor access to logsymkey of group
	edgeData := make(map[string]string)
	edgeData["type"] = global.KEY_TYPE_SYM
	err = key_mgmt_i.AddAccess(stub, auditor.GetPublicKey(), group.GetLogSymKey(), edgeData)
	if err != nil {
		errMsg := "Failed saving log symkey for " + auditor.ID
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}

	return nil
}

// RemoveAuditorPermissionOfGroup removes an auditor's permission to audit group.
// Caller must be admin of group.
func RemoveAuditorPermissionOfGroup(stub cached_stub.CachedStubInterface, caller data_model.User, auditorID string, groupID string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v auditorID: %v groupId: %v", caller.ID, auditorID, groupID)

	// Caller must be admin of group in order to add members/admins
	isCallerAdmin, _, err := user_mgmt_c.IsUserAdminOfGroup(stub, caller.ID, groupID)
	if !isCallerAdmin {
		custom_err := &custom_errors.NotGroupAdminError{UserID: caller.ID, GroupID: groupID}
		logger.Errorf("%v", custom_err)
		return errors.WithStack(custom_err)
	}

	//remove auditor access to logsymkey of group
	err = key_mgmt_i.RevokeAccess(stub, key_mgmt_i.GetPubPrivKeyId(auditorID), key_mgmt_i.GetLogSymKeyId(groupID))
	if err != nil {
		errMsg := "Failed to revoke access to " + groupID + " private key for " + auditorID
		logger.Errorf("%v: %v", errMsg, err)
		return errors.Wrap(err, errMsg)
	}

	return nil
}
