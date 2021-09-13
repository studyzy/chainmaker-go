module chainmaker.org/chainmaker-go/txpool

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/monitor v0.0.0
	chainmaker.org/chainmaker/chainconf/v2 v2.0.0-20210913144615-f27c44059848
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210909033927-2a4cfc146579
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907133316-af00cea33c97
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210901132412-435b75070bf2
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210910112253-04256ae9c5ed
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/prometheus/client_golang v1.9.0
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/chainconf => ./../conf/chainconf

	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf

	chainmaker.org/chainmaker-go/monitor => ../monitor

)
