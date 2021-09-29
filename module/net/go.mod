module chainmaker.org/chainmaker-go/net

go 1.15

require (
	chainmaker.org/chainmaker/chainmaker-net-common v0.0.6-0.20210929022840-ab2afb7445cf
	chainmaker.org/chainmaker/chainmaker-net-libp2p v0.0.11-0.20210929023059-2b89c39cb142
	chainmaker.org/chainmaker/chainmaker-net-liquid v0.0.8-0.20210929023139-c2988ebcabfb
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210928092334-f8be4fb05660
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907134457-53647922a89d
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210901132412-435b75070bf2
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210928092254-cfa32191bfce
	github.com/gogo/protobuf v1.3.2
	github.com/stretchr/testify v1.7.0
)

replace github.com/libp2p/go-libp2p-core => chainmaker.org/chainmaker/libp2p-core v0.0.2
