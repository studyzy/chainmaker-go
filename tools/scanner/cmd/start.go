/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"chainmaker.org/chainmaker-go/tools/scanner/config"
	"chainmaker.org/chainmaker-go/tools/scanner/core"
	"github.com/spf13/cobra"
)

var (
	configPath string
)

func StartCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start command",
		Long:  "Start command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return start()
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "../tools/scanner/config.yml", "specify config file path")

	return cmd
}

func start() error {
	scanConfig, err := config.LoadScanConfig(configPath)
	if err != nil {
		return err
	}
	logScanners := []core.LogScanner{}
	for _, fileConfig := range scanConfig.FileConfigs {
		logScanner, err := core.NewLogScanner(fileConfig)
		if err != nil {
			return err
		}
		go logScanner.Start()
		logScanners = append(logScanners, logScanner)
	}
	errorC := make(chan error, 1)
	go handleExitSignal(errorC)
	e := <-errorC
	if e != nil {
		return e
	}
	for _, logScanner := range logScanners {
		logScanner.Stop()
	}
	fmt.Println("All is stopped!")

	return nil
}

func handleExitSignal(exitC chan<- error) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, os.Interrupt, syscall.SIGINT)
	defer signal.Stop(signalChan)

	for range signalChan {
		exitC <- nil
	}
}
