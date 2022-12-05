/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package init_common contains a single function Init which initializes the indices for all packages in common.
// Init should be called on startup by all solutions.
package init_common

import (
	"common/bchcls/asset_mgmt"
	"common/bchcls/cached_stub"
	"common/bchcls/consent_mgmt"
	"common/bchcls/crypto"
	"common/bchcls/data_model"
	"common/bchcls/datastore"
	"common/bchcls/datastore/datastore_manager"
	"common/bchcls/datatype"
	"common/bchcls/history"
	"common/bchcls/index"
	"common/bchcls/internal/common/metering_connections"
	"common/bchcls/internal/metering_i"
	"common/bchcls/user_mgmt"
	"common/bchcls/utils"

	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("init_common")

// ====================================================================
//               This function is called by init of solution
// ====================================================================

// Init initializes all solutions.
// It constructs and saves index tables for asset, user, consent, and history.
// It also registers the ROOT datatype and sets up default ledger datastore connection.
// All solutions must call init_common.Init during solution set up time.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {

	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	logger.Infof("Global Data Init Called")

	_ = metering_connections.SetMeteringEnv()

	//Index to be used by common package

	//Asset Index
	ret, err := asset_mgmt.Init(stub, logLevel...)
	if err != nil {
		return ret, err
	}

	//User Index
	ret, err = user_mgmt.Init(stub, logLevel...)
	if err != nil {
		return ret, err
	}

	//Consent Index
	ret, err = consent_mgmt.Init(stub, logLevel...)
	if err != nil {
		return ret, err
	}

	//History Index
	ret, err = history.Init(stub, logLevel...)
	if err != nil {
		return ret, err
	}

	//datatype
	ret, err = datatype.Init(stub, logLevel...)
	if err != nil {
		return ret, err
	}

	//datastore
	ret, err = datastore_manager.Init(stub, logLevel...)
	if err != nil {
		return ret, err
	}

	//index
	ret, err = index.Init(stub, logLevel...)
	if err != nil {
		return ret, err
	}

	logger.Infof("Global Data Init Completed")
	return nil, nil
}

// InitDatastore is to be called on startup by solutions that will use the default Cloudant datastore
// It must be called after init_common.Init().
// It initializes the default datastore.
// args = [userID, password, database, host]
// example ["0d993c4d-efd0-49f4-a653-a33c2492f405-bluemix",
//          "2844a1f42798f0e8282f2a77424d779632f08088475068f6013b7f9b17234999",
//          "testdatabase",
//          "https://0d993c4d-efd0-49f4-a653-a33c2492f405-bluemix.cloudantnosqldb.appdomain.cloud"]
func InitDatastore(stub cached_stub.CachedStubInterface, args ...string) ([]byte, error) {

	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	if len(args) < 4 {
		logger.Error("Wrong number of args for InitDatastore")
		return nil, errors.New("Wrong number of args for InitDatastore")
	}
	return datastore_manager.InitDefaultDatastore(stub, args...)
}

// InitSetup does the following three things:
// 1. Run a very simple self test by writing to the ledger and printing out on any errors.
// It could be useful to read the return result and verify network is healthy.
// 2. Initialize common packages by calling init_common.Init() with logLevel.
// 3. Initialize default Cloudant datastore if the fist args is "_cloudant".
//
// optional args = ["_loglevel" loglevel, "_cloudant", username, password, database, host, (drop_table) ]
//
// username, password, database, host will be set by the Network Operator in a production environment.
// Set username, password, database, host values in solution.yaml for your local test.
// drop_table [true|false] is an  optional param - default value is false.
// If it's set to true, database will be dropped and re-created during initialization.
//
// loglevel = "DEBUG" | "INFO" | "NOTICE" | "WARNING" |"ERROR" | "CRITICAL"
// Refer to Shim's LoggingLevel for more information.
func InitSetup(stub cached_stub.CachedStubInterface) (shim.LoggingLevel, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_, args := stub.GetFunctionAndParameters()
	logLevel := shim.LogInfo

	// 1. Run a very simple self test by tryingn to put a ledger item
	// this is a very simple test. let's write to the ledger and error out on any errors
	// it's handy to read this right away to verify network is healthy if it wrote the correct value
	err := stub.PutState("selftest", []byte("init"))
	if err != nil {
		logger.Errorf("self test failed: %v", err)
		return logLevel, errors.New("self test failed") //self-test fail
	}

	// 2. Initialize call common packages by calling init_common.Init() with logLevel
	if len(args) >= 2 && args[0] == "_loglevel" {
		logopt := args[1]
		if logopt == "INFO" {
			logLevel = shim.LogInfo
		} else if logopt == "DEBUG" {
			logLevel = shim.LogDebug
		} else if logopt == "ERROR" {
			logLevel = shim.LogError
		} else if logopt == "CRITICAL" {
			logLevel = shim.LogCritical
		} else if logopt == "NOTICE" {
			logLevel = shim.LogNotice
		} else if logopt == "WARNING" {
			logLevel = shim.LogWarning
		}
		args = args[2:]
	}
	_, err = Init(stub, logLevel)
	if err != nil {
		logger.Errorf("Failed to run common Init: %v", err)
		return logLevel, errors.New("Failed to run common Init")
	}

	// 3. Initialize default datastore if the fist args is "_cloudant"
	if len(args) >= 5 && args[0] == "_cloudant" {
		_, err = InitDatastore(stub, args[1:]...)
		if err != nil {
			logger.Errorf("Failed to run common InitDatastore: %v", err)
			return logLevel, errors.New("Failed to run common InitDatastore")
		}
		// verify that values by trying to instantiate the datastore
		_, err = datastore_manager.GetDatastoreImpl(stub, datastore.DEFAULT_CLOUDANT_DATASTORE_ID)

		if err != nil {
			logger.Errorf("Failed to run common InitDatastore: %v", err)
			return logLevel, errors.New("Failed to run common InitDatastore")
		}

	}
	logger.Info("InitSetup completed successfully")
	return logLevel, nil
}

//InitByInvoke checks the caller and creates the App Admin User
func InitByInvoke(stub cached_stub.CachedStubInterface, args []string) error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))

	_ = metering_i.SetEnvAndAddRow(stub)

	//var login = UserLogin{}
	caller, err := user_mgmt.GetCallerData(stub)
	if err != nil || caller.ID == "" {
		logger.Errorf("login failed: %v", err)
		return errors.New("login failed")
	} else {
		//	json.Unmarshal(loginBytes, &login)
		logger.Debugf("got login: %v", caller)
	}

	// 1. create App Admin User
	var user = data_model.User{}
	var userBytes = []byte(args[0])
	json.Unmarshal(userBytes, &user)
	logger.Debugf("App admin user: %v", user)
	_, err = user_mgmt.RegisterUser(stub, caller, []string{args[0], "true"})
	if err != nil {
		logger.Errorf("Unable to register app admin user: %v", err)
		return errors.New("Unable to register app admin user")
	}
	logger.Info("InitByInvoke completed successfully")
	return nil
}

// InvokeSetup performs following:
// - checks caller's identity and caller's keys are retrieved
// - performs chaincode login
// - args are decrupted if args are encrypted
// - phi_args are retrieved and parsed
// - run InitByInvoke to create
//
// Returns caller, function, args, toReturn, error
// When toReturn is set to true, the caller should return the shim.result (nil)
func InvokeSetup(stub cached_stub.CachedStubInterface) (data_model.User, string, []string, bool, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	function, args := stub.GetFunctionAndParameters()

	// login
	caller, err := user_mgmt.GetCallerData(stub)
	if err != nil || caller.ID == "" {
		logger.Errorf("login error: %v", err)
		return caller, function, args, false, errors.New("login error")
	}
	privkey := caller.PrivateKey

	// parse args
	if len(args) > 0 {
		if args[0] == "encrypted:" && len(args) != 2 {
			logger.Errorf("Invalid number parameter: Number of args must be 2 for encrypted invoke : %v", len(args))
			return caller, function, args, false, errors.New("Invalid number parameter: Number of args must be 2 for encrypted invoke")
		}

		if args[0] == "encrypted:" {
			encryptedBytes, _ := hex.DecodeString(args[1])
			logger.Debugf("encryptedHex: %v", args[1])
			logger.Debugf("encryptedBytes: %x", encryptedBytes)
			decryptedBytes, err := rsa.DecryptPKCS1v15(rand.Reader, privkey, encryptedBytes)
			logger.Errorf("decrypted: %v %v", string(decryptedBytes[:]), err)
			var args2 []string
			json.Unmarshal(decryptedBytes, &args2)
			args = args2
			logger.Debugf("decypted args: %v", args)
		}
	}

	// try to parse phi args
	var tmap map[string][]byte
	tmap, err = stub.GetTransient()
	if err == nil && tmap != nil {
		// get priv key
		numStr, ok := tmap["num_args"]
		if ok && len(numStr) > 0 {
			num, _ := strconv.Atoi(string(numStr))
			num_args := len(args)
			if num > 0 && num_args < num {
				logger.Errorf("number of args and phi args does not match: len args: %v, len phi args: %v", num_args, num)
				return caller, function, args, false, errors.New("invalid phi args: number of args and phi args does not match")
			}
			for i := 0; i <= num; i++ {
				n := strconv.Itoa(i)
				val, ok := tmap["arg"+n]
				if ok {
					//check phi arg hash using sha512
					hash := crypto.Hash(val)
					hexHash := strings.ToUpper(hex.EncodeToString(hash))
					argIndex := num_args - num + i
					argHash := strings.ToUpper(args[argIndex])
					if argHash == hexHash {
						// replace hash with actual value
						args[argIndex] = string(val)
					} else {
						logger.Errorf("hash of phi args does not match for arg" + n)
						logger.Debugf("i:%v, arg i:%v, arg hash: %v, phi arg hash: %v", i, argIndex, argHash, hexHash)
						return caller, function, args, false, errors.New("invalid phi args: hash of phi args does not match")
					}
				}
			}
			logger.Infof("number of phi args successfully parsed: %v", num)
		}
	}

	// Init by Invoke
	toReturn := false
	if function == "init" {
		err = InitByInvoke(stub, args)
		if err != nil {
			logger.Errorf("initByInvoke failed: %v", err)
			return caller, function, args, true, err
		}
		toReturn = true
	}

	logger.Info("InvokeSetup completed successfully")
	return caller, function, args, toReturn, nil
}
