/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package logger

import "log"

//GoLogger is a golang system log implementation of protocol.Logger, it's for unit test
type GoLogger struct{}

func (GoLogger) Debug(args ...interface{}) {
	log.Printf("DEBUG: %v", args)
}

func (GoLogger) Debugf(format string, args ...interface{}) {
	log.Printf("DEBUG: "+format, args...)
}

func (GoLogger) Debugw(msg string, keysAndValues ...interface{}) {
	log.Printf("DEBUG: "+msg+" %v", keysAndValues...)
}

func (GoLogger) Error(args ...interface{}) {
	log.Printf("ERROR: %v", args)
}

func (GoLogger) Errorf(format string, args ...interface{}) {
	log.Printf("ERROR: "+format, args...)
}

func (GoLogger) Errorw(msg string, keysAndValues ...interface{}) {
	log.Printf("ERROR: "+msg+" %v", keysAndValues...)
}

func (GoLogger) Fatal(args ...interface{}) {
	log.Fatal(args...)
}

func (GoLogger) Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

func (GoLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	log.Fatalf(msg+" %v", keysAndValues...)
}

func (GoLogger) Info(args ...interface{}) {
	log.Printf("INFO: %v", args)
}

func (GoLogger) Infof(format string, args ...interface{}) {
	log.Printf("INFO: "+format, args...)
}

func (GoLogger) Infow(msg string, keysAndValues ...interface{}) {
	log.Printf("INFO: "+msg+" %v", keysAndValues...)
}

func (GoLogger) Panic(args ...interface{}) {
	log.Panic(args...)
}

func (GoLogger) Panicf(format string, args ...interface{}) {
	log.Panicf(format, args...)
}

func (GoLogger) Panicw(msg string, keysAndValues ...interface{}) {
	log.Panicf(msg+" %v", keysAndValues...)
}

func (GoLogger) Warn(args ...interface{}) {
	log.Printf("WARN: %v", args)
}

func (GoLogger) Warnf(format string, args ...interface{}) {
	log.Printf("WARN: "+format, args...)
}

func (GoLogger) Warnw(msg string, keysAndValues ...interface{}) {
	log.Printf("WARN: "+msg+" %v", keysAndValues...)
}
