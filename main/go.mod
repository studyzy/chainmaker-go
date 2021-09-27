module chainmaker.org/chainmaker-go/main

go 1.15

require (
	chainmaker.org/chainmaker-go/blockchain v0.0.0
	chainmaker.org/chainmaker-go/monitor v0.0.0
	chainmaker.org/chainmaker-go/rpcserver v0.0.0-00010101000000-000000000000
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20210927063633-0994fcdf3928
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907134457-53647922a89d
	code.cloudfoundry.org/bytefmt v0.0.0-20200131002437-cf55d5288a48
	github.com/common-nighthawk/go-figure v0.0.0-20200609044655-c4b36f998cf2
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../module/accesscontrol
	chainmaker.org/chainmaker-go/blockchain => ../module/blockchain
	chainmaker.org/chainmaker-go/consensus => ./../module/consensus
	chainmaker.org/chainmaker-go/core => ./../module/core
	chainmaker.org/chainmaker-go/monitor => ../module/monitor
	chainmaker.org/chainmaker-go/net => ./../module/net
	chainmaker.org/chainmaker-go/rpcserver => ./../module/rpcserver
	chainmaker.org/chainmaker-go/snapshot => ./../module/snapshot
	chainmaker.org/chainmaker-go/subscriber => ./../module/subscriber
	chainmaker.org/chainmaker-go/sync => ./../module/sync
	chainmaker.org/chainmaker-go/txpool => ./../module/txpool
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
)
