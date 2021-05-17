module chainmaker.org/chainmaker-go/tools/cmc

go 1.15

require (
	chainmaker.org/chainmaker-go/common v0.0.0
	chainmaker.org/chainmaker-sdk-go v0.0.0
	github.com/gogo/protobuf v1.3.2
	github.com/hokaccha/go-prettyjson v0.0.0-20201222001619-a42f9ac2ec8e
	github.com/samkumar/hibe v0.0.0-20171013061409-c1cd171b6178
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	gorm.io/driver/mysql v1.0.6
	gorm.io/gorm v1.21.9
	vuvuzela.io/crypto v0.0.0-20190327123840-80a93a3ed1d6
)

replace (
	chainmaker.org/chainmaker-go/common => ../../common
	chainmaker.org/chainmaker-sdk-go => ../sdk
)
