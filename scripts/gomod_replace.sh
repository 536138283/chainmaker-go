#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

set -x
cd ..
go mod edit -replace github.com/syndtr/goleveldb=chainmaker.org/third_party/goleveldb@chainmaker
go mod tidy
go mod edit -replace 	github.com/RedisBloom/redisbloom-go=chainmaker.org/third_party/redisbloom-go@chainmaker
go mod tidy
go mod edit -replace 	github.com/dgraph-io/badger/v3=chainmaker.org/third_party/badger/v3@chainmaker
go mod tidy
go mod edit -replace 	github.com/linvon/cuckoo-filter=chainmaker.org/third_party/cuckoo-filter@chainmaker
go mod tidy
go mod edit -replace 	github.com/lucas-clemente/quic-go=chainmaker.org/third_party/quic-go@chainmaker
go mod tidy
go mod edit -replace 	github.com/marten-seemann/qtls-go1-15=chainmaker.org/third_party/qtls-go1-15@chainmaker
go mod tidy
go mod edit -replace 	github.com/marten-seemann/qtls-go1-16=chainmaker.org/third_party/qtls-go1-16@chainmaker
go mod tidy
go mod edit -replace 	github.com/marten-seemann/qtls-go1-17=chainmaker.org/third_party/qtls-go1-17@chainmaker
go mod tidy
go mod edit -replace 	github.com/marten-seemann/qtls-go1-18=chainmaker.org/third_party/qtls-go1-18@chainmaker
go mod tidy
go mod edit -replace 	github.com/tikv/client-go=chainmaker.org/third_party/tikv-client-go@chainmaker
go mod tidy
make
make cmc