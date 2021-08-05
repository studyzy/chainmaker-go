module chainmaker.org/chainmaker-go/accesscontrol

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210730091111-cd7813a002c3
	chainmaker.org/chainmaker/pb-go v0.0.0-20210803102358-b7047f9f665d
	chainmaker.org/chainmaker/protocol v0.0.0-20210730061517-ec081b24f778
	github.com/gogo/protobuf v1.3.2
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

	chainmaker.org/chainmaker-go/utils => ../utils
)
