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
)
