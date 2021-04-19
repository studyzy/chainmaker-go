module chainmaker.org/chainmaker-go/consensus

go 1.15

require (
	chainmaker.org/chainmaker-go/chainconf v0.0.0-00010101000000-000000000000
	chainmaker.org/chainmaker-go/common v0.0.0
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/mock v0.0.0-00010101000000-000000000000
	chainmaker.org/chainmaker-go/pb/protogo v0.0.0
	chainmaker.org/chainmaker-go/protocol v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3
	github.com/kr/pretty v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/prometheus/client_golang v1.9.0 // indirect
	github.com/stretchr/testify v1.6.1
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/sys v0.0.0-20201223074533-0d417f636930 // indirect
	golang.org/x/text v0.3.4 // indirect
)

replace (
	chainmaker.org/chainmaker-go/chainconf => ./../conf/chainconf
	chainmaker.org/chainmaker-go/common => ../../common
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ./../logger
	chainmaker.org/chainmaker-go/mock => ../../mock
	chainmaker.org/chainmaker-go/pb/protogo => ../../pb/protogo
	chainmaker.org/chainmaker-go/protocol => ./../../protocol
	chainmaker.org/chainmaker-go/utils => ../utils
	github.com/libp2p/go-libp2p-core => ../net/p2p/libp2pcore
)
