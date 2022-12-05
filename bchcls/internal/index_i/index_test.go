/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package index_i

import (
	"common/bchcls/crypto"
	"common/bchcls/test_utils"

	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func setup(t *testing.T) *test_utils.NewMockStub {
	logger.SetLevel(shim.LogDebug)
	mstub := test_utils.CreateNewMockStub(t)
	return mstub
}

func TestEncryptKeys(t *testing.T) {
	_ = setup(t)

	// hash
	h := crypto.HashShort([]byte("firsthash"))

	// sorted chars
	s := "0123456789!@#\"\\$%^&*():;'<>?,./abcdefghijklmopqrstuvwxABCDEFGHIJKLMNOPQRSTUVWXYZ "
	s1 := []string{}

	for i := 0; i < len(s); i++ {
		s1 = append(s1, s[i:i+1])
	}
	sort.Strings(s1)
	logger.Debugf("sorted:", s1)

	// test strings
	keys := []string{}

	for i := 0; i < len(s1); i++ {
		a := ""
		for j := 0; j < 10; j++ {
			a = a + s1[i]
		}
		keys = append(keys, a)
	}

	//add random keys
	for i := 0; i < 30; i++ {
		n := fmt.Sprintf("%v", rand.Int())
		keys = append(keys, n[0:10])
	}

	// sort keys
	sort.Strings(keys)

	// get encrypted keys
	ekeys := []string{}
	for i := 0; i < len(keys); i++ {
		s := encryptKey(h, keys[i])
		ekeys = append(ekeys, s)
	}

	// copy ekeys
	sekeys := make([]string, len(keys))
	copy(sekeys, ekeys)

	// sort sekeys
	sort.Strings(sekeys)

	// compare results
	allmatched := true
	for i := 0; i < len(keys); i++ {
		logger.Debugf("%v : %v : %v : %v", keys[i], ekeys[i], sekeys[i], ekeys[i] == sekeys[i])
		allmatched = allmatched && ekeys[i] == sekeys[i]
	}

	test_utils.AssertTrue(t, allmatched, "key order din't match")
}
