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

func TestToCamelCase(t *testing.T) {
	param := make(map[string]string)
	param["abc_def"] = "AbcDef"
	param["aBc_deFdd"] = "ABcDeFdd"
	param["abc_1f"] = "Abc1f"
	param[""] = ""

	for k, v := range param {
		val := ToCamelCase(k)
		assert.Equal(t, v, val)
	}
}

func TestIsAnyBlank(t *testing.T) {
	r := IsAnyBlank("1", "1", []byte("1"), make([]byte, 2))
	assert.False(t, r)

	r = IsAnyBlank("", "1", []byte("1"), make([]byte, 2))
	assert.True(t, r)

	r = IsAnyBlank("1", " ", []byte("1"), make([]byte, 2))
	assert.True(t, r)

	r = IsAnyBlank("1", "1", []byte(""), make([]byte, 2))
	assert.True(t, r)

	r = IsAnyBlank("1", "1", []byte("1"), make([]byte, 0))
	assert.True(t, r)

	r = IsAnyBlank(nil)
	assert.True(t, r)

	r = IsAnyBlank("1", []byte("1"), nil)
	assert.True(t, r)
}
func TestIsAllBlank(t *testing.T) {
	r := IsAllBlank("1", "1", []byte("1"), make([]byte, 2))
	assert.False(t, r)

	r = IsAllBlank("", "1", []byte("1"), make([]byte, 2))
	assert.False(t, r)

	r = IsAllBlank("1", " ", []byte("1"), make([]byte, 2))
	assert.False(t, r)

	r = IsAllBlank("1", "1", []byte(""), make([]byte, 2))
	assert.False(t, r)

	r = IsAllBlank("1", "1", []byte("1"), make([]byte, 0))
	assert.False(t, r)

	r = IsAllBlank([]byte("1"))
	assert.False(t, r)

	r = IsAllBlank(nil)
	assert.True(t, r)

	r = IsAllBlank("")
	assert.True(t, r)

	r = IsAllBlank("", []byte(""), nil, make([]byte, 0))
	assert.True(t, r)
}
