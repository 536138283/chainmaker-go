module chainmaker.org/chainmaker-go/docker-go/dockercontainer

go 1.15

require (
	contract-sdk-test1/pb_sdk v0.0.0-00010101000000-000000000000
	github.com/golang/protobuf v1.5.0
	go.uber.org/zap v1.18.1
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.26.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

replace contract-sdk-test1/pb_sdk => ./pb_sdk
