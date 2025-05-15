module chainmaker.org/chainmaker-go

go 1.16

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.3.5-0.20250515091628-cdefec16c2ec
	chainmaker.org/chainmaker/common/v2 v2.3.8-0.20250416094737-df7d34f2ede0
	chainmaker.org/chainmaker/consensus-dpos/v2 v2.3.6-0.20250311094610-74896255665a
	chainmaker.org/chainmaker/consensus-raft/v2 v2.3.6-0.20250311095343-67773d8b289f
	chainmaker.org/chainmaker/consensus-solo/v2 v2.3.6-0.20250311095625-334be7cafa6f
	chainmaker.org/chainmaker/consensus-utils/v2 v2.3.6-0.20250311094111-c601485b5411
	chainmaker.org/chainmaker/localconf/v2 v2.3.6-0.20250508094008-1e0c553ac2a6
	chainmaker.org/chainmaker/logger/v2 v2.3.5-0.20250311075614-ff90ca99cbc1
	chainmaker.org/chainmaker/net-common v1.2.7-0.20250423065129-61d9092537b1
	chainmaker.org/chainmaker/net-libp2p v1.2.9-0.20250425105106-b418145739b1
	chainmaker.org/chainmaker/net-liquid v1.1.3
	chainmaker.org/chainmaker/pb-go/v2 v2.3.7-0.20250515075633-a0b2ad4099ed
	chainmaker.org/chainmaker/protocol/v2 v2.3.9-0.20250407103320-79c9c0f76846
	chainmaker.org/chainmaker/sdk-go/v2 v2.3.8-0.20250509092703-b13228993c0f
	chainmaker.org/chainmaker/store/v2 v2.3.8-0.20250425083527-9765973e8520
	chainmaker.org/chainmaker/utils/v2 v2.3.7-0.20250515084641-2ece4f3000db
	chainmaker.org/chainmaker/vm-docker-go/v2 v2.3.7-0.20250311090804-13f0c6990fc3
	chainmaker.org/chainmaker/vm-engine/v2 v2.3.8-0.20250427070610-0b3a1267dbbd
	chainmaker.org/chainmaker/vm-evm/v2 v2.3.7-0.20250314090743-1e67d63f6959
	chainmaker.org/chainmaker/vm-gasm/v2 v2.3.7-0.20250421030112-c78da87d83a3
	chainmaker.org/chainmaker/vm-native/v2 v2.3.7-0.20250430091550-7d66a4024edb
	chainmaker.org/chainmaker/vm-wasmer/v2 v2.3.7-0.20250314030658-1a38448392a1
	chainmaker.org/chainmaker/vm-wxvm/v2 v2.3.7-0.20250311090156-dd46d6a91839
	chainmaker.org/chainmaker/vm/v2 v2.3.7-0.20250313033547-c4c814d9a06e
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
	github.com/stretchr/testify v1.8.2
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/tidwall/pretty v1.2.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20201229170055-e5319fda7802
	go.uber.org/atomic v1.7.0
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	golang.org/x/time v0.0.0-20210608053304-ed9ce3a009e4
	google.golang.org/grpc v1.47.0
)

require (
	chainmaker.org/chainmaker/consensus-maxbft/v2 v2.3.6-0.20250311095505-0688f55394a0
	chainmaker.org/chainmaker/consensus-tbft/v2 v2.3.6-0.20250513032830-124e2c2f6981
	chainmaker.org/chainmaker/sync/v2 v2.0.0-20250508065913-6488d500a8f7
	chainmaker.org/chainmaker/txpool-batch/v2 v2.3.5-0.20250407020642-f4ec65daa72f
	chainmaker.org/chainmaker/txpool-normal/v2 v2.3.5-0.20250407020622-d14fe5fd1889
	chainmaker.org/chainmaker/txpool-single/v2 v2.3.5-0.20250408023157-f6d52d1f81e3
	github.com/go-echarts/go-echarts/v2 v2.2.4
	github.com/google/flatbuffers v2.0.0+incompatible // indirect
	github.com/gosuri/uilive v0.0.4 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/huin/goupnp v1.0.1-0.20210310174557-0ca763054c88 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-tty v0.0.0-20180907095812-13ff1204f104 // indirect
	github.com/pkg/errors v0.9.1
	github.com/pkg/term v0.0.0-20180730021639-bffc007b7fd5 // indirect
	github.com/robfig/cron/v3 v3.0.1
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/protobuf v1.27.1
)

replace (
	github.com/RedisBloom/redisbloom-go => chainmaker.org/third_party/redisbloom-go v1.0.0
	github.com/dgraph-io/badger/v3 => chainmaker.org/third_party/badger/v3 v3.0.0
	github.com/libp2p/go-conn-security-multistream v0.2.0 => chainmaker.org/third_party/go-conn-security-multistream v1.0.5
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v1.1.1
	github.com/linvon/cuckoo-filter => chainmaker.org/third_party/cuckoo-filter v1.0.0
	github.com/lucas-clemente/quic-go v0.26.0 => chainmaker.org/third_party/quic-go v1.2.2
	github.com/marten-seemann/qtls-go1-16 => chainmaker.org/third_party/qtls-go1-16 v1.1.0
	github.com/marten-seemann/qtls-go1-17 => chainmaker.org/third_party/qtls-go1-17 v1.1.0
	github.com/marten-seemann/qtls-go1-18 => chainmaker.org/third_party/qtls-go1-18 v1.1.0
	github.com/marten-seemann/qtls-go1-19 => chainmaker.org/third_party/qtls-go1-19 v1.0.0
	github.com/syndtr/goleveldb => chainmaker.org/third_party/goleveldb v1.1.0
	github.com/tikv/client-go => chainmaker.org/third_party/tikv-client-go v1.0.0
)
