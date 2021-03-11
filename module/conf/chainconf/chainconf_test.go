/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainconf

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGenesis(t *testing.T) {
	genesis, err := Genesis("./bc1.yml")
	require.Nil(t, err)
	fmt.Println(genesis)
}
