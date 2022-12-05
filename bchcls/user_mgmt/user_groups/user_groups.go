/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package user_groups handles user management functions related to groups.
package user_groups

import (
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/common/graph"
	"common/bchcls/internal/metering_i"
	"common/bchcls/internal/user_mgmt_i"
	"common/bchcls/internal/user_mgmt_i/user_mgmt_c"
	"common/bchcls/utils"

	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("user_groups")

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

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.PutUserInGroup(stub, caller, userID, groupID, isAdmin, keyPaths...)
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

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RegisterSubgroup(stub, caller, args)
}

// RegisterSubgroupWithParams registers a new group as a subgroup of an existing group.
// "WithParams" functions should only be called from within the chaincode.
//
// subgroup is the subgroup to register.
// parentGroupID is the id of the parent group.
// keyPaths are optional parameters. If passed in, they are used to get the parent group's keys.
// The first keyPath is for getting the parent group symKey, and the second keyPath is for getting the parent group privateKey.
func RegisterSubgroupWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, subgroup data_model.User, parentGroupID string, keyPaths ...[]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v, subgroup: %v, parentGroup: %v, keyPath: %v", caller.ID, subgroup.ID, parentGroupID, keyPaths)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RegisterSubgroupWithParams(stub, caller, subgroup, parentGroupID, keyPaths...)
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

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RemoveSubgroupFromGroup(stub, caller, subgroupID, groupID, keyPaths...)
}

// RemoveUserFromGroup removes a user from a group.
//
// args = [userID, groupID, removeSubGroup (optional: default=false)]
// If removeFromSubGroup is true, the function will also traverse the org tree, and remove the user from all
// subgroups of the given groupID. This operation could be slow depending on tree structure.
func RemoveUserFromGroup(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RemoveUserFromGroup(stub, caller, args)
}

// RemoveUserFromGroupWithParams is the internal function for removing a user from a group.
// "WithParams" functions should only be called from within the chaincode.
func RemoveUserFromGroupWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, userID string, groupID string, keyPaths ...[]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v, userID: %v, groupID: %v, keyPaths: %v", caller.ID, userID, groupID, keyPaths)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RemoveUserFromGroupWithParams(stub, caller, userID, groupID, keyPaths...)
}

// IsUserInGroup returns true if a user is either a direct or indirect member of a group.
func IsUserInGroup(stub cached_stub.CachedStubInterface, userID string, groupID string) (bool, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_c.IsUserInGroup(stub, userID, groupID)
}

// IsUserMemberOfGroup returns true if a user is in a group.
// This function does not check indirect membership.
func IsUserMemberOfGroup(stub cached_stub.CachedStubInterface, userID string, groupID string) (bool, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_c.IsUserMemberOfGroup(stub, userID, groupID)
}

// IsUserDirectAdminOfGroup is a helper function that checks if user is a direct admin of a group.
func IsUserDirectAdminOfGroup(stub cached_stub.CachedStubInterface, userID string, groupID string) (bool, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_c.IsUserDirectAdminOfGroup(stub, userID, groupID)
}

// IsUserAdminOfGroup checks if a user is a direct or indirect admin of a group.
// If it is an admin, returns a user admin chain as well. If not, returns an empty list.
func IsUserAdminOfGroup(stub cached_stub.CachedStubInterface, userID string, groupID string) (bool, []string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_c.IsUserAdminOfGroup(stub, userID, groupID)
}

// IsParentGroup returns true if parentGroup is a direct or indirect parent of childGroup, false otherwise.
func IsParentGroup(stub cached_stub.CachedStubInterface, caller data_model.User, parentGroupID string, childGroupID string) bool {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.IsParentGroup(stub, caller, parentGroupID, childGroupID)
}

// SlowGetGroupMemberIDs returns a list of group member and admin IDs.
// This function searches the entire user graph, so it could be slow.
func SlowGetGroupMemberIDs(stub cached_stub.CachedStubInterface, groupID string) ([]string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.SlowGetGroupMemberIDs(stub, groupID)
}

// SlowGetGroupAdminIDs returns a list of group admin IDs.
// This function searches the entire user graph, so it could be slow.
func SlowGetGroupAdminIDs(stub cached_stub.CachedStubInterface, groupID string) ([]string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.SlowGetGroupAdminIDs(stub, groupID)
}

// SlowGetMyGroupIDs returns a list of group IDs for which the given user is either
// a direct or indirect member.
// If adminOnly is set to true, returns only group IDs for which the user is a direct or
// indirect admin.
// This function searches the entire user graph, so it could be slow.
func SlowGetMyGroupIDs(stub cached_stub.CachedStubInterface, caller data_model.User, userID string, adminOnly bool) ([]string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.SlowGetMyGroupIDs(stub, caller, userID, adminOnly)
}

// SlowGetSubgroups returns a list of subgroup IDs of group's child groups.
// This function searches the entire user graph, so it could be slow.
func SlowGetSubgroups(stub cached_stub.CachedStubInterface, groupID string) ([]string, error) {
	filter := fmt.Sprintf(`{"!=": [{"var": "type"}, "%v"]}`, global.SUBGROUP_EDGE)

	_ = metering_i.SetEnvAndAddRow(stub)

	return graph.SlowGetChildren(stub, global.USER_GRAPH, groupID, filter)
}

// GetMyDirectGroupIDs returns a list of group IDs for which the user is a direct member.
func GetMyDirectGroupIDs(stub cached_stub.CachedStubInterface, userID string) ([]string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_c.GetMyDirectGroupIDs(stub, userID)
}

// GetMyDirectAdminGroupIDs returns a list of group IDs for which the user is a direct admin.
func GetMyDirectAdminGroupIDs(stub cached_stub.CachedStubInterface, userID string) ([]string, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_c.GetMyDirectAdminGroupIDs(stub, userID)
}

// GiveAdminPermissionOfGroup gives user admin permission to a group.
// Caller must be an admin of the group.
func GiveAdminPermissionOfGroup(stub cached_stub.CachedStubInterface, caller data_model.User, userID string, groupID string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("userID: %v, groupID: %v", userID, groupID)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GiveAdminPermissionOfGroup(stub, caller, userID, groupID)
}

// RemoveAdminPermissionOfGroup removes group admin permission from a user.
// Caller must be an admin of the group.
//
// args = [userID, groupID]
func RemoveAdminPermissionOfGroup(stub cached_stub.CachedStubInterface, caller data_model.User, args []string) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("args: %v", args)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RemoveAdminPermissionOfGroup(stub, caller, args)
}

// RemoveAdminPermissionOfGroupWithParams is the internal function for removing group admin permission from a user.
// "WithParams" functions should only be called from within the chaincode.
//
// keyPaths are optional parameters. If passed in, they are used to get group's keys.
// The first keyPath is for getting the group symKey, and the second keyPath is for getting the group privateKey.
func RemoveAdminPermissionOfGroupWithParams(stub cached_stub.CachedStubInterface, caller data_model.User, userID string, groupID string, keyPaths ...[]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v userId: %v groupId: %v", caller.ID, userID, groupID)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RemoveAdminPermissionOfGroupWithParams(stub, caller, userID, groupID, keyPaths...)
}

// GiveAuditorPermissionOfGroupById adds group audit permission to a user for the given auditorID and groupID.
// Caller must be direct or indirect admin of group.
// keyPaths are optional parameters. If passed in, they are used to get group's keys.
// The first keyPath is for getting group symKey, the second keyPath is for getting group privateKey.
func GiveAuditorPermissionOfGroupById(stub cached_stub.CachedStubInterface, caller data_model.User, auditorID string, groupID string, keyPaths ...[]string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v auditorID: %v groupId: %v", caller.ID, auditorID, groupID)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GiveAuditorPermissionOfGroupById(stub, caller, auditorID, groupID, keyPaths...)
}

// GiveAuditorPermissionOfGroup adds group audit permission to a user for the given auditor and group objects.
// Caller must be admin of group.
func GiveAuditorPermissionOfGroup(stub cached_stub.CachedStubInterface, caller, auditor, group data_model.User) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v auditorID: %v groupId: %v", caller.ID, auditor.ID, group.ID)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.GiveAuditorPermissionOfGroup(stub, caller, auditor, group)
}

// RemoveAuditorPermissionOfGroup removes group auditor's permission from a user.
// Caller must be admin of group.
func RemoveAuditorPermissionOfGroup(stub cached_stub.CachedStubInterface, caller data_model.User, auditorID string, groupID string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	logger.Debugf("caller: %v auditorID: %v groupId: %v", caller.ID, auditorID, groupID)

	_ = metering_i.SetEnvAndAddRow(stub)

	return user_mgmt_i.RemoveAuditorPermissionOfGroup(stub, caller, auditorID, groupID)
}
