module chainmaker.org/chainmaker-go/wasmer

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker-go/wasi v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210825071035-c1f0524e591e
	chainmaker.org/chainmaker/pb-go v0.0.0-20210825133553-b1953ac0acac
	chainmaker.org/chainmaker/protocol v0.0.0-20210825021221-02ac5d5a967e
)

replace (
	chainmaker.org/chainmaker-go/localconf => ../../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/store => ../../store

	chainmaker.org/chainmaker-go/utils => ../../utils
	chainmaker.org/chainmaker-go/wasi => ../wasi
)
