module chainmaker.org/chainmaker-go

go 1.15

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.2.2
	chainmaker.org/chainmaker/common/v2 v2.2.2-0.20220614062456-558cc969cd2a
	chainmaker.org/chainmaker/consensus-dpos/v2 v2.2.0
	chainmaker.org/chainmaker/consensus-maxbft/v2 v2.2.0
	chainmaker.org/chainmaker/consensus-raft/v2 v2.2.0
	chainmaker.org/chainmaker/consensus-solo/v2 v2.2.0
	chainmaker.org/chainmaker/consensus-tbft/v2 v2.2.2-0.20220612055300-03bf03ed79b4
	chainmaker.org/chainmaker/consensus-utils/v2 v2.2.1
	chainmaker.org/chainmaker/localconf/v2 v2.2.1
	chainmaker.org/chainmaker/logger/v2 v2.2.1
	chainmaker.org/chainmaker/net-common v1.1.1
	chainmaker.org/chainmaker/net-libp2p v1.1.1
	chainmaker.org/chainmaker/net-liquid v1.0.2
	chainmaker.org/chainmaker/pb-go/v2 v2.2.2-0.20220905084343-1e152813dca0
	chainmaker.org/chainmaker/protocol/v2 v2.2.3-0.20220728062301-ef11b347c6be
	chainmaker.org/chainmaker/sdk-go/v2 v2.2.1-0.20220620043312-0432882376cf
	chainmaker.org/chainmaker/store/v2 v2.2.2-0.20220617151628-df429599d0c0
	chainmaker.org/chainmaker/txpool-batch/v2 v2.2.2-0.20220505075429-1188accd427f
	chainmaker.org/chainmaker/txpool-single/v2 v2.2.3-0.20220616100446-bbfd624e39ba
	chainmaker.org/chainmaker/utils/v2 v2.2.3-0.20220614081000-17efb6e04bcb
	chainmaker.org/chainmaker/vm-docker-go/v2 v2.2.2-0.20220808115945-340075b2e41f
	chainmaker.org/chainmaker/vm-evm/v2 v2.2.2-0.20220720083404-b614732c7ad5
	chainmaker.org/chainmaker/vm-gasm/v2 v2.2.2-0.20220720083211-4b5921eaa447
	chainmaker.org/chainmaker/vm-native/v2 v2.2.2-0.20220905084625-4239ddc4c9dd
	chainmaker.org/chainmaker/vm-wasmer/v2 v2.2.2-0.20220720094041-a868c2648408
	chainmaker.org/chainmaker/vm-wxvm/v2 v2.2.2-0.20220720083320-8413c9bb17f0
	chainmaker.org/chainmaker/vm/v2 v2.2.3-0.20220720083050-c03f22d92ed9
	code.cloudfoundry.org/bytefmt v0.0.0-20211005130812-5bb3c17173e5
	github.com/Rican7/retry v0.1.0
	github.com/Workiva/go-datastructures v1.0.53
	github.com/c-bata/go-prompt v0.2.2
	github.com/common-nighthawk/go-figure v0.0.0-20210622060536-734e95fb86be
	github.com/ethereum/go-ethereum v1.10.4
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
	github.com/gosuri/uilive v0.0.4 // indirect
	github.com/gosuri/uiprogress v0.0.1
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/hokaccha/go-prettyjson v0.0.0-20201222001619-a42f9ac2ec8e
	github.com/holiman/uint256 v1.2.0
	github.com/hpcloud/tail v1.0.0
	github.com/kr/pretty v0.2.1 // indirect
	github.com/linvon/cuckoo-filter v0.4.0
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/mapstructure v1.4.2
	github.com/mr-tron/base58 v1.2.0
	github.com/panjf2000/ants/v2 v2.4.7
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/tidwall/pretty v1.2.0
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.7.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/time v0.0.0-20210608053304-ed9ce3a009e4
	google.golang.org/genproto v0.0.0-20210828152312-66f60bf46e71 // indirect
	google.golang.org/grpc v1.41.0
	gorm.io/driver/mysql v1.2.0
	gorm.io/gorm v1.22.3

)

replace (
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v1.0.0
	github.com/linvon/cuckoo-filter => github.com/GuoxinL/cuckoo-filter v0.4.1
	github.com/spf13/afero => github.com/spf13/afero v1.5.1 //for go1.15 build
	github.com/spf13/viper => github.com/spf13/viper v1.7.1 //for go1.15 build
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
