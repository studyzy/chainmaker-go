module chainmaker.org/chainmaker-go/net

go 1.15

require (
	chainmaker.org/chainmaker/chainmaker-net-common v0.0.6-0.20210923074052-df57f27c4692
	chainmaker.org/chainmaker/chainmaker-net-libp2p v0.0.11-0.20210923123609-1694490e1b49
	//chainmaker.org/chainmaker/chainmaker-net-liquid v0.0.7-0.20210909125039-43c3ce7f4308
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210916080251-890936d14d9e
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907133316-af00cea33c97
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210901132412-435b75070bf2
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210917094712-7d18b2f609a1
	github.com/gogo/protobuf v1.3.2
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf

	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
)
