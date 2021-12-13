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
VERSION=v2.1.0
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT = $(shell git log --pretty=format:'%h' -n 1)

AARCH64="aarch64"
CPU=$(shell uname -m)

LOCALCONF_HOME=chainmaker.org/chainmaker-go/blockchain
GOLDFLAGS += -X "${LOCALCONF_HOME}.CurrentVersion=${VERSION}"
GOLDFLAGS += -X "${LOCALCONF_HOME}.BuildDateTime=${DATETIME}"
GOLDFLAGS += -X "${LOCALCONF_HOME}.GitBranch=${GIT_BRANCH}"
GOLDFLAGS += -X "${LOCALCONF_HOME}.GitCommit=${GIT_COMMIT}"

chainmaker:
ifeq ("$(CPU)",$(AARCH64))
ifneq ($(wildcard module/vm/wasmer/wasmer-go/libwasmer.so.aarch64),)
	mv module/vm/wasmer/wasmer-go/libwasmer.so module/vm/wasmer/wasmer-go/libwasmer.so.x86_64
	mv module/vm/wasmer/wasmer-go/libwasmer.so.aarch64 module/vm/wasmer/wasmer-go/libwasmer.so
endif
else
ifneq ($(wildcard module/vm/wasmer/wasmer-go/libwasmer.so.x86_64),)
	mv module/vm/wasmer/wasmer-go/libwasmer.so module/vm/wasmer/wasmer-go/libwasmer.so.aarch64
	mv module/vm/wasmer/wasmer-go/libwasmer.so.x86_64 module/vm/wasmer/wasmer-go/libwasmer.so
endif
endif
	@cd main && go mod tidy && go build -ldflags '${GOLDFLAGS}' -o ../bin/chainmaker

chainmaker-vendor:
	@cd main && go build -mod=vendor -o ../bin/chainmaker

package:
	@cd main && go mod tidy && GOPATH=${GOPATH} go build -ldflags '${GOLDFLAGS}' -o ../bin/chainmaker
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
	@cd main && go mod tidy && go build -ldflags '${GOLDFLAGS}' -o ../bin/chainmaker

cmc:
	@cd tools/cmc && GOPATH=${GOPATH} go build -o ../../bin/cmc

send-tool:
	cd test/send_proposal_request_tool && go build -o ../../bin/send_proposal_request_tool

scanner:
	@cd tools/scanner && GOPATH=${GOPATH} go build -o ../../bin/scanner

dep:
	@go get golang.org/x/tools/cmd/stringer

generate:
	go generate ./...

docker-build:
	rm -rf build/ data/ log/ bin/
	go mod tidy
	docker build -t chainmaker -f ./DOCKER/Dockerfile .
	docker tag chainmaker chainmaker:${VERSION}

docker-build-dev: chainmaker
	docker build -t chainmaker -f ./DOCKER/dev.Dockerfile .
	docker tag chainmaker chainmaker:${VERSION}

docker-compose-start: docker-compose-stop
	docker-compose up -d

docker-compose-stop:
	docker-compose down

ut:
	cd scripts && ./ut_cover.sh

lint:
	cd main && golangci-lint run ./...
	cd module/accesscontrol && golangci-lint run .
	cd module/blockchain && golangci-lint run .
	cd module/consensus && golangci-lint run ./...
	cd module/core && golangci-lint run ./...
	cd module/net && golangci-lint run ./...
	cd module/rpcserver && golangci-lint run ./...
	cd module/snapshot && golangci-lint run ./...
	cd module/subscriber && golangci-lint run ./...
	cd module/sync && golangci-lint run ./...
	cd tools/cmc && golangci-lint run ./...
	cd tools/scanner && golangci-lint run ./...

gomod:
	cd scripts && ./gomod_update.sh

test-deploy:
	cd scripts/test/ && ./quick_deploy.sh

sql-qta:
	echo "clear environment"
	cd test/send_proposal_request_ci && ./stop_sql_tbft_4.sh
	cd test/send_proposal_request_ci && ./clean_sql_log.sh
	echo "start new sql-qta test"
	cd test/send_proposal_request_ci && ./build.sh
	cd test/send_proposal_request_ci && ./start_sql_tbft_4.sh
	cd test/send_proposal_request_sql && go run main.go
	cd test/send_proposal_request_ci && ./stop_sql_tbft_4.sh
	cd test/send_proposal_request_ci && ./clean_sql_log.sh
qta:
	cd test/send_proposal_request_ci && ./build.sh
	cd test/send_proposal_request_ci && ./start_solo.sh
	cd test/send_proposal_request_ci && go run main.go
	cd test/send_proposal_request_ci && ./stop_solo.sh
	cd test/send_proposal_request_ci && ./clean_data_log.sh
