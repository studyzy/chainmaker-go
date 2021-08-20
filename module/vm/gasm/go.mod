module chainmaker.org/chainmaker-go/gasm

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/wasi v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210818084533-a9eaa4199add
	chainmaker.org/chainmaker/pb-go v0.0.0-20210819081938-1bd6048af1d7
	chainmaker.org/chainmaker/protocol v0.0.0-20210810081254-4947fb9a5306
	github.com/golang/groupcache v0.0.0-20191227052852-215e87163ea7
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../logger

	chainmaker.org/chainmaker-go/store => ../../store
	chainmaker.org/chainmaker-go/utils => ../../utils
	chainmaker.org/chainmaker-go/wasi => ../wasi
)
