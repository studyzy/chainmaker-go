#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
rm -rf ./bin/chainmaker
cd ../../main
go build -o ../test/send_proposal_request_4bft/bin/chainmaker