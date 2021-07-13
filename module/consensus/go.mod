module chainmaker.org/chainmaker-go/consensus

go 1.15

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0
	chainmaker.org/chainmaker-go/chainconf v0.0.0
	chainmaker.org/chainmaker-go/dpos v0.0.0
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210709154839-e2c8e4fc62b4
	chainmaker.org/chainmaker/pb-go v0.0.0-20210713015752-33fec271c90a
	chainmaker.org/chainmaker/protocol v0.0.0-20210713021825-63c58dd0297f
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/kr/pretty v0.2.0 // indirect
	github.com/prometheus/client_golang v1.9.0 // indirect
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/thoas/go-funk v0.8.0
	github.com/tidwall/wal v0.1.4
	go.etcd.io/etcd/client/pkg/v3 v3.5.0-beta.4
	go.etcd.io/etcd/raft/v3 v3.5.0-beta.4
	go.etcd.io/etcd/server/v3 v3.5.0-beta.4
	go.uber.org/zap v1.16.1-0.20210329175301-c23abee72d19
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ./../../module/accesscontrol
	chainmaker.org/chainmaker-go/chainconf => ./../conf/chainconf

	chainmaker.org/chainmaker-go/dpos => ./../../module/dpos
	chainmaker.org/chainmaker-go/evm => ./../../module/vm/evm
	chainmaker.org/chainmaker-go/gasm => ./../../module/vm/gasm
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ./../logger

	chainmaker.org/chainmaker-go/store => ./../../module/store
	chainmaker.org/chainmaker-go/utils => ../utils
	chainmaker.org/chainmaker-go/vm => ./../../module/vm
	chainmaker.org/chainmaker-go/wasi => ./../../module/vm/wasi
	chainmaker.org/chainmaker-go/wasmer => ./../../module/vm/wasmer
	chainmaker.org/chainmaker-go/wxvm => ./../../module/vm/wxvm
	github.com/libp2p/go-libp2p-core => ../net/p2p/libp2pcore
)
