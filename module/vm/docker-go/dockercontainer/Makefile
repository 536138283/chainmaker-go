gen:
	cd pb/proto && protoc -I=. --gogofaster_out=plugins=grpc:../protogo --gogofaster_opt=paths=source_relative cdm_message.proto

docker-clear:
	docker stop container1
	docker rm container1
	docker image rm image1
	docker image prune -f