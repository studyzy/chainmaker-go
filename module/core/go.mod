module chainmaker.org/chainmaker-go/core

go 1.15

require (
	chainmaker.org/chainmaker-go/consensus v0.0.0
	chainmaker.org/chainmaker-go/subscriber v0.0.0
	chainmaker.org/chainmaker/chainconf/v2 v2.0.0-20211014142031-0a2a2aac316c
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20211014122130-4ba9d85a64f8
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20211014134424-9431ffcc5bbc
	chainmaker.org/chainmaker/logger/v2 v2.0.1-0.20211014131951-892d098049bc
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211014120010-525e2ffaf04d
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20211014144951-97323532a236
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20211014131421-43de8d9fe869
	chainmaker.org/chainmaker/vm v0.0.0-20211014150836-d6eae08ad3bd
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
