package main

import (
    "chainmaker.org/chainmaker-go/gasm"
    "chainmaker.org/chainmaker-go/logger"
    commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
    "chainmaker.org/chainmaker-go/protocol"
    "chainmaker.org/chainmaker-go/vm/test"
    "chainmaker.org/chainmaker-go/wasmer"
    "fmt"
    "github.com/spf13/cobra"
    "github.com/spf13/pflag"
    "log"
    "net/http"
    _ "net/http/pprof"
    "os"
    "strings"
    "sync"
    "sync/atomic"
    "time"
)

const (
    flagTotalCallTimes = "total-call-times"
    flagVmGoroutineNum = "vm-goroutines-num"
    flagVmType = "vm-type"
    flagWasmFilePath   = "wasm-file-path"
    flagCertFilePath   = "cert-file-path"
)

var (
    totalCallTimes int64
    vmGoroutineNum int64
    vmType string
    wasmFilePath   string
    certFilePath string
)

func main()  {
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
        if i + vmGoroutineNum >= totalCallTimes {
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
        fmt.Printf("finished %d task in %dms, average tps is %d, totalCallTimes: %d, vmGoroutineNum: %d, " +
           "createNum: %d, i: %d, used gas: %d\n",
           finishNum, end-start, finishNum * 1000/(end-start), totalCallTimes, vmGoroutineNum, createNum, i, gas)
    }
    end := time.Now().UnixNano() / 1e6
    f, err := os.OpenFile("Vm Performance Report.md", os.O_APPEND|os.O_CREATE, 0644)
    if err != nil {
        fmt.Println("open report file err", err)
    }
    defer f.Close()
    if vmTypeInt == 0 {
        fmt.Printf("## Gasm VM Benchmarks\n### Contract: %s\n### Tps: %d\n### Total call times: %d\n" +
            "### Go routines(concurrent instances): %d\n### Spend time: %d\n### Used gas: %d\n",
            wasmFilePath, finishNum * 1000/(end-start), finishNum, vmGoroutineNum, end-start, gas)
        f.Write([]byte(fmt.Sprintf("## Gasm VM Benchmarks\n###Contract: %s\n" +
            "### Tps: %d\n### Total call times: %d\n### Go routines(concurrent instances): %d\n" +
            "### Spend time: %d\n### Used gas: %d\n",
            wasmFilePath, finishNum*1000/(end-start), finishNum, vmGoroutineNum, end-start, gas)))
    } else {
        fmt.Printf("## Wasmer VM Benchmarks\n### Contract: %s\n### Tps: %d\n### Total call times: %d\n" +
            "### Go routines: %d\n### Concurrent instances: 10\n### Spend time: %d\n### Used gas: %d\n",
            wasmFilePath, finishNum * 1000/(end-start), finishNum, vmGoroutineNum, end-start, gas)
        f.Write([]byte(fmt.Sprintf("## Wasmer VM Benchmarks\n### Contract: %s\n" +
            "### Tps: %d\n### Total call times: %d\n### Go routines: %d\n### Concurrent instances: 10\n" +
            "### Spend time: %d\n### Used gas: %d\n",
            wasmFilePath, finishNum * 1000/(end-start), finishNum, vmGoroutineNum, end-start, gas)))
    }
}

func invokeContractOfGasm(method string, contractId *commonPb.ContractId, txContext protocol.TxSimContext,
    byteCode []byte)  (contractResult *commonPb.ContractResult) {
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
    attachFlags(perfCmd, []string{flagTotalCallTimes, flagVmGoroutineNum, flagVmType, flagWasmFilePath, flagCertFilePath})
    return perfCmd
}

func initFlagSet() *pflag.FlagSet {
    flags := &pflag.FlagSet{}
    flags.Int64Var(&totalCallTimes, flagTotalCallTimes, 500000, "specify run times")
    flags.Int64Var(&vmGoroutineNum, flagVmGoroutineNum, 50000, "specify goroutines for vm")
    flags.StringVar(&vmType, flagVmType, "gasm", "specify vm type")
    flags.StringVar(&wasmFilePath, flagWasmFilePath, "./counter-go.wasm", "specify the wasm file path")
    flags.StringVar(&certFilePath, flagCertFilePath, "./client1.sign.crt", "specify user's cert file")
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