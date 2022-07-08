module chainmaker.org/chainmaker-go

go 1.16

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.2.3-0.20220607072052-653ff1f72ed5
	chainmaker.org/chainmaker/common/v2 v2.2.2-0.20220628084133-47574edae904
	chainmaker.org/chainmaker/consensus-dpos/v2 v2.0.0-20220627035228-9da92176f1cf
	chainmaker.org/chainmaker/consensus-maxbft/v2 v2.2.1-0.20220708023611-e4fc3ad5412e
	chainmaker.org/chainmaker/consensus-raft/v2 v2.0.0-20220627035329-5bf132e435c7
	chainmaker.org/chainmaker/consensus-solo/v2 v2.2.1-0.20220627031024-6cfa4d15e05e
	chainmaker.org/chainmaker/consensus-tbft/v2 v2.2.2-0.20220705034358-da730eea7350
	chainmaker.org/chainmaker/consensus-utils/v2 v2.2.2-0.20220705094047-e31aec0e6142
	chainmaker.org/chainmaker/localconf/v2 v2.2.2-0.20220607115425-03689a750027
	chainmaker.org/chainmaker/logger/v2 v2.2.2-0.20220613040127-5b976891c91e
	chainmaker.org/chainmaker/net-common v1.1.2-0.20220610083519-e6727dc4f585
	chainmaker.org/chainmaker/net-libp2p v1.1.3-0.20220701093702-c6fd18e07c2e
	chainmaker.org/chainmaker/net-liquid v1.0.3-0.20220701093732-4980d6f9b9fd
	chainmaker.org/chainmaker/pb-go/v2 v2.2.2-0.20220629095211-4ae3dcfd9822
	chainmaker.org/chainmaker/protocol/v2 v2.2.3-0.20220628032755-cebcbb60be67
	chainmaker.org/chainmaker/sdk-go/v2 v2.2.2-0.20220630033703-4b5aa2b89287
	chainmaker.org/chainmaker/store/v2 v2.2.2-0.20220628083247-821a4a6ea69b
	chainmaker.org/chainmaker/txpool-batch/v2 v2.2.3-0.20220704121112-bd723fc4a2a3
	chainmaker.org/chainmaker/txpool-normal/v2 v2.0.0-20220704121129-5356d0b5b297
	chainmaker.org/chainmaker/txpool-single/v2 v2.2.3-0.20220704121147-9f3b57c2a16c
	chainmaker.org/chainmaker/utils/v2 v2.2.3-0.20220629102323-8030046a102a
	chainmaker.org/chainmaker/vm-docker-go/v2 v2.2.3-0.20220707024445-ce98a740ceed
	chainmaker.org/chainmaker/vm-evm/v2 v2.2.2-0.20220627074934-938e05f7646e
	chainmaker.org/chainmaker/vm-gasm/v2 v2.2.1
	chainmaker.org/chainmaker/vm-native/v2 v2.2.3-0.20220630091700-c9f6ccab949a
	chainmaker.org/chainmaker/vm-wasmer/v2 v2.2.2-0.20220629092006-9489741c0707
	chainmaker.org/chainmaker/vm-wxvm/v2 v2.2.1
	chainmaker.org/chainmaker/vm/v2 v2.2.3-0.20220630074844-e45e59cf56f3
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
	github.com/stretchr/testify v1.7.0
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
	chainmaker.org/chainmaker/raftwal/v2 v2.1.1-0.20211220112831-1ebb19d509ff // indirect
	github.com/gosuri/uilive v0.0.4 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/huin/goupnp v1.0.1-0.20210310174557-0ca763054c88 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-tty v0.0.0-20180907095812-13ff1204f104 // indirect
	github.com/pkg/term v0.0.0-20180730021639-bffc007b7fd5 // indirect
)

replace (
	github.com/RedisBloom/redisbloom-go => chainmaker.org/third_party/redisbloom-go v0.0.0-20220429030713-9efb559f09ad
	github.com/dgraph-io/badger/v3 => chainmaker.org/third_party/badger/v3 v3.2103.3-0.20220506101147-b3714597ecc4
	github.com/libp2p/go-conn-security-multistream v0.2.0 => chainmaker.org/third_party/go-conn-security-multistream v0.0.0-20220629104649-989834ad81c4
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v1.0.0
	github.com/linvon/cuckoo-filter => chainmaker.org/third_party/cuckoo-filter v0.0.0-20220601084543-8591df469f8f
	github.com/lucas-clemente/quic-go v0.26.0 => chainmaker.org/third_party/quic-go v1.0.0
	github.com/marten-seemann/qtls-go1-15 => chainmaker.org/third_party/qtls-go1-15 v1.0.0
	github.com/marten-seemann/qtls-go1-16 => chainmaker.org/third_party/qtls-go1-16 v1.0.0
	github.com/marten-seemann/qtls-go1-17 => chainmaker.org/third_party/qtls-go1-17 v1.0.0
	github.com/marten-seemann/qtls-go1-18 => chainmaker.org/third_party/qtls-go1-18 v1.0.0
	github.com/syndtr/goleveldb => chainmaker.org/third_party/goleveldb v1.0.1-0.20220428044327-8b6a49e187b4
	github.com/tikv/client-go => chainmaker.org/third_party/tikv-client-go v0.0.0-20220520083957-392cb085ba4a
// google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
