module chainmaker.org/chainmaker-go/evm

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210818084533-a9eaa4199add
	chainmaker.org/chainmaker/pb-go v0.0.0-20210820090923-daeaf929a7c0
	chainmaker.org/chainmaker/protocol v0.0.0-20210820091045-f54164dfaf0e
	github.com/ethereum/go-ethereum v1.10.3
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	gotest.tools v2.2.0+incompatible

)

replace (
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/utils => ../../utils
)
