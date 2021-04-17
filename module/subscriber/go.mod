module chainmaker.org/chainmaker-go/subscriber

go 1.15

require (
	chainmaker.org/chainmaker-go/common v0.0.0
	chainmaker.org/chainmaker-go/pb/protogo v0.0.0
	github.com/aristanetworks/goarista v0.0.0-20201012165903-2cb20defcd66 // indirect
	github.com/ethereum/go-ethereum v1.9.25
	chainmaker.org/chainmaker-go/logger v0.0.0
)

replace (
	chainmaker.org/chainmaker-go/common => ../../common
	chainmaker.org/chainmaker-go/pb/protogo => ../../pb/protogo
	chainmaker.org/chainmaker-go/logger => ../../logger
)
