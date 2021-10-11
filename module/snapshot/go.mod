module chainmaker.org/chainmaker-go/snapshot

go 1.15

require (
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20211008100315-b70ecfa0c08f
	chainmaker.org/chainmaker/localconf/v2 v2.0.0-20210913154622-9f9774ed7d1b
	chainmaker.org/chainmaker/logger/v2 v2.0.0-20210907134457-53647922a89d
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20211009072509-e7d0967e05e8
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210927062046-68813f263c0b
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
	github.com/stretchr/testify v1.7.0
)

replace chainmaker.org/chainmaker-go/localconf => ../conf/localconf
