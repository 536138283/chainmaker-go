gen:
	cd pb/proto && protoc -I=. --go_out=:../protogo --go_opt=paths=source_relative outside/*.proto
	cd pb/proto && protoc -I=. --go_out=plugins=grpc:../protogo --go_opt=paths=source_relative api/docker_rpc.proto

