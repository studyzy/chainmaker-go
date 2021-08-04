module chainmaker.org/chainmaker-go/snapshot

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210804033544-04ebb04bd759
	chainmaker.org/chainmaker/pb-go v0.0.0-20210723070658-764cafbc33fe
	chainmaker.org/chainmaker/protocol v0.0.0-20210727101110-59285b10f1ef
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

	chainmaker.org/chainmaker-go/utils => ../utils

)
