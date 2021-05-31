module chainmaker.org/chainmaker-go/test/send_proposal_request_tool

go 1.15

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0

	chainmaker.org/chainmaker-go/logger v0.0.0

	chainmaker.org/chainmaker-go/protocol v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	github.com/Rican7/retry v0.1.0
	github.com/ethereum/go-ethereum v1.10.2
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.4.3
	github.com/samkumar/hibe v0.0.0-20171013061409-c1cd171b6178
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.0
	google.golang.org/grpc v1.37.0
	google.golang.org/grpc/examples v0.0.0-20210519181852-3dd75a6888ce // indirect
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../../module/accesscontrol

	chainmaker.org/chainmaker-go/localconf => ../../module/conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../module/logger

	chainmaker.org/chainmaker-go/protocol => ../../protocol
	chainmaker.org/chainmaker-go/utils => ../../module/utils
)
