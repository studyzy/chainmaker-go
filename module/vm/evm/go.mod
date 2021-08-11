module chainmaker.org/chainmaker-go/evm

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210804033544-04ebb04bd759
	chainmaker.org/chainmaker/pb-go v0.0.0-20210810062511-a30ddac7d0e1
	chainmaker.org/chainmaker/protocol v0.0.0-20210727101110-59285b10f1ef
	github.com/ethereum/go-ethereum v1.10.3
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2

)

replace (
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/utils => ../../utils
)
