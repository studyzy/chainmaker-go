module chainmaker.org/chainmaker-go/wasi

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/store v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210818084533-a9eaa4199add
	chainmaker.org/chainmaker/pb-go v0.0.0-20210820090923-daeaf929a7c0
	chainmaker.org/chainmaker/protocol v0.0.0-20210820091045-f54164dfaf0e
	github.com/golang/protobuf v1.4.3 // indirect
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/store => ../../store
	chainmaker.org/chainmaker-go/utils => ../../utils
)
