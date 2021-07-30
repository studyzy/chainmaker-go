/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainconf

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesis, err := Genesis("./testdata/bc1.yml")
	require.Nil(t, err)
	fmt.Println(genesis)
}
