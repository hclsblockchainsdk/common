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
	"common/bchcls/internal/key_mgmt_i/key_mgmt_c/key_mgmt_g"
)

// Key is used for encrypting asset data and other keys on the ledger.
// Type can be KEY_TYPE_PRIVATE, KEY_TYPE_PUBLIC, or KEY_TYPE_SYM.
// Refer to key_mgmt package for more info.
type Key struct {
	ID       string `json:"id"`
	KeyBytes []byte `json:"key"`
	Type     string `json:"type"`
}

// Keys is used in user_mgmt.GetUserKeys to return a user's public, private, and sym keys.
type Keys struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
	SymKey     string `json:"sym_key"`
}

// IsEmpty checks if a given key's ID or keyBytes is empty.
func (k *Key) IsEmpty() bool {
	if len(k.ID) == 0 || len(k.KeyBytes) == 0 {
		return true
	}
	return false
}

// GetLogSymKeyId returns the ID of a log sym key.
func (key *Key) GetLogSymKeyId() string {
	return key_mgmt_g.GetLogSymKeyId(key.ID)
}
