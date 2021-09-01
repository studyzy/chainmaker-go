/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package blockchain

import (
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
)

var _ protocol.Watcher = (*Blockchain)(nil)

// Module
func (bc *Blockchain) Module() string {
	return "BlockChain"
}

// Watch
func (bc *Blockchain) Watch(_ *configPb.ChainConfig) error {
	if err := bc.Init(); err != nil {
		bc.log.Errorf("blockchain init failed when the configuration of blockchain updating, %s", err)
		return err
	}
	bc.StopOnRequirements()
	if err := bc.Start(); err != nil {
		bc.log.Errorf("blockchain start failed when the configuration of blockchain updating, %s", err)
		return err
	}
	return nil
}
