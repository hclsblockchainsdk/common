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

// Datatype represents a type that can be used to classify assets.
// Datatypes are stored in a tree structure. Datatypes can have sub-datatypes.
// All datatype information is public
type Datatype struct {
	DatatypeID  string `json:"datatype_id"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_acive"`
}
