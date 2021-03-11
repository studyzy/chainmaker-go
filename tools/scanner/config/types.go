/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

type RoleConfig struct {
	RoleType   string      `mapstructure:"type"`
	Level      string      `mapstructure:"level"`
	Regex      string      `mapstructure:"regex"`
	Message    string      `mapstructure:"message"`
	Email      bool        `mapstructure:"email"`
	WX         bool        `mapstructure:"wx"`
	WXWork     bool        `mapstructure:"wxwork"`
	MailConfig *MailConfig `mapstructure:"mail_config"`
}

type FileConfig struct {
	FileName    string        `mapstructure:"file_name"`
	RoleConfigs []*RoleConfig `mapstructure:"roles"`
}

type MailConfig struct {
	Address string `mapstructure:"address"`
}

type AlarmCenterConfig struct {
	SendMailURL string `mapstructure:"send_mail_url"`
}

type ScanConfig struct {
	FileConfigs       []*FileConfig      `mapstructure:"file_config"`
	AlarmCenterConfig *AlarmCenterConfig `mapstructure:"alram_center_config"`
}
