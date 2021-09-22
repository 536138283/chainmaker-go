module chainmaker.org/chainmaker-go/net

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker/chainmaker-net-common v0.0.5
	chainmaker.org/chainmaker/chainmaker-net-libp2p v0.0.10
	chainmaker.org/chainmaker/chainmaker-net-liquid v0.0.7
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210906085649-78f6202d8d60
	chainmaker.org/chainmaker/pb-go/v2 v2.0.0
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210906092203-47d66f4908f7
	github.com/gogo/protobuf v1.3.2
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ./../logger

	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
)
