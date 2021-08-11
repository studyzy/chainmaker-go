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
		{"1346478487042892172349946970630736756658846205592", false},
		{"440902816914877365849934251651973913683067062725", false},
		{"94250082384390137379817105468624512268792773790", false},
		{"810227444454088037217518958892195931403787508", false},
		{"4409028169148773658499342516519739136830670625", false},
		{"123", false},
		{"1233", false},
		{"12333", false},
		{"", false},
		{"a123456789B", false},
		{"1abc", false},
		{"0x60acF8D95fd365122e56F414b2C13D9dc7742A07", true},
		{"60acF8D95fd365122e56F414b2C13D9dc7742A07", true},
		{"测试", false},
		//{"aaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbcccccccccccccccccccccddddddddddddddddeeeeeeeeeeeeeeeeeeeeeeeffffffffffffffffffffgggggggggggggggggg", false},
		//{"a", false},
		//{"1346478487042892172349946970630736756658846205592", true},
		//{"440902816914877365849934251651973913683067062725", true},
		//{"94250082384390137379817105468624512268792773790", true},
		//{"810227444454088037217518958892195931403787508", true},
		//{"4409028169148773658499342516519739136830670625", true},
		//{"123", false},
		//{"1233", false},
		//{"12333", true},
		//{"", false},
		//{"a123456789B", false},
		//{"1abc", false},
		//{"0x60acF8D95fd365122e56F414b2C13D9dc7742A07", false},
		//{"60acF8D95fd365122e56F414b2C13D9dc7742A07", false},
		//{"测试", false},
		//{"aaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbcccccccccccccccccccccddddddddddddddddeeeeeeeeeeeeeeeeeeeeeeeffffffffffffffffffffgggggggggggggggggg", false},
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
