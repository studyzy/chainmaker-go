module chainmaker.org/chainmaker-go/monitor

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907133316-af00cea33c97
	github.com/prometheus/client_golang v1.9.0
)

replace (

	chainmaker.org/chainmaker-go/localconf => ../conf/localconf


)
