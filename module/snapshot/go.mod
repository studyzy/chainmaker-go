module chainmaker.org/chainmaker-go/snapshot

go 1.15

require (
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20211101030932-bf3bcb20eb52
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20211101111610-0d268248b5c8
	chainmaker.org/chainmaker/logger/v2 v2.0.1-0.20211015125919-8e5199930ac9
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211021024710-9329804d1c21
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20211014144951-97323532a236
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20211027124954-09b710bd9ce8
	github.com/stretchr/testify v1.7.0
)

replace chainmaker.org/chainmaker-go/localconf => ../conf/localconf
