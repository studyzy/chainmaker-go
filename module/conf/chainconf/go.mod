module chainmaker.org/chainmaker-go/chainconf

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0-00010101000000-000000000000
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210811075857-d3b57d983071
	chainmaker.org/chainmaker/pb-go v0.0.0-20210809091134-f6303e12573d
	chainmaker.org/chainmaker/protocol v0.0.0-20210810081254-4947fb9a5306
	github.com/gogo/protobuf v1.3.2
	github.com/golang/groupcache v0.0.0-20191227052852-215e87163ea7
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../localconf
	chainmaker.org/chainmaker-go/logger => ./../../logger

	chainmaker.org/chainmaker-go/utils => ../../utils
)
