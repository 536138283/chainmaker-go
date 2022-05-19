module chainmaker.org/chainmaker-go

go 1.16

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.2.1-0.20220415025833-8ffe39b33773
	chainmaker.org/chainmaker/common/v2 v2.2.2-0.20220518091326-42bdf196d002
	chainmaker.org/chainmaker/consensus-dpos/v2 v2.0.0-20220420070758-a25aa71c7410
	chainmaker.org/chainmaker/consensus-maxbft/v2 v2.0.0-20220426033108-4aafa46a6f03
	chainmaker.org/chainmaker/consensus-raft/v2 v2.0.0-20220316072055-9c86c51d91fe
	chainmaker.org/chainmaker/consensus-solo/v2 v2.0.0-20220420072039-bc207cc67903
	chainmaker.org/chainmaker/consensus-tbft/v2 v2.2.1-0.20220427090416-d2288b48f6ac
	chainmaker.org/chainmaker/consensus-utils/v2 v2.2.1-0.20220422083335-d2aeacafcfe5
	chainmaker.org/chainmaker/localconf/v2 v2.2.2-0.20220517143546-a82c5a2a4b68
	chainmaker.org/chainmaker/logger/v2 v2.2.2-0.20220517143358-e208fb1bab2d
	chainmaker.org/chainmaker/net-common v1.1.2-0.20220426061105-482b8e05df81
	chainmaker.org/chainmaker/net-libp2p v1.1.2-0.20220426061305-7eb782dcf5f8
	chainmaker.org/chainmaker/net-liquid v1.0.3-0.20220426061931-9011730f7fba
	chainmaker.org/chainmaker/pb-go/v2 v2.2.2-0.20220519000546-25afb023f1b8
	chainmaker.org/chainmaker/protocol/v2 v2.2.3-0.20220519000705-fb966f5ffa52
	chainmaker.org/chainmaker/sdk-go/v2 v2.2.2-0.20220518091801-8d37c755195b
	chainmaker.org/chainmaker/store/v2 v2.2.1-0.20220506093443-87e13815d402
	chainmaker.org/chainmaker/txpool-batch/v2 v2.1.1-0.20220520013831-1a5df2c99a7b
	chainmaker.org/chainmaker/txpool-normal/v2 v2.0.0-20220520013958-71a74d051494
	chainmaker.org/chainmaker/txpool-single/v2 v2.2.3-0.20220520014047-25aa35802ba6
	chainmaker.org/chainmaker/utils/v2 v2.2.3-0.20220517140042-94d5a0c83290
	chainmaker.org/chainmaker/vm-docker-go/v2 v2.1.1-0.20220510070036-9816a868987a
	chainmaker.org/chainmaker/vm-evm/v2 v2.1.1-0.20220518091952-cc31d46894b1
	chainmaker.org/chainmaker/vm-gasm/v2 v2.1.1-0.20220310130906-fc7031ec25c7
	chainmaker.org/chainmaker/vm-native/v2 v2.1.2-0.20220418081245-3081a394cbe5
	chainmaker.org/chainmaker/vm-wasmer/v2 v2.1.1-0.20220224064011-5a7caccf53ed
	chainmaker.org/chainmaker/vm-wxvm/v2 v2.1.1-0.20211223061926-78b8d34d3aa3
	chainmaker.org/chainmaker/vm/v2 v2.1.2-0.20220510065753-a185c1a9224b
	code.cloudfoundry.org/bytefmt v0.0.0-20211005130812-5bb3c17173e5
	github.com/Rican7/retry v0.1.0
	github.com/Workiva/go-datastructures v1.0.53
	github.com/c-bata/go-prompt v0.2.2
	github.com/common-nighthawk/go-figure v0.0.0-20210622060536-734e95fb86be
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
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
	google.golang.org/grpc v1.41.0
	gorm.io/driver/mysql v1.2.0
	gorm.io/gorm v1.22.3
)

require (
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/gosuri/uilive v0.0.4 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/huin/goupnp v1.0.1-0.20210310174557-0ca763054c88 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-tty v0.0.0-20180907095812-13ff1204f104 // indirect
	github.com/pkg/term v0.0.0-20180730021639-bffc007b7fd5 // indirect
	github.com/tjfoc/gmtls v1.2.1 // indirect
	go.opencensus.io v0.23.0 // indirect
	google.golang.org/genproto v0.0.0-20210828152312-66f60bf46e71 // indirect
)

replace (
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
	github.com/linvon/cuckoo-filter => github.com/GuoxinL/cuckoo-filter v0.4.1
	github.com/spf13/afero => github.com/spf13/afero v1.5.1 //for go1.15 build
	github.com/spf13/viper => github.com/spf13/viper v1.7.1 //for go1.15 build
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
