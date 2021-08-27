module chainmaker.org/chainmaker-go/wasi

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/store v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210825071035-c1f0524e591e
	chainmaker.org/chainmaker/pb-go v0.0.0-20210826130850-b78ed618ce07
	chainmaker.org/chainmaker/protocol v0.0.0-20210825021221-02ac5d5a967e
	github.com/golang/protobuf v1.4.3 // indirect
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/store => ../../store
	chainmaker.org/chainmaker-go/utils => ../../utils
)
