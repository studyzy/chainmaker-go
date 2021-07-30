/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

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
		{"", false},
		{"a123456789B", true},
		{"1abc", true},
		{"0x60acF8D95fd365122e56F414b2C13D9dc7742A07", true},
		{"测试", false},
		{"aaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbcccccccccccccccccccccddddddddddddddddeeeeeeeeeeeeeeeeeeeeeeeffffffffffffffffffffgggggggggggggggggg", false},
	}
	for _, testcase := range tests {
		result := CheckContractNameFormat(testcase.name)
		assert.Equal(t, testcase.pass, result, testcase.name)
	}
}

func TestCheckEvmAddressFormat(t *testing.T) {
	tests := []struct {
		name string
		pass bool
	}{
		{"a", false},
		{"", false},
		{"a123456789B", false},
		{"1abc", false},
		{"0x60acF8D95fd365122e56F414b2C13D9dc7742A07", true},
		{"60acF8D95fd365122e56F414b2C13D9dc7742A07", true},
		{"测试", false},
		{"aaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbcccccccccccccccccccccddddddddddddddddeeeeeeeeeeeeeeeeeeeeeeeffffffffffffffffffffgggggggggggggggggg", false},
	}
	for _, testcase := range tests {
		result := CheckEvmAddressFormat(testcase.name)
		assert.Equal(t, testcase.pass, result, testcase.name)
	}
}

func TestCheckChainIdFormat(t *testing.T) {
	tests := []struct {
		name string
		pass bool
	}{
		{"a", true},
		{"", false},
		{"a123456789B", true},
		{"1abc", true},
		{"0x60acF8D95fd365122e56F414b2C13D", false},
		{"60acF8D95fd365122e56F414b2C13D", true},
		{"测试", false},
		{"aaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbcccccccccccccccccccccddddddddddddddddeeeeeeeeeeeeeeeeeeeeeeeffffffffffffffffffffgggggggggggggggggg", false},
	}
	for _, testcase := range tests {
		result := CheckChainIdFormat(testcase.name)
		assert.Equal(t, testcase.pass, result, testcase.name)
	}
}
func TestCheckTxIDFormat(t *testing.T) {
	tests := []struct {
		name string
		pass bool
	}{
		{"a", true},
		{"", false},
		{"a123456789B", true},
		{"1abc", true},
		{"hello world", false},
		{"Fan[asd-23:33]", true},
		{"0x60acF8D95fd365122e56F414b2C13D", true},
		{"60acF8D95fd365122e56F414b2C13D", true},
		{"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", true},
		{"aaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbcccccccccccccccccccccddddddddddddddddeeeeeeeeeeeeeeeeeeeeeeeffffffffffffffffffffgggggggggggggggggg", false},
	}
	for _, testcase := range tests {
		result := CheckTxIDFormat(testcase.name)
		assert.Equal(t, testcase.pass, result, testcase.name)
	}
}
