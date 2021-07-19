module chainmaker.org/chainmaker-go/test

go 1.15

require (
	chainmaker.org/chainmaker-go/accesscontrol v0.0.0
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/net v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0
	chainmaker.org/chainmaker/common v0.0.0-20210716065627-3246c7d5da41
	chainmaker.org/chainmaker/pb-go v0.0.0-20210719032153-653bd8436ef6
	chainmaker.org/chainmaker/protocol v0.0.0-20210719033859-75ccee9a9042
	chainmaker.org/chainmaker/sdk-go v0.0.0-20210706173158-3315f50af4de // indirect
	github.com/aristanetworks/goarista v0.0.0-20170210015632-ea17b1a17847 // indirect
	github.com/dvyukov/go-fuzz v0.0.0-20200318091601-be3528f3a813 // indirect
	github.com/ethereum/go-ethereum v1.10.4
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/mr-tron/base58 v1.2.0
	github.com/pborman/uuid v0.0.0-20170112150404-1b00554d8222 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0
	github.com/rs/xhandler v0.0.0-20160618193221-ed27b6fd6521 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/steakknife/bloomfilter v0.0.0-20180922174646-6819c0d2a570 // indirect
	github.com/steakknife/hamming v0.0.0-20180906055917-c99c65617cd3 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/wsddn/go-ecdh v0.0.0-20161211032359-48726bab9208 // indirect
	golang.org/x/mobile v0.0.0-20200801112145-973feb4309de // indirect
	google.golang.org/genproto v0.0.0-20210303154014-9728d6b83eeb // indirect
	google.golang.org/grpc v1.37.0
)

replace (
	chainmaker.org/chainmaker-go/accesscontrol => ../module/accesscontrol

	chainmaker.org/chainmaker-go/localconf => ./../module/conf/localconf
	chainmaker.org/chainmaker-go/logger => ../module/logger
	chainmaker.org/chainmaker-go/net => ../module/net

	chainmaker.org/chainmaker-go/utils => ../module/utils

	github.com/libp2p/go-libp2p => ../module/net/p2p/libp2p
	github.com/libp2p/go-libp2p-core => ../module/net/p2p/libp2pcore
	github.com/libp2p/go-libp2p-pubsub => ../module/net/p2p/libp2ppubsub
)
