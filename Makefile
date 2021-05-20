ifeq ($(OS),Windows_NT)
  PLATFORM="Windows"
else
  ifeq ($(shell uname),Darwin)
    PLATFORM="MacOS"
  else
    PLATFORM="Linux"
  endif
endif
DATETIME=$(shell date "+%Y%m%d%H%M%S")
VERSION=V1.0.0

chainmaker:
	@cd main && go build -o ../bin/chainmaker

package:
	@cd main && GOPATH=${GOPATH} go build -o ../bin/chainmaker
	@mkdir -p ./release
	@rm -rf ./tmp/chainmaker/
	@mkdir -p ./tmp/chainmaker/
	@mkdir ./tmp/chainmaker/bin
	@mkdir ./tmp/chainmaker/config
	@mkdir ./tmp/chainmaker/log
	@cp bin/chainmaker ./tmp/chainmaker/bin
	@cp -r config ./tmp/chainmaker/
	@cd ./tmp;tar -zcvf chainmaker-$(VERSION).$(DATETIME).$(PLATFORM).tar.gz chainmaker; mv chainmaker-$(VERSION).$(DATETIME).$(PLATFORM).tar.gz ../release
	@rm -rf ./tmp/

compile:
	@cd main && go build -o ../bin/chainmaker

cmc:
	@cd tools/cmc && GOPATH=${GOPATH} go build -o ../../bin/cmc 

scanner:
	@cd tools/scanner && GOPATH=${GOPATH} go build -o ../../bin/scanner 

dep: pb-dep mockgen-dep 
	@go get golang.org/x/tools/cmd/stringer

generate:
	go generate ./...

pb:
	cd pb/proto && protoc -I=. --gogofaster_out=:../protogo --gogofaster_opt=paths=source_relative accesscontrol/*.proto
	cd pb/proto && protoc -I=. --gogofaster_out=:../protogo --gogofaster_opt=paths=source_relative common/*.proto
	cd pb/proto && protoc -I=. --gogofaster_out=:../protogo --gogofaster_opt=paths=source_relative discovery/*.proto
	cd pb/proto && protoc -I=. --gogofaster_out=:../protogo --gogofaster_opt=paths=source_relative net/*.proto
	cd pb/proto && protoc -I=. --gogofaster_out=:../protogo --gogofaster_opt=paths=source_relative store/*.proto
	cd pb/proto && protoc -I=. --gogofaster_out=:../protogo --gogofaster_opt=paths=source_relative sync/*.proto
	cd pb/proto && protoc -I=. --gogofaster_out=:../protogo --gogofaster_opt=paths=source_relative txpool/*.proto
	cd pb/proto && protoc -I=. --gogofaster_out=:../protogo --gogofaster_opt=paths=source_relative consensus/*.proto
	cd pb/proto && protoc -I=. --gogofaster_out=:../protogo --gogofaster_opt=paths=source_relative consensus/tbft/*.proto
	cd pb/proto && protoc -I=. --gogofaster_out=:../protogo --gogofaster_opt=paths=source_relative config/*.proto
	cd pb/proto && protoc -I=. --gogofaster_out=plugins=grpc:../protogo --gogofaster_opt=paths=source_relative api/rpc_node.proto

pb-dep:
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	go get -u github.com/golang/protobuf/protoc-gen-go
	go get -u google.golang.org/grpc
	go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	GO111MODULE=on go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
	go get -u github.com/gogo/protobuf/protoc-gen-gogofaster

clean-pb:
	rm -f pb/*.pb.go
	rm -f pb/api/*.pb.go
	rm -f pb/api/*.gw.go
	rm -f pb/api/*.json
	rm -f pb/api/demo/*.pb.go
	rm -f pb/api/demo/*.gw.go
	rm -f pb/api/demo/*.json


.PHONY: pb

mockgen:
	cd protocol && mockgen -destination ../mock/access_control_mock.go -package mock -source access_control_interface.go
	cd protocol && mockgen -destination ../mock/cache_mock.go -package mock -source cache_interface.go
	cd protocol && mockgen -destination ../mock/consensus_mock.go -package mock -source consensus_interface.go
	cd protocol && mockgen -destination ../mock/core_mock.go -package mock -source core_interface.go
	cd protocol && mockgen -destination ../mock/net_mock.go -package mock -source net_interface.go
	cd protocol && mockgen -destination ../mock/scheduler_mock.go -package mock -source scheduler_interface.go
	cd protocol && mockgen -destination ../mock/snapshot_mock.go -package mock -source snapshot_interface.go
	cd protocol && mockgen -destination ../mock/store_mock.go -package mock -source store_interface.go
	cd protocol && mockgen -destination ../mock/sync_mock.go -package mock -source sync_interface.go
	cd protocol && mockgen -destination ../mock/tx_pool_mock.go -package mock -source tx_pool_interface.go
	cd protocol && mockgen -destination ../mock/vm_mock.go -package mock -source vm_interface.go
	cd protocol && mockgen -destination ../mock/chainconf_mock.go -package mock -source chainconf_interface.go
	cd common/msgbus && mockgen -destination ../../mock/msgbus_mock.go -package mock -source message_bus.go

mockgen-dep:
	go get -u github.com/golang/mock/gomock
	go get -u github.com/golang/mock/mockgen

docker-build:
	docker build -t chainmaker -f ./DOCKER/Dockerfile .
	docker tag chainmaker chainmaker:v1.1.1

docker-compose-start: docker-compose-stop
	docker-compose up -d

docker-compose-stop:
	docker-compose down

ut:
	cd common && go test ./...
	cd main && go test ./...
	cd module/accesscontrol && go test ./...
	cd module/blockchain && go test ./...
	cd module/conf && go test ./...
	cd module/consensus && go test ./...
	cd module/core && go test ./...
	cd module/logger && go test ./...
	cd module/net && go test ./...
	cd module/rpcserver && go test ./...
	cd module/snapshot && go test ./...
	cd module/store && go test ./...
	cd module/subscriber && go test ./...
	cd module/sync && go test ./...
	cd module/txpool && go test ./...
	cd module/utils && go test ./...
	cd module/vm && go test ./...
	cd tools/cmc && go test ./...
	cd tools/scanner && go test ./...
	cd tools/sdk && go test ./...

lint:
	cd common && golangci-lint run ./...
	cd main && golangci-lint run ./...
	cd module/accesscontrol && golangci-lint run ./...
	cd module/blockchain && golangci-lint run ./...
	cd module/conf && golangci-lint run ./...
	cd module/consensus && golangci-lint run ./...
	cd module/core && golangci-lint run ./...
	cd module/logger && golangci-lint run ./...
	cd module/net && golangci-lint run ./...
	cd module/rpcserver && golangci-lint run ./...
	cd module/snapshot && golangci-lint run ./...
	cd module/store && golangci-lint run ./...
	cd module/subscriber && golangci-lint run ./...
	cd module/sync && golangci-lint run ./...
	cd module/txpool && golangci-lint run ./...
	cd module/utils && golangci-lint run ./...
	cd module/vm && golangci-lint run ./...
	cd tools/cmc && golangci-lint run ./...
	cd tools/scanner && golangci-lint run ./...
	cd tools/sdk && golangci-lint run ./...