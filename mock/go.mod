module chainmaker.org/chainmaker-go/mock

go 1.15

require (


	chainmaker.org/chainmaker-go/protocol v0.0.0
	github.com/golang/mock v1.4.4
)

replace (

	chainmaker.org/chainmaker-go/pb/protogo => ../pb/protogo
	chainmaker.org/chainmaker-go/protocol => ../protocol
)
