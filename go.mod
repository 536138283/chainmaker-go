module chainmaker.org/chainmaker-go

go 1.16

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.2.3-0.20220607072052-653ff1f72ed5
	chainmaker.org/chainmaker/common/v2 v2.2.2-0.20220802092829-4f5476e5291b
	chainmaker.org/chainmaker/consensus-dpos/v2 v2.0.0-20220627035228-9da92176f1cf
	chainmaker.org/chainmaker/consensus-maxbft/v2 v2.2.1-0.20220808065629-ec65eb5fad38
	chainmaker.org/chainmaker/consensus-raft/v2 v2.0.0-20220725121227-27a0a9458270
	chainmaker.org/chainmaker/consensus-solo/v2 v2.2.1-0.20220627031024-6cfa4d15e05e
	chainmaker.org/chainmaker/consensus-tbft/v2 v2.2.2-0.20220804063953-03b4311f66fa
	chainmaker.org/chainmaker/consensus-utils/v2 v2.2.2-0.20220727082533-2a5b6fa4e4bc
	chainmaker.org/chainmaker/localconf/v2 v2.2.2-0.20220722082432-17d9a1daf103
	chainmaker.org/chainmaker/logger/v2 v2.2.2-0.20220613040127-5b976891c91e
	chainmaker.org/chainmaker/net-common v1.1.2-0.20220610083519-e6727dc4f585
	chainmaker.org/chainmaker/net-libp2p v1.1.3-0.20220708084550-353d2f219a51
	chainmaker.org/chainmaker/net-liquid v1.0.3-0.20220804124109-8652435136bc
	chainmaker.org/chainmaker/pb-go/v2 v2.2.2-0.20220803084740-fd44bb75b5df
	chainmaker.org/chainmaker/protocol/v2 v2.2.3-0.20220804084551-a2f2cfe55621
	chainmaker.org/chainmaker/sdk-go/v2 v2.2.2-0.20220805083923-1f7edf67b46e
	chainmaker.org/chainmaker/store/v2 v2.2.2-0.20220714081904-fe9f04d19e8f
	chainmaker.org/chainmaker/txpool-batch/v2 v2.2.3-0.20220804091209-d91a8c63a802
	chainmaker.org/chainmaker/txpool-normal/v2 v2.0.0-20220804091131-a853cc122e72
	chainmaker.org/chainmaker/txpool-single/v2 v2.2.3-0.20220804090400-91ae3af54683
	chainmaker.org/chainmaker/utils/v2 v2.2.3-0.20220809070446-5ded62818df3
	chainmaker.org/chainmaker/vm-docker-go/v2 v2.2.2-0.20220808033023-001812a3ecd3
	chainmaker.org/chainmaker/vm-engine/v2 v2.2.3-0.20220809124032-4d8b22ff1cf5
	chainmaker.org/chainmaker/vm-evm/v2 v2.2.2-0.20220802070220-01da1cf03d9c
	chainmaker.org/chainmaker/vm-gasm/v2 v2.2.2-0.20220802030518-73309718e505
	chainmaker.org/chainmaker/vm-native/v2 v2.2.3-0.20220810064406-313d42c8168d
	chainmaker.org/chainmaker/vm-wasmer/v2 v2.2.2-0.20220802070258-414e787b9c8d
	chainmaker.org/chainmaker/vm-wxvm/v2 v2.2.1-0.20220801023609-a749aec413ec
	chainmaker.org/chainmaker/vm/v2 v2.2.3-0.20220808124036-0ef39f80c37d
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
	github.com/go-echarts/go-echarts/v2 v2.2.4
	github.com/gosuri/uilive v0.0.4 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/huin/goupnp v1.0.1-0.20210310174557-0ca763054c88 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-tty v0.0.0-20180907095812-13ff1204f104 // indirect
	github.com/pkg/errors v0.9.1
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
