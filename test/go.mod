module chainmaker.org/chainmaker-go/test

go 1.15

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0
	chainmaker.org/chainmaker-go/net v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210916080817-79b5a4160dae
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210916064951-47123db73430
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210914063622-6f007edc3a98
	chainmaker.org/chainmaker/sdk-go/v2 v2.0.1-0.20210908032542-7ce5c8a979e6
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
	github.com/ethereum/go-ethereum v1.10.4
	github.com/gogo/protobuf v1.3.2
	github.com/mr-tron/base58 v1.2.0
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.0
	google.golang.org/genproto v0.0.0-20210303154014-9728d6b83eeb // indirect
	google.golang.org/grpc v1.37.0
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../module/accesscontrol

	chainmaker.org/chainmaker-go/localconf => ./../module/conf/localconf
	chainmaker.org/chainmaker-go/logger => ../module/logger
	chainmaker.org/chainmaker-go/net => ../module/net
	github.com/libp2p/go-libp2p => ../module/net/p2p/libp2p
	github.com/libp2p/go-libp2p-core => ../module/net/p2p/libp2pcore
	github.com/libp2p/go-libp2p-pubsub => ../module/net/p2p/libp2ppubsub
)
