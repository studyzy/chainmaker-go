/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"github.com/gogo/protobuf/proto"
)

var (
	CHAIN1                         = "chain1"
	IP                             = "localhost"
	PORT                           = 17988
	DEFAULT_CERT_ROOT_PATH         = "../config"
	DEFAULT_WASM_PATH              = "../wasm/fact.wasm"
	DEFAULT_USER_KEY_PATH          = "/crypto-config/%s/user/client1/client1.sign.key"
	DEFAULT_USER_CRT_PATH          = "/crypto-config/%s/user/client1/client1.sign.crt"
	DEFAULT_CA_PATH                = "/crypto-config/%s/ca"
	DEFAULT_USER_ADMIN_PATH        = "/crypto-config/%s/user/admin1/"
	DEFAULT_ORGID                  = "wx-org1.chainmaker.org"
	DEFAULT_UPDATE_HEIGHT_TX_COUNT = 500
	DEFAULT_ORGLIST_STR            = "wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org"
	DEFAULT_CONTRACT_NAME          = "contract1"
	DEFAULT_CONTRACT_TYPE          = "WASMER_RUST"

	prePathFmt          = DEFAULT_CERT_ROOT_PATH + DEFAULT_USER_ADMIN_PATH
	userKeyPath         = DEFAULT_CERT_ROOT_PATH + DEFAULT_USER_KEY_PATH
	userCrtPath         = DEFAULT_CERT_ROOT_PATH + DEFAULT_USER_CRT_PATH
	caPaths             = []string{DEFAULT_CERT_ROOT_PATH + DEFAULT_CA_PATH}
	sk3                 crypto.PrivateKey
	orgId               = DEFAULT_ORGID
	wasmPath            = DEFAULT_WASM_PATH
	contractName        = DEFAULT_CONTRACT_NAME
	contractType        = commonPb.RuntimeType(commonPb.RuntimeType_value[DEFAULT_CONTRACT_TYPE])
	orgList             = []string{}
	updateHeightTxCount = DEFAULT_UPDATE_HEIGHT_TX_COUNT
)

type Benchmarker struct {
	Metrics       *BenchmarkStat
	SendSpanMills int

	Url         string
	Method      string
	ClientCount int

	Sender BenchmarkerSender
}

func NewBenchmarker(url string, clientCount int, spanMills int, method string) *Benchmarker {
	b := &Benchmarker{
		ClientCount:   clientCount,
		Metrics:       NewBenchmarkStat(),
		SendSpanMills: spanMills,
		Url:           url,
		Method:        method,
	}

	switch b.Method {
	case "grpc":
		b.Sender = NewGRPCSender(clientCount, url, true)
	default:
		panic("error bench method")
	}

	return b
}

// StartBench ...
func (b *Benchmarker) StartBench() {
	b.createContract(0)
	time.Sleep(5 * time.Second)
	for i := 0; i < b.ClientCount; i++ {
		b.startPushTxByClientIndex(i)
	}
}

// startPushTxByClientIndex ...
func (b *Benchmarker) startPushTxByClientIndex(n int) error {
	if b.ClientCount < n+1 {
		return fmt.Errorf("index out of range")
	}
	go b.ProcessTxSend(int64(n))
	return nil
}

// createContract ...
func (b *Benchmarker) createContract(index int64) {
	tx, err := genCreateContractTxRequest(orgId, sk3, userCrtPath, CHAIN1)
	if err != nil {
		panic(fmt.Errorf("gen create tx error %v", err))
	}
	// log.Printf("genTX: %d,%s,%+v", tag, tx.GetHash().ToString(), tx.Action.Params)

	res, err := b.Sender.SendTxByClientIndex(b.Url, tx, index)
	if nil != err {
		panic(fmt.Errorf("create contract error :%v", err))
	}

	if res.Code != 0 {
		panic(fmt.Errorf("create contract vm error :%v", err))
	}
}

// ProcessTxSend ...
func (b *Benchmarker) ProcessTxSend(index int64) {
	for {
		time.Sleep(time.Duration(b.SendSpanMills) * time.Millisecond)
		b.Metrics.TpsMeter.Mark(1)
		atomic.AddInt64(b.Metrics.TxSend, 1)

		tx, err := genInvokeContractTxRequest(orgId, sk3, userCrtPath, CHAIN1)
		if err != nil {
			panic(fmt.Errorf("gen invoke tx error %v", err))
		}
		// log.Printf("genTX: %d,%s,%+v", tag, tx.GetHash().ToString(), tx.Action.Params)

		res, err := b.Sender.SendTxByClientIndex(b.Url, tx, index)
		if nil != err {
			atomic.AddInt64(b.Metrics.TxTotalFail, 1)
			log.Printf("send tx error: %s , %+v", tx.Payload.TxId, err)

			continue
		}

		if res.Code != 0 {
			atomic.AddInt64(b.Metrics.TxTotalFail, 1)
			/*
				if res.Code == errors.ErrCodeTxPoolTimeout {
					atomic.AddInt64(b.Metrics.TxTimeout, 1)
				} else if res.Code == errors.ErrCodeTxPoolDuplicatePool ||
					res.Code == errors.ErrCodeTxPoolDuplicatePending ||
					res.Code == errors.ErrCodeTxPoolDuplicateIncrement {

					atomic.AddInt64(b.Metrics.TxDuplicated, 1)
					// log.Printf("%d,%+v\n",tag,tx)
				} else {
					atomic.AddInt64(b.Metrics.TxOtherFail, 1)
				}
			*/
			log.Printf("tx error: %s , code:%d, msg:%s", tx.Payload.TxId, res.Code, res.Message)
		} else {
			atomic.AddInt64(b.Metrics.TxSucc, 1)
			if (*b.Metrics.TxSucc)%int64(updateHeightTxCount) == 0 {
				go b.updateBlockHeight(index, tx.Payload.TxId)
			}
		}
	}
}

func (b *Benchmarker) updateBlockHeight(index int64, txid string) {
	time.Sleep(5 * time.Second)
	getblockTX, err := genGetBlockByTxIDTxRequest(orgId, sk3, txid, CHAIN1)
	if err != nil {
		panic(fmt.Errorf("gen getblockBytxid tx error %v", err))
	}
	res, err := b.Sender.SendTxByClientIndex(b.Url, getblockTX, index)
	if nil != err {
		log.Printf("get tx's block error: %s , %+v", txid, err)
		return
	}
	if res.Code != 0 {
		log.Printf("get tx's block error: %s , %+v", txid, res.Message)
		return
	}
	blockInfo := &commonPb.BlockInfo{}
	err = proto.Unmarshal(res.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf("blockInfo unmarshal error %s\n", err)
		return
	}

	b.Metrics.TxHeight = blockInfo.Block.Header.BlockHeight
}

func main() {
	dest := flag.String("url", fmt.Sprintf("%s:%d", IP, PORT), "node address:port string")
	clientNum := flag.Int("num", 1, "client number")
	spanMills := flag.Int("span", 1000, "sleep span mills for each client")
	method := flag.String("method", "grpc", "grpc")
	certrootPath := flag.String("cert_root_path", DEFAULT_CERT_ROOT_PATH, "cert root file path.default:"+DEFAULT_CERT_ROOT_PATH)
	wasm := flag.String("wasm_path", DEFAULT_WASM_PATH, "wasm path.default:"+DEFAULT_WASM_PATH)
	contractname := flag.String("contract_name", DEFAULT_CONTRACT_NAME, "contract name.default:"+DEFAULT_CONTRACT_NAME)
	contracttype := flag.String("contract_type", DEFAULT_CONTRACT_TYPE, "contract type.default:"+DEFAULT_CONTRACT_TYPE)
	orgid := flag.String("org_id", DEFAULT_ORGID, "org id.default:"+DEFAULT_ORGID)
	updatetx := flag.Int("update_height", DEFAULT_UPDATE_HEIGHT_TX_COUNT, fmt.Sprintf("update height tx count:%d", DEFAULT_UPDATE_HEIGHT_TX_COUNT))
	orglistStr := flag.String("org_list", DEFAULT_ORGLIST_STR, "org list.default:"+DEFAULT_ORGLIST_STR)
	/*
		userkeyPath := flag.String("user_key_path", *certrootPath+DEFAULT_USER_KEY_PATH, "user key file path.default:"+*certrootPath+DEFAULT_USER_KEY_PATH)
		usercrtPath := flag.String("user_crt_path", *certrootPath+DEFAULT_USER_CRT_PATH, "user crt file path.default:"+*certrootPath+DEFAULT_USER_CRT_PATH)
		useradminPath := flag.String("user_admin_path", *certrootPath+DEFAULT_USER_ADMIN_PATH, "user admin file path.default:"+*certrootPath+DEFAULT_USER_ADMIN_PATH)
		caPathsString := flag.String("ca_path", *certrootPath+DEFAULT_CA_PATH, "ca file path.default:"+*certrootPath+DEFAULT_CA_PATH)
		var caPaths []string
		for _, ca := range strings.Split(*caPathsString, ",") {
			caPaths = append(caPaths, ca)
		}
	*/

	flag.Parse()

	if *method != "grpc" {
		fmt.Printf("method %s, not valid\n", *method)
		os.Exit(-1)
	}
	orgId = *orgid
	updateHeightTxCount = *updatetx
	userKeyPath = fmt.Sprintf(*certrootPath+DEFAULT_USER_KEY_PATH, orgId)
	userCrtPath = fmt.Sprintf(*certrootPath+DEFAULT_USER_CRT_PATH, orgId)
	prePathFmt = *certrootPath + DEFAULT_USER_ADMIN_PATH
	caPaths = []string{fmt.Sprintf(*certrootPath+DEFAULT_CA_PATH, orgId)}
	wasmPath = *wasm
	contractName = *contractname
	contractType = commonPb.RuntimeType(commonPb.RuntimeType_value[*contracttype])

	var orglist []string
	for _, org := range strings.Split(*orglistStr, ",") {
		orglist = append(orglist, org)
	}
	orgList = orglist

	file, err := ioutil.ReadFile(userKeyPath)
	if err != nil {
		panic(err)
	}

	sk3, err = asym.PrivateKeyFromPEM(file, nil)
	if err != nil {
		panic(err)
	}

	cpuNum := runtime.NumCPU()
	runtime.GOMAXPROCS(cpuNum)
	fmt.Printf("Start Benchmark Combination:\nUrl:%s,ClientNum:%d, SendSpanMills:%d, CPU:%d, Type:%s\n", *dest, *clientNum, *spanMills, cpuNum, *method)

	benchmarker := NewBenchmarker(*dest, *clientNum, *spanMills, *method)
	benchmarker.StartBench()
	benchmarker.Metrics.PrintInfoLooper(5)

}
