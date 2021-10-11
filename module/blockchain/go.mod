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
	chainmaker.org/chainmaker/chainconf/v2 v2.0.0-20211011073602-d17527b03602
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20211011114226-30eafbbd6523
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20211009063450-f9db84192eea
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907134457-53647922a89d
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211011114556-3bbc2a898d5a
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20211009064056-03cbf6096208
	chainmaker.org/chainmaker/store/v2 v2.0.0-20211009022637-e5e1cba4871b
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20211011062420-24df29961fa2
	chainmaker.org/chainmaker/vm v0.0.0-20210918104424-239140ec3366
	chainmaker.org/chainmaker/vm-evm v0.0.0-20210916091920-b915815eb88b
	chainmaker.org/chainmaker/vm-gasm v0.0.0-20210918095814-3f0ddfe29968
	chainmaker.org/chainmaker/vm-wasmer v0.0.0-20211008132543-79cb76bb2f68
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
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
	google.golang.org/grpc v1.40.0 => google.golang.org/grpc v1.26.0
)
