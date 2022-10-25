module chainmaker.org/chainmaker-go

go 1.16

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.3.1-0.20220906150733-4aa890ffe987
	chainmaker.org/chainmaker/common/v2 v2.3.1-0.20221021080321-6afc98dd49a0
	chainmaker.org/chainmaker/consensus-dpos/v2 v2.3.1-0.20220919121316-150b1649ade2
	chainmaker.org/chainmaker/consensus-maxbft/v2 v2.3.1-0.20220922072457-7e01d4b5bfa9
	chainmaker.org/chainmaker/consensus-raft/v2 v2.3.1-0.20220919093419-1679188d5e4d
	chainmaker.org/chainmaker/consensus-solo/v2 v2.3.1-0.20220922064243-9ec81c55196d
	chainmaker.org/chainmaker/consensus-tbft/v2 v2.3.1-0.20221024081954-c5a16c03db88
	chainmaker.org/chainmaker/consensus-utils/v2 v2.3.1-0.20221017040951-f66e0ad6aba1
	chainmaker.org/chainmaker/localconf/v2 v2.3.1-0.20220930132913-96e38fc556a3
	chainmaker.org/chainmaker/logger/v2 v2.3.1-0.20220906085151-3fe59a8a3fe5
	chainmaker.org/chainmaker/net-common v1.2.0
	chainmaker.org/chainmaker/net-libp2p v1.2.1-0.20220906155423-3c724ff823aa
	chainmaker.org/chainmaker/net-liquid v1.1.1-0.20220906155617-b0a95c2cd5c0
	chainmaker.org/chainmaker/pb-go/v2 v2.3.2-0.20221021075258-7d21f3f42c01
	chainmaker.org/chainmaker/protocol/v2 v2.3.2-0.20221021075557-d411feaae9c2
	chainmaker.org/chainmaker/sdk-go/v2 v2.3.2-0.20221025071841-d30ac404881b
	chainmaker.org/chainmaker/store/v2 v2.3.3-0.20221021083013-adcd42809c60
	chainmaker.org/chainmaker/txpool-batch/v2 v2.3.1-0.20221014095540-f06072edd787
	chainmaker.org/chainmaker/txpool-normal/v2 v2.3.1-0.20221014095531-ad2f6d9b66e2
	chainmaker.org/chainmaker/txpool-single/v2 v2.3.1-0.20221014095522-be39fb70e3f3
	chainmaker.org/chainmaker/utils/v2 v2.3.2-0.20221021075957-8d5aa9784316
	chainmaker.org/chainmaker/vm-docker-go/v2 v2.3.0
	chainmaker.org/chainmaker/vm-engine/v2 v2.3.1-0.20221012075021-4d575e4ff66e
	chainmaker.org/chainmaker/vm-evm/v2 v2.3.1-0.20221014033406-ff250244338b
	chainmaker.org/chainmaker/vm-gasm/v2 v2.3.1-0.20220906161626-e1ccd9b65594
	chainmaker.org/chainmaker/vm-native/v2 v2.3.2-0.20221021080217-cfce5a333a9b
	chainmaker.org/chainmaker/vm-wasmer/v2 v2.3.0
	chainmaker.org/chainmaker/vm-wxvm/v2 v2.3.1-0.20220906162226-ae9199a14411
	chainmaker.org/chainmaker/vm/v2 v2.3.2-0.20221021083400-f78ae3a37a92
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
	github.com/linvon/cuckoo-filter => chainmaker.org/third_party/cuckoo-filter v1.0.0
	github.com/lucas-clemente/quic-go v0.26.0 => chainmaker.org/third_party/quic-go v1.0.0
	github.com/marten-seemann/qtls-go1-15 => chainmaker.org/third_party/qtls-go1-15 v1.0.0
	github.com/marten-seemann/qtls-go1-16 => chainmaker.org/third_party/qtls-go1-16 v1.0.0
	github.com/marten-seemann/qtls-go1-17 => chainmaker.org/third_party/qtls-go1-17 v1.0.0
	github.com/marten-seemann/qtls-go1-18 => chainmaker.org/third_party/qtls-go1-18 v1.0.0
	github.com/syndtr/goleveldb => chainmaker.org/third_party/goleveldb v1.1.0
	github.com/tikv/client-go => chainmaker.org/third_party/tikv-client-go v1.0.0
// google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
