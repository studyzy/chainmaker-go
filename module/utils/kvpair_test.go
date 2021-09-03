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

func TestUnmarshalJsonStrKV2KVPairs(t *testing.T) {
	str := "[{\"key\":\"key\",\"value\":\"counter1\"}]"
	kvpair, err := UnmarshalJsonStrKV2KVPairs(str)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, len(kvpair))
	assert.EqualValues(t, "key", kvpair[0].Key)
	assert.EqualValues(t, "counter1", kvpair[0].Value)
}
