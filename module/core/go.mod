module chainmaker.org/chainmaker-go/core

go 1.15

require (
	chainmaker.org/chainmaker-go/consensus v0.0.0
	chainmaker.org/chainmaker-go/monitor v0.0.0
	chainmaker.org/chainmaker-go/subscriber v0.0.0
	chainmaker.org/chainmaker/chainconf/v2 v2.0.0-20210913144615-f27c44059848
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210928022522-120cf16c8354
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20210914062957-13e84972a921
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907134457-53647922a89d
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211009072509-e7d0967e05e8
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210927062046-68813f263c0b
	chainmaker.org/chainmaker/store/v2 v2.0.0-20210930081040-bd152dcb0224 // indirect
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210916084713-abd13154c26b
	chainmaker.org/chainmaker/vm v0.0.0-20210918104424-239140ec3366
	github.com/gogo/protobuf v1.3.2
	github.com/google/martian v2.1.0+incompatible
	github.com/panjf2000/ants/v2 v2.4.3
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.26.0
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	chainmaker.org/chainmaker-go/consensus => ../consensus
	chainmaker.org/chainmaker-go/consensus/dpos => ./../consensus/dpos

	chainmaker.org/chainmaker-go/monitor => ../monitor
	chainmaker.org/chainmaker-go/subscriber => ../subscriber
)
