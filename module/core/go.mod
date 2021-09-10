module chainmaker.org/chainmaker-go/core

go 1.15

require (
	chainmaker.org/chainmaker-go/chainconf v0.0.0
	chainmaker.org/chainmaker-go/consensus v0.0.0
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/monitor v0.0.0
	chainmaker.org/chainmaker-go/store v0.0.0
	chainmaker.org/chainmaker-go/subscriber v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210909033927-2a4cfc146579
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907133316-af00cea33c97
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210901132412-435b75070bf2
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210910074933-23793f07a247
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
	github.com/gogo/protobuf v1.3.2
	github.com/google/martian v2.1.0+incompatible
	github.com/panjf2000/ants/v2 v2.4.3
	github.com/prometheus/client_golang v1.9.0
	github.com/prometheus/common v0.15.0
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	chainmaker.org/chainmaker-go/chainconf => ./../conf/chainconf
	chainmaker.org/chainmaker-go/consensus => ../consensus
	chainmaker.org/chainmaker-go/consensus/dpos => ./../consensus/dpos
	chainmaker.org/chainmaker-go/evm => ../vm/evm
	chainmaker.org/chainmaker-go/gasm => ../vm/gasm
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf

	chainmaker.org/chainmaker-go/monitor => ../monitor
	chainmaker.org/chainmaker-go/store => ../store
	chainmaker.org/chainmaker-go/subscriber => ../subscriber

	chainmaker.org/chainmaker-go/vm => ../vm
	chainmaker.org/chainmaker-go/wasi => ../vm/wasi
	chainmaker.org/chainmaker-go/wasmer => ../vm/wasmer
	chainmaker.org/chainmaker-go/wxvm => ../vm/wxvm
)
