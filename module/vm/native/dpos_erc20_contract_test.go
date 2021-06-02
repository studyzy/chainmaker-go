/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package native

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBigInteger(t *testing.T) {
	bigInteger := NewBigInteger("1024000000000000000000000000000000000000000000")
	require.NotNil(t, bigInteger)
	bigInteger.Add(NewBigInteger("1024"))
	require.Equal(t, "1024000000000000000000000000000000000000001024", bigInteger.String())
	bigInteger.Sub(NewBigInteger("1024"))
	require.Equal(t, "1024000000000000000000000000000000000000000000", bigInteger.String())
}

