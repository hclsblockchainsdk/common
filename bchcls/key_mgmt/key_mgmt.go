/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package key_mgmt manages lower level key access.
// However, most of the functions are not exposed since key access control
// should be used through higher level packages, such as user_access_ctrl,
// asset_mgmt, and consent_mgmt.
//
// Key graph scenarios:
//
// Adding an asset with datatypes [d1, d2] by asset owner "o"
//      o.pri -> a.sym
//      d1.sym -> a.sym
//      d2.sym -> a.sym
//      a.sym -> assetData
//
// Adding a datatype key "d" for owner "o"
// pd: parent datatype
//      o.sym -> d.sym
//      pd.sym -> d.sym
//      d.sym -> datatypeAsset
//
// New user "u"
//      u.pri -> u.sym
//      u.privHash -> u.pri
//      u.pri -> u.privHash
//      u.sym -> u.log
//      u.sym -> userDataAsset
//
// Set  allowAccess = true  when registering a user (c:caller -> u:user)
//     c.pri -> u.privHash
//
// The following paths get added when a user becomes an admin of a group (u:user -> g:group)
//      u.pri -> g.privHash
//      u.pri -> g.sym
//
// User becomes a member of a group (u:user -> g:group)
//      u.pri -> g.sym
//
// Parent (pg) and sub group (sg) (pg -> sg)
//      pg.pri -> sg.privHash
//      pg.log -> sg.log
//      pg.sym -> sg.sym
//      sg.sym -> pg.sym
//
// User "u" gives auditor permisison to aditor "a"
//      a.pri -> u.log
//
// Owner "o" gives a consent "c" to target "t" for an asset "a"
//      t.pri -> c.sym
//      c.sym -> a.sym
//      a.sym -> c.sym
//      c.sym -> consentAsset
//
// Onwer "o" gives consent "c" to target "t" for datatype "d"
//      o.pri -> c.sym
//      t,pri -> c.sym
//      c.sym -> d.sym
//      c.sym -> consentAsset

package key_mgmt

import (
	"common/bchcls/cached_stub"
	"common/bchcls/internal/common/global"
	"common/bchcls/internal/key_mgmt_i"
	"common/bchcls/internal/metering_i"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("key_mgmt")

///////////////////////////////////////////////////////
// Key management

// KEY_TYPE_PRIVATE is a Key.Type option that specifies a private key.
const KEY_TYPE_PRIVATE = global.KEY_TYPE_PRIVATE

// KEY_TYPE_PUBLIC is a Key.Type option that specifies a public key.
const KEY_TYPE_PUBLIC = global.KEY_TYPE_PUBLIC

// KEY_TYPE_SYM is a Key.Type option that specifies a sym key.
const KEY_TYPE_SYM = global.KEY_TYPE_SYM

// GetPubPrivKeyId returns the ID that should be assigned to a public or private key.
func GetPubPrivKeyId(id string) string {
	return key_mgmt_i.GetPubPrivKeyId(id)
}

// GetSymKeyId returns the ID that should be assigned to a sym key.
func GetSymKeyId(id string) string {
	return key_mgmt_i.GetSymKeyId(id)
}

// GetLogSymKeyId returns the ID that should be assigned to a log sym key.
func GetLogSymKeyId(id string) string {
	return key_mgmt_i.GetLogSymKeyId(id)
}

// GetPrivateKeyHashSymKeyId returns the ID that should be assigned to a sym key derived from the hash of a private key.
func GetPrivateKeyHashSymKeyId(id string) string {
	return key_mgmt_i.GetPrivateKeyHashSymKeyId(id)
}

// KeyExists checks if a key with the given keyId exists in the ledger.
func KeyExists(stub cached_stub.CachedStubInterface, keyId string) bool {

	_ = metering_i.SetEnvAndAddRow(stub)

	return key_mgmt_i.KeyExists(stub, keyId)
}

// GetKey returns the key bytes of the last key in the keyIdList if the given startKey can
// be used to decrypt the second key in the list.
// StartKey's ID should be the first ID in the keyIdList.
func GetKey(stub cached_stub.CachedStubInterface, keyIdList []string, startKey []byte) ([]byte, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return key_mgmt_i.GetKey(stub, keyIdList, startKey)
}

// VerifyAccessPath checks if all edges in the path exist.
func VerifyAccessPath(stub cached_stub.CachedStubInterface, path []string) (bool, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	return key_mgmt_i.VerifyAccessPath(stub, path)
}
