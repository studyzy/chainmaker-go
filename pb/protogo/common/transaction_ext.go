/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import "errors"

func (m *Transaction) GetSenderAccountId() []byte {
	if m != nil && m.Header != nil {
		return m.Header.Sender.MemberInfo
	}
	return nil
}

func (m *Transaction) GetContractName() (string, error) {
	if m == nil || m.Header == nil {
		return "", errors.New("null point")
	}
	if m.Header.TxType == TxType_INVOKE_USER_CONTRACT {
		var payload = &TransactPayload{}
		err := payload.Unmarshal(m.RequestPayload)
		if err != nil {
			return "", err
		}
		return payload.ContractName, nil
	}
	if m.Header.TxType == TxType_MANAGE_USER_CONTRACT {
		return ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), nil //TODO
	}
	if m.Header.TxType == TxType_UPDATE_CHAIN_CONFIG {
		return ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(), nil //TODO
	}
	return "", errors.New("unknown tx type " + m.Header.TxType.String())
}
