module chainmaker.org/chainmaker-go/snapshot

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210831112613-754cd525d627
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210831114653-68cf6bb191f9
	chainmaker.org/chainmaker/protocol/v2 v2.0.0-20210831114940-7b97fb540200
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

	chainmaker.org/chainmaker-go/utils => ../utils

)
