/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import "strings"

type Panic struct {
	Message string `json:"message"`
	Stack   string `json:"stack"`
}

func GetPanic(msgs []string) *Panic {
	panic := &Panic{
		Message: msgs[0][7:],
		Stack:   strings.Join(msgs[3:], "\n"),
	}
	return panic
}

func (p *Panic) Replace(msg string) string {
	result := msg
	result = strings.Replace(result, "${message}", p.Message, -1)
	result = strings.Replace(result, "${stack}", p.Stack, -1)

	result = Replace(p.Message, result)

	return result
}
