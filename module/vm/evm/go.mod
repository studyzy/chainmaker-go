module chainmaker.org/chainmaker-go/evm

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210906095952-6d8f2c6cede0
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210901132412-435b75070bf2
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210901134008-4b83cf573272
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
	github.com/ethereum/go-ethereum v1.10.3
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2

)

replace (
	chainmaker.org/chainmaker-go/logger => ../../logger
)
