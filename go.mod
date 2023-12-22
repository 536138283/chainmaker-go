module chainmaker.org/chainmaker-go

go 1.16

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.3.3-0.20230816083956-847f1567baaa
	chainmaker.org/chainmaker/common/v2 v2.3.3-0.20231019022655-8675e0530915
	chainmaker.org/chainmaker/consensus-dpos/v2 v2.3.0
	chainmaker.org/chainmaker/consensus-raft/v2 v2.3.3-0.20231219093913-447efa0cd327
	chainmaker.org/chainmaker/consensus-solo/v2 v2.3.0
	chainmaker.org/chainmaker/consensus-utils/v2 v2.3.4-0.20231115095741-2b342780d5f5
	chainmaker.org/chainmaker/localconf/v2 v2.3.3-0.20230915124130-a7c865e89eee
	chainmaker.org/chainmaker/logger/v2 v2.3.0
	chainmaker.org/chainmaker/net-common v1.2.4-0.20231213082602-c9a5f77403d1
	chainmaker.org/chainmaker/net-libp2p v1.2.4-0.20231213100056-e71405b638e9
	chainmaker.org/chainmaker/net-liquid v1.1.2-0.20231214034940-1c4f12276cf0
	chainmaker.org/chainmaker/pb-go/v2 v2.3.4-0.20231017064938-42b85038dc6b
	chainmaker.org/chainmaker/protocol/v2 v2.3.4-0.20231204094400-869a4cb2b3e1
	chainmaker.org/chainmaker/sdk-go/v2 v2.3.4-0.20230920063444-01f39c4830c1
	chainmaker.org/chainmaker/store/v2 v2.3.5-0.20231214022652-fff379fc984d
	chainmaker.org/chainmaker/utils/v2 v2.3.4-0.20230927084903-acd55dcc1634
	chainmaker.org/chainmaker/vm-docker-go/v2 v2.3.2
	chainmaker.org/chainmaker/vm-engine/v2 v2.3.5-0.20231220015924-acc5d731790b
	chainmaker.org/chainmaker/vm-evm/v2 v2.3.4-0.20230920075210-f222c32fd983
	chainmaker.org/chainmaker/vm-gasm/v2 v2.3.2
	chainmaker.org/chainmaker/vm-native/v2 v2.3.4-0.20231017071518-396de85fa139
	chainmaker.org/chainmaker/vm-wasmer/v2 v2.3.2
	chainmaker.org/chainmaker/vm-wxvm/v2 v2.3.2
	chainmaker.org/chainmaker/vm/v2 v2.3.4-0.20230913112945-35811a2653bb
	code.cloudfoundry.org/bytefmt v0.0.0-20211005130812-5bb3c17173e5
	github.com/Rican7/retry v0.1.0
	github.com/Workiva/go-datastructures v1.0.53
	github.com/c-bata/go-prompt v0.2.2
	github.com/common-nighthawk/go-figure v0.0.0-20210622060536-734e95fb86be
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
	github.com/gosuri/uiprogress v0.0.1
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hokaccha/go-prettyjson v0.0.0-20201222001619-a42f9ac2ec8e
	github.com/holiman/uint256 v1.2.0
	github.com/hpcloud/tail v1.0.0
	github.com/linvon/cuckoo-filter v0.4.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/mr-tron/base58 v1.2.0
	github.com/panjf2000/ants/v2 v2.4.8
	github.com/prometheus/client_golang v1.11.0
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.8.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/tidwall/pretty v1.2.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20201229170055-e5319fda7802
	go.uber.org/atomic v1.7.0
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	golang.org/x/time v0.0.0-20210608053304-ed9ce3a009e4
	google.golang.org/grpc v1.47.0
)

require (
	chainmaker.org/chainmaker/consensus-maxbft/v2 v2.3.3-0.20231219093759-83566a1cac20
	chainmaker.org/chainmaker/consensus-tbft/v2 v2.3.4-0.20231219093835-0a7a2ae5a1af
	chainmaker.org/chainmaker/libp2p-pubsub v1.1.4-0.20231107023105-c3342b56abd5 // indirect
	chainmaker.org/chainmaker/txpool-batch/v2 v2.3.3-0.20231220040820-8a2d93213247
	chainmaker.org/chainmaker/txpool-normal/v2 v2.3.3-0.20231220040805-fd18d1bd9341
	chainmaker.org/chainmaker/txpool-single/v2 v2.3.3-0.20231220040748-88560918ef14
	github.com/go-echarts/go-echarts/v2 v2.2.4
	github.com/gosuri/uilive v0.0.4 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/huin/goupnp v1.0.1-0.20210310174557-0ca763054c88 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-tty v0.0.0-20180907095812-13ff1204f104 // indirect
	github.com/pkg/errors v0.9.1
	github.com/pkg/term v0.0.0-20180730021639-bffc007b7fd5 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)

replace (
	//chainmaker.org/chainmaker/lws => chainmaker.org/chainmaker/lws v1.2.1-0.20230918025726-ea744645e727
	github.com/RedisBloom/redisbloom-go => chainmaker.org/third_party/redisbloom-go v1.0.0
	github.com/dgraph-io/badger/v3 => chainmaker.org/third_party/badger/v3 v3.0.0
	github.com/libp2p/go-conn-security-multistream v0.2.0 => chainmaker.org/third_party/go-conn-security-multistream v1.0.2
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v1.0.0
	github.com/linvon/cuckoo-filter => chainmaker.org/third_party/cuckoo-filter v1.0.0
	github.com/lucas-clemente/quic-go v0.26.0 => chainmaker.org/third_party/quic-go v1.2.1-0.20230821024043-27eaf3d844cd
	github.com/marten-seemann/qtls-go1-16 => chainmaker.org/third_party/qtls-go1-16 v1.1.0
	github.com/marten-seemann/qtls-go1-17 => chainmaker.org/third_party/qtls-go1-17 v1.1.0
	github.com/marten-seemann/qtls-go1-18 => chainmaker.org/third_party/qtls-go1-18 v1.1.0
	github.com/marten-seemann/qtls-go1-19 => chainmaker.org/third_party/qtls-go1-19 v1.0.0
	github.com/syndtr/goleveldb => chainmaker.org/third_party/goleveldb v1.1.0
	github.com/tikv/client-go => chainmaker.org/third_party/tikv-client-go v1.0.0
// google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
