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

// Consent represents access given to all assets of a particular datatype,
// from one user/group to another, for a specified period of time.
//
// Callers should not pass in the ConsentID, ConsentAssetID, AssetKeyID, or CreatorID fields.
//
// Supply the following fields:
//   - OwnerID: de-identified UUID of the user giving the consent
//   - TargetID: de-identified UUID of the user receiving the consent
//   - DatatypeID: UUID of the datatype consent is given through
//   - Access: level of consent given
//   - ExpirationDate: consent expiration date
//   - ConsentDate: date of the last update to the consent's Access field
// If caller is not the owner, caller must have access to owner’s RSA keys.
// ConsentID is hash(ConsentPrefix + DatatypeID + TargetID + OwnerID).
//
// Optional fields:
//   - Data: arbitrary data specified by the solution developer
//   - ConnectionID: the connection ID for an off-chain datastore. If this is provided, the Consent's encrypted private data will be saved to that datastore.
// De-identified fields:
//   - CreatorID
//   - OwnerID
//   - TargetID
type Consent struct {
	ConsentID      string      `json:"consent_id"`
	ConsentAssetID string      `json:"consent_asset_id"`
	AssetKeyID     string      `json:"asset_key_id"`
	CreatorID      string      `json:"creator_id"`
	OwnerID        string      `json:"owner_id"`
	TargetID       string      `json:"target_id"`
	DatatypeID     string      `json:"datatype_id"`
	Access         string      `json:"access"`
	ExpirationDate int64       `json:"expiration_date"`
	ConsentDate    int64       `json:"consent_date"`
	Data           interface{} `json:"data"`
	ConnectionID   string      `json:"connection_id"`
}
