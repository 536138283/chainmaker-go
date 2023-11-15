/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpcserver

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker-go/module/blockchain"
	"chainmaker.org/chainmaker-go/module/subscriber/model"
	"chainmaker.org/chainmaker/localconf/v3"
	commonPb "chainmaker.org/chainmaker/pb-go/v3/common"
	"chainmaker.org/chainmaker/protocol/v3"
)

var (
	// singleton tx result dispatcher
	dispatcher *txResultDispatcher
)

// txResultExt extend Result
type txResultExt struct {
	Result        *commonPb.Result
	TxTimestamp   int64
	TxBlockHeight uint64
}

// transactionExt extend Transaction
type transactionExt struct {
	Transaction *commonPb.Transaction
	BlockHeight uint64
}

// SendRequestSync - deal received TxRequest, sync tx result and send response
func (s *ApiService) SendRequestSync(ctx context.Context, req *commonPb.TxRequest) (*commonPb.TxResponse, error) {
	if req.Payload.TxType == commonPb.TxType_ETH_TX {
		return s.sendRawEthTransaction(ctx, req.Payload.GetParameter("RAWTX"), true)
	}

	s.log.DebugDynamic(func() string {
		return fmt.Sprintf("SendRequestSync[%s],payload:%#v,\n----signer:%v\n----endorsers:%+v",
			req.Payload.TxId, req.Payload, req.Sender, req.Endorsers)
	})

	startTime := time.Now()
	resp := s.invoke(ctx, &commonPb.Transaction{
		Payload:   req.Payload,
		Sender:    req.Sender,
		Endorsers: req.Endorsers,
		Result:    nil,
		Payer:     req.Payer,
	}, protocol.RPC, true)
	elapsed := time.Since(startTime)

	// audit log format: ip:port|orgId|chainId|TxType|TxId|Timestamp|ContractName|Method|retCode|retCodeMsg|retMsg
	// |invokeElapsed
	s.logBrief.Infof("|%s|%s|%s|%s|%s|%d|%s|%s|%d|%s|%s|%d", GetClientAddr(ctx), req.Sender.Signer.OrgId,
		req.Payload.ChainId, req.Payload.TxType, req.Payload.TxId, req.Payload.Timestamp, req.Payload.ContractName,
		req.Payload.Method, resp.Code, resp.Code, resp.Message, elapsed.Milliseconds())

	return resp, nil
}

type txResultDispatcher struct {
	// chainMakerServer instance
	chainMakerServer *blockchain.ChainMakerServer
	// mux protect childs and txRegistrations
	mux sync.Mutex
	// one child dispatcher per chain
	// key: chainId value: chan struct{} for stop signal
	childs map[string]chan struct{}
	// txRegistrations key: chainId_txId value: chan *commonPb.ContractResult
	txRegistrations map[string]chan *txResultExt
	// count of transactions are waiting for results
	txCount int64

	// the stop channel for stop all goroutines
	rootStopC chan struct{}
}

// newTxResultDispatcher returns a new txResultDispatcher
func newTxResultDispatcher(chainMakerServer *blockchain.ChainMakerServer) *txResultDispatcher {
	d := &txResultDispatcher{
		chainMakerServer: chainMakerServer,
		childs:           make(map[string]chan struct{}),
		txRegistrations:  make(map[string]chan *txResultExt),
		rootStopC:        make(chan struct{}),
	}
	go d.printStatistics()
	return d
}

// startChild - start a child dispatcher for a chain, run in a goroutine
func (d *txResultDispatcher) startChild(chainId string, childStopC chan struct{}) error {
	eventSubscriber, err := d.chainMakerServer.GetEventSubscribe(chainId)
	if err != nil {
		return err
	}

	go func() {
		blockEventC := make(chan model.NewBlockEvent, 1)
		sub := eventSubscriber.SubscribeBlockEvent(blockEventC)
		defer sub.Unsubscribe()

		for {
			select {
			case ev := <-blockEventC:
				log.Debugf("tx result dispatcher [%s] received block height: %d tx count: %d", chainId,
					ev.BlockInfo.Block.Header.BlockHeight, len(ev.BlockInfo.Block.Txs))
				for _, tx := range ev.BlockInfo.Block.Txs {
					txExt := &transactionExt{
						Transaction: tx,
						BlockHeight: ev.BlockInfo.Block.Header.BlockHeight,
					}
					d.trySendTxResult(txExt)
				}
			case <-d.rootStopC:
				return
			case <-childStopC:
				return
			}
		}
	}()
	return nil
}

// stop - stop all goroutines, include child dispatchers
func (d *txResultDispatcher) stop() {
	close(d.rootStopC)
}

// trySendTxResult - try to send a tx result.
// if txExt not registered, do nothing
func (d *txResultDispatcher) trySendTxResult(txExt *transactionExt) {
	d.mux.Lock()
	defer d.mux.Unlock()

	k := txExt.Transaction.Payload.ChainId + "_" + txExt.Transaction.Payload.TxId
	if txResultC, exists := d.txRegistrations[k]; exists {
		result := &txResultExt{
			Result:        txExt.Transaction.Result,
			TxTimestamp:   txExt.Transaction.Payload.Timestamp,
			TxBlockHeight: txExt.BlockHeight,
		}
		// non-blocking write to channel to ignore txResultC buffer is full in extreme cases
		select {
		case txResultC <- result:
		default:
		}
	}
}

// register for transaction result events.
// Note that unregister must be called when the registration is no longer needed.
//   - chainId is the chain ID for which events are to be received
//   - txId is the transaction ID for which events are to be received
//   - Returns the channel that is used to receive result. The channel
//     is closed when unregister is called.
func (d *txResultDispatcher) register(chainId, txId string) (chan *txResultExt, error) {
	d.mux.Lock()
	defer d.mux.Unlock()

	// if child dispatcher not exists, start it
	if _, exists := d.childs[chainId]; !exists {
		childStopC := make(chan struct{})
		err := d.startChild(chainId, childStopC)
		if err != nil {
			log.Errorf("start tx result dispatcher [%s] failed, %s", chainId, err.Error())
			return nil, err
		}
		d.childs[chainId] = childStopC
	}

	k := chainId + "_" + txId
	if txResultC, exists := d.txRegistrations[k]; exists {
		return txResultC, nil
	}
	atomic.AddInt64(&d.txCount, 1)
	txResultC := make(chan *txResultExt, 1)
	d.txRegistrations[k] = txResultC
	return txResultC, nil
}

// unregister removes the given registration and closes the event channel.
func (d *txResultDispatcher) unregister(chainId, txId string) {
	d.mux.Lock()
	defer d.mux.Unlock()

	k := chainId + "_" + txId
	if txResultC, exists := d.txRegistrations[k]; exists {
		atomic.AddInt64(&d.txCount, -1)
		delete(d.txRegistrations, k)
		close(txResultC)
	}
}

// evacuateChilds - stop child dispatchers that should not run any more based on current chains
func (d *txResultDispatcher) evacuateChilds() {
	currentChainIds := make(map[string]struct{})
	for _, chainConfig := range localconf.ChainMakerConfig.BlockChainConfig {
		currentChainIds[chainConfig.ChainId] = struct{}{}
	}

	d.mux.Lock()
	defer d.mux.Unlock()

	for chainId, childStopC := range d.childs {
		if _, exists := currentChainIds[chainId]; !exists {
			delete(d.childs, chainId)
			close(childStopC)
		}
	}
}

// printStatistics - logging statistics
func (d *txResultDispatcher) printStatistics() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			count := atomic.LoadInt64(&d.txCount)
			if count > 0 {
				log.Infof("[%d] transactions are waiting for results", count)
			}
		case <-d.rootStopC:
			return
		}
	}
}
