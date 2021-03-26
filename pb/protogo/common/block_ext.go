/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import "encoding/hex"

func (b *Block) GetBlockHashStr() string {
	return hex.EncodeToString(b.Header.BlockHash)
}
