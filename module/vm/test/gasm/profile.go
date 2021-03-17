package main

import (
    "chainmaker.org/chainmaker-go/gasm"
    "chainmaker.org/chainmaker-go/logger"
    commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
    "chainmaker.org/chainmaker-go/protocol"
    "chainmaker.org/chainmaker-go/vm/test"
    "fmt"
    "github.com/spf13/cobra"
    "github.com/spf13/pflag"
    "log"
    "net/http"
    _ "net/http/pprof"
    "sync"
    "sync/atomic"
    "time"
)

const (
    flagTotalCallTimes = "total-call-times"
    flagVmGoroutineNum = "vm-goroutines-num"
    flagWasmFilePath   = "wasm-file-path"
    flagCertFilePath   = "cert-file-path"
)

var (
    totalCallTimes int64
    vmGoroutineNum int64
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
    contractId, txContext, byteCode := test.InitContextTest(commonPb.RuntimeType_GASM)

    if len(byteCode) == 0 {
        panic("error byteCode==0")
    }

    finishNum := int64(0)
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
        for j := int64(0); j < createNum; j++ {
            wg.Add(1)
            go func() {
                defer wg.Done()
                invokeCallContract("increase", int32(i), contractId, txContext, byteCode)
                end := time.Now().UnixNano() / 1e6
                finished := atomic.AddInt64(&finishNum, 1)
                //atomic.AddInt64(&finishNum, 1)
                if (end-start)/1000 > 0 && finished % 100 == 0 {
                   fmt.Printf("【tps】 %d/s 【spend】%dms, finished=%d \n",
                       finished * 1000/int64(end-start), end-start, finished)
                }
            }()
        }
        wg.Wait()
        end := time.Now().UnixNano() / 1e6
        fmt.Printf("finished %d task in %dms, average tps is %d, totalCallTimes: %d, vmGoroutineNum: %d, " +
            "createNum: %d, i: %d\n",
            finishNum, end-start, finishNum * 1000/(end-start), totalCallTimes, vmGoroutineNum, createNum, i)
    }
}

func invokeCallContract(method string, id int32, contractId *commonPb.ContractId, txContext protocol.TxSimContext, byteCode []byte) {
   parameters := make(map[string]string)
   test.BaseParam(parameters)
   //parameters["contract_name"] = test.ContractNameTest
   //parameters["method"] = "query"
   //parameters[protocol.ContractTxIdParam] = parameters["tx_id"]

   runtimeInstance := &gasm.RuntimeInstance{
       Log: logger.GetLogger(logger.MODULE_VM),
   }
   runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
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
    attachFlags(perfCmd, []string{flagTotalCallTimes, flagVmGoroutineNum, flagWasmFilePath, flagCertFilePath})
    return perfCmd
}

func initFlagSet() *pflag.FlagSet {
    flags := &pflag.FlagSet{}
    flags.Int64Var(&totalCallTimes, flagTotalCallTimes, 500000, "specify run times")
    flags.Int64Var(&vmGoroutineNum, flagVmGoroutineNum, 50000, "specify goroutines for vm")
    flags.StringVar(&wasmFilePath, flagWasmFilePath, "./counter.wasm", "specify the wasm file path")
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