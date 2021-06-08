module chainmaker.org/chainmaker-go/snapshot

go 1.15

require (

	chainmaker.org/chainmaker-go/logger v0.0.0


	chainmaker.org/chainmaker-go/utils v0.0.0
	github.com/pingcap/parser v0.0.0-20200623164729-3a18f1e5dceb // indirect
	github.com/stretchr/testify v1.7.0
)

replace (

	chainmaker.org/chainmaker-go/logger => ../logger


	chainmaker.org/chainmaker-go/utils => ../utils
)
