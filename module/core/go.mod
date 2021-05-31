module chainmaker.org/chainmaker-go/core

go 1.15

require (
	chainmaker.org/chainmaker-go/chainconf v0.0.0
	chainmaker.org/chainmaker-go/consensus v0.0.0
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/monitor v0.0.0
	chainmaker.org/chainmaker-go/store v0.0.0
	chainmaker.org/chainmaker-go/subscriber v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210531130733-8190437182e6
	chainmaker.org/chainmaker/pb-go v0.0.0-20210531071221-ccada476876b
	chainmaker.org/chainmaker/protocol v0.0.0-20210531063313-99b65ccc50e6
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.5.0
	github.com/google/martian v2.1.0+incompatible
	github.com/panjf2000/ants/v2 v2.4.3
	github.com/prometheus/client_golang v1.9.0
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	chainmaker.org/chainmaker-go/chainconf => ./../conf/chainconf

	chainmaker.org/chainmaker-go/consensus => ../consensus
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger
	chainmaker.org/chainmaker-go/monitor => ../monitor

	chainmaker.org/chainmaker-go/store => ../store
	chainmaker.org/chainmaker-go/subscriber => ../subscriber
	chainmaker.org/chainmaker-go/utils => ../utils
)
