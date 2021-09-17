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
	chainmaker.org/chainmaker/chainconf/v2 v2.0.0-20210913144615-f27c44059848
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210915091854-ee36ee89dc57
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20210914062957-13e84972a921
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907134457-53647922a89d
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210901132412-435b75070bf2
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210914063622-6f007edc3a98
	chainmaker.org/chainmaker/store/v2 v2.0.0-20210913121309-9387122bc431
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
	chainmaker.org/chainmaker/vm v0.0.0-20210914114029-6004e1ebd15e
	chainmaker.org/chainmaker/vm-evm v0.0.0-20210914075919-6899e681474a
	chainmaker.org/chainmaker/vm-gasm v0.0.0-20210914080309-c93c4afc11e8
	chainmaker.org/chainmaker/vm-wasmer v0.0.0-20210914123344-d12baf7f7cbe
	chainmaker.org/chainmaker/vm-wxvm v0.0.0-20210914075731-ec7bc0766d57
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
)
