/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package custom_errors defines our custom error types.
//
// Custom types are useful for:
// 1) allowing callers to do type-checking to see the cause of the error.
// 2) re-using messages for common errors.
// If neither scenario applies, it's perfectly fine to instead use errors.New("some message").
//
// A custom error can be wrapped by another error when returned using errors.Wrap(err, custom_err.Error()).
// To return a custom error with stack trace, use errors.WithStack(custom_err).
// If returning a custom error for type checking, it must be returned without a wrapper.
package custom_errors

import (
	"fmt"
)

// IterError provides an error message for Iter.Next() failure.
type IterError struct{}

func (e *IterError) Error() string {
	return "Error reading next KV"
}

// MarshalError provides an error message for json.Marshal failure.
type MarshalError struct {
	Type string
}

func (e *MarshalError) Error() string {
	return fmt.Sprintf("Failed to marshal %v", e.Type)
}

// UnmarshalError provides an error message for json.Unmarshal failure.
type UnmarshalError struct {
	Type string
}

func (e *UnmarshalError) Error() string {
	return fmt.Sprintf("Failed to unmarshal %v", e.Type)
}

// LengthCheckingError provides an error message for an incorrect slice or string length.
type LengthCheckingError struct {
	Type string
}

func (e *LengthCheckingError) Error() string {
	return fmt.Sprintf("Length of %v does not match expected", e.Type)
}

// TypeAssertionError provides an error message for an incorrect type.
type TypeAssertionError struct {
	Item string
	Type string
}

func (e *TypeAssertionError) Error() string {
	return fmt.Sprintf("%v is not of %v type", e.Item, e.Type)
}

// Ledger

// CreateCompositeKeyError provides an error message for stub.CreateCompositeKey failure.
type CreateCompositeKeyError struct {
	Type string
}

func (e *CreateCompositeKeyError) Error() string {
	return fmt.Sprintf("Failed to create composite key for %v", e.Type)
}

// SplitCompositeKeyError provides an error message for stub.SplitCompositeKey failure.
type SplitCompositeKeyError struct {
	Key string
}

func (e *SplitCompositeKeyError) Error() string {
	return fmt.Sprintf("Failed to split composite key %v", e.Key)
}

// GetLedgerError provides an error message for failure to retrieve an item from the ledger.
type GetLedgerError struct {
	LedgerKey  string
	LedgerItem string
}

func (e *GetLedgerError) Error() string {
	return fmt.Sprintf("Failed to get ledger item \"%v\" from ledger with ledger key \"%v\"", e.LedgerItem, e.LedgerKey)
}

// PutLedgerError provides an error message for failure to save an item to the ledger.
type PutLedgerError struct {
	LedgerKey string
}

func (e *PutLedgerError) Error() string {
	return fmt.Sprintf("Failed to put %v in ledger", e.LedgerKey)
}

// DeleteLedgerError provides an error message for failure to delete an item from the ledger.
type DeleteLedgerError struct {
	LedgerKey string
}

func (e *DeleteLedgerError) Error() string {
	return fmt.Sprintf("Failed to delete %v from ledger", e.LedgerKey)
}

// Crypto

// InvalidSymKeyError provides an error message for an invalid symmetric key.
type InvalidSymKeyError struct{}

func (e *InvalidSymKeyError) Error() string {
	return "Invalid symKey: length must be 32 bytes"
}

// InvalidPrivateKeyError provides an error message for an invalid RSA private key.
type InvalidPrivateKeyError struct{}

func (e *InvalidPrivateKeyError) Error() string {
	return "Invalid private key by caller"
}

// InvalidPublicKeyError provides an error message for an invalid RSA public key.
type InvalidPublicKeyError struct{}

func (e *InvalidPublicKeyError) Error() string {
	return "Invalid public key by caller"
}

// EncryptionError provides an error message for encryption failure.
type EncryptionError struct {
	ToEncrypt     string
	EncryptionKey string
}

func (e *EncryptionError) Error() string {
	return fmt.Sprintf("Failed to encrypt %v with %v", e.ToEncrypt, e.EncryptionKey)
}

// DecryptionError provides an error message for decryption failure.
type DecryptionError struct {
	ToDecrypt     string
	DecryptionKey string
}

func (e *DecryptionError) Error() string {
	return fmt.Sprintf("Failed to decrypt %v with %v", e.ToDecrypt, e.DecryptionKey)
}

// CiphertextEmptyError provides an error message for empty ciphertext.
type CiphertextEmptyError struct{}

func (e *CiphertextEmptyError) Error() string {
	return "ciphertext is empty"
}

// CiphertextLengthError provides an error message for ciphertext length that is too short.
type CiphertextLengthError struct{}

func (e *CiphertextLengthError) Error() string {
	return "ciphertext too short"
}

// CiphertextBlockSizeError provides an error message for ciphertext length that is not a multiple of block size.
type CiphertextBlockSizeError struct{}

func (e *CiphertextBlockSizeError) Error() string {
	return "ciphertext is not a multiple of the block size"
}

// Key Management

// InvalidKeyError provides an error message for an invalid key.
type InvalidKeyError struct {
	KeyId string
}

func (e *InvalidKeyError) Error() string {
	return fmt.Sprintf("Key \"%v\" doesn't match existing key in graph", e.KeyId)
}

// AddAccessError provides an error message for failure to add access from one key to another.
type AddAccessError struct {
	Key string
}

func (e *AddAccessError) Error() string {
	return fmt.Sprintf("Failed to AddAccess to %v", e.Key)
}

// VerifyAccessError provides an error message for failure to verify access from one key to another.
type VerifyAccessError struct {
	StartKeyId  string
	TargetKeyId string
}

func (e *VerifyAccessError) Error() string {
	return fmt.Sprintf("Failed to VerifyAccess from %v to %v", e.StartKeyId, e.TargetKeyId)
}

// VerifyAccessAndGetKeyError provides an error message for failure to verify access and retrieve target key.
type VerifyAccessAndGetKeyError struct{}

func (e *VerifyAccessAndGetKeyError) Error() string {
	return "Failed to VerifyAccess and GetKey"
}

// ParseKeyError provides an error message for failure to parse a key.
type ParseKeyError struct {
	Type string
}

func (e *ParseKeyError) Error() string {
	return fmt.Sprintf("Unable to parse %v", e.Type)
}

// ValidateKeyError provides an error message for failure to validate a key.
type ValidateKeyError struct {
	KeyId string
}

func (e *ValidateKeyError) Error() string {
	return fmt.Sprintf("Error cross-checking \"%v\" with existing key", e.KeyId)
}

// Graph

// GetNodesError provides an error message for failure to get nodes in a graph.
type GetNodesError struct{}

func (e *GetNodesError) Error() string {
	return "Failed to get nodes"
}

// GetEdgeError provides an error message for failure to get an edge in a graph.
type GetEdgeError struct {
	ParentNode string
	ChildNode  string
}

func (e *GetEdgeError) Error() string {
	return fmt.Sprintf("Failed to get graph edge from %v to %v", e.ParentNode, e.ChildNode)
}

// PutEdgeError provides an error message for failure to save an edge in a graph.
type PutEdgeError struct {
	ParentNode string
	ChildNode  string
}

func (e *PutEdgeError) Error() string {
	return fmt.Sprintf("Failed to put graph edge from %v to %v", e.ParentNode, e.ChildNode)
}

// GetDirectChildrenError provides an error message for failure to get child nodes of a parent node in a graph.
type GetDirectChildrenError struct {
	Parent string
}

func (e *GetDirectChildrenError) Error() string {
	return fmt.Sprintf("Failed to get direct children of %v", e.Parent)
}

// GetDirectParentsError provides an error message for failure to get parent nodes of a child node in a graph.
type GetDirectParentsError struct {
	Child string
}

func (e *GetDirectParentsError) Error() string {
	return fmt.Sprintf("Failed to get direct parents of %v", e.Child)
}

// Asset Management

// ReplaceAssetKeyIdError provides an error message for attempt to change an asset's asset key id.
type ReplaceAssetKeyIdError struct{}

func (e *ReplaceAssetKeyIdError) Error() string {
	return "Asset exists on ledger with different AssetKeyId"
}

// ReplaceAssetKeyHashError provides an error message for attempt to change an asset's asset key.
type ReplaceAssetKeyHashError struct{}

func (e *ReplaceAssetKeyHashError) Error() string {
	return "Asset exists on ledger with different AssetKey"
}

// GetAssetDataError provides an error message for failure to get asset data.
type GetAssetDataError struct {
	AssetId string
}

func (e *GetAssetDataError) Error() string {
	return fmt.Sprintf("Failed to get asset data for %v", e.AssetId)
}

// User Management

// GetUserError provides an error message for failure to get user data.
type GetUserError struct {
	ID string
}

func (e *GetUserError) Error() string {
	return fmt.Sprintf("Failed to get user %v", e.ID)
}

// NotGroupAdminError provides an error message for a user not being a group admin when required.
type NotGroupAdminError struct {
	UserID  string
	GroupID string
}

func (e *NotGroupAdminError) Error() string {
	return fmt.Sprintf("%v is not an admin of %v", e.UserID, e.GroupID)
}

// CannotBeGroupError provides an error message for a user being a group when not allowed.
type CannotBeGroupError struct {
	GroupID string
}

func (e *CannotBeGroupError) Error() string {
	return fmt.Sprintf("userID cannot be a group: \"%v\"", e.GroupID)
}

// RegisterOrgInvalidFieldError provides an error message for an invalid org field during registration.
type RegisterOrgInvalidFieldError struct {
	ID    string
	Field string
}

func (e *RegisterOrgInvalidFieldError) Error() string {
	return fmt.Sprintf("RegisterOrg invalid field for %v: %v", e.ID, e.Field)
}

// Datatype

// CycleError provides an error message for attempt to add a datatype relationship that would cause a cycle.
type CycleError struct {
	Parent string
	Child  string
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("Failed to create edge from %v to %v to prevent cycle", e.Parent, e.Child)
}

// GetDatatypeError provides an error message for failure to get datatype_i.
type GetDatatypeError struct {
	Datatype string
}

func (e *GetDatatypeError) Error() string {
	return fmt.Sprintf("Failed to get datatype %v", e.Datatype)
}

// GetParentDatatypesError provides an error message for failure to get parent datatypes.
type GetParentDatatypesError struct {
	Datatype string
}

func (e *GetParentDatatypesError) Error() string {
	return fmt.Sprintf("Failed to get parent datatypes for %v", e.Datatype)
}

// GetChildDatatypesError provides an error message for failure to get child datatypes.
type GetChildDatatypesError struct {
	Datatype string
}

func (e *GetChildDatatypesError) Error() string {
	return fmt.Sprintf("Failed to get child datatypes for %v", e.Datatype)
}

// PutDatatypeError provides an error message for failure to save a datatype_i.
type PutDatatypeError struct {
	Datatype string
}

func (e *PutDatatypeError) Error() string {
	return fmt.Sprintf("Failed to put datatype %v", e.Datatype)
}

// AddRelationshipError provides an error message for failure to add a datatype relationship.
type AddRelationshipError struct {
	Parent string
	Child  string
}

func (e *AddRelationshipError) Error() string {
	return fmt.Sprintf("Failed to add relationship between datatypes %v and %v", e.Parent, e.Child)
}

// RemoveRelationshipError provides an error message for failure to remove a datatype relationship.
type RemoveRelationshipError struct {
	Parent string
	Child  string
}

func (e *RemoveRelationshipError) Error() string {
	return fmt.Sprintf("Failed to remove relationship between datatypes %v and %v", e.Parent, e.Child)
}

// RoleAccessPrivilegeError provides an error message for access denied for an Action due to callers Role
type RoleAccessPrivilegeError struct {
	Role string
}

func (e *RoleAccessPrivilegeError) Error() string {
	return fmt.Sprintf("Your role does not have privilege to perform this action: Role name %v", e.Role)
}

// Index

// IndexError provides an error message for an index related action failure.
type IndexError struct {
	Index  string
	Action string
}

func (e *IndexError) Error() string {
	return fmt.Sprintf("Index %v failed during %v", e.Index, e.Action)
}

// Consent

// ValidateConsentError provides an error message for validate consent failure.
type ValidateConsentError struct {
	ConsentId string
}

func (e *ValidateConsentError) Error() string {
	return fmt.Sprintf("Failed to validate consent for %v", e.ConsentId)
}

// ConsentAccessError provides an error message for deny or mismatched consent access.
type ConsentAccessError struct {
	ConsentId string
}

func (e *ConsentAccessError) Error() string {
	return fmt.Sprintf("Consent access is deny or mismatch for consent %v", e.ConsentId)
}

// GetConsentError provides an error message for failure to get consent.
type GetConsentError struct {
	ConsentID string
}

func (e *GetConsentError) Error() string {
	return fmt.Sprintf("Failed to get consent for %v", e.ConsentID)
}

// GetConsentsError provides an error message for failure to get consents.
type GetConsentsError struct {
	SortOrder      []string
	PartialKeyList []string
}

func (e *GetConsentsError) Error() string {
	return fmt.Sprintf("Consents not found with sort order: %v and partial key list: %v", e.SortOrder, e.PartialKeyList)
}

// ConsentAssetIsNilError provides an error message for a nil consent asset.
type ConsentAssetIsNilError struct{}

func (e *ConsentAssetIsNilError) Error() string {
	return "Consent asset is nil"
}

// User Access Control

// HasWriteAccessError provides an error message for not having write access when required.
type HasWriteAccessError struct {
	AssetId string
	UserId  string
}

func (e *HasWriteAccessError) Error() string {
	return fmt.Sprintf("%v does not have write access to %v", e.UserId, e.AssetId)
}

// Cached Stub

// MethodNotImplementedError provides an error message for an unknown method inside ChainStub.
type MethodNotImplementedError struct {
	Method string
}

func (e *MethodNotImplementedError) Error() string {
	return fmt.Sprintf("%v is not implemented in ChaincoddeStub", e.Method)
}
