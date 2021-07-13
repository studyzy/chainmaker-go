module chainmaker.org/chainmaker-go/wasmer

go 1.15

require (

	chainmaker.org/chainmaker-go/logger v0.0.0


	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker-go/wasi v0.0.0
	chainmaker.org/chainmaker-go/store v0.0.0
)

replace (

	chainmaker.org/chainmaker-go/logger => ../../logger


	chainmaker.org/chainmaker-go/utils => ../../utils
	chainmaker.org/chainmaker-go/wasi => ../wasi
	chainmaker.org/chainmaker-go/store => ../../store
	chainmaker.org/chainmaker-go/localconf => ../../conf/localconf
)
