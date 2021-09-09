/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package localconf record all the values of the local config options.
package localconf

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	flagSets                        = make([]*pflag.FlagSet, 0)
	checkNewCmBlockChainConfigFlagC = make(chan struct{}, 1)
	// FindNewBlockChainNotifyC is the chan for finding new block chain configs.
	FindNewBlockChainNotifyC = make(chan string)
	// ChainMakerConfig is the CMConfig instance for global.
	ChainMakerConfig = &CMConfig{StorageConfig: map[string]interface{}{}}
)

// InitLocalConfig init local config.
func InitLocalConfig(cmd *cobra.Command) error {
	// 1. init CMConfig
	cmConfig, err := initCmConfig(cmd)
	if err != nil {
		return err
	}

	// 2. set log config
	logger.SetLogConfig(&cmConfig.LogConfig)

	// 3. set global chainmaker config
	ChainMakerConfig = cmConfig

	return nil
}

func initCmConfig(cmd *cobra.Command) (*CMConfig, error) {
	// 0. load env
	cmViper := viper.New()
	err := cmViper.BindPFlags(cmd.PersistentFlags())
	if err != nil {
		return nil, err
	}
	// 1. load the path of the config files
	ymlFile := ConfigFilepath
	if !filepath.IsAbs(ymlFile) {
		ymlFile, _ = filepath.Abs(ymlFile)
		ConfigFilepath = ymlFile
	}

	// 2. load the config file
	cmViper.SetConfigFile(ymlFile)
	if err := cmViper.ReadInConfig(); err != nil {
		return nil, err
	}
	logConfigFile := cmViper.GetString("log.config_file")
	if logConfigFile != "" {
		cmViper.SetConfigFile(logConfigFile)
		if err := cmViper.MergeInConfig(); err != nil {
			return nil, err
		}
	}

	for _, command := range cmd.Commands() {
		flagSets = append(flagSets, command.PersistentFlags())
		err := cmViper.BindPFlags(command.PersistentFlags())
		if err != nil {
			return nil, err
		}
	}

	// 3. create new CMConfig instance
	cmConfig := &CMConfig{}
	if err := cmViper.Unmarshal(cmConfig); err != nil {
		return nil, err
	}
	return cmConfig, nil
}

func CheckNewCmBlockChainConfig() error {
	select {
	case checkNewCmBlockChainConfigFlagC <- struct{}{}:
	default:
		return errors.New("the task is in progress. try again later pls")
	}
	defer func() { <-checkNewCmBlockChainConfigFlagC }()
	// 0. load env
	cmViper := viper.New()
	// 1. load the config file
	if err := loadConfigFile(cmViper); err != nil {
		return err
	}
	// 2. create new CMConfig instance
	newCmConfig := &CMConfig{}
	if err := cmViper.Unmarshal(newCmConfig); err != nil {
		return err
	}
	// 3. compare new CMConfig with the current
	compareThenSetNewCMConfigWithCurrent(newCmConfig)
	return nil
}

func loadConfigFile(cmViper *viper.Viper) error {
	ymlFile := ConfigFilepath
	if !filepath.IsAbs(ymlFile) {
		ymlFile, _ = filepath.Abs(ymlFile)
	}
	cmViper.SetConfigFile(ymlFile)
	if err := cmViper.ReadInConfig(); err != nil {
		return err
	}
	logConfigFile := cmViper.GetString("log.config_file")
	if logConfigFile != "" {
		cmViper.SetConfigFile(logConfigFile)
		if err := cmViper.MergeInConfig(); err != nil {
			return err
		}
	}
	for idx := range flagSets {
		err := cmViper.BindPFlags(flagSets[idx])
		if err != nil {
			return err
		}
	}
	return nil
}

func compareThenSetNewCMConfigWithCurrent(newCmConfig *CMConfig) {
	// 3.1 load existed chains.
	existedChain := make(map[string]struct{})
	for idx := range ChainMakerConfig.BlockChainConfig {
		existedChain[ChainMakerConfig.BlockChainConfig[idx].ChainId] = struct{}{}
	}
	// 3.2 make sure new chains existed
	newChainIdsFound := make([]string, 0)
	for idx := range newCmConfig.BlockChainConfig {
		newChainId := newCmConfig.BlockChainConfig[idx].ChainId
		_, ok := existedChain[newChainId]
		if !ok {
			// new chain found
			newChainIdsFound = append(newChainIdsFound, newChainId)
		}
	}
	if len(newChainIdsFound) > 0 {
		// 3.3 set new chain config
		if newCmConfig.NodeConfig.NodeId == "" {
			newCmConfig.SetNodeId(ChainMakerConfig.NodeConfig.NodeId)
		}
		ChainMakerConfig = newCmConfig
		// 3.4 send new block chain found notify
		for idx := range newChainIdsFound {
			FindNewBlockChainNotifyC <- newChainIdsFound[idx]
		}
	}
}

func initCmConfigForLogOnly() (*CMConfig, error) {
	// 1. load the path of the logger config files
	logYmlFile := ChainMakerConfig.LogConfig.ConfigFile

	// 2. create new viper
	cmViper := viper.New()
	cmViper.SetConfigFile(logYmlFile)
	if err := cmViper.ReadInConfig(); err != nil {
		return nil, err
	}
	// 3. create new CMConfig instance
	cmConfig := &CMConfig{}
	if err := cmViper.Unmarshal(cmConfig); err != nil {
		return nil, err
	}
	cmConfig.LogConfig.ConfigFile = logYmlFile
	return cmConfig, nil
}

// PrettyJson print with json.
func (c *CMConfig) PrettyJson() (string, error) {
	ret, err := prettyjson.Marshal(c)
	if err != nil {
		return "", err
	}

	return string(ret), nil
}

// SetNodeId - 设置NodeId
func (c *CMConfig) SetNodeId(nodeId string) {
	c.NodeConfig.NodeId = nodeId
}

// RefreshLogLevelsConfig refresh the levels of the loggers with the logger config file.
func RefreshLogLevelsConfig() error {
	newCmConfig, err := initCmConfigForLogOnly()
	if err != nil {
		return err
	}
	// refresh global loggers' logLevel
	logger.RefreshLogConfig(&newCmConfig.LogConfig)

	return nil
}

// UpdateDebugConfig refresh the switches of the debug mode.
func UpdateDebugConfig(pairs []*config.ConfigKeyValue) error {
	value := reflect.ValueOf(&ChainMakerConfig.DebugConfig)
	elem := value.Elem()
	for _, pair := range pairs {
		if _, ok := elem.Type().FieldByName(pair.Key); !ok {
			continue
		}
		elem.FieldByName(pair.Key).SetBool(strings.ToLower(pair.Value) == "true")
	}
	return nil
}
