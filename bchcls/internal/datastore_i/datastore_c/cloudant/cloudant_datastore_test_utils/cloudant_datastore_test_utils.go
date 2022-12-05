/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package cloudant_datastore_test_utils

import (
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
	"common/bchcls/datastore"
	"common/bchcls/internal/datastore_i"
	"common/bchcls/test_utils"
	"common/bchcls/utils"
	"os"

	"net/url"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("data_store/cloudant_datastore_test_utils")

// SetupDatastore instantiates an offchain Cloudant DB
// Used in consent_mgmt_test, user_mgmt_test, and history_test
func SetupDatastore(mstub *test_utils.NewMockStub, caller data_model.User, datastoreConnectionID string) error {
	mstub.MockTransactionStart("t1")
	stub := cached_stub.NewCachedStub(mstub)

	username := "admin"
	password := "pass"
	database := "test"
	host := "http://127.0.0.1:9080"
	// Get values from environment variables
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_USERNAME")) {
		username = os.Getenv("CLOUDANT_USERNAME")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_PASSWORD")) {
		password = os.Getenv("CLOUDANT_PASSWORD")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_DATABASE")) {
		database = os.Getenv("CLOUDANT_DATABASE")
	}
	if !utils.IsStringEmpty(os.Getenv("CLOUDANT_HOST")) {
		host = os.Getenv("CLOUDANT_HOST")
	}

	params := url.Values{}
	params.Add("username", username)
	params.Add("password", password)
	params.Add("database", database)
	params.Add("host", host)

	connection := datastore.DatastoreConnection{
		ID:         datastoreConnectionID,
		Type:       datastore.DATASTORE_TYPE_DEFAULT_CLOUDANT,
		ConnectStr: params.Encode(),
	}

	err := datastore_i.PutDatastoreConnection(stub, caller, connection)
	mstub.MockTransactionEnd("t1")

	return err
}
