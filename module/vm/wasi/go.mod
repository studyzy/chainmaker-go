module chainmaker.org/chainmaker-go/wasi

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0

	chainmaker.org/chainmaker-go/protocol v0.0.0
	chainmaker.org/chainmaker-go/store v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.4.3
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
)

replace (

	chainmaker.org/chainmaker-go/localconf => ./../../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/pb/protogo => ../../../pb/protogo
	chainmaker.org/chainmaker-go/protocol => ../../../protocol
	chainmaker.org/chainmaker-go/store => ../../store
	chainmaker.org/chainmaker-go/utils => ../../utils
)
