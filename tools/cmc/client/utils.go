/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"chainmaker.org/chainmaker/common/random/uuid"
	sdkPbCommon "chainmaker.org/chainmaker-sdk-go/pb/protogo/common"
	"errors"
	"fmt"
)

func checkProposalRequestResp(resp *sdkPbCommon.TxResponse, needContractResult bool) error {
	if resp.Code != sdkPbCommon.TxStatusCode_SUCCESS {
		return errors.New(resp.Message)
	}

	if needContractResult && resp.ContractResult == nil {
		return fmt.Errorf("contract result is nil")
	}

	if resp.ContractResult != nil && resp.ContractResult.Code != sdkPbCommon.ContractResultCode_OK {
		return errors.New(resp.ContractResult.Message)
	}

	return nil
}
func maxi(i, j int) int {
	if j > i {
		return j
	}
	return i
}

// return hex string format random transaction id with length = 64
func GetRandTxId() string {
	return uuid.GetUUID() + uuid.GetUUID()
}
