module chainmaker.org/chainmaker-go/evm

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210825071035-c1f0524e591e
	chainmaker.org/chainmaker/pb-go v0.0.0-20210825133553-b1953ac0acac
	chainmaker.org/chainmaker/protocol v0.0.0-20210825021221-02ac5d5a967e
	github.com/ethereum/go-ethereum v1.10.3
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	gotest.tools v2.2.0+incompatible

)

replace (
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/utils => ../../utils
)
