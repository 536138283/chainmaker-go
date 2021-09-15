#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -x
BRANCH=develop

cd ../module/accesscontrol
go mod tidy
# go test ./...
cd ../blockchain
go mod tidy
# go test ./...
cd ../conf/chainconf
go mod tidy
cd ../localconf
go mod tidy
cd ../../consensus
go mod tidy
# go test ./...
cd ../core
go mod tidy
# go test ./...
cd ../dpos
go mod tidy
cd ../logger
go mod tidy
# go test ./...
cd ../net
go mod tidy
# go test ./...
cd ../rpcserver
go mod tidy
## go test ./...
cd ../snapshot
go mod tidy
cd ../store
go mod tidy
# go test ./...
cd ../subscriber

go mod tidy
# go test ./...
cd ../sync
go mod tidy
# go test ./...
cd ../txpool
go mod tidy
# go test ./...
cd ../utils
go mod tidy
cd ../vm
go mod tidy
cd gasm
go mod tidy
cd ../evm
go mod tidy
cd ../wasi
go mod tidy
cd ../wasmer
go mod tidy
cd ../wxvm
go mod tidy
cd ../../../test



go mod tidy
#go build ./...
cd ../tools/cmc


go mod tidy
## go test ./...
go build .
cd ../scanner
go mod tidy
cd ../../main
go mod tidy
go build .