/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"encoding/csv"
	"fmt"
	"net/http"

	_ "net/http/pprof"
	"os"
	"os/signal"
	"path"
	"runtime"
	"syscall"
	"time"

	"chainmaker.org/chainmaker-go/blockchain"
	"chainmaker.org/chainmaker-go/module/monitor"
	"chainmaker.org/chainmaker-go/rpcserver"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	"code.cloudfoundry.org/bytefmt"
	"github.com/spf13/cobra"
)

var log = logger.GetLogger(logger.MODULE_CLI)

func StartCMD() *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Startup ChainMaker",
		Long:  "Startup ChainMaker",
		RunE: func(cmd *cobra.Command, _ []string) error {
			initLocalConfig(cmd)
			mainStart()
			fmt.Println("ChainMaker exit")
			return nil
		},
	}
	attachFlags(startCmd, []string{flagNameOfConfigFilepath})
	return startCmd
}

func mainStart() {
	if localconf.ChainMakerConfig.DebugConfig.IsTraceMemoryUsage {
		traceMemoryUsage()
	}

	// init chainmaker server
	chainMakerServer := blockchain.NewChainMakerServer()
	if err := chainMakerServer.Init(); err != nil {
		log.Errorf("chainmaker server init failed, %s", err.Error())
		return
	}

	// init rpc server
	rpcServer, err := rpcserver.NewRPCServer(chainMakerServer)
	if err != nil {
		log.Errorf("rpc server init failed, %s", err.Error())
		return
	}

	// init monitor server
	monitorServer := monitor.NewMonitorServer()

	//// p2p callback to validate
	//txpool.RegisterCallback(rpcServer.Gateway().Invoke)

	// new an error channel to receive errors
	errorC := make(chan error, 1)

	// handle exit signal in separate go routines
	go handleExitSignal(errorC)

	// start blockchains in separate go routines
	if err := chainMakerServer.Start(); err != nil {
		log.Errorf("chainmaker server startup failed, %s", err.Error())
		return
	}

	// start rpc server and listen in another go routine
	if err := rpcServer.Start(); err != nil {
		errorC <- err
	}

	// start monitor server and listen in another go routine
	if err := monitorServer.Start(); err != nil {
		errorC <- err
	}

	if localconf.ChainMakerConfig.PProfConfig.Enabled {
		startPProf()
	}

	printLogo()

	// listen error signal in main function
	errC := <-errorC
	if errC != nil {
		log.Error("chainmaker encounters error ", errC)
	}
	rpcServer.Stop()
	chainMakerServer.Stop()
	log.Info("All is stopped!")

}

func handleExitSignal(exitC chan<- error) {

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, os.Interrupt, syscall.SIGINT)
	defer signal.Stop(signalChan)

	for sig := range signalChan {
		log.Infof("received signal: %d (%s)", sig, sig)
		exitC <- nil
	}
}

func printLogo() {
	log.Infof(logo())
}

func startPProf() {
	go func() {
		addr := fmt.Sprintf(":%d", localconf.ChainMakerConfig.PProfConfig.Port)
		log.Infof("pprof start at [%s]", addr)
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			fmt.Println(err)
		}
	}()
}

func traceMemoryUsage() {
	go func() {
		p := path.Join(path.Dir(localconf.ChainMakerConfig.LogConfig.SystemLog.FilePath), "mem.csv")
		f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			panic(err)
		}
		w := csv.NewWriter(f)
		err = w.Write([]string{
			"Alloc", "TotalAlloc", "Sys", "Mallocs", "Frees", "HeapAlloc", "HeapSys",
			"HeapIdle", "HeapInuse", "HeapReleased", "HeapObjects", "StackInuse",
			"StackSys", "MSpanInuse", "MSpanSys", "MCacheInuse", "MCacheSys",
			"BuckHashSys", "GCSys", "OtherSys",
		})
		if err != nil {
			panic(err)
		}
		for range time.Tick(time.Second) {
			mem := new(runtime.MemStats)
			runtime.ReadMemStats(mem)
			err = w.Write([]string{
				bytefmt.ByteSize(mem.Alloc),
				bytefmt.ByteSize(mem.TotalAlloc),
				bytefmt.ByteSize(mem.Sys),
				bytefmt.ByteSize(mem.Mallocs),
				bytefmt.ByteSize(mem.Frees),
				bytefmt.ByteSize(mem.HeapAlloc),
				bytefmt.ByteSize(mem.HeapSys),
				bytefmt.ByteSize(mem.HeapIdle),
				bytefmt.ByteSize(mem.HeapInuse),
				bytefmt.ByteSize(mem.HeapReleased),
				bytefmt.ByteSize(mem.HeapObjects),
				bytefmt.ByteSize(mem.StackInuse),
				bytefmt.ByteSize(mem.StackSys),
				bytefmt.ByteSize(mem.MSpanInuse),
				bytefmt.ByteSize(mem.MSpanSys),
				bytefmt.ByteSize(mem.MCacheInuse),
				bytefmt.ByteSize(mem.MCacheSys),
				bytefmt.ByteSize(mem.BuckHashSys),
				bytefmt.ByteSize(mem.GCSys),
				bytefmt.ByteSize(mem.OtherSys),
			})
			if err != nil {
				panic(err)
			}
			w.Flush()
		}
	}()
}
