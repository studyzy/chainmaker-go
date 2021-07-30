module chainmaker.org/chainmaker-go/test/subscribe_test_tool

go 1.16

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210722032200-380ced605d25
	chainmaker.org/chainmaker/pb-go v0.0.0-20210723070658-764cafbc33fe
	chainmaker.org/chainmaker/protocol v0.0.0-20210727101110-59285b10f1ef
	github.com/gogo/protobuf v1.3.2
	github.com/spf13/cobra v1.2.1
	google.golang.org/grpc v1.38.0

)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../../module/accesscontrol
	chainmaker.org/chainmaker-go/localconf => ../../module/conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../module/logger
	chainmaker.org/chainmaker-go/utils => ../../module/utils
)
