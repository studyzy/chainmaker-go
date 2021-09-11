module chainmaker.org/chainmaker-go/gasm

go 1.15

require (
	chainmaker.org/chainmaker-go/wasi v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210909033927-2a4cfc146579
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907133316-af00cea33c97
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210901132412-435b75070bf2
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210910112253-04256ae9c5ed
	github.com/golang/groupcache v0.0.0-20191227052852-215e87163ea7
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../logger

	chainmaker.org/chainmaker-go/store => ../../store

	chainmaker.org/chainmaker-go/wasi => ../wasi
)
