#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

set -x

cd ../module/accesscontrol
go mod download
cd ../blockchain
go mod download
cd ../conf/chainconf
go mod download
cd ../localconf
go mod download
cd ../../consensus
go mod download
cd ../core
go mod download
cd ../logger
go mod download
cd ../net
go mod download
cd ../rpcserver
go mod download
cd ../snapshot
go mod download
cd ../store
go mod download
cd ../subscriber
go mod download
cd ../sync
go mod download
cd ../txpool
go mod download
cd ../utils
go mod download
cd ../vm
go mod download
cd ../../test
go mod download
cd ../tools/cmc
go mod download
