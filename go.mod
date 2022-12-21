module chainmaker.org/chainmaker-go

go 1.16

require (
	chainmaker.org/chainmaker/net-common v1.2.2-0.20221215160619-96074a3a1951
	chainmaker.org/chainmaker/net-libp2p v1.2.2-0.20221215163047-b6984c944e7c
	chainmaker.org/chainmaker/net-liquid v1.1.1-0.20221215161714-c11970d0394f
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
	github.com/stretchr/testify v1.8.1
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
	chainmaker.org/chainmaker/chainconf/v3 v3.0.0-20221216071835-0e68877c2cf6
	chainmaker.org/chainmaker/common/v3 v3.0.0-20221219095751-8403621d5d51
	chainmaker.org/chainmaker/consensus-abft/v3 v3.0.0-20221215170435-69b73b81a806
	chainmaker.org/chainmaker/consensus-dpos/v3 v3.0.0-20221215165135-ef9b875e02ee
	chainmaker.org/chainmaker/consensus-maxbft/v3 v3.0.0-20221215164818-7d218f843768
	chainmaker.org/chainmaker/consensus-raft/v3 v3.0.0-20221221033359-b10228f150b9
	chainmaker.org/chainmaker/consensus-solo/v3 v3.0.0-20221215165852-2da7a814209b
	chainmaker.org/chainmaker/consensus-tbft/v3 v3.0.0-20221221072010-adf289dad645
	chainmaker.org/chainmaker/consensus-utils/v3 v3.0.0-20221215163218-859d20eb13ba
	chainmaker.org/chainmaker/localconf/v3 v3.0.0-20221216145445-586baf4ff3c7
	chainmaker.org/chainmaker/logger/v3 v3.0.0-20221215144630-dc51ec90e4d2
	chainmaker.org/chainmaker/pb-go/v3 v3.0.0-20221221064008-f515d53ac20d
	chainmaker.org/chainmaker/protocol/v3 v3.0.0-20221216071513-a00aa38168f2
	chainmaker.org/chainmaker/sdk-go/v3 v3.0.0-20221220072953-6589f4a83fd8
	chainmaker.org/chainmaker/store-huge/v3 v3.0.0-20221216090235-a261ffcc5ac7
	chainmaker.org/chainmaker/store/v3 v3.0.0-20221216074351-8fd0ca065af9
	chainmaker.org/chainmaker/txpool-batch/v3 v3.0.0-20221216084034-f97990921694
	chainmaker.org/chainmaker/txpool-normal/v3 v3.0.0-20221216083954-eb9907237f6b
	chainmaker.org/chainmaker/txpool-single/v3 v3.0.0-20221216083918-679d4048e5db
	chainmaker.org/chainmaker/utils/v3 v3.0.0-20221216142307-223160f75f87
	chainmaker.org/chainmaker/vm-docker-go/v3 v3.0.0-20221215155255-a82afbd98e70
	chainmaker.org/chainmaker/vm-engine/v3 v3.0.0-20221219025124-19d5268422c9
	chainmaker.org/chainmaker/vm-evm/v3 v3.0.0-20221216070931-9f418041a3d2
	chainmaker.org/chainmaker/vm-gasm/v3 v3.0.0-20221215154235-fc87247c7d80
	chainmaker.org/chainmaker/vm-native/v3 v3.0.0-20221221092600-c51f4f126afe
	chainmaker.org/chainmaker/vm-wasmer/v3 v3.0.0-20221215154918-3b910a1efecb
	chainmaker.org/chainmaker/vm-wxvm/v3 v3.0.0-20221215151317-d5335ed0efce
	chainmaker.org/chainmaker/vm/v3 v3.0.0-20221216144325-84451089a89f
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
