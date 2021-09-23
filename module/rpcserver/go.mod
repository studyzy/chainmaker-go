module chainmaker.org/chainmaker-go/rpcserver

go 1.15

require (
	chainmaker.org/chainmaker-go/blockchain v0.0.0
	chainmaker.org/chainmaker-go/monitor v0.0.0
	chainmaker.org/chainmaker-go/subscriber v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210922025418-6abc6bbf1be9
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20210914062957-13e84972a921
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907134457-53647922a89d
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210916064951-47123db73430
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210914063622-6f007edc3a98
	chainmaker.org/chainmaker/store/v2 v2.0.0-20210922083030-e886493a2be9
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210916084713-abd13154c26b
	chainmaker.org/chainmaker/vm-native v0.0.0-20210917091516-85e8d7855fe5
	github.com/gogo/protobuf v1.3.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/prometheus/client_golang v1.11.0
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	google.golang.org/grpc v1.38.0
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
	github.com/libp2p/go-libp2p => ../net/p2p/libp2p
	github.com/libp2p/go-libp2p-core => ../net/p2p/libp2pcore
	github.com/libp2p/go-libp2p-pubsub => ../net/p2p/libp2ppubsub
)
