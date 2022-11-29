module chainmaker.org/chainmaker-go

go 1.16

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.3.1-0.20221128034452-1d2cdc87e1ef
	chainmaker.org/chainmaker/common/v2 v2.3.1-0.20221128030901-8463111fc821
	chainmaker.org/chainmaker/consensus-dpos/v2 v2.3.1-0.20220928123352-fe127bc35022
	chainmaker.org/chainmaker/consensus-maxbft/v2 v2.3.1-0.20221129070434-5ce2fe0c21af
	chainmaker.org/chainmaker/consensus-raft/v2 v2.3.1-0.20221128072133-aa9a72e627de
	chainmaker.org/chainmaker/consensus-solo/v2 v2.3.1-0.20220922081441-580676b65f8a
	chainmaker.org/chainmaker/consensus-tbft/v2 v2.3.2-0.20221027085131-68df9a194ae4
	chainmaker.org/chainmaker/consensus-utils/v2 v2.3.2-0.20221129033539-e6ffb2710b7c
	chainmaker.org/chainmaker/localconf/v2 v2.3.1-0.20220930132913-96e38fc556a3
	chainmaker.org/chainmaker/logger/v2 v2.3.1-0.20220906085151-3fe59a8a3fe5
	chainmaker.org/chainmaker/net-common v1.2.1
	chainmaker.org/chainmaker/net-libp2p v1.2.2-0.20221110034948-500bb43f4bf2
	chainmaker.org/chainmaker/net-liquid v1.1.1-0.20221109062455-bd9d2851ee5a
	chainmaker.org/chainmaker/pb-go/v2 v2.3.2-0.20221128032314-f49253fc6fe3
	chainmaker.org/chainmaker/protocol/v2 v2.3.2-0.20221128031232-be0784b5f827
	chainmaker.org/chainmaker/sdk-go/v2 v2.3.2-0.20221128041045-d587b9d75354
	chainmaker.org/chainmaker/store/v2 v2.3.3-0.20221128034143-67d184b620f1
	chainmaker.org/chainmaker/txpool-batch/v2 v2.3.1-0.20221128064035-9ad38482dc2f
	chainmaker.org/chainmaker/txpool-normal/v2 v2.3.1-0.20221128062542-1062c48d1047
	chainmaker.org/chainmaker/txpool-single/v2 v2.3.1-0.20221128063021-dbd93633275d
	chainmaker.org/chainmaker/utils/v2 v2.3.2-0.20221102024459-3ee62f3bfb98
	chainmaker.org/chainmaker/vm-docker-go/v2 v2.3.2-0.20221128055924-2b0062a036b9
	chainmaker.org/chainmaker/vm-engine/v2 v2.3.2-0.20221128064014-1d291ebb2fe6
	chainmaker.org/chainmaker/vm-evm/v2 v2.3.2-0.20221128060606-f4faf7308595
	chainmaker.org/chainmaker/vm-gasm/v2 v2.3.2-0.20221128071457-42f3ae39d90a
	chainmaker.org/chainmaker/vm-native/v2 v2.3.2-0.20221128040348-1c586d46cb4e
	chainmaker.org/chainmaker/vm-wasmer/v2 v2.3.2-0.20221121061121-d88f996f1669
	chainmaker.org/chainmaker/vm-wxvm/v2 v2.3.2-0.20221128071546-ff96ef5d93be
	chainmaker.org/chainmaker/vm/v2 v2.3.2-0.20221128080135-cdae0bb7f86d
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
	gorm.io/driver/mysql v1.2.0
	gorm.io/gorm v1.22.3
)

require (
	chainmaker.org/chainmaker/consensus-abft/v2 v2.0.0-20221020064225-9ff9d0ec5a39
	chainmaker.org/chainmaker/lws v1.1.1-0.20220713075428-5bed3f300ef9 // indirect
	github.com/go-echarts/go-echarts/v2 v2.2.4
	github.com/gosuri/uilive v0.0.4 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/huin/goupnp v1.0.1-0.20210310174557-0ca763054c88 // indirect
	github.com/mattn/go-tty v0.0.0-20180907095812-13ff1204f104 // indirect
	github.com/pkg/term v0.0.0-20180730021639-bffc007b7fd5 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/protobuf v1.28.0
)

replace (
	github.com/RedisBloom/redisbloom-go => chainmaker.org/third_party/redisbloom-go v1.0.0
	github.com/dgraph-io/badger/v3 => chainmaker.org/third_party/badger/v3 v3.0.0
	github.com/libp2p/go-conn-security-multistream v0.2.0 => chainmaker.org/third_party/go-conn-security-multistream v1.0.0
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v1.0.0
	github.com/linvon/cuckoo-filter => chainmaker.org/third_party/cuckoo-filter v1.0.1-0.20220601084543-8591df469f8f
	github.com/lucas-clemente/quic-go => chainmaker.org/third_party/quic-go v1.0.0
	github.com/marten-seemann/qtls-go1-15 => chainmaker.org/third_party/qtls-go1-15 v1.0.0
	github.com/marten-seemann/qtls-go1-16 => chainmaker.org/third_party/qtls-go1-16 v1.0.0
	github.com/marten-seemann/qtls-go1-17 => chainmaker.org/third_party/qtls-go1-17 v1.0.0
	github.com/marten-seemann/qtls-go1-18 => chainmaker.org/third_party/qtls-go1-18 v1.0.0
	github.com/syndtr/goleveldb => chainmaker.org/third_party/goleveldb v1.1.0
	github.com/tikv/client-go => chainmaker.org/third_party/tikv-client-go v1.0.0
// google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
