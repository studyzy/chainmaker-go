#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -x
BRANCH=develop

cd ../module/accesscontrol
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go mod tidy
# go test ./...
cd ../blockchain
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go get chainmaker.org/chainmaker/store/v2@${BRANCH}
go get chainmaker.org/chainmaker/vm-native@${BRANCH}
go get chainmaker.org/chainmaker/vm-wasmer@${BRANCH}
go get chainmaker.org/chainmaker/vm-gasm@${BRANCH}
go get chainmaker.org/chainmaker/vm-wxvm@${BRANCH}
go get chainmaker.org/chainmaker/vm-evm@${BRANCH}
go mod tidy
# go test ./...
cd ../consensus
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go get chainmaker.org/chainmaker/vm-native@${BRANCH}
go mod tidy
# go test ./...
cd ../core
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go get chainmaker.org/chainmaker/vm-native@${BRANCH}
go get chainmaker.org/chainmaker/vm@${BRANCH}
go mod tidy
# go test ./...

# go test ./...
cd ../net
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go mod tidy
# go test ./...
cd ../rpcserver
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go get chainmaker.org/chainmaker/vm-native@${BRANCH}
go get chainmaker.org/chainmaker/utils/v2@${BRANCH}
go get chainmaker.org/chainmaker/store/v2@${BRANCH}
go mod tidy
## go test ./...
cd ../snapshot
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go mod tidy
# go test ./...
cd ../subscriber
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go mod tidy
# go test ./...
cd ../sync
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go mod tidy
# go test ./...
cd ../txpool
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go mod tidy
# go test ./...
#cd ../vm
#go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
#go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
#go get chainmaker.org/chainmaker/common/v2@${BRANCH}
#go mod tidy
#cd gasm
#go mod tidy
#cd ../evm
#go mod tidy
#cd ../wasi
#go mod tidy
#cd ../wasmer
#go mod tidy
#cd ../wxvm
#go mod tidy
cd ../../test
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go mod tidy
go build ./...
cd ../tools/cmc
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/sdk-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go mod tidy
## go test ./...
go build .
cd ../scanner
go mod tidy
cd ../../main
go mod tidy
go build .
