module chainmaker.org/chainmaker-go/main

go 1.15

require (
	chainmaker.org/chainmaker-go/blockchain v0.0.0
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/monitor v0.0.0
	chainmaker.org/chainmaker-go/rpcserver v0.0.0
	code.cloudfoundry.org/bytefmt v0.0.0-20200131002437-cf55d5288a48
	github.com/common-nighthawk/go-figure v0.0.0-20200609044655-c4b36f998cf2
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	gorm.io/driver/sqlite v1.1.4 // indirect
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../module/accesscontrol
	chainmaker.org/chainmaker-go/blockchain => ../module/blockchain
	chainmaker.org/chainmaker-go/chainconf => ./../module/conf/chainconf
	chainmaker.org/chainmaker-go/common => ./../common
	chainmaker.org/chainmaker-go/consensus => ./../module/consensus
	chainmaker.org/chainmaker-go/core => ./../module/core
	chainmaker.org/chainmaker-go/gasm => ../module/vm/gasm
	chainmaker.org/chainmaker-go/localconf => ./../module/conf/localconf
	chainmaker.org/chainmaker-go/logger => ../module/logger
	chainmaker.org/chainmaker-go/mock => ../mock
	chainmaker.org/chainmaker-go/monitor => ../module/monitor
	chainmaker.org/chainmaker-go/net => ./../module/net
	chainmaker.org/chainmaker-go/pb/protogo => ./../pb/protogo
	chainmaker.org/chainmaker-go/protocol => ./../protocol
	chainmaker.org/chainmaker-go/rpcserver => ./../module/rpcserver
	chainmaker.org/chainmaker-go/snapshot => ./../module/snapshot
	chainmaker.org/chainmaker-go/spv => ./../module/spv
	chainmaker.org/chainmaker-go/store => ./../module/store
	chainmaker.org/chainmaker-go/subscriber => ./../module/subscriber
	chainmaker.org/chainmaker-go/sync => ./../module/sync
	chainmaker.org/chainmaker-go/txpool => ./../module/txpool
	chainmaker.org/chainmaker-go/txpool/batchtxpool => ./../module/txpool/batch
	chainmaker.org/chainmaker-go/utils => ./../module/utils
	chainmaker.org/chainmaker-go/vm => ./../module/vm
	chainmaker.org/chainmaker-go/wasi => ../module/vm/wasi
	chainmaker.org/chainmaker-go/wasmer => ../module/vm/wasmer
	chainmaker.org/chainmaker-go/wxvm => ../module/vm/wxvm
	github.com/libp2p/go-libp2p => ./../module/net/p2p/libp2p
	github.com/libp2p/go-libp2p-core => ./../module/net/p2p/libp2pcore
	github.com/libp2p/go-libp2p-pubsub => ./../module/net/p2p/libp2ppubsub
)
