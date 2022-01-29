module chainmaker.org/chainmaker-go

go 1.15

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.1.2-0.20220128025317-57dfc5f3d371
	chainmaker.org/chainmaker/common/v2 v2.1.2-0.20220127064703-4da19a869e9d
	chainmaker.org/chainmaker/consensus-dpos/v2 v2.0.0-20220129032058-2fb694281387
	chainmaker.org/chainmaker/consensus-maxbft/v2 v2.0.0-20220129082242-0ebca81d186b
	chainmaker.org/chainmaker/consensus-raft/v2 v2.0.0-20211231065915-33ec8814372a
	chainmaker.org/chainmaker/consensus-solo/v2 v2.0.0-20220129082508-3ed277948aeb
	chainmaker.org/chainmaker/consensus-tbft/v2 v2.0.0-20220129082948-e2988c87e5e0
	chainmaker.org/chainmaker/consensus-utils/v2 v2.0.0-20220129030546-2a2892b25e58
	chainmaker.org/chainmaker/localconf/v2 v2.1.1-0.20220129103344-6ec43cc786d6
	chainmaker.org/chainmaker/logger/v2 v2.1.1-0.20220128022235-c984177a37cc
	chainmaker.org/chainmaker/net-common v1.0.2-0.20220120084355-9be05b200365
	chainmaker.org/chainmaker/net-libp2p v1.0.2-0.20220125065325-34de4d8e90b6
	chainmaker.org/chainmaker/net-liquid v1.0.2-0.20220125072102-53ec8e660e73
	chainmaker.org/chainmaker/pb-go/v2 v2.1.1-0.20220128100409-ce390cb69a32
	chainmaker.org/chainmaker/protocol/v2 v2.1.2-0.20220128021940-8fd70a8c2c75
	chainmaker.org/chainmaker/sdk-go/v2 v2.0.1-0.20220129084708-75eef4494978
	chainmaker.org/chainmaker/store/v2 v2.1.1-0.20220128070304-a958ce8c7661
	chainmaker.org/chainmaker/txpool-batch/v2 v2.1.1-0.20220128062951-d5758846d3bd
	chainmaker.org/chainmaker/txpool-single/v2 v2.1.1-0.20220128061857-855bf0e3e65b
	chainmaker.org/chainmaker/utils/v2 v2.1.1-0.20220128023017-5bf8279342f1
	chainmaker.org/chainmaker/vm-docker-go/v2 v2.1.1-0.20220129024947-88192498b431
	chainmaker.org/chainmaker/vm-evm/v2 v2.1.1-0.20220129021120-8f8d5490deba
	chainmaker.org/chainmaker/vm-gasm/v2 v2.1.1-0.20220128100259-9ef774d53506
	chainmaker.org/chainmaker/vm-native/v2 v2.1.2-0.20220128071131-d9aa2c021f61
	chainmaker.org/chainmaker/vm-wasmer/v2 v2.1.1-0.20220128101120-31ffdd74d4f1
	chainmaker.org/chainmaker/vm-wxvm/v2 v2.1.1-0.20220129020915-aae993ae1ac6
	chainmaker.org/chainmaker/vm/v2 v2.1.2-0.20220128094534-8b0f7f34ac08
	code.cloudfoundry.org/bytefmt v0.0.0-20211005130812-5bb3c17173e5
	github.com/Rican7/retry v0.1.0
	github.com/Workiva/go-datastructures v1.0.53
	github.com/c-bata/go-prompt v0.2.2
	github.com/common-nighthawk/go-figure v0.0.0-20210622060536-734e95fb86be
	github.com/ethereum/go-ethereum v1.10.4
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
	github.com/gosuri/uilive v0.0.4 // indirect
	github.com/gosuri/uiprogress v0.0.1
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hokaccha/go-prettyjson v0.0.0-20201222001619-a42f9ac2ec8e
	github.com/holiman/uint256 v1.2.0
	github.com/hpcloud/tail v1.0.0
	github.com/mitchellh/mapstructure v1.4.2
	github.com/mr-tron/base58 v1.2.0
	github.com/panjf2000/ants/v2 v2.4.7
	github.com/prometheus/client_golang v1.11.0
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/tidwall/pretty v1.0.2
	github.com/tmc/grpc-websocket-proxy v0.0.0-20201229170055-e5319fda7802
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.7.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
	golang.org/x/time v0.0.0-20210608053304-ed9ce3a009e4
	google.golang.org/genproto v0.0.0-20210828152312-66f60bf46e71 // indirect
	google.golang.org/grpc v1.41.0
	gorm.io/driver/mysql v1.2.0
	gorm.io/gorm v1.22.3
)

replace (
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
	github.com/spf13/afero => github.com/spf13/afero v1.5.1 //for go1.15 build
	github.com/spf13/viper => github.com/spf13/viper v1.7.1 //for go1.15 build
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
