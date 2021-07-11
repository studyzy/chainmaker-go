/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package contractmgr

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheckContractName(t *testing.T) {
	tests := []struct {
		name string
		pass bool
	}{
		{"a", true},
		{"a123456789B", true},
		{"1abc", false},
		{"测试", false},
	}
	for _, testcase := range tests {
		result := checkContractName(testcase.name)
		assert.Equal(t, testcase.pass, result)
	}
}
