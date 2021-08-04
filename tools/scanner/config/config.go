/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"fmt"

	"github.com/spf13/viper"
)

var GlobalConfig *ScanConfig

func LoadScanConfig(path string) (*ScanConfig, error) {
	scanConfig := &ScanConfig{}

	if err := scanConfig.loadConfig(path); err != nil {
		return nil, fmt.Errorf("Load config [%s] failed, %s", path, err)
	}

	GlobalConfig = scanConfig
	return scanConfig, nil
}

func (c *ScanConfig) loadConfig(path string) error {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return err
	}

	return v.Unmarshal(c)
}
