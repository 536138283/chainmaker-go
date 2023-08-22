module chainmaker.org/chainmaker-go

go 1.16

require (
	chainmaker.org/chainmaker/chainconf/v3 v3.0.1-0.20230720141624-cbf816ebbaad
	chainmaker.org/chainmaker/common/v3 v3.0.1-0.20230724053752-2714154e9069
	chainmaker.org/chainmaker/consensus-abft/v3 v3.0.1-0.20230314074553-134b54d23351
	chainmaker.org/chainmaker/consensus-dpos/v3 v3.0.1-0.20230407102347-455cf7024e05
	chainmaker.org/chainmaker/consensus-maxbft/v3 v3.0.1-0.20230703081204-fa2aa05da9eb
	chainmaker.org/chainmaker/consensus-raft/v3 v3.0.1-0.20230724024044-966261e08ee3
	chainmaker.org/chainmaker/consensus-solo/v3 v3.0.1-0.20230314074753-015c3abcec07
	chainmaker.org/chainmaker/consensus-tbft/v3 v3.0.1-0.20230728083550-6240b6c33aee
	chainmaker.org/chainmaker/consensus-utils/v3 v3.0.1-0.20230221100751-02e92d01a8d6
	chainmaker.org/chainmaker/localconf/v3 v3.0.1-0.20230720141317-c3b4502e2792
	chainmaker.org/chainmaker/logger/v3 v3.0.0
	chainmaker.org/chainmaker/net-common v1.3.1-0.20230728092037-e374a414e360
	chainmaker.org/chainmaker/net-libp2p v1.3.1-0.20230728092025-dfbeab83c976
	chainmaker.org/chainmaker/net-liquid v1.3.1-0.20230819082858-17777e0b8d45
	chainmaker.org/chainmaker/pb-go/v3 v3.0.1-0.20230821080207-ff9a5ec9e5de
	chainmaker.org/chainmaker/protocol/v3 v3.0.1-0.20230822085606-90f975acdbd4
	chainmaker.org/chainmaker/sdk-go/v3 v3.0.1-0.20230803070130-f6042a6a7b55
	chainmaker.org/chainmaker/store-huge/v3 v3.0.1-0.20230724101603-b4b3d6f0b527
	chainmaker.org/chainmaker/store/v3 v3.0.1-0.20230726073456-c2221a8d7e93
	chainmaker.org/chainmaker/txpool-batch/v3 v3.0.1-0.20230728091946-3b94e1b4a27b
	chainmaker.org/chainmaker/txpool-normal/v3 v3.0.1-0.20230728092001-675a90acbd59
	chainmaker.org/chainmaker/txpool-single/v3 v3.0.1-0.20230728092013-89d88d4b768b
	chainmaker.org/chainmaker/utils/v3 v3.0.1-0.20230810060544-368b1884aa38
	chainmaker.org/chainmaker/vm-docker-go/v3 v3.0.1-0.20230221095955-3188d613b0c1
	chainmaker.org/chainmaker/vm-engine/v3 v3.0.1-0.20230811092531-e7604015a070
	chainmaker.org/chainmaker/vm-evm/v3 v3.0.1-0.20230724112908-eeaf88648204
	chainmaker.org/chainmaker/vm-gasm/v3 v3.0.1-0.20230221095629-aa99ef9de0ca
	chainmaker.org/chainmaker/vm-native/v3 v3.0.1-0.20230822100332-4dbda3f1a73e
	chainmaker.org/chainmaker/vm-wasmer/v3 v3.0.1-0.20230726065243-a0dd6f3544e3
	chainmaker.org/chainmaker/vm-wxvm/v3 v3.0.1-0.20230221095707-a6ef1151257e
	chainmaker.org/chainmaker/vm/v3 v3.0.1-0.20230822095744-dc5e5113c537
)

require (
	code.cloudfoundry.org/bytefmt v0.0.0-20211005130812-5bb3c17173e5
	github.com/Rican7/retry v0.1.0
	github.com/Workiva/go-datastructures v1.0.53
	github.com/c-bata/go-prompt v0.2.2
	github.com/common-nighthawk/go-figure v0.0.0-20210622060536-734e95fb86be
	github.com/go-echarts/go-echarts/v2 v2.2.4
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
	github.com/gosuri/uilive v0.0.4 // indirect
	github.com/gosuri/uiprogress v0.0.1
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20201222001619-a42f9ac2ec8e
	github.com/holiman/uint256 v1.2.0
	github.com/hpcloud/tail v1.0.0
	github.com/huin/goupnp v1.0.1-0.20210310174557-0ca763054c88 // indirect
	github.com/linvon/cuckoo-filter v0.4.0
	github.com/mattn/go-tty v0.0.0-20180907095812-13ff1204f104 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/mapstructure v1.5.0
	github.com/mr-tron/base58 v1.2.0
	github.com/panjf2000/ants/v2 v2.4.8
	github.com/pkg/term v0.0.0-20180730021639-bffc007b7fd5 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.8.1
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/tidwall/pretty v1.2.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20201229170055-e5319fda7802
	go.uber.org/atomic v1.7.0
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/time v0.0.0-20210608053304-ed9ce3a009e4
	google.golang.org/grpc v1.47.0
	google.golang.org/protobuf v1.28.0
)

replace (
	github.com/RedisBloom/redisbloom-go => chainmaker.org/third_party/redisbloom-go v1.0.0
	github.com/dgraph-io/badger/v3 => chainmaker.org/third_party/badger/v3 v3.0.0
	github.com/gomodule/redigo => github.com/gomodule/redigo v1.8.2
	github.com/libp2p/go-conn-security-multistream v0.2.0 => chainmaker.org/third_party/go-conn-security-multistream v1.0.0
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v1.0.0
	github.com/linvon/cuckoo-filter => chainmaker.org/third_party/cuckoo-filter v1.0.1
	github.com/lucas-clemente/quic-go => chainmaker.org/third_party/quic-go v1.2.0
	github.com/marten-seemann/qtls-go1-16 => chainmaker.org/third_party/qtls-go1-16 v1.2.0
	github.com/marten-seemann/qtls-go1-17 => chainmaker.org/third_party/qtls-go1-17 v1.2.0
	github.com/marten-seemann/qtls-go1-18 => chainmaker.org/third_party/qtls-go1-18 v1.2.0
	github.com/marten-seemann/qtls-go1-19 => chainmaker.org/third_party/qtls-go1-19 v1.1.0
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.14.8
	github.com/oleiade/reflections => github.com/oleiade/reflections v1.0.1
	github.com/syndtr/goleveldb => chainmaker.org/third_party/goleveldb v1.1.0
	github.com/tikv/client-go => chainmaker.org/third_party/tikv-client-go v1.0.0
)
