module chainmaker.org/chainmaker-go/test/send_proposal_request_tool

go 1.15

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210819015845-c6b778b6349a
	chainmaker.org/chainmaker/pb-go v0.0.0-20210823032707-b3e96f797849
	chainmaker.org/chainmaker/protocol v0.0.0-20210825021221-02ac5d5a967e
	github.com/Rican7/retry v0.1.0
	github.com/ethereum/go-ethereum v1.10.2
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.4.3
	github.com/mr-tron/base58 v1.2.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/pretty v1.2.0
	github.com/tjfoc/gmtls v1.2.1 // indirect
	google.golang.org/genproto v0.0.0-20200806141610-86f49bd18e98 // indirect
	google.golang.org/grpc v1.37.0
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../../module/accesscontrol

	chainmaker.org/chainmaker-go/localconf => ../../module/conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../module/logger

	chainmaker.org/chainmaker-go/utils => ../../module/utils
)
