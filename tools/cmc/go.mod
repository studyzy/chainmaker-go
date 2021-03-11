module chainmaker.org/chainmaker-go/tools/cmc

go 1.15

require (
	chainmaker.org/chainmaker-go/common v0.0.0
	chainmaker.org/chainmaker-sdk-go v0.0.0
	github.com/gogo/protobuf v1.3.2
	github.com/hokaccha/go-prettyjson v0.0.0-20201222001619-a42f9ac2ec8e
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	google.golang.org/grpc v1.36.0
)

replace (
	chainmaker.org/chainmaker-go/common => ../../common
	chainmaker.org/chainmaker-sdk-go => ../sdk
)
