module chainmaker.org/chainmaker-go/snapshot

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210719075615-247a2254ad3d
	chainmaker.org/chainmaker/pb-go v0.0.0-20210719032153-653bd8436ef6
	chainmaker.org/chainmaker/protocol v0.0.0-20210719033859-75ccee9a9042
	github.com/pingcap/parser v0.0.0-20200623164729-3a18f1e5dceb // indirect
	github.com/spf13/cobra v1.1.1 // indirect
	github.com/stretchr/testify v1.7.0
)

replace (
	chainmaker.org/chainmaker-go/localconf => ../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

	chainmaker.org/chainmaker-go/utils => ../utils

)
