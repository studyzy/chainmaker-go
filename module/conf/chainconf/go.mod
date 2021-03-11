module chainmaker.org/chainmaker-go/chainconf

go 1.15

require (
	chainmaker.org/chainmaker-go/common v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/pb/protogo v0.0.0
	chainmaker.org/chainmaker-go/protocol v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6
	github.com/golang/protobuf v1.4.3
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
)

replace (
	chainmaker.org/chainmaker-go/common => ./../../../common
	chainmaker.org/chainmaker-go/logger => ./../../logger
	chainmaker.org/chainmaker-go/pb/protogo => ./../../../pb/protogo
	chainmaker.org/chainmaker-go/protocol => ./../../../protocol
	chainmaker.org/chainmaker-go/utils => ../../utils
)
