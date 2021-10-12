module chainmaker.org/chainmaker-go/consensus

go 1.15

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0
	chainmaker.org/chainmaker/chainconf/v2 v2.0.0-20210913144615-f27c44059848
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20211011114226-30eafbbd6523
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20210914062957-13e84972a921
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907134457-53647922a89d
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211011114556-3bbc2a898d5a
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20211009064056-03cbf6096208
	chainmaker.org/chainmaker/raftwal/v2 v2.0.3
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
	chainmaker.org/chainmaker/vm-native v0.0.0-20210922090336-9f8289cf0433
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/kr/pretty v0.2.0 // indirect
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/thoas/go-funk v0.8.0
	go.etcd.io/etcd/client/pkg/v3 v3.5.0
	go.etcd.io/etcd/raft/v3 v3.5.0
	go.etcd.io/etcd/server/v3 v3.5.0
	go.uber.org/zap v1.19.1
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../accesscontrol
	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
)
