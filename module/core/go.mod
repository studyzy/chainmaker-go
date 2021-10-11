module chainmaker.org/chainmaker-go/core

go 1.15

require (
	chainmaker.org/chainmaker-go/consensus v0.0.0
	chainmaker.org/chainmaker-go/monitor v0.0.0
	chainmaker.org/chainmaker-go/subscriber v0.0.0
	chainmaker.org/chainmaker/chainconf/v2 v2.0.0-20211011064002-0be7c9f79898
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20211009071412-cec96328e5d6
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20210928020228-3ab2986d5ecd
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907134457-53647922a89d
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211009063815-ca059c45bcf3
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20211009064056-03cbf6096208
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20211011062420-24df29961fa2
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
