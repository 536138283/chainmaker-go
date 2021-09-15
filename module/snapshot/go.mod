module chainmaker.org/chainmaker-go/snapshot

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210915075633-90ea007220a9
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210915083256-3cba3b585dd4
	chainmaker.org/chainmaker/protocol/v2 v2.0.0
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

	chainmaker.org/chainmaker-go/utils => ../utils

)
