module chainmaker.org/chainmaker-go/snapshot

go 1.15

require (

	chainmaker.org/chainmaker-go/logger v0.0.0

	chainmaker.org/chainmaker-go/protocol v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	github.com/stretchr/testify v1.6.1
)

replace (

	chainmaker.org/chainmaker-go/logger => ../logger

	chainmaker.org/chainmaker-go/protocol => ../../protocol
	chainmaker.org/chainmaker-go/utils => ../utils
)
