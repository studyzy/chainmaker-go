/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbprovider

import "chainmaker.org/chainmaker-go/protocol"

type Provider interface {

	// GetDBHandle returns db handle given dbName
	GetDBHandle(dbName string) protocol.DBHandle

	// Close closes database
	Close() error
}
