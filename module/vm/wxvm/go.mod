module chainmaker.org/chainmaker-go/wxvm

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210825071035-c1f0524e591e
	chainmaker.org/chainmaker/pb-go v0.0.0-20210825133553-b1953ac0acac
	chainmaker.org/chainmaker/protocol v0.0.0-20210825021221-02ac5d5a967e
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
