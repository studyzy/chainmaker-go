module chainmaker.org/chainmaker-go/txpool/batchtxpool

go 1.15

require (
	chainmaker.org/chainmaker-go/common v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/pb/protogo v0.0.0
	chainmaker.org/chainmaker-go/protocol v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0-00010101000000-000000000000
	github.com/gogo/protobuf v1.3.2
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.6.1
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)

replace (
	chainmaker.org/chainmaker-go/common => ../../../common
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/pb/protogo => ../../../pb/protogo
	chainmaker.org/chainmaker-go/protocol => ../../../protocol
	chainmaker.org/chainmaker-go/utils => ../../utils
)
