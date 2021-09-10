module chainmaker.org/chainmaker-go/wasmer

go 1.15

require (
	chainmaker.org/chainmaker-go/wasi v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210909033927-2a4cfc146579
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907133316-af00cea33c97
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210901132412-435b75070bf2
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210910074933-23793f07a247
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
)

replace (
	chainmaker.org/chainmaker-go/localconf => ../../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/store => ../../store

	chainmaker.org/chainmaker-go/wasi => ../wasi
)
