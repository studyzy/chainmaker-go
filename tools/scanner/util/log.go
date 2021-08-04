/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"regexp"
	"strconv"
	"strings"
)

type Log struct {
	Time     string `json:"time"`
	Level    string `json:"level"`
	Module   string `json:"module"`
	ChainId  string `json:"chainId"`
	Position string `json:"position"`
	Message  string `json:"message"`
}

func GetLog(msg string) *Log {
	log := &Log{}

	index := strings.Index(msg[:], "\t")
	log.Time = msg[0:index]

	startIndex := index + 2
	index = strings.Index(msg[startIndex:], "\t") + startIndex
	log.Level = msg[startIndex : index-1]

	startIndex = index + 2
	index = strings.Index(msg[startIndex:], "\t") + startIndex
	log.Module = msg[startIndex:index]

	if strings.Contains(log.Module, "@") {
		i := strings.Index(log.Module, "@")
		log.ChainId = log.Module[i+1:]
		log.Module = log.Module[:i-2]
	} else {
		log.Module = msg[startIndex : index-1]
	}

	startIndex = index + 1
	index = strings.Index(msg[startIndex:], "\t") + startIndex
	log.Position = msg[startIndex:index]

	log.Message = msg[index+1:]

	return log
}

func (l *Log) Replace(msg string) string {
	result := msg
	result = strings.Replace(result, "${time}", l.Time, -1)
	result = strings.Replace(result, "${level}", l.Level, -1)
	result = strings.Replace(result, "${module}", l.Module, -1)
	result = strings.Replace(result, "${chainId}", l.ChainId, -1)
	result = strings.Replace(result, "${position}", l.Position, -1)
	result = strings.Replace(result, "${message}", l.Message, -1)

	result = Replace(l.Message, result)

	return result
}

func Replace(template, msg string) string {
	templateReg := regexp.MustCompile(`\[.*?\]`)
	templateRegResult := templateReg.FindAllString(template, -1)

	msgReg := regexp.MustCompile(`\$\{[0-9]*\}`)
	msgRegResult := msgReg.FindAllString(msg, -1)

	result := msg
	for _, m := range msgRegResult {
		index, err := strconv.Atoi(m[2 : len(m)-1])
		if err != nil {
			continue
		}
		if index < 0 || index >= len(templateRegResult) {
			continue
		}
		t := templateRegResult[index][1 : len(templateRegResult[index])-1]
		result = strings.Replace(result, m, t, -1)
	}
	return result
}
