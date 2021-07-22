module chainmaker.org/chainmaker-go/accesscontrol

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210719075615-247a2254ad3d
	chainmaker.org/chainmaker/pb-go v0.0.0-20210721063744-9464c7cbe8a9
	chainmaker.org/chainmaker/protocol v0.0.0-20210721064218-27953860b84b
	github.com/gogo/protobuf v1.3.2
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

	chainmaker.org/chainmaker-go/utils => ../utils
)
