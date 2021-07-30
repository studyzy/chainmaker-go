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
	val := ToCamelCase("abc_def")
	assert.Equal(t, "AbcDef", val)

	val = ToCamelCase("aBc_deFdd")
	assert.Equal(t, "ABcDeFdd", val)

	val = ToCamelCase("abc_1f")
	assert.Equal(t, "Abc1f", val)
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

	r = IsAllBlank(nil)
	assert.True(t, r)

	r = IsAllBlank("")
	assert.True(t, r)

	r = IsAllBlank("", []byte(""), nil, make([]byte, 0))
	assert.True(t, r)
}
