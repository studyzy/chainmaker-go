module chainmaker.org/chainmaker-go

go 1.15

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0
	chainmaker.org/chainmaker-go/blockchain v0.0.0
	chainmaker.org/chainmaker-go/net v0.0.0
	chainmaker.org/chainmaker-go/rpcserver v0.0.0
	chainmaker.org/chainmaker-go/txpool v0.0.0
	chainmaker.org/chainmaker-go/vm v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.1.1-0.20211108032042-88ed7467a15c
	chainmaker.org/chainmaker/localconf/v2 v2.1.0
	chainmaker.org/chainmaker/logger/v2 v2.1.0
	chainmaker.org/chainmaker/pb-go/v2 v2.1.0
	chainmaker.org/chainmaker/protocol/v2 v2.1.1-0.20211129092325-fd10970cbf25
	chainmaker.org/chainmaker/sdk-go/v2 v2.1.0
	chainmaker.org/chainmaker/txpool-batch/v2 v2.1.0
	chainmaker.org/chainmaker/txpool-single/v2 v2.1.0
	chainmaker.org/chainmaker/utils/v2 v2.1.0
	chainmaker.org/chainmaker/vm-docker-go v0.0.0-20211125021937-fa215c973c9f
	chainmaker.org/chainmaker/vm-evm/v2 v2.1.1-0.20211129130757-c04c939844ba
	chainmaker.org/chainmaker/vm-gasm/v2 v2.1.1-0.20211129130918-a4eae896c132
	chainmaker.org/chainmaker/vm-wasmer/v2 v2.1.1-0.20211129130826-b242ad73cae2
	chainmaker.org/chainmaker/vm-wxvm/v2 v2.1.1-0.20211129093651-d06e02ed593e
	code.cloudfoundry.org/bytefmt v0.0.0-20200131002437-cf55d5288a48
	github.com/common-nighthawk/go-figure v0.0.0-20200609044655-c4b36f998cf2
	github.com/ethereum/go-ethereum v1.10.4
	github.com/gogo/protobuf v1.3.2
	github.com/mr-tron/base58 v1.2.0
	github.com/prometheus/client_golang v1.11.0
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	google.golang.org/grpc v1.41.0
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
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v1.0.0
	google.golang.org/grpc v1.40.0 => google.golang.org/grpc v1.26.0
)
