module chainmaker.org/chainmaker-go/evm

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/pb/protogo v0.0.0
	chainmaker.org/chainmaker-go/protocol v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	github.com/ethereum/go-ethereum v1.9.25
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/sys v0.0.0-20201221093633-bc327ba9c2f0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)

replace (
	chainmaker.org/chainmaker-go/common => ../../../common
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/pb/protogo => ../../../pb/protogo
	chainmaker.org/chainmaker-go/protocol => ../../../protocol
	chainmaker.org/chainmaker-go/utils => ../../utils
)
