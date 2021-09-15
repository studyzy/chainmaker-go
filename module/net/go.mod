module chainmaker.org/chainmaker-go/net

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker/chainmaker-net-common v0.0.5-0.20210906094918-bc30c12a8ff9
	chainmaker.org/chainmaker/chainmaker-net-libp2p v0.0.10-0.20210906104859-01678aa80176
	chainmaker.org/chainmaker/chainmaker-net-liquid v0.0.7-0.20210909125039-43c3ce7f4308
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210906095952-6d8f2c6cede0
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907133316-af00cea33c97
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210901132412-435b75070bf2
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210906092203-47d66f4908f7
	github.com/gogo/protobuf v1.3.2
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf

	github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
)
