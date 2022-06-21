module chainmaker.org/chainmaker-go

go 1.16

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.2.3-0.20220601093135-49f1a82f5679
	chainmaker.org/chainmaker/common/v2 v2.2.2-0.20220621121723-440f87bf25bd
	chainmaker.org/chainmaker/consensus-dpos/v2 v2.0.0-20220420070758-a25aa71c7410
	chainmaker.org/chainmaker/consensus-raft/v2 v2.0.0-20220420114456-4bd051c6e220
	chainmaker.org/chainmaker/consensus-solo/v2 v2.0.0-20220420072039-bc207cc67903
	chainmaker.org/chainmaker/consensus-tbft/v2 v2.2.2-0.20220602085258-a0e3a94e67d5
	chainmaker.org/chainmaker/consensus-utils/v2 v2.2.2-0.20220610092515-91f118b0e1f1
	chainmaker.org/chainmaker/localconf/v2 v2.2.2-0.20220615040355-97ca5754a1eb
	chainmaker.org/chainmaker/logger/v2 v2.2.2-0.20220616100243-4fa06732e9a3
	chainmaker.org/chainmaker/net-common v1.1.2-0.20220610083519-e6727dc4f585
	chainmaker.org/chainmaker/net-libp2p v1.1.3-0.20220610084257-d720400089e4
	chainmaker.org/chainmaker/net-liquid v1.0.3-0.20220609031411-3e1c47ac7cfb
	chainmaker.org/chainmaker/pb-go/v2 v2.2.2-0.20220617081650-cc775e8ce4db
	chainmaker.org/chainmaker/protocol/v2 v2.2.3-0.20220621122141-d136370aa6d1
	chainmaker.org/chainmaker/sdk-go/v2 v2.2.2-0.20220616062219-1f80af699445
	chainmaker.org/chainmaker/store/v2 v2.2.2-0.20220527181405-be81673f609d
	chainmaker.org/chainmaker/txpool-batch/v2 v2.2.3-0.20220621131403-9487ec165329
	chainmaker.org/chainmaker/txpool-normal/v2 v2.0.0-20220530072011-9e7b5c90b3b5
	chainmaker.org/chainmaker/txpool-single/v2 v2.2.3-0.20220621132803-3cb4dbfc3e64
	chainmaker.org/chainmaker/utils/v2 v2.2.3-0.20220615103110-6849a8ada870
	chainmaker.org/chainmaker/vm-docker-go/v2 v2.2.3-0.20220601071506-dd01ec670925
	chainmaker.org/chainmaker/vm-evm/v2 v2.2.2-0.20220527114415-d0cc67f922c9
	chainmaker.org/chainmaker/vm-gasm/v2 v2.1.1-0.20220310130906-fc7031ec25c7
	chainmaker.org/chainmaker/vm-native/v2 v2.2.3-0.20220617091652-9b1d69633ae1
	chainmaker.org/chainmaker/vm-wasmer/v2 v2.2.2-0.20220511025424-a3cda7f42c9c
	chainmaker.org/chainmaker/vm-wxvm/v2 v2.2.2-0.20220601100322-0b18a7a72339
	chainmaker.org/chainmaker/vm/v2 v2.2.3-0.20220613020953-5896c3d78e45
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
	google.golang.org/grpc v1.43.0
	gorm.io/driver/mysql v1.2.0
	gorm.io/gorm v1.22.3
)

require (
	chainmaker.org/chainmaker/consensus-maxbft/v2 v2.0.0-20220621094854-96d966b704d5
	github.com/gosuri/uilive v0.0.4 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/huin/goupnp v1.0.1-0.20210310174557-0ca763054c88 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-tty v0.0.0-20180907095812-13ff1204f104 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/pkg/term v0.0.0-20180730021639-bffc007b7fd5 // indirect
)

replace (
	github.com/RedisBloom/redisbloom-go => chainmaker.org/third_party/redisbloom-go v0.0.0-20220429030713-9efb559f09ad
	github.com/dgraph-io/badger/v3 => chainmaker.org/third_party/badger/v3 v3.2103.3-0.20220507023220-88f2a60cb489
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
	github.com/linvon/cuckoo-filter => chainmaker.org/third_party/cuckoo-filter v0.0.0-20220601084543-8591df469f8f
	github.com/lucas-clemente/quic-go => chainmaker.org/third_party/quic-go v1.0.0
	github.com/marten-seemann/qtls-go1-15 => chainmaker.org/third_party/qtls-go1-15 v1.0.0
	github.com/marten-seemann/qtls-go1-16 => chainmaker.org/third_party/qtls-go1-16 v1.0.0
	github.com/marten-seemann/qtls-go1-17 => chainmaker.org/third_party/qtls-go1-17 v1.0.0
	github.com/marten-seemann/qtls-go1-18 => chainmaker.org/third_party/qtls-go1-18 v1.0.0
	github.com/syndtr/goleveldb => chainmaker.org/third_party/goleveldb v1.0.1-0.20220507023041-7cd1cad083a4
	github.com/tikv/client-go => chainmaker.org/third_party/tikv-client-go v0.0.0-20220520083957-392cb085ba4a
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
