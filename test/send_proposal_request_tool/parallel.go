/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker/common/v2/ca"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/logger/v2"

	"chainmaker.org/chainmaker/utils/v2"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log = logger.GetLogger(logger.MODULE_CLI)

var (
	threadNum   int
	loopNum     int
	timeout     int
	printTime   int
	sleepTime   int
	climbTime   int
	checkResult bool
	recordLog   bool
	showKey     bool

	hostsString        string
	userCrtPathsString string
	userKeyPathsString string
	orgIDsString       string

	hosts        []string
	userCrtPaths []string
	userKeyPaths []string
	orgIDs       []string

	nodeNum int

	fileCache = NewFileCacheReader()
	certCache = NewCertFileCacheReader()

	abiCache     = NewFileCacheReader()
	outputResult bool
)

type KeyValuePair struct {
	Key        string `json:"key,omitempty"`
	Value      string `json:"value,omitempty"`
	Unique     bool   `json:"unique,omitempty"`
	RandomRate int    `json:"randomRate,omitempty"`
}

type Detail struct {
	TPS          float32                `json:"tps"`
	SuccessCount int                    `json:"successCount"`
	FailCount    int                    `json:"failCount"`
	Count        int                    `json:"count"`
	MinTime      int64                  `json:"minTime"`
	MaxTime      int64                  `json:"maxTime"`
	AvgTime      float32                `json:"avgTime"`
	StartTime    string                 `json:"startTime"`
	EndTime      string                 `json:"endTime"`
	Elapsed      float32                `json:"elapsed"`
	ThreadNum    int                    `json:"threadNum"`
	LoopNum      int                    `json:"loopNum"`
	Nodes        map[string]interface{} `json:"nodes"`
}

type Statistician struct {
	completedCount int32
	completedTimes []int64
	completedState []bool
	completedId    []int

	lastIndex     int
	lastStartTime time.Time

	startTime time.Time
	endTime   time.Time
}

func ParallelCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parallel",
		Short: "Parallel",
		Long:  "Parallel",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			caPaths = strings.Split(caPathsString, ",")
			hosts = strings.Split(hostsString, ",")
			userCrtPaths = strings.Split(userCrtPathsString, ",")
			userKeyPaths = strings.Split(userKeyPathsString, ",")
			orgIDs = strings.Split(orgIDsString, ",")
			if len(hosts) != len(userCrtPaths) || len(hosts) != len(userKeyPaths) || len(hosts) != len(caPaths) || len(hosts) != len(orgIDs) {
				panic(fmt.Sprintf("hosts[%d], user-crts[%d], user-keys[%d], ca-path[%d], orgIDs[%d] length invalid",
					len(hosts), len(userCrtPaths), len(userKeyPaths), len(caPaths), len(orgIDs)))
			}
			nodeNum = len(hosts)
			if len(pairsFile) != 0 {
				bytes, err := ioutil.ReadFile(pairsFile)
				if err != nil {
					panic(err)
				}
				pairsString = string(bytes)
			}
			fmt.Println("tx content: ", pairsString)
		},
	}

	flags := cmd.PersistentFlags()
	flags.IntVarP(&threadNum, "threadNum", "N", 10, "specify thread number")
	flags.IntVarP(&loopNum, "loopNum", "l", 1000, "specify loop number")
	flags.IntVarP(&timeout, "timeout", "T", 2, "specify timeout(unit: s)")
	flags.IntVarP(&printTime, "printTime", "r", 1, "specify print time(unit: s)")
	flags.IntVarP(&sleepTime, "sleepTime", "S", 100, "specify sleep time(unit: ms)")
	flags.IntVarP(&climbTime, "climbTime", "L", 10, "specify climb time(unit: s)")
	flags.StringVarP(&hostsString, "hosts", "H", "localhost:17988,localhost:27988", "specify hosts")
	flags.StringVarP(&userCrtPathsString, "user-crts", "K", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/client1/client1.tls.crt", "specify user crt path")
	flags.StringVarP(&userKeyPathsString, "user-keys", "u", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key,../../config/crypto-config/wx-org2.chainmaker.org/user/client1/client1.tls.key", "specify user key path")
	flags.StringVarP(&orgIDsString, "org-IDs", "I", "wx-org1,wx-org2", "specify user key path")
	flags.BoolVarP(&checkResult, "check-result", "Y", false, "specify whether check result")
	flags.BoolVarP(&recordLog, "record-log", "g", false, "specify whether record log")
	flags.BoolVarP(&outputResult, "output-result", "", false, "output rpc result, eg: txid")
	flags.BoolVarP(&showKey, "showKey", "", false, "bool")

	cmd.AddCommand(invokeCMD())
	cmd.AddCommand(queryCMD())
	cmd.AddCommand(createContractCMD())
	cmd.AddCommand(upgradeContractCMD())

	return cmd
}

var (
	invokerMethod = "invoke"
)

func invokeCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   invokerMethod,
		Short: "Invoke",
		RunE: func(_ *cobra.Command, _ []string) error {
			return parallel(invokerMethod)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&pairsString, "pairs", "a", "[{\"key\":\"key\",\"value\":\"counter1\",\"unique\":false}]", "specify pairs")
	flags.StringVarP(&pairsFile, "pairs-file", "A", "", "specify pairs file, if used, set --pairs=\"\"")
	flags.StringVarP(&method, "method", "m", "increase", "specify contract method")
	flags.StringVarP(&abiPath, "abi-path", "", "", "abi file path")

	return cmd
}

func queryCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query",
		Long:  "Query",
		RunE: func(_ *cobra.Command, _ []string) error {
			return parallel("query")
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&pairsString, "pairs", "a", "[{\"key\":\"key\",\"value\":\"counter1\",\"unique\":false}]", "specify pairs")
	flags.StringVarP(&pairsFile, "pairs-file", "A", "", "specify pairs file, if used, set --pairs=\"\"")
	flags.StringVarP(&method, "method", "m", "increase", "specify contract method")

	return cmd
}

var (
	createContractStr  = "createContract"
	upgradeContractStr = "upgradeContract"
)

func createContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   createContractStr,
		Short: "Create Contract",
		Long:  "Create Contract",
		RunE: func(_ *cobra.Command, _ []string) error {
			return parallel(createContractStr)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&wasmPath, "wasm-path", "w", "../wasm/counter-go.wasm", "specify wasm path")
	flags.Int32VarP(&runTime, "run-time", "m", int32(commonPb.RuntimeType_GASM), "specify run time")

	return cmd
}

func upgradeContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   upgradeContractStr,
		Short: "Upgrade Contract",
		Long:  "Upgrade Contract",
		RunE: func(_ *cobra.Command, _ []string) error {
			return parallel(upgradeContractStr)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&wasmPath, "wasm-path", "w", "../wasm/counter-go.wasm", "specify wasm path")
	flags.Int32VarP(&runTime, "run-time", "R", int32(commonPb.RuntimeType_GASM), "specify run time")
	flags.StringVarP(&version, "version", "v", "2.0.0", "specify contract version")

	return cmd
}

func parallel(parallelMethod string) error {
	if nodeNum > threadNum {
		//fmt.Println("threadNum:", threadNum, "less nodeNum:", nodeNum, "change threadNum=nodeNum")
		threadNum = nodeNum
	}
	timeoutChan := make(chan struct{}, threadNum)
	doneChan := make(chan struct{}, threadNum)
	doneCount := 0
	statistician := &Statistician{
		completedTimes: make([]int64, threadNum*loopNum),
		completedState: make([]bool, threadNum*loopNum),
		completedId:    make([]int, threadNum*loopNum),
	}

	var threads []*Thread
	for i := 0; i < threadNum; i++ {
		t := &Thread{
			id:           i,
			loopNum:      loopNum,
			doneChan:     doneChan,
			timeoutChan:  timeoutChan,
			statistician: statistician,
		}
		switch parallelMethod {
		case invokerMethod:
			t.operationName = invokerMethod
			t.handler = &invokeHandler{threadId: i}
		case "query":
			t.operationName = "query"
			t.handler = &queryHandler{threadId: i}
		case createContractStr:
			t.operationName = createContractStr
			t.handler = &createContractHandler{threadId: i}
		case upgradeContractStr:
			t.operationName = upgradeContractStr
			t.handler = &upgradeContractHandler{threadId: i}
		}
		threads = append(threads, t)
	}

	statistician.startTime = time.Now()
	statistician.lastStartTime = time.Now()

	for _, thread := range threads {
		if err := thread.Init(); err != nil {
			return err
		}
	}

	go parallelStart(threads)

	printTicker := time.NewTicker(time.Duration(printTime) * time.Second)
	timeoutTicker := time.NewTicker(time.Duration(timeout) * time.Second)
	timeoutOnce := sync.Once{}
	for {
		if doneCount >= threadNum {
			break
		}
		select {
		case <-doneChan:
			doneCount++
		case <-printTicker.C:
			go statistician.PrintDetails(false)
		case <-timeoutTicker.C:
			go func() {
				timeoutOnce.Do(func() {
					for i := 0; i < threadNum; i++ {
						timeoutChan <- struct{}{}
					}
				})
			}()
		}
	}

	statistician.endTime = time.Now()

	fmt.Println("Statistics for the entire test")
	statistician.PrintDetails(true)

	close(timeoutChan)
	close(doneChan)
	printTicker.Stop()
	timeoutTicker.Stop()
	for _, t := range threads {
		t.Stop()
	}
	return nil
}

func parallelStart(threads []*Thread) {
	count := threadNum / 10
	if count == 0 {
		count = 1
	}
	interval := time.Duration(climbTime/count) * time.Second
	for index := 0; index < threadNum; {
		for j := 0; j < 10; j++ {
			go threads[index].Start()
			index++
			if index >= threadNum {
				break
			}
		}
		time.Sleep(interval)
	}
}

func (s *Statistician) PrintDetails(all bool) {
	nodeMin := make([]int64, nodeNum)
	nodeMax := make([]int64, nodeNum)
	nodeSum := make([]int64, nodeNum)
	nodeSuccessCount := make([]int, nodeNum)
	nodeCount := make([]int, nodeNum)

	for i := 0; i < nodeNum; i++ {
		nodeMin[i] = math.MaxInt16
		nodeMax[i] = 0
	}

	last := 0
	if !all {
		last = s.lastIndex
	}
	nowCount := atomic.LoadInt32(&s.completedCount)
	nowTime := time.Now()
	min, max, sum, successCount, count := s.completedTimes[0], s.completedTimes[0], int64(0), 0, int64(0)
	for i := last; i < int(nowCount); i++ {
		nodeId := s.completedId[i]

		sum += s.completedTimes[i]
		nodeSum[nodeId] += s.completedTimes[i]

		count++
		nodeCount[nodeId]++

		if s.completedState[i] {
			successCount++
			nodeSuccessCount[nodeId]++
		}
		if s.completedTimes[i] < min {
			min = s.completedTimes[i]
		}
		if s.completedTimes[i] < nodeMin[nodeId] {
			nodeMin[nodeId] = s.completedTimes[i]
		}

		if s.completedTimes[i] > max {
			max = s.completedTimes[i]
		}
		if s.completedTimes[i] > nodeMax[nodeId] {
			nodeMax[nodeId] = s.completedTimes[i]
		}
	}

	detail := s.statisticsResults(&numberResults{count: int(count), successCount: successCount,
		max: max, min: min, sum: sum, nodeSuccessCount: nodeSuccessCount, nodeCount: nodeCount,
		nodeMin: nodeMin, nodeMax: nodeMax, nodeSum: nodeSum}, all, nowTime)
	s.lastIndex = int(nowCount)
	s.lastStartTime = time.Now()

	bytes, err := json.Marshal(detail)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(bytes))
	fmt.Println()
}

type numberResults struct {
	count, successCount         int
	min, max, sum               int64
	nodeSuccessCount, nodeCount []int
	nodeMin, nodeMax, nodeSum   []int64
}

func (s *Statistician) statisticsResults(ret *numberResults, all bool, nowTime time.Time) (detail *Detail) {
	detail = &Detail{
		Nodes: make(map[string]interface{}),
	}
	if ret.count > 0 {
		detail = &Detail{
			SuccessCount: ret.successCount,
			FailCount:    ret.count - ret.successCount,
			Count:        ret.count,
			MinTime:      ret.min,
			MaxTime:      ret.max,
			AvgTime:      float32(ret.sum) / float32(ret.count),
			ThreadNum:    threadNum,
			LoopNum:      loopNum,
			Nodes:        make(map[string]interface{}),
		}
		for i := 0; i < nodeNum; i++ {
			detail.Nodes[fmt.Sprintf("node%d_successCount", i)] = ret.nodeSuccessCount[i]
			detail.Nodes[fmt.Sprintf("node%d_failCount", i)] = ret.nodeCount[i] - ret.nodeSuccessCount[i]
			detail.Nodes[fmt.Sprintf("node%d_count", i)] = ret.nodeCount[i]
			detail.Nodes[fmt.Sprintf("node%d_minTime", i)] = ret.nodeMin[i]
			detail.Nodes[fmt.Sprintf("node%d_maxTime", i)] = ret.nodeMax[i]
			detail.Nodes[fmt.Sprintf("node%d_avgTime", i)] = float32(ret.nodeSum[i]) / float32(ret.nodeCount[i])
		}
	}
	if all {
		detail.StartTime = s.startTime.Format("2006-01-04 15:04:05.000")
		detail.EndTime = s.endTime.Format("2006-01-03 15:04:05.000")
		detail.Elapsed = float32(s.endTime.Sub(s.startTime).Milliseconds()) / 1000
		detail.TPS = float32(ret.successCount) / float32(s.endTime.Sub(s.startTime).Seconds())
		for i := 0; i < nodeNum; i++ {
			detail.Nodes[fmt.Sprintf("node%d_tps", i)] = float32(ret.nodeSuccessCount[i]) / float32(s.endTime.Sub(s.startTime).Seconds())
		}
	} else {
		detail.StartTime = s.lastStartTime.Format("2006-02-02 15:04:05.000")
		detail.EndTime = nowTime.Format("2006-01-02 15:04:05.000")
		detail.Elapsed = float32(nowTime.Sub(s.lastStartTime).Milliseconds()) / 1000
		detail.TPS = float32(ret.successCount) / float32(nowTime.Sub(s.lastStartTime).Seconds())
		for i := 0; i < nodeNum; i++ {
			detail.Nodes[fmt.Sprintf("node%d_tps", i)] = float32(ret.nodeSuccessCount[i]) / float32(nowTime.Sub(s.lastStartTime).Seconds())
		}
	}
	return detail
}

type Thread struct {
	id            int
	loopNum       int
	doneChan      chan struct{}
	timeoutChan   chan struct{}
	handler       Handler
	statistician  *Statistician
	operationName string

	conn   *grpc.ClientConn
	client apiPb.RpcNodeClient
	sk3    crypto.PrivateKey
	index  int
}

func (t *Thread) Init() error {
	var err error
	t.index = t.id % len(hosts)
	t.conn, err = t.initGRPCConnect(useTLS, t.index)
	if err != nil {
		return err
	}
	t.client = apiPb.NewRpcNodeClient(t.conn)

	file, err := ioutil.ReadFile(userKeyPaths[t.index])
	if err != nil {
		return err
	}

	t.sk3, err = asym.PrivateKeyFromPEM(file, nil)
	if err != nil {
		return err
	}

	return nil
}

func (t *Thread) Start() {
	infos, err := t.getPairInfos()
	if err != nil {
		t.doneChan <- struct{}{}
		return
	}

	for i := 0; i < t.loopNum; i++ {
		select {
		case <-t.timeoutChan:
			t.doneChan <- struct{}{}
			return
		default:
			start := time.Now()
			err := t.handler.handle(t.client, t.sk3, orgIDs[t.index], userCrtPaths[t.index], i, infos)
			elapsed := time.Since(start)

			index := atomic.AddInt32(&t.statistician.completedCount, 1)
			t.statistician.completedTimes[index-1] = elapsed.Milliseconds()
			t.statistician.completedState[index-1] = err == nil
			t.statistician.completedId[index-1] = t.index

			if recordLog && err != nil {
				log.Errorf("threadId: %d, loopId: %d, nodeId: %d, err: %s", t.id, i, t.index, err)
			}

			time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		}
	}
	t.doneChan <- struct{}{}
}

func (t *Thread) getPairInfos() ([]*KeyValuePair, error) {
	if t.operationName == createContractStr || t.operationName == upgradeContractStr {
		return nil, nil
	}
	var ps []*KeyValuePair
	err := json.Unmarshal([]byte(pairsString), &ps)
	if err != nil {
		log.Errorf("unmarshal pair content failed, origin content: %s, "+
			"threadId: %d, nodeId: %d, err: %s", pairsString, t.id, t.index, err)
		return nil, err
	}

	return ps, nil
}

func (t *Thread) Stop() {
	t.conn.Close()
}

func (t *Thread) initGRPCConnect(useTLS bool, index int) (*grpc.ClientConn, error) {
	url := hosts[index]

	if useTLS {
		tlsClient := ca.CAClient{
			ServerName: "chainmaker.org",
			CaPaths:    []string{caPaths[index]},
			CertFile:   userCrtPaths[index],
			KeyFile:    userKeyPaths[index],
		}

		c, err := tlsClient.GetCredentialsByCA()
		if err != nil {
			return nil, err
		}
		return grpc.Dial(url, grpc.WithTransportCredentials(*c))
	} else {
		return grpc.Dial(url, grpc.WithInsecure())
	}
}

type Handler interface {
	handle(client apiPb.RpcNodeClient, sk3 crypto.PrivateKey, orgId string, userCrtPath string, loopId int, ps []*KeyValuePair) error
}

type invokeHandler struct {
	threadId int
}

var (
	respStr     = "proposalRequest error, resp: %+v"
	templateStr = "%s_%d_%d_%d"
	resultStr   = "exec result, orgid: %s, loop_id: %d, method1: %s, txid: %s, resp: %+v"
)

var randomRate = 0
var totalSentTxs = 1
var totalRandomSentTxs = 1

func (h *invokeHandler) handle(client apiPb.RpcNodeClient, sk3 crypto.PrivateKey, orgId string, userCrtPath string, loopId int, ps []*KeyValuePair) error {
	txId := utils.GetRandTxId()

	// 构造Payload
	pairs := make([]*commonPb.KeyValuePair, 0)
	randomRateTmp := 0
	for _, p := range ps {
		if p.RandomRate > 100 || p.RandomRate < 0 {
			panic("randomRate must int in [0,100]")
		}

		if p.RandomRate > 0 {
			if randomRateTmp > 0 {
				panic("randomRate used once by one key")
			}
			randomRateTmp = p.RandomRate
			randomRate = p.RandomRate
		}

		key := p.Key
		val := []byte(p.Value)
		totalSentTxs += 1
		if randomRate > 0 && p.RandomRate > 0 {
			if randomRate > (totalRandomSentTxs * 100 / totalSentTxs) {
				val = []byte(fmt.Sprintf(templateStr, p.Value, h.threadId, loopId, time.Now().UnixNano()))
				totalRandomSentTxs += 1
			}
		} else if p.Unique {
			val = []byte(fmt.Sprintf(templateStr, p.Value, h.threadId, loopId, time.Now().UnixNano()))
		}
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   key,
			Value: val,
		})
	}
	if showKey {
		j, err := json.Marshal(pairs)
		if err != nil {
			fmt.Println(err)
		}
		rate := totalRandomSentTxs * 100 / totalSentTxs
		if totalRandomSentTxs == 1 {
			rate = 0
		}
		fmt.Printf("totalSentTxs:%d\t totalRandomSentTxs:%d\t randomRate:%d \t param:%s\t \n", totalSentTxs, totalRandomSentTxs-1, rate, string(j))
	}

	// 支持evm
	//var err error
	var abiData *[]byte
	if abiPath != "" {
		abiData = abiCache.Read(abiPath)
		runTime = 5 //evm
	}

	method1, pairs1, err := makePairs(method, abiPath, pairs, commonPb.RuntimeType(runTime), abiData)

	//fmt.Println("[exec_handle]orgId: ", orgId, ", userCrtPath: ", userCrtPath, ", loopId: ", loopId, ", method1: ", method1, ", pairs1: ", pairs1, ", method: ", method, ", pairs: ", pairs)
	payloadBytes, err := constructInvokePayload(chainId, contractName, method1, pairs1)
	if err != nil {
		return err
	}

	resp, err = sendRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT,
		txId: txId, chainId: chainId}, orgId, userCrtPath, payloadBytes, nil)
	if err != nil {
		return err
	}

	if outputResult {
		msg := fmt.Sprintf(resultStr, orgId, loopId, method1, txId, resp)
		fmt.Println(msg)
	}

	if checkResult && resp.Code != commonPb.TxStatusCode_SUCCESS {
		return fmt.Errorf(respStr, resp)
	}

	return nil
}

type queryHandler struct {
	threadId int
}

func (h *queryHandler) handle(client apiPb.RpcNodeClient, sk3 crypto.PrivateKey, orgId string, userCrtPath string, loopId int, ps []*KeyValuePair) error {
	txId := utils.GetRandTxId()

	// 构造Payload
	//var ps []*KeyValuePair
	//err := json.Unmarshal([]byte(pairsString), &ps)
	//if err != nil {
	//	return err
	//}
	pairs := []*commonPb.KeyValuePair{}
	for _, p := range ps {
		key := p.Key
		val := []byte(p.Value)
		if p.Unique {
			val = []byte(fmt.Sprintf(templateStr, p.Value, h.threadId, loopId, time.Now().UnixNano()))
		}
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   key,
			Value: val,
		})
		if showKey {
			fmt.Printf("key:%s val:%s\n", key, val)
		}
	}

	payloadBytes, err := constructQueryPayload(chainId, contractName, method, pairs)
	if err != nil {
		return err
	}

	resp, err = sendRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_QUERY_CONTRACT,
		txId: txId, chainId: chainId}, orgId, userCrtPath, payloadBytes, nil)
	if err != nil {
		return err
	}

	if checkResult && resp.Code != commonPb.TxStatusCode_SUCCESS {
		return fmt.Errorf(respStr, resp)
	}

	return nil
}

type createContractHandler struct {
	threadId int
}

func (h *createContractHandler) handle(client apiPb.RpcNodeClient, sk3 crypto.PrivateKey, orgId string, userCrtPath string, loopId int, ps []*KeyValuePair) error {
	txId := utils.GetRandTxId()

	wasmBin, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		return err
	}
	var pairs []*commonPb.KeyValuePair
	payload, _ := utils.GenerateInstallContractPayload(fmt.Sprintf(templateStr, contractName, h.threadId, loopId, time.Now().Unix()),
		"1.0.0", commonPb.RuntimeType(runTime), wasmBin, pairs)

	//
	//method := syscontract.ContractManageFunction_INIT_CONTRACT.String()
	//
	//payload := &commonPb.Payload{
	//	ChainId: chainId,
	//	Contract: &commonPb.Contract{
	//		ContractName:    fmt.Sprintf(templateStr, contractName, h.threadId, loopId, time.Now().Unix()),
	//		ContractVersion: "1.0.0",
	//		RuntimeType:     commonPb.RuntimeType(runTime),
	//	},
	//	Method:      method,
	//	Parameters:  pairs,
	//	ByteCode:    wasmBin,
	//	Endorsement: nil,
	//}

	endorsement, err := acSign(payload)
	if err != nil {
		return err
	}

	resp, err = sendRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT,
		txId: txId, chainId: chainId}, orgId, userCrtPath, payload, endorsement)
	if err != nil {
		return err
	}

	if checkResult && resp.Code != commonPb.TxStatusCode_SUCCESS {
		return fmt.Errorf(respStr, resp)
	}

	return nil
}

type upgradeContractHandler struct {
	threadId int
}

func (h *upgradeContractHandler) handle(client apiPb.RpcNodeClient, sk3 crypto.PrivateKey, orgId string, userCrtPath string, loopId int, ps []*KeyValuePair) error {
	txId := utils.GetRandTxId()

	wasmBin, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		return err
	}

	var pairs []*commonPb.KeyValuePair
	payload, _ := GenerateUpgradeContractPayload(fmt.Sprintf(templateStr, contractName, h.threadId, loopId, time.Now().Unix()),
		version, commonPb.RuntimeType(runTime), wasmBin, pairs)
	payload.TxId = txId
	payload.ChainId = chainId
	payload.Timestamp = time.Now().Unix()
	endorsement, err := acSign(payload)
	if err != nil {
		return err
	}

	resp, err = sendRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT,
		txId: txId, chainId: chainId}, orgId, userCrtPath, payload, endorsement)
	if err != nil {
		return err
	}

	if checkResult && resp.Code != commonPb.TxStatusCode_SUCCESS {
		return fmt.Errorf(respStr, resp)
	}

	return nil
}

func GenerateUpgradeContractPayload(contractName, version string, runtimeType commonPb.RuntimeType, bytecode []byte,
	initParameters []*commonPb.KeyValuePair) (*commonPb.Payload, error) {
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.UpgradeContract_CONTRACT_NAME.String(),
		Value: []byte(contractName),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.UpgradeContract_CONTRACT_VERSION.String(),
		Value: []byte(version),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.UpgradeContract_CONTRACT_RUNTIME_TYPE.String(),
		Value: []byte(runtimeType.String()),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.UpgradeContract_CONTRACT_BYTECODE.String(),
		Value: bytecode,
	})
	for _, kv := range initParameters {
		pairs = append(pairs, kv)
	}
	payload := &commonPb.Payload{
		ContractName: syscontract.SystemContract_CONTRACT_MANAGE.String(),
		Method:       syscontract.ContractManageFunction_UPGRADE_CONTRACT.String(),
		Parameters:   pairs,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
	}
	return payload, nil
}

func sendRequest(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, msg *InvokerMsg,
	orgId, userCrtPath string, payload *commonPb.Payload, endorsers []*commonPb.EndorsementEntry) (*commonPb.TxResponse, error) {

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(time.Duration(requestTimeout)*time.Second)))
	defer cancel()

	file := fileCache.Read(userCrtPath)

	// 构造Sender
	senderFull := &acPb.Member{
		OrgId:      orgId,
		MemberInfo: *file,
		//IsFullCert: true,
	}

	var sender *acPb.Member
	if useShortCrt {
		certId, err := certCache.Read(userCrtPath, senderFull.MemberInfo, hashAlgo)
		if err != nil {
			return nil, fmt.Errorf("fail to compute the identity for certificate [%v]", err)
		}
		sender = &acPb.Member{
			OrgId:      senderFull.OrgId,
			MemberInfo: *certId,
			MemberType: acPb.MemberType_CERT_HASH,
		}
	} else {
		sender = senderFull
	}

	// 构造Header

	req := &commonPb.TxRequest{
		Payload: payload,
		Sender: &commonPb.EndorsementEntry{
			Signer: sender,
		},
	}
	if len(endorsers) > 0 {
		req.Endorsers = endorsers
	}
	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		return nil, err
	}

	signer, err := getSigner(sk3, senderFull)
	if err != nil {
		return nil, err
	}
	signBytes, err := signer.Sign("SHA256", rawTxBytes)
	if err != nil {
		return nil, err
	}

	req.Sender.Signature = signBytes

	result, err := client.SendRequest(ctx, req)
	if err != nil {
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
			return nil, fmt.Errorf("client.call err: deadline\n")
		}
		return nil, fmt.Errorf("client.call err: %v\n", err)
	}
	return result, nil
}
