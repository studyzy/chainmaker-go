module chainmaker.org/chainmaker-go/dpos

go 1.15

require (
	chainmaker.org/chainmaker-go/pb/protogo v0.0.0
	chainmaker.org/chainmaker-go/protocol v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker-go/vm v0.0.0
	github.com/golang/protobuf v1.4.3
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	chainmaker.org/chainmaker-go/chainconf => ../conf/chainconf
	chainmaker.org/chainmaker-go/common => ../../common
	chainmaker.org/chainmaker-go/evm => ../vm/evm
	chainmaker.org/chainmaker-go/gasm => ../vm/gasm
	chainmaker.org/chainmaker-go/localconf => ../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger
	chainmaker.org/chainmaker-go/pb/protogo => ../../pb/protogo
	chainmaker.org/chainmaker-go/protocol => ../../protocol
	chainmaker.org/chainmaker-go/store => ../store
	chainmaker.org/chainmaker-go/utils => ../utils
	chainmaker.org/chainmaker-go/vm => ../vm
	chainmaker.org/chainmaker-go/vm/native => ../vm/native
	chainmaker.org/chainmaker-go/wasi => ../vm/wasi
	chainmaker.org/chainmaker-go/wasmer => ../vm/wasmer
	chainmaker.org/chainmaker-go/wxvm => ../vm/wxvm
)
