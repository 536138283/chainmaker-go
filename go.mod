module chainmaker.org/chainmaker-go

go 1.15

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.2.1-0.20220321090753-4b8da8e9a502
	chainmaker.org/chainmaker/common/v2 v2.2.1-0.20220321141225-ddd44f7eae82
	chainmaker.org/chainmaker/consensus-dpos/v2 v2.0.0-20220214082315-669156ba05b1
	chainmaker.org/chainmaker/consensus-maxbft/v2 v2.0.0-20220328071036-401a7d7f53eb
	chainmaker.org/chainmaker/consensus-raft/v2 v2.0.0-20220316072055-9c86c51d91fe
	chainmaker.org/chainmaker/consensus-solo/v2 v2.0.0-20220214113925-c976b365d11e
	chainmaker.org/chainmaker/consensus-tbft/v2 v2.0.0-20220311064722-5d9de0d56f1e
	chainmaker.org/chainmaker/consensus-utils/v2 v2.2.1-0.20220318073856-51a3d09172d5
	chainmaker.org/chainmaker/localconf/v2 v2.1.1-0.20220321090903-1507deaf95a7
	chainmaker.org/chainmaker/logger/v2 v2.2.0
	chainmaker.org/chainmaker/net-common v1.1.1-0.20220317094854-1f309816cacf
	chainmaker.org/chainmaker/net-libp2p v1.1.1-0.20220318022330-f23874b250e9
	chainmaker.org/chainmaker/net-liquid v1.0.2-0.20220318022337-c9e131793e19
	chainmaker.org/chainmaker/pb-go/v2 v2.2.1-0.20220321090254-034bd4bb5d37
	chainmaker.org/chainmaker/protocol/v2 v2.2.1-0.20220322121112-a46927c36654
	chainmaker.org/chainmaker/sdk-go/v2 v2.0.1-0.20220318050403-1f1d893eb879
	chainmaker.org/chainmaker/store/v2 v2.1.1-0.20220315021427-b6d7d4084f2a
	chainmaker.org/chainmaker/txpool-batch/v2 v2.1.1-0.20220317082215-d725aee9972b
	chainmaker.org/chainmaker/txpool-normal/v2 v2.0.0-20220317065147-c95df5eee311
	chainmaker.org/chainmaker/txpool-single/v2 v2.1.1-0.20220317065135-10a1bd0f5667
	chainmaker.org/chainmaker/utils/v2 v2.2.0
	chainmaker.org/chainmaker/vm-docker-go/v2 v2.1.1-0.20220315062103-65871bc67efa
	chainmaker.org/chainmaker/vm-evm/v2 v2.1.1-0.20220210034424-6444fc8f91bd
	chainmaker.org/chainmaker/vm-gasm/v2 v2.1.1-0.20220310130906-fc7031ec25c7
	chainmaker.org/chainmaker/vm-native/v2 v2.1.2-0.20220322052827-07e75d9b3f16
	chainmaker.org/chainmaker/vm-wasmer/v2 v2.1.1-0.20220310131354-75dde8ecad7c
	chainmaker.org/chainmaker/vm-wxvm/v2 v2.1.1-0.20220210034407-cb55533fd090
	chainmaker.org/chainmaker/vm/v2 v2.1.2-0.20220322053256-5b94babc8cea
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
	github.com/panjf2000/ants/v2 v2.4.8
	github.com/prometheus/client_golang v1.11.0
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/tidwall/pretty v1.2.0
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
