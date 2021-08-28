module chainmaker-go/tools/wal-parse

go 1.16

replace (
	chainmaker.org/chainmaker-go/common => ../../common
	chainmaker.org/chainmaker-go/localconf => ../../module/conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../module/logger
	chainmaker.org/chainmaker-go/pb/protogo => ../../pb/protogo
)

require (
	chainmaker.org/chainmaker-go/common v0.0.0
	chainmaker.org/chainmaker-go/pb/protogo v0.0.0
	github.com/gogo/protobuf v1.3.2
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
)
