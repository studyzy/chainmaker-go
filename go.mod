module chainmaker.org/chainmaker-go

go 1.15

require (
	chainmaker.org/chainmaker-go/blockchain v0.0.0
	chainmaker.org/chainmaker-go/rpcserver v0.0.0-00010101000000-000000000000
	chainmaker.org/chainmaker-go/txpool v0.0.0
	chainmaker.org/chainmaker-go/vm v0.0.0
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20211014134424-9431ffcc5bbc
	chainmaker.org/chainmaker/logger/v2 v2.0.1-0.20211015125919-8e5199930ac9
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20211022113918-f11dc73904c1
	chainmaker.org/chainmaker/txpool-batch/v2 v2.0.0-20211019074609-46e3d29f0908
	chainmaker.org/chainmaker/txpool-single/v2 v2.0.0-20211018131403-7eb37f80a128
	chainmaker.org/chainmaker/vm-evm v0.0.0-20211022115217-ea9fef83d452
	chainmaker.org/chainmaker/vm-gasm v0.0.0-20211022122255-a9820b48eeb2
	chainmaker.org/chainmaker/vm-wasmer v0.0.0-20211022121804-8c562b28d334
	chainmaker.org/chainmaker/vm-wxvm v0.0.0-20211022122830-78371cdbd3a9
	code.cloudfoundry.org/bytefmt v0.0.0-20200131002437-cf55d5288a48
	github.com/common-nighthawk/go-figure v0.0.0-20200609044655-c4b36f998cf2
	github.com/ethereum/go-ethereum v1.10.4 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ./module/accesscontrol
	chainmaker.org/chainmaker-go/blockchain => ./module/blockchain
	chainmaker.org/chainmaker-go/consensus => ./module/consensus
	chainmaker.org/chainmaker-go/core => ./module/core
	chainmaker.org/chainmaker-go/net => ./module/net
	chainmaker.org/chainmaker-go/rpcserver => ./module/rpcserver
	chainmaker.org/chainmaker-go/snapshot => ./module/snapshot
	chainmaker.org/chainmaker-go/subscriber => ./module/subscriber
	chainmaker.org/chainmaker-go/sync => ./module/sync
	chainmaker.org/chainmaker-go/txpool => ./module/txpool
	chainmaker.org/chainmaker-go/vm => ./module/vm
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
	google.golang.org/grpc v1.40.0 => google.golang.org/grpc v1.26.0
)
