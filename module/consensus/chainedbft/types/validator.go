/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package types

//Validator define chainedbft validator
type Validator struct {
	NodeID string `json:"nodeID,"`
	Index  uint64 `json:"index,"`
}
