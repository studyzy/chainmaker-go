module chainmaker.org/chainmaker-go/chainconf

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0-00010101000000-000000000000
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210531062058-beb572d07e38
	chainmaker.org/chainmaker/pb-go v0.0.0-20210531071221-ccada476876b
	chainmaker.org/chainmaker/protocol v0.0.0-20210531063313-99b65ccc50e6
	github.com/gogo/protobuf v1.3.2
	github.com/golang/groupcache v0.0.0-20191227052852-215e87163ea7
	github.com/golang/protobuf v1.4.3
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	google.golang.org/grpc/examples v0.0.0-20210519181852-3dd75a6888ce // indirect
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../localconf
	chainmaker.org/chainmaker-go/logger => ./../../logger
	chainmaker.org/chainmaker-go/utils => ../../utils
)
