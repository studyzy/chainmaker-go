module chainmaker.org/chainmaker-go/core

go 1.15

require (
	chainmaker.org/chainmaker-go/consensus v0.0.0
	chainmaker.org/chainmaker-go/subscriber v0.0.0
	chainmaker.org/chainmaker/chainconf/v2 v2.0.0-20211022083213-24febecb45ee
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20211020133403-66958ba788b8
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20211014134424-9431ffcc5bbc
	chainmaker.org/chainmaker/logger/v2 v2.0.1-0.20211015125919-8e5199930ac9
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211021024710-9329804d1c21
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20211022113918-f11dc73904c1
	chainmaker.org/chainmaker/txpool-batch/v2 v2.0.0-20211019074609-46e3d29f0908
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20211020061817-1e64a279af4f
	chainmaker.org/chainmaker/vm v0.0.0-20211022085604-9ff0d4318eff
	github.com/ethereum/go-ethereum v1.10.3 // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/panjf2000/ants/v2 v2.4.3
	github.com/prometheus/client_golang v1.11.0
	github.com/stretchr/testify v1.7.0
	google.golang.org/grpc v1.40.0 // indirect
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	chainmaker.org/chainmaker-go/consensus => ../consensus
	chainmaker.org/chainmaker-go/consensus/dpos => ./../consensus/dpos

	chainmaker.org/chainmaker-go/monitor => ../monitor
	chainmaker.org/chainmaker-go/subscriber => ../subscriber
)
