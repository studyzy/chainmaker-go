module chainmaker.org/chainmaker-go/evm

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210811075857-d3b57d983071
	chainmaker.org/chainmaker/pb-go v0.0.0-20210809091134-f6303e12573d
	chainmaker.org/chainmaker/protocol v0.0.0-20210810081254-4947fb9a5306
	github.com/ethereum/go-ethereum v1.10.3
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2

)

replace (
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/utils => ../../utils
)
