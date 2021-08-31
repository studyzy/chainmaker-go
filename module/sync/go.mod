module chainmaker.org/chainmaker-go/sync

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210831112613-754cd525d627
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210831160840-24d6ff2d780c
	chainmaker.org/chainmaker/protocol/v2 v2.0.0-20210831114940-7b97fb540200
	github.com/Workiva/go-datastructures v1.0.52
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.4.2
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

)
