module chainmaker.org/chainmaker-go/consensus

go 1.15

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0
	chainmaker.org/chainmaker-go/chainconf v0.0.0
	chainmaker.org/chainmaker-go/common v0.0.0
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/mock v0.0.0
	chainmaker.org/chainmaker-go/pb/protogo v0.0.0
	chainmaker.org/chainmaker-go/protocol v0.0.0
	chainmaker.org/chainmaker-go/raftwal v0.0.0
	chainmaker.org/chainmaker-go/store v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker-go/vm v0.0.0-00010101000000-000000000000
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.5.0
	github.com/golang/protobuf v1.5.2
	github.com/jfcg/sorty v1.0.15
	github.com/kr/pretty v0.2.0 // indirect
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/thoas/go-funk v0.8.0
	go.etcd.io/etcd/client/pkg/v3 v3.5.0
	go.etcd.io/etcd/raft/v3 v3.5.0
	go.etcd.io/etcd/server/v3 v3.5.0
	go.uber.org/zap v1.19.1
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ./../../module/accesscontrol
	chainmaker.org/chainmaker-go/chainconf => ./../conf/chainconf
	chainmaker.org/chainmaker-go/common => ../../common
	chainmaker.org/chainmaker-go/evm => ./../../module/vm/evm
	chainmaker.org/chainmaker-go/gasm => ./../../module/vm/gasm
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ./../logger
	chainmaker.org/chainmaker-go/mock => ../../mock
	chainmaker.org/chainmaker-go/pb/protogo => ../../pb/protogo
	chainmaker.org/chainmaker-go/protocol => ./../../protocol
	chainmaker.org/chainmaker-go/raftwal => ./raft/raftwal
	chainmaker.org/chainmaker-go/store => ./../../module/store
	chainmaker.org/chainmaker-go/utils => ../utils
	chainmaker.org/chainmaker-go/vm => ./../../module/vm
	chainmaker.org/chainmaker-go/wasi => ./../../module/vm/wasi
	chainmaker.org/chainmaker-go/wasmer => ./../../module/vm/wasmer
	chainmaker.org/chainmaker-go/wxvm => ./../../module/vm/wxvm
	github.com/libp2p/go-libp2p-core => ../net/p2p/libp2pcore
)
