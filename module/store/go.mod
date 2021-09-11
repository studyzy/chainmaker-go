module chainmaker.org/chainmaker-go/store

go 1.15

require (
	chainmaker.org/chainmaker-go/localconf v0.0.0
	chainmaker.org/chainmaker/common/v2 v2.0.1-0.20210909033927-2a4cfc146579
	chainmaker.org/chainmaker/pb-go/v2 v2.0.1-0.20210901132412-435b75070bf2
	chainmaker.org/chainmaker/protocol/v2 v2.0.1-0.20210910112253-04256ae9c5ed
	chainmaker.org/chainmaker/store-badgerdb/v2 v2.0.0-20210909150251-a7a79b6b6f24
	chainmaker.org/chainmaker/store-leveldb/v2 v2.0.0-20210909122843-d0874400838a
	chainmaker.org/chainmaker/store-sqldb/v2 v2.0.0-20210911163035-4e8cbc0401a8 // indirect
	chainmaker.org/chainmaker/utils/v2 v2.0.0-20210907033606-84c6c841cbdb
	github.com/emirpasic/gods v1.12.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gogo/protobuf v1.3.2
	github.com/google/flatbuffers v2.0.0+incompatible // indirect
	github.com/mattn/go-sqlite3 v2.0.1+incompatible
	github.com/mitchellh/mapstructure v1.1.2
	github.com/pingcap/errors v0.11.5-0.20201029093017-5a7df2af2ac7 // indirect
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca
	github.com/tecbot/gorocksdb v0.0.0-20191217155057-f0fad39f321c
	github.com/yiyanwannian/gorocksdb v0.0.0-20210414112040-54bce342c6b6
	go.uber.org/atomic v1.7.0 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gorm.io/driver/mysql v1.0.3
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.20.8
)

replace chainmaker.org/chainmaker-go/localconf => ./../conf/localconf
