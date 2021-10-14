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
	chainmaker.org/chainmaker-go/vm v0.0.0
	chainmaker.org/chainmaker/chainconf/v2 v2.0.0-20211014142031-0a2a2aac316c
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20211014122130-4ba9d85a64f8
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20211014134424-9431ffcc5bbc
	chainmaker.org/chainmaker/logger/v2 v2.0.1-0.20211014131951-892d098049bc
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211014120010-525e2ffaf04d
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20211014144951-97323532a236
	chainmaker.org/chainmaker/store/v2 v2.0.0-20211014154101-7b199f0df636
	chainmaker.org/chainmaker/txpool-batch/v2 v2.0.0-20211014143342-e04da749db9b
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20211014131421-43de8d9fe869
	chainmaker.org/chainmaker/vm v0.0.0-20211014150836-d6eae08ad3bd
	chainmaker.org/chainmaker/vm-evm v0.0.0-20211014155012-e69085fedd2f
	chainmaker.org/chainmaker/vm-gasm v0.0.0-20211014154622-4c1c925eacdd
	chainmaker.org/chainmaker/vm-native v0.0.0-20211014145655-91f98409dbbd // indirect
	chainmaker.org/chainmaker/vm-wasmer v0.0.0-20211014154818-0f4ed6551187
	chainmaker.org/chainmaker/vm-wxvm v0.0.0-20211014155330-6a66e3935c65
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
	chainmaker.org/chainmaker-go/vm => ../vm
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
	google.golang.org/grpc v1.40.0 => google.golang.org/grpc v1.26.0
)
