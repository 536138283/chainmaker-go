/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"fmt"
	"strconv"

	"chainmaker.org/chainmaker/pb-go/v2/config"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/localconf/v2"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	blockVersion230 = uint32(230)
)

type CommitBlock struct {
	store                   protocol.BlockchainStore
	log                     protocol.Logger
	snapshotManager         protocol.SnapshotManager
	ledgerCache             protocol.LedgerCache
	chainConf               protocol.ChainConf
	txFilter                protocol.TxFilter
	msgBus                  msgbus.MessageBus
	metricBlockSize         *prometheus.HistogramVec // metric block size
	metricBlockCounter      *prometheus.CounterVec   // metric block counter
	metricTxCounter         *prometheus.CounterVec   // metric transaction counter
	metricTpsGauge          *prometheus.GaugeVec     // metric real-time transaction per second (TPS)
	metricBlockCommitTime   *prometheus.HistogramVec // metric block commit time
	metricBlockIntervalTime *prometheus.HistogramVec // metric block interval time
}

type CommitBlockConf struct {
	Store                   protocol.BlockchainStore
	Log                     protocol.Logger
	SnapshotManager         protocol.SnapshotManager
	TxPool                  protocol.TxPool
	LedgerCache             protocol.LedgerCache
	ChainConf               protocol.ChainConf
	TxFilter                protocol.TxFilter
	MsgBus                  msgbus.MessageBus
	MetricBlockSize         *prometheus.HistogramVec // metric block size
	MetricBlockCounter      *prometheus.CounterVec   // metric block counter
	MetricTxCounter         *prometheus.CounterVec   // metric transaction counter
	MetricTpsGauge          *prometheus.GaugeVec     // metric real-time transaction per second (TPS)
	MetricBlockCommitTime   *prometheus.HistogramVec // metric block commit time
	MetricBlockIntervalTime *prometheus.HistogramVec // metric block interval time
}

// NewCommitBlock new commit block
func NewCommitBlock(cbConf *CommitBlockConf) *CommitBlock {
	commitBlock := &CommitBlock{
		store:           cbConf.Store,
		txFilter:        cbConf.TxFilter,
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
		commitBlock.metricBlockIntervalTime = cbConf.MetricBlockIntervalTime
		commitBlock.metricTpsGauge = cbConf.MetricTpsGauge
	}
	return commitBlock
}

// CommitBlock the action that all consensus types do when a block is committed
func (cb *CommitBlock) CommitBlock(
	block *commonpb.Block,
	rwSetMap map[string]*commonpb.TxRWSet,
	conEventMap map[string][]*commonpb.ContractEvent) (
	dbLasts, snapshotLasts, confLasts, otherLasts, pubEventLasts, filterLasts int64, blockInfo *commonpb.BlockInfo, err error) {
	// record block
	rwSet := RearrangeRWSet(block, rwSetMap)
	// record contract event
	events := rearrangeContractEvent(block, conEventMap)

	if block.Header.BlockVersion >= blockVersion230 {
		// notify chainConf to update config before put block
		startConfTick := utils.CurrentTimeMillisSeconds()
		if err = cb.NotifyMessage(block, events); err != nil {
			return 0, 0, 0, 0, 0, 0, nil, err
		}
		confLasts = utils.CurrentTimeMillisSeconds() - startConfTick
	}
	// put block
	startDBTick := utils.CurrentTimeMillisSeconds()
	if err = cb.store.PutBlock(block, rwSet); err != nil {
		// if put db error, then panic
		cb.log.Error(err)
		panic(err)
	}
	cb.ledgerCache.SetLastCommittedBlock(block)
	dbLasts = utils.CurrentTimeMillisSeconds() - startDBTick

	// TxFilter adds
	filterLasts = utils.CurrentTimeMillisSeconds()
	// The default filter type does not run AddsAndSetHeight
	if localconf.ChainMakerConfig.TxFilter.Type != int32(config.TxFilterType_None) {
		err = cb.txFilter.AddsAndSetHeight(utils.GetTxIds(block.Txs), block.Header.GetBlockHeight())
		if err != nil {
			// if add filter error, then panic
			cb.log.Error(err)
			panic(err)
		}
	}
	filterLasts = utils.CurrentTimeMillisSeconds() - filterLasts

	// clear snapshot
	startSnapshotTick := utils.CurrentTimeMillisSeconds()
	if err = cb.snapshotManager.NotifyBlockCommitted(block); err != nil {
		err = fmt.Errorf("notify snapshot error [%d](hash:%x)",
			block.Header.BlockHeight, block.Header.BlockHash)
		cb.log.Error(err)
		return 0, 0, 0, 0, 0, 0, nil, err
	}
	snapshotLasts = utils.CurrentTimeMillisSeconds() - startSnapshotTick
	// v220_compat Deprecated
	if block.Header.BlockVersion < blockVersion230 {
		// notify chainConf to update config when config block committed
		startConfTick := utils.CurrentTimeMillisSeconds()
		if err = NotifyChainConf(block, cb.chainConf); err != nil {
			return 0, 0, 0, 0, 0, 0, nil, err
		}
		confLasts = utils.CurrentTimeMillisSeconds() - startConfTick
	}
	// contract event
	pubEventLasts = cb.publishContractEvent(block, events)

	// monitor
	startOtherTick := utils.CurrentTimeMillisSeconds()
	blockInfo = &commonpb.BlockInfo{
		Block:     block,
		RwsetList: rwSet,
	}
	// async is ok
	go cb.MonitorCommit(blockInfo)
	otherLasts = utils.CurrentTimeMillisSeconds() - startOtherTick

	return
}

// publishContractEvent publish contract event, return time used
func (cb *CommitBlock) publishContractEvent(block *commonpb.Block, events []*commonpb.ContractEvent) int64 {
	if len(events) == 0 {
		return 0
	}

	startPublishContractEventTick := utils.CurrentTimeMillisSeconds()
	cb.log.DebugDynamic(func() string {
		return fmt.Sprintf("start publish contractEventsInfo: block[%d] ",
			block.Header.BlockHeight)
	})
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

// MonitorCommit buried point
func (cb *CommitBlock) MonitorCommit(bi *commonpb.BlockInfo) {
	if !localconf.ChainMakerConfig.MonitorConfig.Enabled {
		return
	}
	raw, err := proto.Marshal(bi)
	if err != nil {
		cb.log.Errorw("marshal BlockInfo failed", "err", err)
		return
	}
	(*cb.metricBlockSize).WithLabelValues(bi.Block.Header.ChainId).Observe(float64(len(raw)))
	(*cb.metricBlockCounter).WithLabelValues(bi.Block.Header.ChainId).Inc()
	(*cb.metricTxCounter).WithLabelValues(bi.Block.Header.ChainId).Add(float64(bi.Block.Header.TxCount))
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

// NotifyMessage Notify other subscription modules of chain configuration and certificate management events
func (cb *CommitBlock) NotifyMessage(block *commonpb.Block, events []*commonpb.ContractEvent) (err error) {
	if block == nil || len(block.GetTxs()) == 0 {
		return nil
	}

	if native, _ := utils.IsNativeTx(block.Txs[0]); !native {
		return nil
	}

	for _, event := range events { // one by one
		data := event.EventData
		if len(data) == 0 {
			continue
		}
		topicEnum, err := strconv.Atoi(event.Topic)
		if err != nil {
			continue
		}
		topic := msgbus.Topic(topicEnum)
		cb.msgBus.PublishSync(topic, data) // data is a []string, hexToString(proto.Marshal(data))
	}
	return nil
}
