module chainmaker.org/chainmaker-go/dpos

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/store v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker-go/vm v0.0.0
	chainmaker.org/chainmaker/pb-go v0.0.0-20210630123814-27dfc86893cf
	chainmaker.org/chainmaker/protocol v0.0.0-20210630150922-63cd365c6511
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.4.3
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	chainmaker.org/chainmaker-go/chainconf => ../conf/chainconf

	chainmaker.org/chainmaker-go/evm => ../vm/evm
	chainmaker.org/chainmaker-go/gasm => ../vm/gasm
	chainmaker.org/chainmaker-go/localconf => ../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

	chainmaker.org/chainmaker-go/store => ../store
	chainmaker.org/chainmaker-go/utils => ../utils
	chainmaker.org/chainmaker-go/vm => ../vm
	chainmaker.org/chainmaker-go/vm/native => ../vm/native
	chainmaker.org/chainmaker-go/wasi => ../vm/wasi
	chainmaker.org/chainmaker-go/wasmer => ../vm/wasmer
	chainmaker.org/chainmaker-go/wxvm => ../vm/wxvm
)
