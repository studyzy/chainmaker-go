module chainmaker.org/chainmaker-go/wasmer

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker-go/wasi v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210804033713-22bae864e5c4
	chainmaker.org/chainmaker/pb-go v0.0.0-20210802022101-dc6daf93f738
	chainmaker.org/chainmaker/protocol v0.0.0-20210804070206-a6b73cec5918
)

replace (
	chainmaker.org/chainmaker-go/localconf => ../../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/store => ../../store

	chainmaker.org/chainmaker-go/utils => ../../utils
	chainmaker.org/chainmaker-go/wasi => ../wasi
)
