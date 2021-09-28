module chainmaker.org/chainmaker-go/blockchain

go 1.15

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0
	chainmaker.org/chainmaker-go/consensus v0.0.0
	chainmaker.org/chainmaker-go/core v0.0.0
	chainmaker.org/chainmaker-go/net v0.0.0
	chainmaker.org/chainmaker-go/snapshot v0.0.0
	chainmaker.org/chainmaker-go/subscriber v0.0.0
	chainmaker.org/chainmaker-go/sync v0.0.0
	chainmaker.org/chainmaker-go/txpool v0.0.0
	chainmaker.org/chainmaker/chainconf/v2 v2.0.0-20210928032929-371e51d10202
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210927025216-3d740cb6258e
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20210928020228-3ab2986d5ecd
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907134457-53647922a89d
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210916064951-47123db73430
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210927062046-68813f263c0b
	chainmaker.org/chainmaker/store/v2 v2.0.0-20210927063334-95fec89a7435
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210916084713-abd13154c26b
	chainmaker.org/chainmaker/vm v0.0.0-20210918104424-239140ec3366
	chainmaker.org/chainmaker/vm-evm v0.0.0-20210916091920-b915815eb88b
	chainmaker.org/chainmaker/vm-gasm v0.0.0-20210918095814-3f0ddfe29968
	chainmaker.org/chainmaker/vm-wasmer v0.0.0-20210918173526-7cd16f1a1d3a
	chainmaker.org/chainmaker/vm-wxvm v0.0.0-20210918101823-dce1c76fb189
	github.com/mitchellh/mapstructure v1.4.1
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	chainmaker.org/chainmaker-go/consensus => ../consensus
	chainmaker.org/chainmaker-go/core => ../core
	chainmaker.org/chainmaker-go/monitor => ../monitor
	chainmaker.org/chainmaker-go/net => ../net
	chainmaker.org/chainmaker-go/snapshot => ../snapshot
	chainmaker.org/chainmaker-go/subscriber => ../subscriber
	chainmaker.org/chainmaker-go/sync => ../sync
	chainmaker.org/chainmaker-go/txpool => ../txpool
	//chainmaker.org/chainmaker-go/txpool/batchtxpool => ./../txpool/batch
	github.com/libp2p/go-libp2p => ../net/p2p/libp2p
	github.com/libp2p/go-libp2p-core => ../net/p2p/libp2pcore
	github.com/libp2p/go-libp2p-pubsub => ../net/p2p/libp2ppubsub
	google.golang.org/grpc v1.40.0 => google.golang.org/grpc v1.26.0
)
