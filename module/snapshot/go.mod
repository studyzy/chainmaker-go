module chainmaker.org/chainmaker-go/snapshot

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210812042900-40fd24729b4a
	chainmaker.org/chainmaker/pb-go v0.0.0-20210812134751-dc3557ac21d8
	chainmaker.org/chainmaker/protocol v0.0.0-20210813101412-754cc134be43
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

	chainmaker.org/chainmaker-go/utils => ../utils

)
