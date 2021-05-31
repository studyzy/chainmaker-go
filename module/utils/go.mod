module chainmaker.org/chainmaker-go/utils

go 1.15

require (
	chainmaker.org/chainmaker-go/protocol v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210531062058-beb572d07e38
	chainmaker.org/chainmaker/pb-go v0.0.0-20210531071221-ccada476876b
	github.com/gogo/protobuf v1.3.2
	github.com/pingcap/parser v0.0.0-20200623164729-3a18f1e5dceb
	github.com/pingcap/tidb v1.1.0-beta.0.20200630082100-328b6d0a955c
	github.com/stretchr/testify v1.7.0
	github.com/studyzy/sqlparse v0.0.0-20210520090832-d40c792e1576
	google.golang.org/grpc v1.37.0 // indirect
)

replace (
	chainmaker.org/chainmaker-go/protocol => ../../protocol
	google.golang.org/grpc v1.37.0 => google.golang.org/grpc v1.26.0
)
