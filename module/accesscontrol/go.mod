module chainmaker.org/chainmaker-go/accesscontrol

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210906095952-6d8f2c6cede0
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907133316-af00cea33c97
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210910073253-777869c183c7
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210901134008-4b83cf573272
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
	github.com/gogo/protobuf v1.3.2
	github.com/stretchr/testify v1.7.0
)

replace chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
