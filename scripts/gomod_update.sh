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
go get chainmaker.org/chainmaker/localconf/v2@${BRANCH}
go get chainmaker.org/chainmaker/utils/v2@${BRANCH}
go mod tidy

cd ../consensus
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go get chainmaker.org/chainmaker/localconf/v2@${BRANCH}
go get chainmaker.org/chainmaker/chainconf/v2@${BRANCH}
go get chainmaker.org/chainmaker/utils/v2@${BRANCH}
go get chainmaker.org/chainmaker/vm-native@${BRANCH}
go get chainmaker.org/chainmaker/logger/v2@${BRANCH}
go mod tidy

cd ../core
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go get chainmaker.org/chainmaker/localconf/v2@${BRANCH}
go get chainmaker.org/chainmaker/chainconf/v2@${BRANCH}
go get chainmaker.org/chainmaker/utils/v2@${BRANCH}
go get chainmaker.org/chainmaker/vm@${BRANCH}
go get chainmaker.org/chainmaker/logger/v2@${BRANCH}
go mod tidy


cd ../net
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go get chainmaker.org/chainmaker/logger/v2@${BRANCH}
go get chainmaker.org/chainmaker/chainmaker-net-common@${BRANCH}
go get chainmaker.org/chainmaker/chainmaker-net-liquid@${BRANCH}
go get chainmaker.org/chainmaker/chainmaker-net-libp2p@${BRANCH}
go mod tidy

cd ../rpcserver
go get chainmaker.org/chainmaker/logger/v2@${BRANCH}
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go get chainmaker.org/chainmaker/store/v2@${BRANCH}
go get chainmaker.org/chainmaker/vm-native@${BRANCH}
go get chainmaker.org/chainmaker/localconf/v2@${BRANCH}
go get chainmaker.org/chainmaker/utils/v2@${BRANCH}
go mod tidy

cd ../snapshot
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go get chainmaker.org/chainmaker/localconf/v2@${BRANCH}
go get chainmaker.org/chainmaker/logger/v2@${BRANCH}
go mod tidy

cd ../subscriber
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go mod tidy

cd ../sync
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go get chainmaker.org/chainmaker/localconf/v2@${BRANCH}
go get chainmaker.org/chainmaker/logger/v2@${BRANCH}
go mod tidy

cd ../txpool
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go get chainmaker.org/chainmaker/txpool-batch/v2@${BRANCH}
go get chainmaker.org/chainmaker/txpool-single/v2@${BRANCH}
go mod tidy

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
go get chainmaker.org/chainmaker/localconf/v2@${BRANCH}
go get chainmaker.org/chainmaker/chainconf/v2@${BRANCH}
go get chainmaker.org/chainmaker/txpool-batch/v2@${BRANCH}
go get chainmaker.org/chainmaker/vm@${BRANCH}
go get chainmaker.org/chainmaker/utils/v2@${BRANCH}
go mod tidy

cd ../vm
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go mod tidy

#cd ../tools/cmc
#go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
#go get chainmaker.org/chainmaker/sdk-go/v2@${BRANCH}
#go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
#go get chainmaker.org/chainmaker/common/v2@${BRANCH}
#go mod tidy

cd ../../
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/vm-wasmer@${BRANCH}
go get chainmaker.org/chainmaker/vm-gasm@${BRANCH}
go get chainmaker.org/chainmaker/vm-wxvm@${BRANCH}
go get chainmaker.org/chainmaker/vm-evm@${BRANCH}
go get chainmaker.org/chainmaker/localconf/v2@${BRANCH}
go get chainmaker.org/chainmaker/txpool-batch/v2@${BRANCH}
go get chainmaker.org/chainmaker/txpool-single/v2@${BRANCH}
go get chainmaker.org/chainmaker/logger/v2@${BRANCH}
go mod tidy

cd ./main
go build -o chainmaker
