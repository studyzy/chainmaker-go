/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"fmt"

	"chainmaker.org/chainmaker-go/chainconf"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/localconf"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type CommitBlock struct {
	store                 protocol.BlockchainStore
	log                   protocol.Logger
	snapshotManager       protocol.SnapshotManager
	ledgerCache           protocol.LedgerCache
	chainConf             protocol.ChainConf
	msgBus                msgbus.MessageBus
	metricBlockSize       *prometheus.HistogramVec // metric block size
	metricBlockCounter    *prometheus.CounterVec   // metric block counter
	metricTxCounter       *prometheus.CounterVec   // metric transaction counter
	metricBlockCommitTime *prometheus.HistogramVec // metric block commit time
}

type CommitBlockConf struct {
	Store                 protocol.BlockchainStore
	Log                   protocol.Logger
	SnapshotManager       protocol.SnapshotManager
	TxPool                protocol.TxPool
	LedgerCache           protocol.LedgerCache
	ChainConf             protocol.ChainConf
	MsgBus                msgbus.MessageBus
	MetricBlockSize       *prometheus.HistogramVec // metric block size
	MetricBlockCounter    *prometheus.CounterVec   // metric block counter
	MetricTxCounter       *prometheus.CounterVec   // metric transaction counter
	MetricBlockCommitTime *prometheus.HistogramVec // metric block commit time
}

func NewCommitBlock(cbConf *CommitBlockConf) *CommitBlock {
	commitBlock := &CommitBlock{
		store:           cbConf.Store,
		log:             cbConf.Log,
		snapshotManager: cbConf.SnapshotManager,
		ledgerCache:     cbConf.LedgerCache,
		chainConf:       cbConf.ChainConf,
		msgBus:          cbConf.MsgBus,
	}
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		commitBlock.metricBlockSize = cbConf.MetricBlockSize
		commitBlock.metricBlockCounter = cbConf.MetricBlockCounter
		commitBlock.metricTxCounter = cbConf.MetricTxCounter
		commitBlock.metricBlockCommitTime = cbConf.MetricBlockCommitTime
	}
	return commitBlock
}

//CommitBlock the action that all consensus types do when a block is committed
func (cb *CommitBlock) CommitBlock(
	block *commonpb.Block,
	rwSetMap map[string]*commonpb.TxRWSet,
	conEventMap map[string][]*commonpb.ContractEvent) (dbLasts, snapshotLasts, confLasts, otherLasts, pubEvent int64, err error) {
	// record block
	rwSet := RearrangeRWSet(block, rwSetMap)
	// record contract event
	events := rearrangeContractEvent(block, conEventMap)

	startDBTick := utils.CurrentTimeMillisSeconds()
	if err = cb.store.PutBlock(block, rwSet); err != nil {
		// if put db error, then panic
		cb.log.Error(err)
		panic(err)
	}
	dbLasts = utils.CurrentTimeMillisSeconds() - startDBTick

	// clear snapshot
	startSnapshotTick := utils.CurrentTimeMillisSeconds()
	if err = cb.snapshotManager.NotifyBlockCommitted(block); err != nil {
		err = fmt.Errorf("notify snapshot error [%d](hash:%x)",
			block.Header.BlockHeight, block.Header.BlockHash)
		cb.log.Error(err)
		return 0, 0, 0, 0, 0, err
	}
	snapshotLasts = utils.CurrentTimeMillisSeconds() - startSnapshotTick

	// notify chainConf to update config when config block committed
	startConfTick := utils.CurrentTimeMillisSeconds()
	cb.ledgerCache.SetLastCommittedBlock(block)
	if err = NotifyChainConf(block, cb.chainConf); err != nil {
		return 0, 0, 0, 0, 0, err
	}
	confLasts = utils.CurrentTimeMillisSeconds() - startConfTick

	// publish contract event
	var startPublishContractEventTick int64
	if len(events) > 0 {
		startPublishContractEventTick = utils.CurrentTimeMillisSeconds()
		cb.log.Infof("start publish contractEventsInfo: block[%d] ,time[%d]", block.Header.BlockHeight, startPublishContractEventTick)
		var eventsInfo []*commonpb.ContractEventInfo
		for _, t := range events {
			eventInfo := &commonpb.ContractEventInfo{
				BlockHeight:     block.Header.BlockHeight,
				ChainId:         block.Header.GetChainId(),
				Topic:           t.Topic,
				TxId:            t.TxId,
				ContractName:    t.ContractName,
				ContractVersion: t.ContractVersion,
				EventData:       t.EventData,
			}
			eventsInfo = append(eventsInfo, eventInfo)
		}
		cb.msgBus.Publish(msgbus.ContractEventInfo, &commonpb.ContractEventInfoList{ContractEvents: eventsInfo})
		pubEvent = utils.CurrentTimeMillisSeconds() - startPublishContractEventTick
	}
	startOtherTick := utils.CurrentTimeMillisSeconds()
	bi := &commonpb.BlockInfo{
		Block:     block,
		RwsetList: rwSet,
	}
	// synchronize new block height to consensus and sync module
	cb.msgBus.PublishSafe(msgbus.BlockInfo, bi)
	if err = cb.MonitorCommit(bi); err != nil {
		return 0, 0, 0, 0, 0, err
	}
	otherLasts = utils.CurrentTimeMillisSeconds() - startOtherTick

	return
}

func (cb *CommitBlock) MonitorCommit(bi *commonpb.BlockInfo) error {
	if !localconf.ChainMakerConfig.MonitorConfig.Enabled {
		return nil
	}
	raw, err := proto.Marshal(bi)
	if err != nil {
		cb.log.Errorw("marshal BlockInfo failed", "err", err)
		return err
	}
	(*cb.metricBlockSize).WithLabelValues(bi.Block.Header.ChainId).Observe(float64(len(raw)))
	(*cb.metricBlockCounter).WithLabelValues(bi.Block.Header.ChainId).Inc()
	(*cb.metricTxCounter).WithLabelValues(bi.Block.Header.ChainId).Add(float64(bi.Block.Header.TxCount))
	return nil
}

func NotifyChainConf(block *commonpb.Block, chainConf protocol.ChainConf) (err error) {
	if block != nil && block.GetTxs() != nil && len(block.GetTxs()) > 0 {
		tx := block.GetTxs()[0]
		if _, ok := chainconf.IsNativeTx(tx); ok {
			if err = chainConf.CompleteBlock(block); err != nil {
				return fmt.Errorf("chainconf block complete, %s", err)
			}
		}
	}
	return nil
}

func rearrangeContractEvent(block *commonpb.Block, conEventMap map[string][]*commonpb.ContractEvent) []*commonpb.ContractEvent {
	conEvent := make([]*commonpb.ContractEvent, 0)
	if conEventMap == nil {
		return conEvent
	}
	for _, tx := range block.Txs {
		if event, ok := conEventMap[tx.Header.TxId]; ok {
			for _, e := range event {
				conEvent = append(conEvent, e)
			}
		}
	}
	return conEvent
}
