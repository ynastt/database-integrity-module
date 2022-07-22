#!/bin/bash
export GOPATH=`pwd`
go get github.com/arangodb/go-driver
go get github.com/Toorop/go-bitcoind
go build ./src/bitcoin_rpc
go build ./src/arango
go build ./src/check_fields
go install ./src/main
