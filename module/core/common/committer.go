/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"chainmaker.org/chainmaker-go/chainconf"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/monitor"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type CommitBlock struct {
	store                 protocol.BlockchainStore
	log                   *logger.CMLogger
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
	Store           protocol.BlockchainStore
	Log             *logger.CMLogger
	SnapshotManager protocol.SnapshotManager
	TxPool          protocol.TxPool
	LedgerCache     protocol.LedgerCache
	ChainConf       protocol.ChainConf
	MsgBus          msgbus.MessageBus
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
		commitBlock.metricBlockSize = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_COMMITTER, monitor.MetricBlockSize,
			monitor.HelpCurrentBlockSizeMetric, prometheus.ExponentialBuckets(1024, 2, 12), monitor.ChainId)

		commitBlock.metricBlockCounter = monitor.NewCounterVec(monitor.SUBSYSTEM_CORE_COMMITTER, monitor.MetricBlockCounter,
			monitor.HelpBlockCountsMetric, monitor.ChainId)

		commitBlock.metricTxCounter = monitor.NewCounterVec(monitor.SUBSYSTEM_CORE_COMMITTER, monitor.MetricTxCounter,
			monitor.HelpTxCountsMetric, monitor.ChainId)

		commitBlock.metricBlockCommitTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_COMMITTER, monitor.MetricBlockCommitTime,
			monitor.HelpBlockCommitTimeMetric, []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, monitor.ChainId)
	}
	return commitBlock
}

//CommitBlock the action that all consensus types do when a block is committed
func (cb *CommitBlock) CommitBlock(block *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet) error {
	startTick := utils.CurrentTimeMillisSeconds()
	// record block
	rwSet := RearrangeRWSet(block, rwSetMap)

	contractEventMap := make(map[string][]*commonpb.ContractEvent)
	for _, tx := range block.Txs {
		event := tx.Result.ContractResult.ContractEvent
		contractEventMap[tx.Header.TxId] = event
	}
	// record contract event
	events := cb.rearrangeContractEvent(block, contractEventMap)

	startDBTick := utils.CurrentTimeMillisSeconds()
	if err := cb.store.PutBlock(block, rwSet); err != nil {
		// if put db error, then panic
		cb.log.Error(err)
		panic(err)
	}
	dbLasts := utils.CurrentTimeMillisSeconds() - startDBTick

	// clear snapshot
	startSnapshotTick := utils.CurrentTimeMillisSeconds()
	if err := cb.snapshotManager.NotifyBlockCommitted(block); err != nil {
		err = fmt.Errorf("notify snapshot error [%d](hash:%x)",
			block.Header.BlockHeight, block.Header.BlockHash)
		cb.log.Error(err)
		return err
	}
	snapshotLasts := utils.CurrentTimeMillisSeconds() - startSnapshotTick

	// notify chainConf to update config when config block committed
	startConfTick := utils.CurrentTimeMillisSeconds()
	if err := notifyChainConf(block, cb.chainConf); err != nil {
		return err
	}
	confLasts := utils.CurrentTimeMillisSeconds() - startConfTick

	// publish contract event
	var startPublishContractEventTick int64
	var pubEvent int64
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
		cb.msgBus.Publish(msgbus.ContractEventInfo, eventsInfo)
		pubEvent = utils.CurrentTimeMillisSeconds() - startPublishContractEventTick
	}
	startOtherTick := utils.CurrentTimeMillisSeconds()
	cb.ledgerCache.SetLastCommittedBlock(block)
	bi := &commonpb.BlockInfo{
		Block:     block,
		RwsetList: rwSet,
	}
	// synchronize new block height to consensus and sync module
	cb.msgBus.Publish(msgbus.BlockInfo, bi)
	if err := cb.monitorCommit(bi); err != nil {
		return err
	}
	otherLasts := utils.CurrentTimeMillisSeconds() - startOtherTick
	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	cb.log.Infof("commit block [%d](count:%d,hash:%x), time used(db:%d,ss:%d,conf:%d,pubConEvent:%d,other:%d,total:%d)",
		block.Header.BlockHeight, block.Header.TxCount, block.Header.BlockHash, dbLasts, snapshotLasts, confLasts, pubEvent, otherLasts, elapsed)
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		cb.metricBlockCommitTime.WithLabelValues(cb.chainConf.ChainConfig().ChainId).Observe(float64(elapsed) / 1000)
	}
	return nil
}

func (cb *CommitBlock) monitorCommit(bi *commonpb.BlockInfo) error {
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

func notifyChainConf(block *commonpb.Block, chainConf protocol.ChainConf) (err error) {
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

func (cb *CommitBlock) rearrangeContractEvent(block *commonpb.Block, conEventMap map[string][]*commonpb.ContractEvent) []*commonpb.ContractEvent {
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
