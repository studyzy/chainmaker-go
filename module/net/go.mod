module chainmaker.org/chainmaker-go/net

go 1.15

require (
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210928022522-120cf16c8354
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20210913154622-9f9774ed7d1b
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907134457-53647922a89d
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211009072509-e7d0967e05e8
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210927062046-68813f263c0b
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
	github.com/gogo/protobuf v1.3.2
	github.com/libp2p/go-libp2p v0.11.0
	github.com/libp2p/go-libp2p-circuit v0.3.1
	github.com/libp2p/go-libp2p-core v0.6.1
	github.com/libp2p/go-libp2p-discovery v0.5.0
	github.com/libp2p/go-libp2p-kad-dht v0.10.0
	github.com/libp2p/go-libp2p-pubsub v0.3.5
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/stretchr/testify v1.7.0
	github.com/tjfoc/gmsm v1.4.1
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ./../logger

	github.com/libp2p/go-libp2p => ./p2p/libp2p
	github.com/libp2p/go-libp2p-core => ./p2p/libp2pcore
	github.com/libp2p/go-libp2p-pubsub => ./p2p/libp2ppubsub
)
