/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logger

// LogConfig: the config of log module
type LogConfig struct {
	ConfigFile string        `mapstructure:"config_file"`
	SystemLog  LogNodeConfig `mapstructure:"system"`
	BriefLog   LogNodeConfig `mapstructure:"brief"`
	EventLog   LogNodeConfig `mapstructure:"event"`
}

// LogNodeConfig: the log config of node
type LogNodeConfig struct {
	LogLevelDefault string            `mapstructure:"log_level_default"`
	LogLevels       map[string]string `mapstructure:"log_levels"`
	FilePath        string            `mapstructure:"file_path"`
	MaxAge          int               `mapstructure:"max_age"`
	RotationTime    int               `mapstructure:"rotation_time"`
	LogInConsole    bool              `mapstructure:"log_in_console"`
	ShowColor       bool              `mapstructure:"show_color"`
}
