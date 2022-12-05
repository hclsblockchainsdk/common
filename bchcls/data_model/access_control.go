/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package data_model contains structs used across packages to prevent circular imports.
// For example, the User struct is needed by both asset_mgmt and user_mgmt, but user_mgmt
// depends on functions in asset_mgmt.
// They can't import each other, so the shared structs live here.
package data_model

import (
	"common/bchcls/internal/common/global"
)

// AccessControl represents a user's read or write access to an asset.
// UserKey is optional
type AccessControl struct {
	UserId   string `json:"userid"`
	UserKey  *Key   `json:"user_key"`
	AssetId  string `json:"assetid"`
	AssetKey *Key   `json:"asset_key"`
	Access   string `json:"access"`
}

// IsValid checks if an AccessControl object's fields are valid
func (a *AccessControl) IsValid() bool {
	if len(a.UserId) == 0 {
		return false
	}
	if len(a.AssetId) == 0 {
		return false
	}
	if a.Access != global.ACCESS_READ &&
		a.Access != global.ACCESS_READ_ONLY &&
		a.Access != global.ACCESS_WRITE &&
		a.Access != global.ACCESS_WRITE_ONLY {
		return false
	}
	return true
}

// AccessControlFilters are filters for key traversal functions, which can be passed along each function call.
// Used with the SlowCheckAccessToKey function.
type AccessControlFilters struct {
	AssetFilters    []string
	OwnerFilters    []string
	DatatypeFilters []string
}
