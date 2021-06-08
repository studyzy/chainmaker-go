module chainmaker.org/chainmaker-go/vm

go 1.15

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0
	chainmaker.org/chainmaker-go/chainconf v0.0.0
	chainmaker.org/chainmaker-go/common v0.0.0
	chainmaker.org/chainmaker-go/evm v0.0.0
	chainmaker.org/chainmaker-go/gasm v0.0.0
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/pb/protogo v0.0.0
	chainmaker.org/chainmaker-go/protocol v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker-go/wasi v0.0.0
	chainmaker.org/chainmaker-go/wasmer v0.0.0
	chainmaker.org/chainmaker-go/wxvm v0.0.0
	github.com/ethereum/go-ethereum v1.9.25
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/mr-tron/base58 v1.2.0
	github.com/shopspring/decimal v1.2.0
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca // indirect
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	chainmaker.org/chainmaker-go/chainconf => ../conf/chainconf
	chainmaker.org/chainmaker-go/common => ../../common
	chainmaker.org/chainmaker-go/evm => ./evm
	chainmaker.org/chainmaker-go/gasm => ./gasm
	chainmaker.org/chainmaker-go/localconf => ../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger
	chainmaker.org/chainmaker-go/pb/protogo => ../../pb/protogo
	chainmaker.org/chainmaker-go/protocol => ../../protocol
	chainmaker.org/chainmaker-go/store => ../store
	chainmaker.org/chainmaker-go/utils => ../utils
	chainmaker.org/chainmaker-go/wasi => ./wasi
	chainmaker.org/chainmaker-go/wasmer => ./wasmer
	chainmaker.org/chainmaker-go/wxvm => ./wxvm
)
