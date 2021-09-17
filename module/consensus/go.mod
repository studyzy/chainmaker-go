module chainmaker.org/chainmaker-go/consensus

go 1.15

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0
	chainmaker.org/chainmaker/chainconf/v2 v2.0.0-20210913144615-f27c44059848
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210915091854-ee36ee89dc57
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20210914062957-13e84972a921
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907134457-53647922a89d
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210901132412-435b75070bf2
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210914063622-6f007edc3a98
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
	chainmaker.org/chainmaker/vm-native v0.0.0-20210915072159-1598396850f0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/kr/pretty v0.2.0 // indirect
	github.com/prometheus/client_golang v1.9.0 // indirect
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/thoas/go-funk v0.8.0
	go.etcd.io/etcd/client/pkg/v3 v3.5.0-beta.4
	go.etcd.io/etcd/raft/v3 v3.5.0-beta.4
	go.etcd.io/etcd/server/v3 v3.5.0-beta.4
	go.uber.org/zap v1.16.1-0.20210329175301-c23abee72d19
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	github.com/libp2p/go-libp2p-core => ../net/p2p/libp2pcore
)
