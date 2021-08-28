module chainmaker.org/chainmaker-go/accesscontrol

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210828064653-da1cfc1db5ea
	chainmaker.org/chainmaker/pb-go v0.0.0-20210826130850-b78ed618ce07
	chainmaker.org/chainmaker/protocol v1.2.3-0.20210828065550-3d6fac33d331
	github.com/gogo/protobuf v1.3.2
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

	chainmaker.org/chainmaker-go/utils => ../utils
)
