module chainmaker.org/chainmaker-go/localconf

go 1.15

require (
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker/pb-go v0.0.0-20210816122318-5281b78fdd2e
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20201222001619-a42f9ac2ec8e
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
)

replace chainmaker.org/chainmaker-go/logger => ./../../logger
