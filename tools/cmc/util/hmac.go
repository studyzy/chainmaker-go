// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/hex"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/hash"
)

func SM3(data []byte) (string, error) {
	bz, err := hash.Get(crypto.HASH_TYPE_SM3, data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bz), nil
}
