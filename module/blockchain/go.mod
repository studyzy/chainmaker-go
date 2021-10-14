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
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20211011130949-b332c3193ef5
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20211013023845-6792e74fdd6d
	chainmaker.org/chainmaker/logger/v2 v2.0.0
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211011124513-b828aaef61ff
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210929132906-a818e778c41b
	chainmaker.org/chainmaker/store/v2 v2.0.0-20211009100304-856ce14a7318
	chainmaker.org/chainmaker/txpool-batch/v2 v2.0.0-20211013035004-190b3acb8eb8
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210916084713-abd13154c26b
	chainmaker.org/chainmaker/vm v0.0.0-20211009033650-5123c6160898
	chainmaker.org/chainmaker/vm-evm v0.0.0-20210916091920-b915815eb88b
	chainmaker.org/chainmaker/vm-gasm v0.0.0-20211011113052-073922cd3e28
	chainmaker.org/chainmaker/vm-wasmer v0.0.0-20211011030923-6b5440e1e9f7
	chainmaker.org/chainmaker/vm-wxvm v0.0.0-20210918101823-dce1c76fb189
	github.com/fatih/color v1.13.0 // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20210113012101-fb4e108d2519 // indirect
	github.com/mattn/go-colorable v0.1.11 // indirect
	github.com/mitchellh/mapstructure v1.4.2
	github.com/spf13/cobra v1.2.1 // indirect
	github.com/spf13/viper v1.9.0 // indirect
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	chainmaker.org/chainmaker-go/consensus => ../consensus
	chainmaker.org/chainmaker-go/core => ../core
	chainmaker.org/chainmaker-go/net => ../net
	chainmaker.org/chainmaker-go/snapshot => ../snapshot
	chainmaker.org/chainmaker-go/subscriber => ../subscriber
	chainmaker.org/chainmaker-go/sync => ../sync
	chainmaker.org/chainmaker-go/txpool => ../txpool
	github.com/libp2p/go-libp2p => ../net/p2p/libp2p
	github.com/libp2p/go-libp2p-core => ../net/p2p/libp2pcore
	github.com/libp2p/go-libp2p-pubsub => ../net/p2p/libp2ppubsub
	google.golang.org/grpc v1.40.0 => google.golang.org/grpc v1.26.0
)
