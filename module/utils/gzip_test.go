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

func TestGZipCompressBytes(t *testing.T) {
	input := []byte("Hello World!Hello World!Hello World!Hello World!Hello World!Hello World!Hello World!")
	t.Logf("input length:%d", len(input))
	zipData, err := GZipCompressBytes(input)
	assert.Nil(t, err)
	t.Logf("gziped length:%d", len(zipData))
	unzipData, err := GZipDeCompressBytes(zipData)
	assert.Nil(t, err)
	t.Logf("unzip text:%s", string(unzipData))
	assert.Equal(t, unzipData, input)
}
