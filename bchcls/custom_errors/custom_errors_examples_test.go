/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package custom_errors

import (
	"fmt"
)

func ExampleAddAccessError_Error() {
	custom_err := &AddAccessError{Key: "key1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to AddAccess to key1
}

func ExampleAddRelationshipError_Error() {
	custom_err := &AddRelationshipError{Parent: "datatype1", Child: "datatype2"}

	fmt.Println(custom_err.Error())
	// Output: Failed to add relationship between datatypes datatype1 and datatype2
}

func ExampleCannotBeGroupError_Error() {
	custom_err := &CannotBeGroupError{GroupID: "user1"}

	fmt.Println(custom_err.Error())
	// Output: userID cannot be a group: "user1"
}

func ExampleCiphertextBlockSizeError_Error() {
	custom_err := &CiphertextBlockSizeError{}

	fmt.Println(custom_err.Error())
	// Output: ciphertext is not a multiple of the block size

}
func ExampleCiphertextEmptyError_Error() {
	custom_err := &CiphertextEmptyError{}

	fmt.Println(custom_err.Error())
	// Output: ciphertext is empty
}

func ExampleCiphertextLengthError_Error() {
	custom_err := &CiphertextLengthError{}

	fmt.Println(custom_err.Error())
	// Output: ciphertext too short
}

func ExampleConsentAccessError_Error() {
	custom_err := &ConsentAccessError{ConsentId: "consent1"}

	fmt.Println(custom_err.Error())
	// Output: Consent access is deny or mismatch for consent consent1
}

func ExampleConsentAssetIsNilError_Error() {
	custom_err := &ConsentAssetIsNilError{}

	fmt.Println(custom_err.Error())
	// Output: Consent asset is nil
}

func ExampleCreateCompositeKeyError_Error() {
	custom_err := &CreateCompositeKeyError{Type: "Asset"}

	fmt.Println(custom_err.Error())
	// Output: Failed to create composite key for Asset
}

func ExampleCycleError_Error() {
	custom_err := &CycleError{Parent: "datatype1", Child: "datatype2"}

	fmt.Println(custom_err.Error())
	// Output: Failed to create edge from datatype1 to datatype2 to prevent cycle
}

func ExampleDecryptionError_Error() {
	custom_err := &DecryptionError{ToDecrypt: "asset", DecryptionKey: "sym_key"}

	fmt.Println(custom_err.Error())
	// Output: Failed to decrypt asset with sym_key
}

func ExampleDeleteLedgerError_Error() {
	custom_err := &DeleteLedgerError{LedgerKey: "ledger_key1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to delete ledger_key1 from ledger
}

func ExampleEncryptionError_Error() {
	custom_err := &EncryptionError{ToEncrypt: "asset", EncryptionKey: "sym_key"}

	fmt.Println(custom_err.Error())
	// Output: Failed to encrypt asset with sym_key
}

func ExampleGetAssetDataError_Error() {
	custom_err := &GetAssetDataError{AssetId: "asset_id1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to get asset data for asset_id1
}

func ExampleGetChildDatatypesError_Error() {
	custom_err := &GetChildDatatypesError{Datatype: "datatype1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to get child datatypes for datatype1
}

func ExampleGetConsentError_Error() {
	custom_err := &GetConsentError{ConsentID: "consent1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to get consent for consent1
}

func ExampleGetConsentsError_Error() {
	custom_err := &GetConsentsError{SortOrder: []string{"owner_id"}, PartialKeyList: []string{"user1"}}

	fmt.Println(custom_err.Error())
	// Output: Consents not found with sort order: [owner_id] and partial key list: [user1]
}

func ExampleGetDatatypeError_Error() {
	custom_err := &GetDatatypeError{Datatype: "datatype1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to get datatype datatype1
}

func ExampleGetDirectChildrenError_Error() {
	custom_err := &GetDirectChildrenError{Parent: "node1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to get direct children of node1
}

func ExampleGetDirectParentsError_Error() {
	custom_err := &GetDirectParentsError{Child: "node1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to get direct parents of node1
}

func ExampleGetEdgeError_Error() {
	custom_err := &GetEdgeError{ParentNode: "node1", ChildNode: "node2"}

	fmt.Println(custom_err.Error())
	// Output: Failed to get graph edge from node1 to node2
}

func ExampleGetLedgerError_Error() {
	custom_err := &GetLedgerError{LedgerKey: "ledger_key1", LedgerItem: "ledger_item1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to get ledger item "ledger_item1" from ledger with ledger key "ledger_key1"
}

func ExampleGetNodesError_Error() {
	custom_err := &GetNodesError{}

	fmt.Println(custom_err.Error())
	// Output: Failed to get nodes
}

func ExampleGetParentDatatypesError_Error() {
	custom_err := &GetParentDatatypesError{Datatype: "datatype1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to get parent datatypes for datatype1
}

func ExampleGetUserError_Error() {
	custom_err := &GetUserError{ID: "user1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to get user user1
}

func ExampleHasWriteAccessError_Error() {
	custom_err := &HasWriteAccessError{AssetId: "asset_id1", UserId: "user1"}

	fmt.Println(custom_err.Error())
	// Output: user1 does not have write access to asset_id1
}

func ExampleIndexError_Error() {
	custom_err := &IndexError{Index: "Asset", Action: "UpdateRow"}

	fmt.Println(custom_err.Error())
	// Output: Index Asset failed during UpdateRow
}

func ExampleInvalidKeyError_Error() {
	custom_err := &InvalidKeyError{KeyId: "key1"}

	fmt.Println(custom_err.Error())
	// Output: Key "key1" doesn't match existing key in graph
}

func ExampleInvalidPrivateKeyError_Error() {
	custom_err := &InvalidPrivateKeyError{}

	fmt.Println(custom_err.Error())
	// Output: Invalid private key by caller
}

func ExampleInvalidPublicKeyError_Error() {
	custom_err := &InvalidPublicKeyError{}

	fmt.Println(custom_err.Error())
	// Output: Invalid public key by caller
}

func ExampleInvalidSymKeyError_Error() {
	custom_err := &InvalidSymKeyError{}

	fmt.Println(custom_err.Error())
	// Output: Invalid symKey: length must be 32 bytes
}

func ExampleIterError_Error() {
	custom_err := &IterError{}

	fmt.Println(custom_err.Error())
	// Output: Error reading next KV
}

func ExampleLengthCheckingError_Error() {
	custom_err := &LengthCheckingError{Type: "datatypeID"}

	fmt.Println(custom_err.Error())
	// Output: Length of datatypeID does not match expected
}

func ExampleMarshalError_Error() {
	custom_err := &MarshalError{Type: "Datatype"}

	fmt.Println(custom_err.Error())
	// Output: Failed to marshal Datatype
}

func ExampleNotGroupAdminError_Error() {
	custom_err := &NotGroupAdminError{UserID: "user1", GroupID: "group1"}

	fmt.Println(custom_err.Error())
	// Output: user1 is not an admin of group1
}

func ExampleParseKeyError_Error() {
	custom_err := &ParseKeyError{Type: "private key"}

	fmt.Println(custom_err.Error())
	// Output: Unable to parse private key
}

func ExamplePutDatatypeError_Error() {
	custom_err := &PutDatatypeError{Datatype: "datatype1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to put datatype datatype1
}

func ExamplePutEdgeError_Error() {
	custom_err := &PutEdgeError{ParentNode: "node1", ChildNode: "node2"}

	fmt.Println(custom_err.Error())
	// Output: Failed to put graph edge from node1 to node2
}

func ExamplePutLedgerError_Error() {
	custom_err := &PutLedgerError{LedgerKey: "ledger_key1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to put ledger_key1 in ledger
}

func ExampleRegisterOrgInvalidFieldError_Error() {
	custom_err := &RegisterOrgInvalidFieldError{ID: "org1", Field: "IsGroup"}

	fmt.Println(custom_err.Error())
	// Output: RegisterOrg invalid field for org1: IsGroup
}

func ExampleRemoveRelationshipError_Error() {
	custom_err := &RemoveRelationshipError{Parent: "datatype1", Child: "datatype2"}

	fmt.Println(custom_err.Error())
	// Output: Failed to remove relationship between datatypes datatype1 and datatype2
}

func ExampleReplaceAssetKeyHashError_Error() {
	custom_err := &ReplaceAssetKeyHashError{}

	fmt.Println(custom_err.Error())
	// Output: Asset exists on ledger with different AssetKey
}

func ExampleReplaceAssetKeyIdError_Error() {
	custom_err := &ReplaceAssetKeyIdError{}

	fmt.Println(custom_err.Error())
	// Output: Asset exists on ledger with different AssetKeyId
}

func ExampleSplitCompositeKeyError_Error() {
	custom_err := &SplitCompositeKeyError{Key: "ledger_key1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to split composite key ledger_key1
}

func ExampleTypeAssertionError_Error() {
	custom_err := &TypeAssertionError{Item: "assetId", Type: "string"}

	fmt.Println(custom_err.Error())
	// Output: assetId is not of string type
}

func ExampleUnmarshalError_Error() {
	custom_err := &UnmarshalError{Type: "Datatype"}

	fmt.Println(custom_err.Error())
	// Output: Failed to unmarshal Datatype
}

func ExampleValidateConsentError_Error() {
	custom_err := &ValidateConsentError{ConsentId: "consent1"}

	fmt.Println(custom_err.Error())
	// Output: Failed to validate consent for consent1
}

func ExampleValidateKeyError_Error() {
	custom_err := &ValidateKeyError{KeyId: "key1"}

	fmt.Println(custom_err.Error())
	// Output: Error cross-checking "key1" with existing key
}

func ExampleVerifyAccessAndGetKeyError_Error() {
	custom_err := &VerifyAccessAndGetKeyError{}

	fmt.Println(custom_err.Error())
	// Output: Failed to VerifyAccess and GetKey
}

func ExampleVerifyAccessError_Error() {
	custom_err := &VerifyAccessError{StartKeyId: "key1", TargetKeyId: "key2"}

	fmt.Println(custom_err.Error())
	// Output: Failed to VerifyAccess from key1 to key2
}
