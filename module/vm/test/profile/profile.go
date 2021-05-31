/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker-go/gasm"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
	"chainmaker.org/chainmaker-go/vm/test"
	"chainmaker.org/chainmaker-go/wasmer"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	flagTotalCallTimes = "total-call-times"
	flagVmGoroutineNum = "vm-goroutines-num"
	flagVmType         = "vm-type"
	flagWasmFilePath   = "wasm-file-path"
	flagCertFilePath   = "cert-file-path"
	flagReportFilePath = "report-file-path"
)

var (
	totalCallTimes int64
	vmGoroutineNum int64
	vmType         string
	wasmFilePath   string
	certFilePath   string
	reportFilePath string
)

func main() {
	go func() {
		if err := http.ListenAndServe(":6060", nil); err != nil {
			log.Fatal(err)
		}
	}()

	mainCmd := &cobra.Command{Use: "profile"}
	mainCmd.AddCommand(Perf())
	err := mainCmd.Execute()
	if err != nil {
		fmt.Println(err)
	}

	return
}

func startPerf() {
	test.WasmFile = wasmFilePath
	test.CertFilePath = certFilePath
	var contractId *commonPb.ContractId
	var txContext *test.TxContextMockTest
	var byteCode []byte
	var vmTypeInt int
	if strings.ToLower(vmType) == "gasm" {
		vmTypeInt = 0
		contractId, txContext, byteCode = test.InitContextTest(commonPb.RuntimeType_GASM)
	} else if strings.ToLower(vmType) == "wasmer" {
		vmTypeInt = 1
		contractId, txContext, byteCode = test.InitContextTest(commonPb.RuntimeType_WASMER)
	} else {
		log.Fatal("unknown vm type: ", vmType, ", only support gasm or wasmer")
	}

	if len(byteCode) == 0 {
		panic("error byteCode==0")
	}

	finishNum := int64(0)
	var gas int64
	start := time.Now().UnixNano() / 1e6
	for i := int64(0); i < totalCallTimes; {
		var createNum int64
		if i+vmGoroutineNum >= totalCallTimes {
			createNum = totalCallTimes - i
		} else {
			createNum = vmGoroutineNum
		}
		i += createNum
		wg := sync.WaitGroup{}
		pool := wasmer.NewVmPoolManager("Counter")
		for j := int64(0); j < createNum; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var result *commonPb.ContractResult
				if vmTypeInt == 0 {
					result = invokeContractOfGasm("increase", contractId, txContext, byteCode)
				} else {
					result = invokeContractOfWasmer("increase", contractId, txContext, pool, byteCode)
				}

				//end := time.Now().UnixNano() / 1e6
				finished := atomic.AddInt64(&finishNum, 1)
				if finished == 1 {
					gas = result.GasUsed
				}
				//if (end-start)/1000 > 0 && finished % 1000 == 0 {
				//   fmt.Printf("【tps】 %d/s 【spend】%dms, finished=%d, used gas=%d\n",
				//       finished * 1000/int64(end-start), end-start, finished, result.GasUsed)
				//}
			}()
		}
		wg.Wait()
		end := time.Now().UnixNano() / 1e6
		fmt.Printf("finished %d task in %dms, average tps is %d, totalCallTimes: %d, vmGoroutineNum: %d, "+
			"createNum: %d, i: %d, used gas: %d\n",
			finishNum, end-start, finishNum*1000/(end-start), totalCallTimes, vmGoroutineNum, createNum, i, gas)
	}
	end := time.Now().UnixNano() / 1e6
	var f *os.File
	var err error
	f, err = os.OpenFile(reportFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("open report file err", err)
		return
	}
	fileInfo, err := f.Stat()
	if err != nil {
		fmt.Println("file stat err: ", err)
		return
	}
	if fileInfo.Size() > 4096 {
		f.Close()
		os.Rename(reportFilePath, reportFilePath+"-"+time.Now().Format(time.RFC3339))
		f, err = os.OpenFile(reportFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("open report file again err", err)
			return
		}
	}
	defer f.Close()
	var reportStr string
	now := time.Now()
	if vmTypeInt == 0 {
		fmt.Printf("## Gasm VM Benchmark Test\n### Time: %d-%02d-%02d %02d:%02d:%02d\n"+
			"### Contract: %s\n### Tps: %d\n### Total call times: %d\n"+
			"### Go routines(concurrent instances): %d\n### Spend time: %d\n### Used gas: %d\n",
			now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(),
			wasmFilePath, finishNum*1000/(end-start), finishNum, vmGoroutineNum, end-start, gas)
		reportStr = fmt.Sprintf("<h1>Gasm VM Benchmark Test</h1>\n<h3>Time: %d-%02d-%02d %02d:%02d:%02d</h3>\n"+
			"<h3>Contract: %s</h3>\n<h3> Tps: %d</h3>\n<h3>Total call times: %d</h3>\n<h3>Go routines(concurrent instances): %d</h3>\n"+
			"<h3>Spend time: %dms</h3>\n<h3> Used gas: %d</h3>\n",
			now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(),
			wasmFilePath, finishNum*1000/(end-start), finishNum, vmGoroutineNum, end-start, gas)
	} else {
		fmt.Printf("## Wasmer VM Benchmark Test\n### Time: %d-%02d-%02d %02d:%02d:%02d\n"+
			"### Contract: %s\n### Tps: %d\n### Total call times: %d\n"+
			"### Go routines: %d\n### Concurrent instances: 10\n### Spend time: %d\n### Used gas: %d\n",
			now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(),
			wasmFilePath, finishNum*1000/(end-start), finishNum, vmGoroutineNum, end-start, gas)
		//reportStr = fmt.Sprintf("## Wasmer VM Benchmark Test\n### Time: %d-%02d-%02d %02d:%02d:%02d\n" +
		//    "### Contract: %s\n### Tps: %d\n### Total call times: %d\n### Go routines: %d\n" +
		//    "### Concurrent instances: 10\n### Spend time: %dms\n### Used gas: %d\n",
		//    now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(),
		//    wasmFilePath, finishNum * 1000/(end-start), finishNum, vmGoroutineNum, end-start, gas)
		reportStr = fmt.Sprintf("<h1>Wasmer VM Benchmark Test</h1>\n<h3>Time: %d-%02d-%02d %02d:%02d:%02d</h3>\n"+
			"<h3>Contract: %s</h3>\n<h3>Tps: %d</h3>\n<h3>Total call times: %d</h3>\n<h3>Go routines: %d</h3>\n"+
			"<h3>Concurrent instances: 10</h3>\n<h3>Spend time: %dms</h3>\n<h3>Used gas: %d</h3>\n",
			now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(),
			wasmFilePath, finishNum*1000/(end-start), finishNum, vmGoroutineNum, end-start, gas)
	}
	writeLen, err := f.Write([]byte(reportStr))
	if err != nil {
		fmt.Println("write file error: ", err)
	}
	if writeLen != len(reportStr) {
		fmt.Print("write file len is not equal real data len")
	}
}

func invokeContractOfGasm(method string, contractId *commonPb.ContractId, txContext protocol.TxSimContext,
	byteCode []byte) (contractResult *commonPb.ContractResult) {
	parameters := make(map[string]string)
	test.BaseParam(parameters)
	//parameters["contract_name"] = test.ContractNameTest
	//parameters["method"] = "query"
	//parameters[protocol.ContractTxIdParam] = parameters["tx_id"]

	runtimeInstance := &gasm.RuntimeInstance{
		Log: logger.GetLogger(logger.MODULE_VM),
	}
	return runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeContractOfWasmer(method string, contractId *commonPb.ContractId, txContext protocol.TxSimContext,
	pool *wasmer.VmPoolManager, byteCode []byte) (contractResult *commonPb.ContractResult) {
	parameters := make(map[string]string)
	test.BaseParam(parameters)
	//parameters["key"] = "key"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	return runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func Perf() *cobra.Command {
	perfCmd := &cobra.Command{
		Use:   "perf",
		Short: "start perf",
		Long:  "start perf",
		RunE: func(_ *cobra.Command, _ []string) error {
			startPerf()
			return nil
		},
	}
	attachFlags(perfCmd, []string{flagTotalCallTimes, flagVmGoroutineNum, flagVmType, flagWasmFilePath,
		flagCertFilePath, flagReportFilePath})
	return perfCmd
}

func initFlagSet() *pflag.FlagSet {
	flags := &pflag.FlagSet{}
	flags.Int64Var(&totalCallTimes, flagTotalCallTimes, 500000, "specify run times")
	flags.Int64Var(&vmGoroutineNum, flagVmGoroutineNum, 50000, "specify goroutines for vm")
	flags.StringVar(&vmType, flagVmType, "gasm", "specify vm type")
	flags.StringVar(&wasmFilePath, flagWasmFilePath, "./counter-go.wasm", "specify the wasm file path")
	flags.StringVar(&certFilePath, flagCertFilePath, "./client1.sign.crt", "specify user's cert file")
	flags.StringVar(&reportFilePath, flagReportFilePath, "./Vm Performance Report.html", "specify the report file")
	return flags
}

func attachFlags(cmd *cobra.Command, flagNames []string) {
	flags := initFlagSet()
	cmdFlags := cmd.Flags()
	for _, flagName := range flagNames {
		if flag := flags.Lookup(flagName); flag != nil {
			cmdFlags.AddFlag(flag)
		}
	}
}
