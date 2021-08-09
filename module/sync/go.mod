module chainmaker.org/chainmaker-go/sync

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210804033713-22bae864e5c4
	chainmaker.org/chainmaker/pb-go v0.0.0-20210809091134-f6303e12573d
	chainmaker.org/chainmaker/protocol v0.0.0-20210809025435-1ca089468862
	github.com/Workiva/go-datastructures v1.0.52
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.4.2
	github.com/stretchr/testify v1.6.1
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

)
