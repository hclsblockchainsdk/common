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
	"common/bchcls/crypto"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/key_mgmt_i/key_mgmt_c/key_mgmt_g"
	"common/bchcls/internal/user_mgmt_i/user_mgmt_c/user_mgmt_g"

	"crypto/rsa"
	"encoding/json"
	"reflect"
)

// Org is currently used by the GetCallerData function to check if caller ID is valid.
type Org struct {
	Id   string      `json:"id"`
	Name string      `json:"name"`
	Data interface{} `json:"data"`
}

// User represents either a person or a group.
// A group is an organization and can have admins, members, and subgroups.
// De-identified fields:
//   - ID
//   - Name
//   - Org
type User struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	Role               string         `json:"role"`
	PublicKey          *rsa.PublicKey `json:"-"`
	PublicKeyB64       string         `json:"public_key"`
	IsGroup            bool           `json:"is_group"`
	Status             string         `json:"status"`
	SolutionPublicData interface{}    `json:"solution_public_data"`
	ConnectionID       string         `json:"connection_id"`

	// private data
	Email               string          `json:"email"`
	PrivateKey          *rsa.PrivateKey `json:"-"`
	PrivateKeyB64       string          `json:"private_key"`
	SymKey              []byte          `json:"-"`
	SymKeyB64           string          `json:"sym_key"`
	KmsPublicKeyId      string          `json:"kms_public_key_id"`
	KmsPrivateKeyId     string          `json:"kms_private_key_id"`
	KmsSymKeyId         string          `json:"kms_sym_key_id"`
	Secret              string          `json:"secret"`
	SolutionPrivateData interface{}     `json:"solution_private_data"`
}

// UserPublicData is public data of the user object.
type UserPublicData struct {
	ID                 string      `json:"id"`
	Name               string      `json:"name"`
	Role               string      `json:"role"`
	PublicKeyB64       string      `json:"public_key"`
	IsGroup            bool        `json:"is_group"`
	Status             string      `json:"status"`
	SolutionPublicData interface{} `json:"solution_public_data"`
	ConnectionID       string      `json:"connection_id"`
}

// UserPrivateData is private data of the user object.
type UserPrivateData struct {
	Email               string      `json:"email"`
	KmsPublicKeyId      string      `json:"kms_public_key_id"`
	KmsPrivateKeyId     string      `json:"kms_private_key_id"`
	KmsSymKeyId         string      `json:"kms_sym_key_id"`
	Secret              string      `json:"secret"`
	SolutionPrivateData interface{} `json:"solution_private_data"`
}

// IsSystemAdmin returns true if user's role is ROLE_SYSTEM_ADMIN.
func (u *User) IsSystemAdmin() bool {
	return u.Role == global.ROLE_SYSTEM_ADMIN
}

// Equal returns true if two users objects are equal.
func (u *User) Equal(other User) bool {
	equal := true
	equal = equal && u.ID == other.ID
	equal = equal && u.Name == other.Name
	equal = equal && u.Role == other.Role
	equal = equal && u.PublicKeyB64 == other.PublicKeyB64
	if len(u.PrivateKeyB64) > 0 && len(other.PrivateKeyB64) > 0 {
		equal = equal && u.PrivateKeyB64 == other.PrivateKeyB64
	}
	if len(u.SymKeyB64) > 0 && len(other.SymKeyB64) > 0 {
		equal = equal && u.SymKeyB64 == other.SymKeyB64
	}
	equal = equal && u.IsGroup == other.IsGroup
	equal = equal && u.Status == other.Status
	equal = equal && u.Email == other.Email
	equal = equal && u.KmsPublicKeyId == other.KmsPublicKeyId
	equal = equal && u.KmsPrivateKeyId == other.KmsPrivateKeyId
	equal = equal && u.KmsSymKeyId == other.KmsSymKeyId
	equal = equal && u.Secret == other.Secret
	equal = equal && reflect.DeepEqual(u.SolutionPublicData, other.SolutionPublicData)
	equal = equal && reflect.DeepEqual(u.SolutionPrivateData, other.SolutionPrivateData)
	equal = equal && reflect.DeepEqual(u.ConnectionID, other.ConnectionID)
	return equal
}

// IsSameUser checks if two users are the same by checking only minimally required fields
// Does not compare Email, Status, IsGroup, Secret, SolutionPublicData, and SolutionPrivateData.
func (u *User) IsSameUser(other User) bool {
	logger.Debugf("u %v", *u)
	logger.Debugf("user %v", other)
	equal := true
	equal = equal && u.ID == other.ID
	equal = equal && u.Role == other.Role
	equal = equal && u.PublicKeyB64 == other.PublicKeyB64
	if len(u.PrivateKeyB64) > 0 && len(other.PrivateKeyB64) > 0 {
		equal = equal && u.PrivateKeyB64 == other.PrivateKeyB64
	}
	if len(u.SymKeyB64) > 0 && len(other.SymKeyB64) > 0 {
		equal = equal && u.SymKeyB64 == other.SymKeyB64
	}

	return equal
}

// ConvertToAsset converts a user to an asset.
func (u *User) ConvertToAsset() Asset {
	asset := Asset{}
	asset.AssetId = user_mgmt_g.GetUserAssetID(u.ID)
	asset.Datatypes = []string{}
	asset.PublicData = u.GetPublicDataBytes()
	asset.PrivateData = u.GetPrivateDataBytes()
	asset.IndexTableName = global.INDEX_USER
	asset.OwnerIds = []string{u.ID}

	// if an off-chain datastore is specified, save the id so that the asset can be saved there
	if len(u.ConnectionID) != 0 {
		asset.SetDatastoreConnectionID(u.ConnectionID)
	}

	return asset
}

// LoadFromAsset converts an asset to a user object.
func (u *User) LoadFromAsset(asset *Asset) *User {
	var publicData UserPublicData
	var privateData UserPrivateData

	err := json.Unmarshal(asset.PublicData, &publicData)
	if err == nil {
		u.ID = publicData.ID
		u.Name = publicData.Name
		u.Role = publicData.Role
		u.PublicKeyB64 = publicData.PublicKeyB64
		u.IsGroup = publicData.IsGroup
		u.Status = publicData.Status
		u.SolutionPublicData = publicData.SolutionPublicData
		u.ConnectionID = publicData.ConnectionID
	}

	err = json.Unmarshal(asset.PrivateData, &privateData)
	if err == nil {
		u.Email = privateData.Email
		u.KmsPrivateKeyId = privateData.KmsPrivateKeyId
		u.KmsPublicKeyId = privateData.KmsPublicKeyId
		u.KmsSymKeyId = privateData.KmsSymKeyId
		u.Secret = privateData.Secret
		u.SolutionPrivateData = privateData.SolutionPrivateData
	}

	u.PublicKey, _ = crypto.ParsePublicKeyB64(u.PublicKeyB64)
	u.PrivateKey = nil
	u.SymKey = nil

	return u
}

// GetPubPrivKeyId returns the ID of the public/private key of the user.
func (u *User) GetPubPrivKeyId() string {
	return key_mgmt_g.GetPubPrivKeyId(u.ID)
}

// GetPrivateKey returns the private key of the user.
func (u *User) GetPrivateKey() Key {
	keyBytes, err := crypto.DecodeStringB64(u.PrivateKeyB64)
	if err != nil || len(keyBytes) == 0 {
		keyBytes = crypto.PrivateKeyToBytes(u.PrivateKey)
	}
	return Key{
		ID:       u.GetPubPrivKeyId(),
		KeyBytes: keyBytes,
		Type:     global.KEY_TYPE_PRIVATE,
	}
}

// GetPublicKey returns the public key of the user.
func (u *User) GetPublicKey() Key {
	return Key{
		ID:       u.GetPubPrivKeyId(),
		KeyBytes: crypto.PublicKeyToBytes(u.PublicKey),
		Type:     global.KEY_TYPE_PUBLIC,
	}
}

// GetSymKeyId returns the ID of the sym key of the user.
func (u *User) GetSymKeyId() string {
	return key_mgmt_g.GetSymKeyId(u.ID)
}

// GetSymKey returns the sym key of a user.
func (u *User) GetSymKey() Key {
	return Key{
		ID:       u.GetSymKeyId(),
		KeyBytes: u.SymKey,
		Type:     global.KEY_TYPE_SYM,
	}
}

// GetLogSymKeyId returns the ID of the log sym key of the user.
func (u *User) GetLogSymKeyId() string {
	return key_mgmt_g.GetLogSymKeyId(u.ID)
}

// GetLogSymKey deterministically generates and returns a log sym key for the user.
func (u *User) GetLogSymKey() Key {
	logSymKeyBytes := append(u.SymKey, "logSymKey"...)
	return Key{
		ID:       u.GetLogSymKeyId(),
		KeyBytes: crypto.GetSymKeyFromHash(logSymKeyBytes),
		Type:     global.KEY_TYPE_SYM,
	}
}

// GetPrivateKeyHashSymKeyId returns the ID of the private-key-hash sym key of the user.
func (u *User) GetPrivateKeyHashSymKeyId() string {
	return key_mgmt_g.GetPrivateKeyHashSymKeyId(u.ID)
}

// GetPrivateKeyHashSymKey deterministically generates and returns a sym key from hash of the user's private key.
func (u *User) GetPrivateKeyHashSymKey() Key {
	privkeyBytes, err := crypto.DecodeStringB64(u.PrivateKeyB64)
	if err != nil || len(privkeyBytes) == 0 {
		privkeyBytes = crypto.PrivateKeyToBytes(u.PrivateKey)
	}
	return Key{
		ID:       u.GetPrivateKeyHashSymKeyId(),
		KeyBytes: crypto.GetSymKeyFromHash(privkeyBytes),
		Type:     global.KEY_TYPE_SYM,
	}
}

// GetPublicDataBytes turns user's public data into bytes.
func (u *User) GetPublicDataBytes() []byte {
	publicData := UserPublicData{}
	publicData.ID = u.ID
	publicData.Name = u.Name
	publicData.PublicKeyB64 = u.PublicKeyB64
	publicData.Role = u.Role
	publicData.Status = u.Status
	publicData.IsGroup = u.IsGroup
	publicData.SolutionPublicData = u.SolutionPublicData
	publicData.ConnectionID = u.ConnectionID
	publicBytes, _ := json.Marshal(&publicData)
	return publicBytes
}

// GetPrivateDataBytes turns user's private data into bytes.
func (u *User) GetPrivateDataBytes() []byte {
	privateData := UserPrivateData{}
	privateData.SolutionPrivateData = u.SolutionPrivateData
	privateData.Email = u.Email
	privateData.KmsPrivateKeyId = u.KmsPrivateKeyId
	privateData.KmsPublicKeyId = u.KmsPublicKeyId
	privateData.KmsSymKeyId = u.KmsSymKeyId
	privateData.Secret = u.Secret
	privateBytes, _ := json.Marshal(&privateData)
	return privateBytes
}
