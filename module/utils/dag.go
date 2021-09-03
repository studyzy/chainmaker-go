/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"fmt"

	"chainmaker.org/chainmaker/common/v2/crypto/hash"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/gogo/protobuf/proto"
)

// CalcDagHash calculate DAG hash
func CalcDagHash(hashType string, dag *commonPb.DAG) ([]byte, error) {
	if dag == nil {
		return nil, fmt.Errorf("calc hash block == nil")
	}

	dagBytes, err := proto.Marshal(dag)
	if err != nil {
		return nil, fmt.Errorf("marshal DAG error, %s", err)
	}

	hashByte, err := hash.GetByStrType(hashType, dagBytes)
	if err != nil {
		return nil, err
	}
	return hashByte, nil
}
