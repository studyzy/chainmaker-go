/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import "strings"

// ToCamelCase as: abc_def -> AbcDef
func ToCamelCase(field string) string {
	if field == "" {
		return ""
	}

	var str string
	temp := strings.Split(field, "_")
	for _, v := range temp {
		r := []rune(v)
		if len(r) > 0 {
			if r[0] >= 'a' && r[0] <= 'z' {
				r[0] -= 32
			}
			str += string(r)
		}
	}
	return str
}

func IsAnyBlank(args ...string) bool {
	for i := 0; i < len(args); i++ {
		if len(args[i]) == 0 || strings.TrimSpace(args[i]) == "" {
			return true
		}
	}
	return false
}

func IsAllBlank(args ...string) bool {
	for i := 0; i < len(args); i++ {
		if len(args[i]) != 0 && strings.TrimSpace(args[i]) != "" {
			return false
		}
	}
	return true
}
