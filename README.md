# Common SDK
This repo hosts Blockchain for Healthcare Common Go packages and is intended to be used by Solution developers as sdk in developing Go chaincodes for Hyperledger Fabric. It provides common functionality like Key management, User management, User access control, Consent, Asset management, Data encryption/decryption etc.


### Prerequisites
- Common SDK was built and verified on
  - golang 1.12.12
  - Hyperledger Fabric 1.4.7 
----


## Development


#### Cloning

To test & develop this repo by itself, clone it to your `GOPATH`, e.g. `Users\JohnDoe\go\src\common`.

To use it in a chaincode solution, add `common` as git `submodule` in the solution's `vendor` directory (for e.g.  solution_chaincode/vendor/common).

#### Building Common SDK Go Code
```
export GO111MODULE=auto

cd bchcls
go build --tags nopkcs11 ./...
```

#### Running Go Tests

The offchain db unit tests, require IBM Cloudant dev container to be running:
```
#To start cloudant container, do following root folder
docker-compose -f docker-compose-cloudant.yaml up -d
```

```
cd bchcls
go test --tags nopkcs11 ./...
```

#### Best practices
- Format your Go Code via IDE or use
```
go fmt ./...
```
