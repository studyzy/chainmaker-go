module chainmaker.org/chainmaker-go/provider

go 1.15

require (
	chainmaker.org/chainmaker-go/common v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/protocol v0.0.0
	chainmaker.org/chainmaker-go/subscriber v0.0.0-00010101000000-000000000000
)

replace (
	chainmaker.org/chainmaker-go/common => ./../../common
	chainmaker.org/chainmaker-go/logger => ../logger
	chainmaker.org/chainmaker-go/pb/protogo => ../../pb/protogo
	chainmaker.org/chainmaker-go/protocol => ./../../protocol
	chainmaker.org/chainmaker-go/subscriber => ../subscriber

)
