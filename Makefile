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
VERSION=V1.2.0

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

dep:
	@go get golang.org/x/tools/cmd/stringer

generate:
	go generate ./...

docker-build:
	docker build -t chainmaker -f ./DOCKER/Dockerfile .
	docker tag chainmaker chainmaker:v1.2.0

docker-build-dev: chainmaker
	docker build -t chainmaker -f ./DOCKER/dev.Dockerfile .
	docker tag chainmaker chainmaker:v1.2.0

docker-compose-start: docker-compose-stop
	docker-compose up -d

docker-compose-stop:
	docker-compose down

ut:
	cd main && go test -cover ./...
	cd module/accesscontrol && go test -cover ./...
	cd module/blockchain && go test -cover ./...
	cd module/conf/chainconf && go test -cover ./...
	cd module/conf/localconf && go test -cover ./...
	cd module/consensus && go test -cover ./...
	cd module/core && go test -cover ./...
	cd module/logger && go test -cover ./...
	cd module/net && go test -cover ./...
	cd module/rpcserver && go test -cover ./...
	cd module/snapshot && go test -cover ./...
	cd module/store && go test -cover ./...
	cd module/subscriber && go test -cover ./...
	cd module/sync && go test -cover ./...
	cd module/txpool && go test -cover ./...
	cd module/utils && go test -cover ./...
	cd module/vm && go test -cover ./...
	#cd tools/cmc && go test -cover ./...
	#cd tools/scanner && go test -cover ./...

lint:
#	cd common && golangci-lint run ./...
#	cd main && golangci-lint run ./...
#	cd module/accesscontrol && golangci-lint run ./...
#	cd module/blockchain && golangci-lint run ./...
	cd module/conf/chainconf && golangci-lint run ./...
	cd module/conf/localconf && golangci-lint run ./...
	cd module/consensus/raft && golangci-lint run ./...
#	cd module/core && golangci-lint run ./...
	cd module/logger && golangci-lint run ./...
#	cd module/net && golangci-lint run ./...
#	cd module/rpcserver && golangci-lint run ./...
#	cd module/snapshot && golangci-lint run ./...
	cd module/store && golangci-lint run ./...
#	cd module/subscriber && golangci-lint run ./...
#	cd module/sync && golangci-lint run ./...
#	cd module/txpool && golangci-lint run ./...
	cd module/utils && golangci-lint run ./...
	cd module/vm/gasm && golangci-lint run ./...
#	cd tools/cmc && golangci-lint run ./...
#	cd tools/scanner && golangci-lint run ./...
#	cd tools/sdk && golangci-lint run ./...

gomod:
	cd scripts && ./gomod_update.sh

test-deploy:
	cd scripts/test/ && ./quick_deploy.sh