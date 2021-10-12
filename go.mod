module chainmaker.org/chainmaker-go

go 1.15

require (
	chainmaker.org/chainmaker-go/blockchain v0.0.0
	chainmaker.org/chainmaker-go/rpcserver v0.0.0-00010101000000-000000000000
	chainmaker.org/chainmaker-go/txpool v0.0.0
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20211012062258-a0408edb4a1c
	chainmaker.org/chainmaker/logger/v2 v2.0.0
	chainmaker.org/chainmaker/txpool-batch/v2 v2.0.0-20211013035004-190b3acb8eb8
	chainmaker.org/chainmaker/txpool-single/v2 v2.0.0-20211013035055-0b3b640215ad
	code.cloudfoundry.org/bytefmt v0.0.0-20200131002437-cf55d5288a48
	github.com/common-nighthawk/go-figure v0.0.0-20200609044655-c4b36f998cf2
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
	github.com/libp2p/go-libp2p => ./module/net/p2p/libp2p
	github.com/libp2p/go-libp2p-core => ./module/net/p2p/libp2pcore
	github.com/libp2p/go-libp2p-pubsub => ./module/net/p2p/libp2ppubsub
	google.golang.org/grpc v1.40.0 => google.golang.org/grpc v1.26.0
)
