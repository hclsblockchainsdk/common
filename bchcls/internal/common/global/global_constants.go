/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// global package contains global data, variables, constants, or functions to be
// used across all bchcls packages.
// This should be the lowest level package (below data_model and common).

// In global package, only the following bchcls packages are allowed to be imported:
// 	"common/bchcls/cached_stub"
//	"common/bchcls/crypto"
//	"common/bchcls/custom_errors"
//	"common/bchcls/internal/common/graph"
//	"common/bchcls/internal/common/rb_tree"
//

package global

import (
	"unicode/utf8"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("common")

///////////////////////////////////////////////////////
// Common

const MIN_UNICODE_RUNE_VALUE = 0            //U+0000
const MAX_UNICODE_RUNE_VALUE = utf8.MaxRune //U+10FFFF - maximum (and unallocated) code point
const COMPOSITE_KEY_NAMESPACE = "\x00"
const EMPTY_KEY_SUBSTITUDE = "\x01"

///////////////////////////////////////////////////////
// User management

// Asset namespace for User object
const USER_ASSET_NAMESPACE = "data_model.User"

// INDEX_USER stores the name of the user index table.
const INDEX_USER = "User"

const USER_GRAPH = "UserGraph"

const ADMIN_EDGE = "admin"
const MEMBER_EDGE = "member"
const SUBGROUP_EDGE = "subgroup"

// ROLE_SYSTEM_ADMIN is a User.Role option that specifies a system admin.
const ROLE_SYSTEM_ADMIN = "system"

// ROLE_USER is a User.Role option that specifies a user.
const ROLE_USER = "user"

// ROLE_ORG is a User.Role option that specifies an org.
const ROLE_ORG = "org"

// ROLE_AUDIT is a User.Role option that specifies an auditor.
const ROLE_AUDIT = "audit"

/////////////////////////////////////////////////////
// Asset management

// Prefix for all Asset ledger keys.
const ASSET_ID_PREFIX = "asset_"

// Prefix for Asset Cache
const ASSET_CACHE_PREFIX = "assetCache_"
const ASSET_PRIVATE_CACHE_PREFIX = "assetPrivateCache_"

//////////////////////////////////////////////////////
// Access

// ACCESS_READ is an AccessControl.Access or Consent.Access option that specifies read access.
const ACCESS_READ = "read"

// ACCESS_READ_ONLY is an AccessControl.Access option that specifies read access without write access.
// This only to be used when verifying access.
// If this is used when adding access, it is treated as ACCESS_READ.
const ACCESS_READ_ONLY = "read_only"

// ACCESS_WRITE is an AccessControl.Access or Consent.Access option that specifies write access.
const ACCESS_WRITE = "write"

// ACCESS_WRITE_ONLY is an AccessControl.Access option that specifies write access without read access.
const ACCESS_WRITE_ONLY = "write_only"

// ACCESS_DENY is a Consent.Access option that specifies deny access.
const ACCESS_DENY = "deny"

// EDGEDATA_ACCESS_TYPE is a key to be used for edgedata map[string]string.
const EDGEDATA_ACCESS_TYPE = "AccessType"

////////////////////////////////////////////////////////////
// Consent

// CONSENT_PREFIX is the prefix for all consent ledger keys.
const CONSENT_PREFIX = "Consent"
const CONSENT_EDGE = "ConsentEdge"

// INDEX_CONSENT stores the name of the consent index table.
const INDEX_CONSENT = "Consent"

// CONSENT_ASSET_NAMESPACE is the asset namespace for consents.
const CONSENT_ASSET_NAMESPACE = "data_model.Consent"

/////////////////////////////////////////////////////////////
// Datatype

const DATATYPE_GRAPH = "DatatypeGraph"
const DATATYPE_PREFIX = "Datatype"

// ROOT_DATATYPE_ID is id of ROOT datatype_i. All other datatypes are children of ROOT.
const ROOT_DATATYPE_ID = "ROOT"

/////////////////////////////////////////////////////////////
// Datastore

//DATASTORE_TYPE_DEFAULT_CLOUDANT is type for Cloudant DB. This is the default off-chain datastore type
const DATASTORE_TYPE_DEFAULT_CLOUDANT = "ds.Cloudant"

//DATASTORE_TYPE_DEFAULT_LEDGER is the type for default Hyperledger on-chain storage
const DATASTORE_TYPE_DEFAULT_LEDGER = "ds.Ledger"

//DATASTORE_ASSET_METADATA_KEY is the key to define datastore metadata in asset metadata
const DATASTORE_CONNECTION_ID_METADATA_KEY = "ds.ConnectionID"

// default data store IDs
const DEFAULT_LEDGER_DATASTORE_ID = "ledger_"
const DEFAULT_CLOUDANT_DATASTORE_ID = "cloudant_"

/////////////////////////////////////////////////////////////
// Key management

// Prefix for all edges in KeyGraph
const KEY_GRAPH_PREFIX = "KeyGraph"
const REVERSE_GRAPH_PREFIX = "REVERSE_"

// Prefix for all nodes in the KeyGraph
const KEY_NODE_PREFIX = "KeyNode"

// KEY_TYPE_PRIVATE is a Key.Type option that specifies a private key.
const KEY_TYPE_PRIVATE = "private"

// KEY_TYPE_PUBLIC is a Key.Type option that specifies a public key.
const KEY_TYPE_PUBLIC = "public"

// KEY_TYPE_SYM is a Key.Type option that specifies a sym key.
const KEY_TYPE_SYM = "sym"

const KEY_PREFIX_PUB_PRIV = "pub-priv"
const KEY_PREFIX_SYM_KEY = "sym"
const KEY_PREFIX_LOG_SYM_KEY = "log-sym"
const KEY_PREFIX_PRIV_HASH = "private-hash"

/////////////////////////////////////////////////////////////
// History

// TRANSACTION_LOG_ASSET_NAMESPACE is the asset namespace for transaction logs.
const TRANSACTION_LOG_ASSET_NAMESPACE = "history.TransactionLog"

// INDEX_HISTORY stores the name of the history index table.
const INDEX_HISTORY = "History"

/////////////////////////////////////////////////////////////
// Metering

// IBM Cloud Cloudant instance service credentials
// These values will be set by metering_connections when reading config file
var Cloudant_password = ""
var Cloudant_url = ""
var Cloudant_username = ""

// DevelopmentEnv will be dynamically set during Instantiation by the init_common.Init function
var DevelopmentEnv = ""

// ProductionEnvString indicates the code is running in a production environment
const ProductionEnvString = "production"

// StagingEnvString indicates the code is running in a staging environment
const StagingEnvString = "staging"

// DevelopmentEnvString indicates the code is running in a development environment
const DevelopmentEnvString = "development"
