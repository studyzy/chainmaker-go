module chainmaker.org/chainmaker-go/evm

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210819015845-c6b778b6349a
	chainmaker.org/chainmaker/pb-go v0.0.0-20210823032707-b3e96f797849
	chainmaker.org/chainmaker/protocol v0.0.0-20210823033144-bcf0422b11ea
	github.com/ethereum/go-ethereum v1.10.3
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	gotest.tools v2.2.0+incompatible

)

replace (
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/utils => ../../utils
)
