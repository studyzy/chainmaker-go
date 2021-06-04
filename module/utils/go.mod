module chainmaker.org/chainmaker-go/utils

go 1.15

require (
	chainmaker.org/chainmaker-go/common v0.0.0
	chainmaker.org/chainmaker-go/pb/protogo v0.0.0
	chainmaker.org/chainmaker-go/protocol v0.0.0
	github.com/gogo/protobuf v1.3.2
	github.com/mr-tron/base58 v1.2.0
	github.com/pingcap/parser v0.0.0-20200623164729-3a18f1e5dceb
	github.com/pingcap/tidb v1.1.0-beta.0.20200630082100-328b6d0a955c
	github.com/stretchr/testify v1.7.0
	github.com/studyzy/sqlparse v0.0.0-20210520090832-d40c792e1576
	google.golang.org/grpc v1.37.0 // indirect
)

replace (
	chainmaker.org/chainmaker-go/common => ../../common
	chainmaker.org/chainmaker-go/pb/protogo => ../../pb/protogo
	chainmaker.org/chainmaker-go/protocol => ../../protocol
	google.golang.org/grpc v1.37.0 => google.golang.org/grpc v1.26.0
)
