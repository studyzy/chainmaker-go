module chainmaker.org/chainmaker-go/wxvm

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210901114756-9114511c2b70
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210901132412-435b75070bf2
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210901134008-4b83cf573272
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/pingcap/errors v0.11.5-0.20201029093017-5a7df2af2ac7 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gopkg.in/yaml.v2 v2.3.0 // indirect
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../logger

	chainmaker.org/chainmaker-go/store => ../../store
	chainmaker.org/chainmaker-go/utils => ../../utils
	chainmaker.org/chainmaker-go/wasi => ../wasi
)
