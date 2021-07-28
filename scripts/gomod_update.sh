#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -x
BRANCH=v2.0.1_dev

cd ../module/accesscontrol
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../blockchain
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../conf/chainconf
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../localconf
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../../consensus
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../core
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../logger
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../net
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../rpcserver
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
#go test ./...
cd ../snapshot
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../store
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../subscriber
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../sync
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../txpool
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../utils
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../vm
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go test ./...
cd ../../test
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
go build ./...
cd ../tools/cmc
go get chainmaker.org/chainmaker/protocol@${BRANCH}
go get chainmaker.org/chainmaker/sdk-go@${BRANCH}
go get chainmaker.org/chainmaker/pb-go@${BRANCH}
go get chainmaker.org/chainmaker/common@${BRANCH}
go mod tidy
#go test ./...
go build .
cd ../../main
go mod tidy
go build .