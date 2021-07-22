/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package contractmgr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckContractName(t *testing.T) {
	tests := []struct {
		name string
		pass bool
	}{
		{"a", true},
		{"a123456789B", true},
		{"1abc", true},
		{"0x60acF8D95fd365122e56F414b2C13D9dc7742A07", true},
		{"测试", false},
		{"aaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbcccccccccccccccccccccddddddddddddddddeeeeeeeeeeeeeeeeeeeeeeeffffffffffffffffffffgggggggggggggggggg", false},
	}
	for _, testcase := range tests {
		result := checkContractName(testcase.name)
		assert.Equal(t, testcase.pass, result, testcase.name)
	}
}
