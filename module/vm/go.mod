module chainmaker.org/chainmaker-go/vm

go 1.15

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0
	chainmaker.org/chainmaker-go/chainconf v0.0.0
	chainmaker.org/chainmaker-go/evm v0.0.0
	chainmaker.org/chainmaker-go/gasm v0.0.0
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/wasmer v0.0.0
	chainmaker.org/chainmaker-go/wxvm v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210909033927-2a4cfc146579
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907133316-af00cea33c97
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210901132412-435b75070bf2
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210910112253-04256ae9c5ed
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
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

	chainmaker.org/chainmaker-go/store => ../store

	chainmaker.org/chainmaker-go/wasi => ./wasi
	chainmaker.org/chainmaker-go/wasmer => ./wasmer
	chainmaker.org/chainmaker-go/wxvm => ./wxvm
)
