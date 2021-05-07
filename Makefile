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
	docker tag chainmaker chainmaker:v1.0.0_r

docker-compose-start: docker-compose-stop
	docker-compose up -d

docker-compose-stop:
	docker-compose down
