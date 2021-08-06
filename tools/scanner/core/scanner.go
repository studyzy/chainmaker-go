/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/tools/scanner/config"
	"chainmaker.org/chainmaker-go/tools/scanner/util"
	"github.com/hpcloud/tail"
)

type LogScanner interface {
	Start()
	Stop()
}

const (
	defaultBufferSize = 100
	PANIC             = "panic"
	format            = `^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]\ [0-9][0-9]\:[0-9][0-9]\:[0-9][0-9]\.[0-9][0-9][0-9]`
)

type logScannerImpl struct {
	config     *config.FileConfig
	buffer     []string
	bufferSize int
	tail       *tail.Tail

	index          int
	panicStartLine int

	stopOnce *sync.Once
	stopCh   chan struct{}
}

func NewLogScanner(config *config.FileConfig) (LogScanner, error) {
	l := &logScannerImpl{
		config:         config,
		buffer:         make([]string, defaultBufferSize),
		bufferSize:     defaultBufferSize,
		panicStartLine: -1,
		stopOnce:       &sync.Once{},
		stopCh:         make(chan struct{}),
	}
	var err error
	l.tail, err = tail.TailFile(l.config.FileName, tail.Config{Follow: true})
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (l *logScannerImpl) Start() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-l.stopCh:
			ticker.Stop()
			return
		case line := <-l.tail.Lines:
			l.buffer[l.index] = line.Text

			for _, role := range l.config.RoleConfigs {
				if l.panicStartLine >= 0 && strings.ToLower(role.RoleType) == PANIC {
					l.handlePanic(role, true)
				}

				l.handle(role, line.Text)
			}

			l.index = (l.index + 1) % l.bufferSize
		case <-ticker.C:
			for _, role := range l.config.RoleConfigs {
				if l.panicStartLine >= 0 && strings.ToLower(role.RoleType) == PANIC {
					l.handlePanic(role, false)
				}
			}
		}
	}
}

func (l *logScannerImpl) Stop() {
	stopFunc := func() {
		close(l.stopCh)
		err := l.tail.Stop()
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	l.stopOnce.Do(stopFunc)
}

func (l *logScannerImpl) handle(role *config.RoleConfig, line string) {
	var msg string
	switch strings.ToLower(role.RoleType) {
	case "normal":
		msg = l.handleNormal(role, line)
	case PANIC:
		if strings.HasPrefix(line, "panic: ") {
			l.panicStartLine = l.index
		}
	}

	l.sendMsg(role, msg)
}

func (l *logScannerImpl) handleNormal(role *config.RoleConfig, line string) string {
	logReg := regexp.MustCompile(format)
	if len(logReg.FindAllStringIndex(line, -1)) == 0 {
		return ""
	}

	reg := regexp.MustCompile(role.Regex)
	if reg != nil {
		result := reg.FindAllStringIndex(line, -1)
		if len(result) > 0 {
			log := util.GetLog(line)
			if log.Level == strings.ToUpper(role.Level) {
				return log.Replace(role.Message)
			}
		}
	}

	return ""
}

func (l *logScannerImpl) sendMsg(role *config.RoleConfig, msg string) {
	if msg != "" {
		if role.Email {
			err := l.email(role, msg)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
		//if role.WX {
		//	l.wx(role, msg)
		//}
		//if role.WXWork {
		//	l.wxwork(role, msg)
		//}
	}
}

func (l *logScannerImpl) handlePanic(role *config.RoleConfig, newLine bool) {
	if newLine {

		logReg := regexp.MustCompile(format)
		panicReg := regexp.MustCompile(`^panic\:`)
		line := l.buffer[l.index]
		if len(logReg.FindAllStringIndex(line, -1)) == 0 && len(panicReg.FindAllStringIndex(line, -1)) == 0 {
			return
		}
	}
	msgs := []string{}
	for i := l.panicStartLine; i != l.index; i = (i + 1) % l.bufferSize {
		msgs = append(msgs, l.buffer[i])
	}
	msg := util.GetPanic(msgs).Replace(role.Message)

	l.panicStartLine = -1

	if msg != "" {
		if role.Email {
			err := l.email(role, msg)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
		//if role.WX {
		//	l.wx(role, msg)
		//}
		//if role.WXWork {
		//	l.wxwork(role, msg)
		//}
	}
}

func (l *logScannerImpl) email(role *config.RoleConfig, msg string) error {
	return SendMail("日志扫描信息", msg, role.MailConfig.Address, "", "")
}

//func (l *logScannerImpl) wx(role *config.RoleConfig, msg string) error {
//	fmt.Println(msg)
//	return nil
//}
//
//func (l *logScannerImpl) wxwork(role *config.RoleConfig, msg string) error {
//	fmt.Println(msg)
//	return nil
//}
