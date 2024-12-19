module chainmaker.org/chainmaker-go

go 1.16

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.3.5-0.20241213075657-c7f01d72fd96
	chainmaker.org/chainmaker/common/v2 v2.3.7-0.20241213064647-fa975ac686cb
	chainmaker.org/chainmaker/consensus-dpos/v2 v2.3.6-0.20240924071500-a03ab82d2671
	chainmaker.org/chainmaker/consensus-raft/v2 v2.3.6-0.20240924072002-668273cc596d
	chainmaker.org/chainmaker/consensus-solo/v2 v2.3.6-0.20240924072158-663b8cb446c9
	chainmaker.org/chainmaker/consensus-utils/v2 v2.3.6-0.20240924071242-12a9f842357f
	chainmaker.org/chainmaker/localconf/v2 v2.3.6-0.20241126015413-78af079db796
	chainmaker.org/chainmaker/logger/v2 v2.3.5-0.20241213073308-5477ab2482dc
	chainmaker.org/chainmaker/net-common v1.2.6-0.20241014030838-c56b32eb1a36
	chainmaker.org/chainmaker/net-libp2p v1.2.7-0.20241014080127-3b97a24253a9
	chainmaker.org/chainmaker/net-liquid v1.1.3
	chainmaker.org/chainmaker/pb-go/v2 v2.3.7-0.20241213070034-7553f600ceef
	chainmaker.org/chainmaker/protocol/v2 v2.3.8-0.20241213071011-7f965411db4e
	chainmaker.org/chainmaker/sdk-go/v2 v2.3.7-0.20241204023454-4ee5b4615669
	chainmaker.org/chainmaker/store/v2 v2.3.7-0.20241017081801-7ffa81831e84
	chainmaker.org/chainmaker/utils/v2 v2.3.7-0.20241213073938-00acb6333cd5
	chainmaker.org/chainmaker/vm-docker-go/v2 v2.3.7-0.20240924070303-058bed13cc63
	chainmaker.org/chainmaker/vm-engine/v2 v2.3.8-0.20241219085246-c81ced637f6e
	chainmaker.org/chainmaker/vm-evm/v2 v2.3.7-0.20241025091146-aeeef899298c
	chainmaker.org/chainmaker/vm-gasm/v2 v2.3.7-0.20240924062528-6f24f64a6742
	chainmaker.org/chainmaker/vm-native/v2 v2.3.7-0.20240924032826-32d500e3eeb8
	chainmaker.org/chainmaker/vm-wasmer/v2 v2.3.7-0.20240924062152-404d6c15e68a
	chainmaker.org/chainmaker/vm-wxvm/v2 v2.3.7-0.20240924062724-9b9401d612fb
	chainmaker.org/chainmaker/vm/v2 v2.3.7-0.20241203054730-7213c0cd222b
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
	chainmaker.org/chainmaker/consensus-maxbft/v2 v2.3.6-0.20240924071711-78dda56ee275
	chainmaker.org/chainmaker/consensus-tbft/v2 v2.3.6-0.20240924072420-436ad82fb309
	chainmaker.org/chainmaker/txpool-batch/v2 v2.3.5-0.20240924031757-153aee2bd699
	chainmaker.org/chainmaker/txpool-normal/v2 v2.3.5-0.20240924031528-14fc6d9c8524
	chainmaker.org/chainmaker/txpool-single/v2 v2.3.5-0.20240924031653-b69086d742fb
	github.com/go-echarts/go-echarts/v2 v2.2.4
	github.com/gosuri/uilive v0.0.4 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/huin/goupnp v1.0.1-0.20210310174557-0ca763054c88 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-tty v0.0.0-20180907095812-13ff1204f104 // indirect
	github.com/pkg/errors v0.9.1
	github.com/pkg/term v0.0.0-20180730021639-bffc007b7fd5 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/protobuf v1.27.1
)

replace (
	chainmaker.org/chainmaker/protocol/v2 v2.3.7-0.20240924025857-9a2a74f56eac => chainmaker.org/chainmaker/protocol/v2 v2.3.8-0.20241213071011-7f965411db4e
	chainmaker.org/chainmaker/store/v2 v2.3.7-0.20240924032400-263a9e5cfe49 => chainmaker.org/chainmaker/store/v2 v2.3.7-0.20241017081801-7ffa81831e84
	chainmaker.org/chainmaker/vm/v2 v2.3.7-0.20240924033049-66e6cb45263b => chainmaker.org/chainmaker/vm/v2 v2.3.7-0.20241203054730-7213c0cd222b
	github.com/RedisBloom/redisbloom-go => chainmaker.org/third_party/redisbloom-go v1.0.0
	github.com/dgraph-io/badger/v3 => chainmaker.org/third_party/badger/v3 v3.0.0
	github.com/libp2p/go-conn-security-multistream v0.2.0 => chainmaker.org/third_party/go-conn-security-multistream v1.0.5
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v1.1.0
	github.com/linvon/cuckoo-filter => chainmaker.org/third_party/cuckoo-filter v1.0.0
	github.com/lucas-clemente/quic-go v0.26.0 => chainmaker.org/third_party/quic-go v1.2.2
	github.com/marten-seemann/qtls-go1-16 => chainmaker.org/third_party/qtls-go1-16 v1.1.0
	github.com/marten-seemann/qtls-go1-17 => chainmaker.org/third_party/qtls-go1-17 v1.1.0
	github.com/marten-seemann/qtls-go1-18 => chainmaker.org/third_party/qtls-go1-18 v1.1.0
	github.com/marten-seemann/qtls-go1-19 => chainmaker.org/third_party/qtls-go1-19 v1.0.0
	github.com/syndtr/goleveldb => chainmaker.org/third_party/goleveldb v1.1.0
	github.com/tikv/client-go => chainmaker.org/third_party/tikv-client-go v1.0.0
)
