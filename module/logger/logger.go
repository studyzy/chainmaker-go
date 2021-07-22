/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package logger used to get a logger for modules to write log
package logger

import (
	"strings"
	"sync"

	"go.uber.org/zap/zapcore"

	"chainmaker.org/chainmaker/common/log"
	"go.uber.org/zap"
)

const (
	// output system.log
	MODULE_BLOCKCHAIN = "[Blockchain]"
	MODULE_NET        = "[Net]"
	MODULE_STORAGE    = "[Storage]"
	MODULE_SNAPSHOT   = "[Snapshot]"
	MODULE_CONSENSUS  = "[Consensus]"
	MODULE_TXPOOL     = "[TxPool]"
	MODULE_CORE       = "[Core]"
	MODULE_VM         = "[Vm]"
	MODULE_RPC        = "[Rpc]"
	MODULE_LEDGER     = "[Ledger]"
	MODULE_CLI        = "[Cli]"
	MODULE_CHAINCONF  = "[ChainConf]"
	MODULE_ACCESS     = "[Access]"
	MODULE_MONITOR    = "[Monitor]"
	MODULE_SYNC       = "[Sync]"
	MODULE_DPOS       = "[DPoS]"
	// output brief.log
	MODULE_BRIEF = "[Brief]"

	// output to event.log
	MODULE_EVENT = "[Event]"
)

var (
	// map[module-name]map[module-name+chainId]zap.AtomicLevel
	loggerLevels = make(map[string]map[string]zap.AtomicLevel)
	loggerMutex  sync.Mutex
	logConfig    *LogConfig

	// map[moduleName+chainId]*CMLogger
	cmLoggers = sync.Map{}
)

// CMLogger is an implementation of chainmaker logger.
type CMLogger struct {
	zlog     *zap.SugaredLogger
	name     string
	chainId  string
	lock     sync.RWMutex
	logLevel log.LOG_LEVEL
}

func (l *CMLogger) Logger() *zap.SugaredLogger {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.zlog
}

func (l *CMLogger) Debug(args ...interface{}) {
	l.zlog.Debug(args...)
}
func (l *CMLogger) Debugf(format string, args ...interface{}) {
	l.zlog.Debugf(format, args...)
}
func (l *CMLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.zlog.Debugw(msg, keysAndValues...)
}
func (l *CMLogger) Error(args ...interface{}) {
	l.zlog.Error(args...)
}
func (l *CMLogger) Errorf(format string, args ...interface{}) {
	l.zlog.Errorf(format, args...)
}
func (l *CMLogger) Errorw(msg string, keysAndValues ...interface{}) {
	l.zlog.Errorw(msg, keysAndValues...)
}
func (l *CMLogger) Fatal(args ...interface{}) {
	l.zlog.Fatal(args...)
}
func (l *CMLogger) Fatalf(format string, args ...interface{}) {
	l.zlog.Fatalf(format, args...)
}
func (l *CMLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.zlog.Fatalw(msg, keysAndValues...)
}
func (l *CMLogger) Info(args ...interface{}) {
	l.zlog.Info(args...)
}
func (l *CMLogger) Infof(format string, args ...interface{}) {
	l.zlog.Infof(format, args...)
}
func (l *CMLogger) Infow(msg string, keysAndValues ...interface{}) {
	l.zlog.Infow(msg, keysAndValues...)
}
func (l *CMLogger) Panic(args ...interface{}) {
	l.zlog.Panic(args...)
}
func (l *CMLogger) Panicf(format string, args ...interface{}) {
	l.zlog.Panicf(format, args...)
}
func (l *CMLogger) Panicw(msg string, keysAndValues ...interface{}) {
	l.zlog.Panicw(msg, keysAndValues...)
}
func (l *CMLogger) Warn(args ...interface{}) {
	l.zlog.Warn(args...)
}
func (l *CMLogger) Warnf(format string, args ...interface{}) {
	l.zlog.Warnf(format, args...)
}
func (l *CMLogger) Warnw(msg string, keysAndValues ...interface{}) {
	l.zlog.Warnw(msg, keysAndValues...)
}

func (l *CMLogger) DebugDynamic(getStr func() string) {
	if l.logLevel == log.LEVEL_DEBUG {
		str := getStr()
		l.zlog.Debug(str)
	}
}
func (l *CMLogger) InfoDynamic(getStr func() string) {
	if l.logLevel == log.LEVEL_DEBUG || l.logLevel == log.LEVEL_INFO {
		l.zlog.Info(getStr())
	}
}

// SetLogger set logger.
func (l *CMLogger) SetLogger(logger *zap.SugaredLogger) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.zlog = logger
}

// newCMLogger create a new CMLogger.
func newCMLogger(name string, chainId string, logger *zap.SugaredLogger, logLevel log.LOG_LEVEL) *CMLogger {
	return &CMLogger{name: name, chainId: chainId, zlog: logger, logLevel: logLevel}
}

// SetLogConfig set the config of logger module, called in initialization of config module
func SetLogConfig(config *LogConfig) {
	logConfig = config
	RefreshLogConfig(logConfig)
}

// GetLogger find or create a CMLogger with module name, usually called in initialization of all module.
// After one module get the logger, the module can use it forever until the program terminate.
func GetLogger(name string) *CMLogger {
	return GetLoggerByChain(name, "")
}

// GetLoggerByChain find the CMLogger object with module name and chainId, usually called in initialization of all module.
// One module can get a logger for each chain, then logger can be use forever until the program terminate.
func GetLoggerByChain(name, chainId string) *CMLogger {
	logHeader := name + chainId
	var logger *CMLogger
	loggerVal, ok := cmLoggers.Load(logHeader)
	if ok {
		logger, _ = loggerVal.(*CMLogger)
		return logger
	} else {
		zapLogger, logLevel := createLoggerByChain(name, chainId)

		logger = newCMLogger(name, chainId, zapLogger, logLevel)
		loggerVal, ok = cmLoggers.LoadOrStore(logHeader, logger)
		if ok {
			logger, _ = loggerVal.(*CMLogger)
		}
		return logger
	}
}

func createLoggerByChain(name, chainId string) (*zap.SugaredLogger, log.LOG_LEVEL) {
	var config log.LogConfig
	var pureName string

	if logConfig == nil {
		logConfig = DefaultLogConfig()
	}

	if logConfig.SystemLog.LogLevelDefault == "" {
		defaultLogNode := GetDefaultLogNodeConfig()
		config = log.LogConfig{
			Module:       "[DEFAULT]",
			ChainId:      chainId,
			LogPath:      defaultLogNode.FilePath,
			LogLevel:     log.GetLogLevel(defaultLogNode.LogLevelDefault),
			MaxAge:       defaultLogNode.MaxAge,
			RotationTime: defaultLogNode.RotationTime,
			JsonFormat:   false,
			ShowLine:     true,
			LogInConsole: defaultLogNode.LogInConsole,
			ShowColor:    defaultLogNode.ShowColor,
		}
	} else {
		if name == MODULE_BRIEF {
			config = log.LogConfig{
				Module:       name,
				ChainId:      chainId,
				LogPath:      logConfig.BriefLog.FilePath,
				LogLevel:     log.GetLogLevel(logConfig.BriefLog.LogLevelDefault),
				MaxAge:       logConfig.BriefLog.MaxAge,
				RotationTime: logConfig.BriefLog.RotationTime,
				JsonFormat:   false,
				ShowLine:     true,
				LogInConsole: logConfig.BriefLog.LogInConsole,
				ShowColor:    logConfig.BriefLog.ShowColor,
			}
		} else if name == MODULE_EVENT {
			config = log.LogConfig{
				Module:       name,
				ChainId:      chainId,
				LogPath:      logConfig.EventLog.FilePath,
				LogLevel:     log.GetLogLevel(logConfig.EventLog.LogLevelDefault),
				MaxAge:       logConfig.EventLog.MaxAge,
				RotationTime: logConfig.EventLog.RotationTime,
				JsonFormat:   false,
				ShowLine:     true,
				LogInConsole: logConfig.EventLog.LogInConsole,
				ShowColor:    logConfig.EventLog.ShowColor,
			}
		} else {
			pureName = strings.ToLower(strings.Trim(name, "[]"))
			value, exists := logConfig.SystemLog.LogLevels[pureName]
			if !exists {
				value = logConfig.SystemLog.LogLevelDefault
			}
			config = log.LogConfig{
				Module:       name,
				ChainId:      chainId,
				LogPath:      logConfig.SystemLog.FilePath,
				LogLevel:     log.GetLogLevel(value),
				MaxAge:       logConfig.SystemLog.MaxAge,
				RotationTime: logConfig.SystemLog.RotationTime,
				JsonFormat:   false,
				ShowLine:     true,
				LogInConsole: logConfig.SystemLog.LogInConsole,
				ShowColor:    logConfig.SystemLog.ShowColor,
			}
		}
	}
	logger, level := log.InitSugarLogger(&config)
	if pureName != "" {
		if _, exist := loggerLevels[pureName]; !exist {
			loggerLevels[pureName] = make(map[string]zap.AtomicLevel)
		}
		logHeader := name + chainId
		loggerLevels[pureName][logHeader] = level
	}
	return logger, config.LogLevel
}

func refreshAllLoggerOfCmLoggers() {
	cmLoggers.Range(func(_, value interface{}) bool {
		cmLogger, _ := value.(*CMLogger)
		newLogger, logLevel := createLoggerByChain(cmLogger.name, cmLogger.chainId)
		cmLogger.SetLogger(newLogger)
		cmLogger.logLevel = logLevel
		return true
	})
}

// RefreshLogConfig refresh log levels of modules at initiation time of log module
// or refresh log levels of modules dynamiclly at running time.
func RefreshLogConfig(config *LogConfig) {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	// scan loggerLevels and find the level from config, if can't find level, set it to default
	for name, loggers := range loggerLevels {
		var logLevevl zapcore.Level
		var strlevel string
		var exist bool
		if strlevel, exist = config.SystemLog.LogLevels[name]; !exist {
			strlevel = config.SystemLog.LogLevelDefault
		}
		switch log.GetLogLevel(strlevel) {
		case log.LEVEL_DEBUG:
			logLevevl = zap.DebugLevel
		case log.LEVEL_INFO:
			logLevevl = zap.InfoLevel
		case log.LEVEL_WARN:
			logLevevl = zap.WarnLevel
		case log.LEVEL_ERROR:
			logLevevl = zap.ErrorLevel
		default:
			logLevevl = zap.InfoLevel
		}
		for _, aLevel := range loggers {
			aLevel.SetLevel(logLevevl)
		}
	}

	refreshAllLoggerOfCmLoggers()
}

// DefaultLogConfig create default config for log module
func DefaultLogConfig() *LogConfig {
	defaultLogNode := GetDefaultLogNodeConfig()
	config := &LogConfig{
		SystemLog: LogNodeConfig{
			LogLevelDefault: defaultLogNode.LogLevelDefault,
			FilePath:        defaultLogNode.FilePath,
			MaxAge:          defaultLogNode.MaxAge,
			RotationTime:    defaultLogNode.RotationTime,
			LogInConsole:    defaultLogNode.LogInConsole,
		},
		BriefLog: LogNodeConfig{
			LogLevelDefault: defaultLogNode.LogLevelDefault,
			FilePath:        defaultLogNode.FilePath,
			MaxAge:          defaultLogNode.MaxAge,
			RotationTime:    defaultLogNode.RotationTime,
			LogInConsole:    defaultLogNode.LogInConsole,
		},
		EventLog: LogNodeConfig{
			LogLevelDefault: defaultLogNode.LogLevelDefault,
			FilePath:        defaultLogNode.FilePath,
			MaxAge:          defaultLogNode.MaxAge,
			RotationTime:    defaultLogNode.RotationTime,
			LogInConsole:    defaultLogNode.LogInConsole,
		},
	}
	return config
}

// GetDefaultLogNodeConfig create a default log config of node
func GetDefaultLogNodeConfig() LogNodeConfig {
	return LogNodeConfig{
		LogLevelDefault: log.DEBUG,
		FilePath:        "./default.log",
		MaxAge:          log.DEFAULT_MAX_AGE,
		RotationTime:    log.DEFAULT_ROTATION_TIME,
		LogInConsole:    true,
		ShowColor:       true,
	}
}
