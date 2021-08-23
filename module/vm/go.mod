module chainmaker.org/chainmaker-go/vm

go 1.15

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0
	chainmaker.org/chainmaker-go/chainconf v0.0.0
	chainmaker.org/chainmaker-go/evm v0.0.0
	chainmaker.org/chainmaker-go/gasm v0.0.0
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker-go/wasmer v0.0.0
	chainmaker.org/chainmaker-go/wxvm v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210819015845-c6b778b6349a
	chainmaker.org/chainmaker/pb-go v0.0.0-20210823032707-b3e96f797849
	chainmaker.org/chainmaker/protocol v0.0.0-20210823033144-bcf0422b11ea
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.4.3
	github.com/google/uuid v1.1.5
	github.com/mr-tron/base58 v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	gotest.tools v2.2.0+incompatible
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	chainmaker.org/chainmaker-go/chainconf => ../conf/chainconf
	chainmaker.org/chainmaker-go/evm => ./evm
	chainmaker.org/chainmaker-go/gasm => ./gasm
	chainmaker.org/chainmaker-go/localconf => ../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger
	chainmaker.org/chainmaker-go/store => ../store
	chainmaker.org/chainmaker-go/utils => ../utils
	chainmaker.org/chainmaker-go/wasi => ./wasi
	chainmaker.org/chainmaker-go/wasmer => ./wasmer
	chainmaker.org/chainmaker-go/wxvm => ./wxvm
)
