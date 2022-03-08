/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"fmt"
	"strconv"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/localconf/v2"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
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
	conEventMap map[string][]*commonpb.ContractEvent) (
	dbLasts, snapshotLasts, confLasts, otherLasts, pubEventLasts int64, blockInfo *commonpb.BlockInfo, err error) {
	// record block
	rwSet := RearrangeRWSet(block, rwSetMap)
	// record contract event
	events := rearrangeContractEvent(block, conEventMap)

	// notify chainConf to update config before put block
	startConfTick := utils.CurrentTimeMillisSeconds()
	if err = cb.NotifyMessage(block, cb.chainConf); err != nil {
		return 0, 0, 0, 0, 0, nil, err
	}
	confLasts = utils.CurrentTimeMillisSeconds() - startConfTick

	// put block
	startDBTick := utils.CurrentTimeMillisSeconds()
	if err = cb.store.PutBlock(block, rwSet); err != nil {
		// if put db error, then panic
		cb.log.Error(err)
		panic(err)
	}
	cb.ledgerCache.SetLastCommittedBlock(block)
	dbLasts = utils.CurrentTimeMillisSeconds() - startDBTick

	// clear snapshot
	startSnapshotTick := utils.CurrentTimeMillisSeconds()
	if err = cb.snapshotManager.NotifyBlockCommitted(block); err != nil {
		err = fmt.Errorf("notify snapshot error [%d](hash:%x)",
			block.Header.BlockHeight, block.Header.BlockHash)
		cb.log.Error(err)
		return 0, 0, 0, 0, 0, nil, err
	}
	snapshotLasts = utils.CurrentTimeMillisSeconds() - startSnapshotTick

	// contract event
	pubEventLasts = cb.publishContractEvent(block, events)

	// monitor
	startOtherTick := utils.CurrentTimeMillisSeconds()
	blockInfo = &commonpb.BlockInfo{
		Block:     block,
		RwsetList: rwSet,
	}
	if err = cb.MonitorCommit(blockInfo); err != nil {
		return 0, 0, 0, 0, 0, nil, err
	}
	otherLasts = utils.CurrentTimeMillisSeconds() - startOtherTick

	return
}

// publishContractEvent publish contract event, return time used
func (cb *CommitBlock) publishContractEvent(block *commonpb.Block, events []*commonpb.ContractEvent) int64 {
	if len(events) == 0 {
		return 0
	}

	startPublishContractEventTick := utils.CurrentTimeMillisSeconds()
	cb.log.Infof(
		"start publish contractEventsInfo: block[%d] ,time[%d]",
		block.Header.BlockHeight,
		startPublishContractEventTick,
	)
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
	return utils.CurrentTimeMillisSeconds() - startPublishContractEventTick
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

func rearrangeContractEvent(block *commonpb.Block,
	conEventMap map[string][]*commonpb.ContractEvent) []*commonpb.ContractEvent {
	conEvent := make([]*commonpb.ContractEvent, 0)
	if conEventMap == nil {
		return conEvent
	}
	for _, tx := range block.Txs {
		if event, ok := conEventMap[tx.Payload.TxId]; ok {
			conEvent = append(conEvent, event...)
		}
	}
	return conEvent
}

func (cb *CommitBlock) NotifyMessage(block *commonpb.Block, chainConf protocol.ChainConf) (err error) {
	if block == nil || len(block.GetTxs()) == 0 {
		return nil
	}

	//if ok, _ := utils.IsNativeTx(block.GetTxs()[0]); !ok && utils.HasDPosTxWritesInHeader(block, chainConf) {
	//	return
	//}

	for _, tx := range block.Txs { // one by one
		if tx.Result == nil || tx.Result.ContractResult == nil {
			continue
		}
		for _, event := range tx.Result.ContractResult.ContractEvent {
			data := event.EventData
			if len(data) == 0 {
				continue
			}
			topicEnum, err := strconv.Atoi(event.Topic)
			if err != nil {
				continue
			}
			topic := msgbus.Topic(topicEnum)
			for _, payload := range data {
				cb.msgBus.PublishSync(topic, payload) // data is a string, base64(proto.Marshal(struct))
			}
		}
	}
	return nil
}
