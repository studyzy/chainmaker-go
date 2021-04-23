module chainmaker.org/chainmaker-go/utils

go 1.15

require (
	chainmaker.org/chainmaker-go/common v0.0.0
	chainmaker.org/chainmaker-go/pb/protogo v0.0.0
	chainmaker.org/chainmaker-go/protocol v0.0.0
	github.com/gogo/protobuf v1.3.2
	github.com/pingcap/parser v0.0.0-20200623164729-3a18f1e5dceb // indirect
	github.com/pingcap/tidb v1.1.0-beta.0.20200630082100-328b6d0a955c // indirect
	github.com/stretchr/testify v1.6.1
)

replace (
	chainmaker.org/chainmaker-go/common => ../../common
	chainmaker.org/chainmaker-go/pb/protogo => ../../pb/protogo
	chainmaker.org/chainmaker-go/protocol => ../../protocol
)
