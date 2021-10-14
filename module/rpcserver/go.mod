module chainmaker.org/chainmaker-go/rpcserver

go 1.15

require (
	chainmaker.org/chainmaker-go/blockchain v0.0.0-00010101000000-000000000000
	chainmaker.org/chainmaker-go/subscriber v0.0.0
	chainmaker.org/chainmaker/chainconf/v2 v2.0.0-20211014142031-0a2a2aac316c // indirect
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20211014122130-4ba9d85a64f8
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20211014134424-9431ffcc5bbc
	chainmaker.org/chainmaker/logger/v2 v2.0.1-0.20211014131951-892d098049bc
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211014120010-525e2ffaf04d
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20211014144951-97323532a236
	chainmaker.org/chainmaker/store/v2 v2.0.0-20211014154101-7b199f0df636
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20211014131421-43de8d9fe869
	chainmaker.org/chainmaker/vm v0.0.0-20211014150836-d6eae08ad3bd // indirect
	chainmaker.org/chainmaker/vm-evm v0.0.0-20211014155012-e69085fedd2f // indirect
	chainmaker.org/chainmaker/vm-gasm v0.0.0-20211014154622-4c1c925eacdd // indirect
	chainmaker.org/chainmaker/vm-native v0.0.0-20211014145655-91f98409dbbd
	github.com/gogo/protobuf v1.3.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/prometheus/client_golang v1.11.0
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	google.golang.org/grpc v1.41.0
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	chainmaker.org/chainmaker-go/blockchain => ../blockchain
	chainmaker.org/chainmaker-go/consensus => ../consensus
	chainmaker.org/chainmaker-go/core => ../core
	chainmaker.org/chainmaker-go/net => ../net
	chainmaker.org/chainmaker-go/snapshot => ../snapshot
	chainmaker.org/chainmaker-go/subscriber => ../subscriber
	chainmaker.org/chainmaker-go/sync => ../sync
	chainmaker.org/chainmaker-go/txpool => ../txpool
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
	google.golang.org/grpc v1.40.0 => google.golang.org/grpc v1.26.0
)
