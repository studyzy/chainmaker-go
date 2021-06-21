module chainmaker.org/chainmaker-go/sync

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210609023657-282d880dd032
	chainmaker.org/chainmaker/pb-go v0.0.0-20210608121517-b3fe5e4784c1
	chainmaker.org/chainmaker/protocol v0.0.0-20210609024825-0db378505f4e
	github.com/Workiva/go-datastructures v1.0.52
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.4.2
	github.com/stretchr/testify v1.6.1
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

)
