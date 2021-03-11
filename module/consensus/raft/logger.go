/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package raft

import "go.uber.org/zap"

// Logger implements raft.Logger interface with wraping zap.SugaredLogger
type Logger struct {
	*zap.SugaredLogger
}

// NewLogger creates a new Logger instance
func NewLogger(lg *zap.SugaredLogger) *Logger {
	return &Logger{
		SugaredLogger: lg,
	}
}

func (l *Logger) Warning(v ...interface{}) {
	l.SugaredLogger.Warn(v)
}

func (l *Logger) Warningf(format string, v ...interface{}) {
	l.SugaredLogger.Warnf(format, v)
}
