/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package logger

import "testing"

var l = &GoLogger{}
var arg0 = &LogNodeConfig{
	FilePath: "/test",
	MaxAge:   123,
}

func TestGoLogger_Debug(t *testing.T) {
	l.Debug("message", 1)
	l.Debugf("%s-%d", arg0.FilePath, arg0.MaxAge)
	l.Debugw("config", arg0)
}
func TestGoLogger_Info(t *testing.T) {
	l.Info("message", 1)
	l.Infof("%s-%d", arg0.FilePath, arg0.MaxAge)
	l.Infow("config", arg0)
}
func TestGoLogger_Warn(t *testing.T) {
	l.Warn("message", 1)
	l.Warnf("%s-%d", arg0.FilePath, arg0.MaxAge)
	l.Warnw("config", arg0)
}
func TestGoLogger_Error(t *testing.T) {
	l.Error("message", 1)
	l.Errorf("%s-%d", arg0.FilePath, arg0.MaxAge)
	l.Errorw("config", arg0)
}
