module chainmaker.org/chainmaker-go/rpcserver

go 1.15

require (
	chainmaker.org/chainmaker-go/blockchain v0.0.0-00010101000000-000000000000
	chainmaker.org/chainmaker-go/monitor v0.0.0
	chainmaker.org/chainmaker-go/subscriber v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20211011114226-30eafbbd6523
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20211009063450-f9db84192eea
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210927081951-999ab4a3fad6
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211011114556-3bbc2a898d5a
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20211009064056-03cbf6096208
	chainmaker.org/chainmaker/store/v2 v2.0.0-20211009022637-e5e1cba4871b
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20211012064332-ba20dd386083
	chainmaker.org/chainmaker/vm-native v0.0.0-20211012092026-c61f9d7b1e7c
	github.com/gogo/protobuf v1.3.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/prometheus/client_golang v1.11.0
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	google.golang.org/grpc v1.40.0
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	chainmaker.org/chainmaker-go/blockchain => ../blockchain
	chainmaker.org/chainmaker-go/consensus => ../consensus
	chainmaker.org/chainmaker-go/core => ../core
	chainmaker.org/chainmaker-go/monitor => ../monitor
	chainmaker.org/chainmaker-go/net => ../net
	chainmaker.org/chainmaker-go/snapshot => ../snapshot
	chainmaker.org/chainmaker-go/subscriber => ../subscriber
	chainmaker.org/chainmaker-go/sync => ../sync
	chainmaker.org/chainmaker-go/txpool => ../txpool
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
	google.golang.org/grpc v1.40.0 => google.golang.org/grpc v1.26.0
)
