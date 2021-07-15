module chainmaker.org/chainmaker-go/txpool

go 1.15

require (
	chainmaker.org/chainmaker-go/chainconf v0.0.0
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/monitor v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210714055243-e02c9a0323b2
	chainmaker.org/chainmaker/pb-go v0.0.0-20210714113815-1228e0322111
	chainmaker.org/chainmaker/protocol v0.0.0-20210714073836-8ec1557557b0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/prometheus/client_golang v1.9.0
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/chainconf => ./../conf/chainconf

	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

	chainmaker.org/chainmaker-go/monitor => ../monitor

	chainmaker.org/chainmaker-go/utils => ../utils
)
