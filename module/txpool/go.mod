module chainmaker.org/chainmaker-go/txpool

go 1.15

require (
	chainmaker.org/chainmaker-go/chainconf v0.0.0
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/monitor v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210609023657-282d880dd032
	chainmaker.org/chainmaker/pb-go v0.0.0-20210621034028-d765d0e95b61
	chainmaker.org/chainmaker/protocol v0.0.0-20210609024825-0db378505f4e
	github.com/gogo/protobuf v1.3.2
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
