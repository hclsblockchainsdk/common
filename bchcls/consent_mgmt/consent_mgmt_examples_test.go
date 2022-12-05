/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package consent_mgmt

import (
	"common/bchcls/cached_stub"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/test_utils"

	"encoding/json"
	"strconv"
	"time"
)

func ExamplePutConsent() {
	mstub := test_utils.CreateExampleMockStub()

	// assume owner, target, and datatype1 exist on the ledger
	ownerUser := test_utils.CreateTestUser("owner")
	targetUser := test_utils.CreateTestUser("target")
	dtype := data_model.Datatype{
		DatatypeID:  "datatype1",
		Description: "test datatype",
		IsActive:    true,
	}

	consentDate := time.Now().Unix()
	consent := data_model.Consent{
		OwnerID:        ownerUser.ID,
		TargetID:       targetUser.ID,
		DatatypeID:     dtype.DatatypeID,
		Access:         ACCESS_READ,
		ConsentDate:    consentDate,
		ExpirationDate: consentDate + 60*60*24,
		Data:           make(map[string]interface{}),
	}
	consentBytes, _ := json.Marshal(&consent)

	consentKey := test_utils.GenerateSymKey()
	consentKeyB64 := crypto.EncodeToB64String(consentKey)

	mstub.MockTransactionStart("transaction1")
	stub := cached_stub.NewCachedStub(mstub)
	PutConsent(stub, ownerUser, []string{string(consentBytes), consentKeyB64})
	mstub.MockTransactionEnd("transaction1")
}

func ExampleGetConsent() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	// assume owner, target, datatype1, and consents exist on the ledger
	ownerUser := test_utils.CreateTestUser("owner")
	targetUser := test_utils.CreateTestUser("target")

	// get datatype consent
	GetConsent(stub, targetUser, []string{"datatype1", targetUser.ID, ownerUser.ID})
}

func ExampleValidateConsent() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	// assume owner, target, datatype1, and consents exist on the ledger
	ownerUser := test_utils.CreateTestUser("owner")
	targetUser := test_utils.CreateTestUser("target")
	currTime := time.Now().Unix()

	args := []string{"datatype1", "", ownerUser.ID, targetUser.ID, ACCESS_READ, strconv.FormatInt(currTime, 10)}
	ValidateConsent(stub, targetUser, args)
}

func ExampleGetConsentsWithOwnerID() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	// assume owner and consents exist on the ledger
	ownerUser := test_utils.CreateTestUser("owner")

	GetConsentsWithOwnerID(stub, ownerUser, []string{ownerUser.ID})
}

func ExampleGetConsentsWithTargetID() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	// assume target and consents exist on the ledger
	targetUser := test_utils.CreateTestUser("target")

	GetConsentsWithTargetID(stub, targetUser, []string{targetUser.ID})
}

func ExampleGetConsentsWithCallerID() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	// assume caller and consents exist on the ledger
	caller := test_utils.CreateTestUser("creator")

	GetConsentsWithCallerID(stub, caller, []string{})
}

func ExampleGetConsentsWithOwnerIDAndDatatypeID() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	// assume owner, datatype1, and consents exist on the ledger
	ownerUser := test_utils.CreateTestUser("owner")

	GetConsentsWithOwnerIDAndDatatypeID(stub, ownerUser, []string{ownerUser.ID, "datatype1"})
}

func ExampleGetConsentsWithTargetIDAndDatatypeID() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	// assume target, datatype1, and consents exist on the ledger
	targetUser := test_utils.CreateTestUser("target")

	GetConsentsWithTargetIDAndDatatypeID(stub, targetUser, []string{targetUser.ID, "datatype1"})
}

func ExampleGetConsentsWithTargetIDAndOwnerID() {
	stub := cached_stub.NewCachedStub(test_utils.CreateExampleMockStub())

	// assume owner, target, and consents exist on the ledger
	ownerUser := test_utils.CreateTestUser("owner")
	targetUser := test_utils.CreateTestUser("target")

	GetConsentsWithTargetIDAndOwnerID(stub, ownerUser, []string{targetUser.ID, ownerUser.ID})
}
