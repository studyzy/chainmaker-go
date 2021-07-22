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

// IsAnyBlank args type only string/[]byte
func IsAnyBlank(args ...interface{}) bool {
	for i := 0; i < len(args); i++ {
		if args[i] == nil {
			return true
		}

		switch v := args[i].(type) {
		case string:
			if len(v) == 0 || strings.TrimSpace(v) == "" {
				return true
			}
		case []byte:
			if len(v) == 0 {
				return true
			}
		}
	}
	return false
}

// IsAllBlank args type only string/[]byte
func IsAllBlank(args ...interface{}) bool {
	for i := 0; i < len(args); i++ {
		if args[i] == nil {
			continue
		}

		switch v := args[i].(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return false
			}
		case []byte:
			if len(v) != 0 {
				return false
			}
		}
	}
	return true
}
