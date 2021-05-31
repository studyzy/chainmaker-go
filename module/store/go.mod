module chainmaker.org/chainmaker-go/store

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/protocol v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210531062058-beb572d07e38
	chainmaker.org/chainmaker/pb-go v0.0.0-20210531071221-ccada476876b
	github.com/emirpasic/gods v1.12.0
	github.com/facebookgo/ensure v0.0.0-20200202191622-63f1cf65ac4c // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/facebookgo/subset v0.0.0-20200203212716-c811ad88dec4 // indirect
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gogo/protobuf v1.3.2
	github.com/hokaccha/go-prettyjson v0.0.0-20201222001619-a42f9ac2ec8e // indirect
	github.com/mattn/go-sqlite3 v2.0.1+incompatible
	github.com/pingcap/errors v0.11.5-0.20201029093017-5a7df2af2ac7 // indirect
	github.com/pingcap/parser v0.0.0-20200623164729-3a18f1e5dceb
	github.com/pingcap/tidb v1.1.0-beta.0.20200630082100-328b6d0a955c
	github.com/pkg/errors v0.9.1
	github.com/spf13/viper v1.7.1 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca
	github.com/tecbot/gorocksdb v0.0.0-20191217155057-f0fad39f321c
	github.com/tidwall/wal v0.1.4
	go.uber.org/atomic v1.7.0 // indirect
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	gorm.io/driver/mysql v1.0.3
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.20.8
)

replace (
	chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../logger

	chainmaker.org/chainmaker-go/protocol => ../../protocol
	chainmaker.org/chainmaker-go/utils => ../utils
)
