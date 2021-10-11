module chainmaker.org/chainmaker-go/net

go 1.15

require (
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20211008100315-b70ecfa0c08f
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20210913154622-9f9774ed7d1b
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907134457-53647922a89d
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211009072509-e7d0967e05e8
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210927062046-68813f263c0b
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
	chainmaker.org/chainmaker/chainmaker-net-common v0.0.6-0.20210929043521-02e40bf96300
	chainmaker.org/chainmaker/chainmaker-net-libp2p v0.0.11-0.20210929043636-4e46c072735d
	chainmaker.org/chainmaker/chainmaker-net-liquid v0.0.8-0.20210929043651-03cf1a4650f8
	github.com/gogo/protobuf v1.3.2
	github.com/stretchr/testify v1.7.0
)

replace github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
