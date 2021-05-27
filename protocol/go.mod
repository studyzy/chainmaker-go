module chainmaker.org/chainmaker-go/protocol

go 1.15

require (
	chainmaker.org/chainmaker-go/common v0.0.0
	chainmaker.org/chainmaker-go/pb/protogo v0.0.0
	github.com/syndtr/goleveldb v1.0.0
)

replace (
	chainmaker.org/chainmaker-go/common => ../common
	chainmaker.org/chainmaker-go/pb/protogo => ../pb/protogo
)
