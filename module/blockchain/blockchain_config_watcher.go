package blockchain

import (
	configPb "chainmaker.org/chainmaker-go/pb/protogo/config"
	"chainmaker.org/chainmaker-go/protocol"
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
