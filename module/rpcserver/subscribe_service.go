/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpcserver

import (
	"errors"
	"fmt"

	commonErr "chainmaker.org/chainmaker-go/common/errors"
	apiPb "chainmaker.org/chainmaker-go/pb/protogo/api"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/subscriber"
	"chainmaker.org/chainmaker-go/subscriber/model"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Subscribe - deal block/tx subscribe request
func (s *ApiService) Subscribe(req *commonPb.TxRequest, server apiPb.RpcNode_SubscribeServer) error {
	var (
		errCode commonErr.ErrCode
		errMsg  string
	)

	tx := &commonPb.Transaction{
		Header:           req.Header,
		RequestPayload:   req.Payload,
		RequestSignature: req.Signature,
		Result:           nil}

	errCode, errMsg = s.validate(tx)
	if errCode != commonErr.ERR_CODE_OK {
		return status.Error(codes.Unauthenticated, errMsg)
	}

	switch req.Header.TxType {
	case commonPb.TxType_SUBSCRIBE_BLOCK_INFO:
		return s.dealBlockSubscription(tx, server)
	case commonPb.TxType_SUBSCRIBE_TX_INFO:
		return s.dealTxSubscription(tx, server)
	case commonPb.TxType_SUBSCRIBE_CONTRACT_EVENT_INFO:
		return s.dealContractEventSubscription(tx, server)
	}

	return nil
}

// dealTxSubscription - deal tx subscribe request
func (s *ApiService) dealTxSubscription(tx *commonPb.Transaction, server apiPb.RpcNode_SubscribeServer) error {
	var (
		err     error
		errMsg  string
		errCode commonErr.ErrCode
		payload commonPb.SubscribeTxPayload
		db      protocol.BlockchainStore
	)

	if err = proto.Unmarshal(tx.RequestPayload, &payload); err != nil {
		errCode = commonErr.ERR_CODE_SYSTEM_CONTRACT_PB_UNMARSHAL
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.InvalidArgument, errMsg)
	}

	if err = s.checkSubscribePayload(payload.StartBlock, payload.EndBlock); err != nil {
		errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_SUBSCRIBE_TX
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.InvalidArgument, errMsg)
	}

	s.log.Infof("Recv block subscribe request: [start:%d]/[end:%d]/[txType:%d]/[txIds:%+v]",
		payload.StartBlock, payload.EndBlock, payload.TxType, payload.TxIds)

	chainId := tx.Header.ChainId
	if db, err = s.chainMakerServer.GetStore(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.Internal, errMsg)
	}

	return s.doSendTx(tx, db, server, payload)
}

func (s *ApiService) doSendTx(tx *commonPb.Transaction, db protocol.BlockchainStore,
	server apiPb.RpcNode_SubscribeServer, payload commonPb.SubscribeTxPayload) error {

	var (
		txIdsMap                      = make(map[string]struct{})
		alreadySendHistoryBlockHeight int64
		err                           error
	)

	for _, txId := range payload.TxIds {
		txIdsMap[txId] = struct{}{}
	}

	if payload.StartBlock == -1 && payload.EndBlock == -1 {
		return s.sendNewTx(db, tx, server, payload, txIdsMap, -1)
	}

	if alreadySendHistoryBlockHeight, err = s.doSendHistoryTx(db, server, payload, txIdsMap); err != nil {
		return err
	}

	if alreadySendHistoryBlockHeight == 0 {
		return status.Error(codes.OK, "OK")
	}

	return s.sendNewTx(db, tx, server, payload, txIdsMap, alreadySendHistoryBlockHeight)
}

func (s *ApiService) doSendHistoryTx(db protocol.BlockchainStore, server apiPb.RpcNode_SubscribeServer,
	payload commonPb.SubscribeTxPayload, txIdsMap map[string]struct{}) (int64, error) {

	var (
		err             error
		errMsg          string
		errCode         commonErr.ErrCode
		lastBlockHeight int64
	)

	var startBlockHeight int64
	if payload.StartBlock > startBlockHeight {
		startBlockHeight = payload.StartBlock
	}

	if lastBlockHeight, err = s.checkAndGetLastBlockHeight(db, payload.StartBlock); err != nil {
		errCode = commonErr.ERR_CODE_GET_LAST_BLOCK
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return -1, status.Error(codes.Internal, errMsg)
	}

	if payload.EndBlock != -1 && payload.EndBlock <= lastBlockHeight {
		err, _ := s.sendHistoryTx(db, server, startBlockHeight, payload.EndBlock, payload.TxType, payload.TxIds, txIdsMap)
		if err != nil {
			s.log.Errorf("sendHistoryTx failed, %s", err)
			return -1, err
		}

		return 0, status.Error(codes.OK, "OK")
	}

	if len(payload.TxIds) > 0 && len(txIdsMap) == 0 {
		return 0, status.Error(codes.OK, "OK")
	}

	err, alreadySendHistoryBlockHeight := s.sendHistoryTx(db, server, startBlockHeight, payload.EndBlock, payload.TxType, payload.TxIds, txIdsMap)
	if err != nil {
		s.log.Errorf("sendHistoryTx failed, %s", err)
		return -1, err
	}

	if len(payload.TxIds) > 0 && len(txIdsMap) == 0 {
		return 0, status.Error(codes.OK, "OK")
	}

	s.log.Debugf("after sendHistoryBlock, alreadySendHistoryBlockHeight is %d", alreadySendHistoryBlockHeight)

	return alreadySendHistoryBlockHeight, nil
}

// dealBlockSubscription - deal block subscribe request
func (s *ApiService) dealBlockSubscription(tx *commonPb.Transaction, server apiPb.RpcNode_SubscribeServer) error {
	var (
		err             error
		errMsg          string
		errCode         commonErr.ErrCode
		payload         commonPb.SubscribeBlockPayload
		db              protocol.BlockchainStore
		lastBlockHeight int64
	)

	if err = proto.Unmarshal(tx.RequestPayload, &payload); err != nil {
		errCode = commonErr.ERR_CODE_SYSTEM_CONTRACT_PB_UNMARSHAL
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.InvalidArgument, errMsg)
	}

	if err = s.checkSubscribePayload(payload.StartBlock, payload.EndBlock); err != nil {
		errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_SUBSCRIBE_BLOCK
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.InvalidArgument, errMsg)
	}

	s.log.Infof("Recv block subscribe request: [start:%d]/[end:%d]/[withRWSet:%v]",
		payload.StartBlock, payload.EndBlock, payload.WithRwSet)

	chainId := tx.Header.ChainId
	if db, err = s.chainMakerServer.GetStore(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.Internal, errMsg)
	}

	if lastBlockHeight, err = s.checkAndGetLastBlockHeight(db, payload.StartBlock); err != nil {
		errCode = commonErr.ERR_CODE_GET_LAST_BLOCK
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.Internal, errMsg)
	}

	var startBlockHeight int64
	if payload.StartBlock > startBlockHeight {
		startBlockHeight = payload.StartBlock
	}

	if payload.StartBlock == -1 && payload.EndBlock == -1 {
		return s.sendNewBlock(db, tx, server, payload.EndBlock, payload.WithRwSet, -1)
	}

	if payload.EndBlock != -1 && payload.EndBlock <= lastBlockHeight {
		err, _ := s.sendHistoryBlock(db, server, startBlockHeight, payload.EndBlock, payload.WithRwSet)
		if err != nil {
			s.log.Errorf("sendHistoryBlock failed, %s", err)
			return err
		}

		return status.Error(codes.OK, "OK")
	}

	err, alreadySendHistoryBlockHeight := s.sendHistoryBlock(db, server, startBlockHeight, payload.EndBlock, payload.WithRwSet)
	if err != nil {
		s.log.Errorf("sendHistoryBlock failed, %s", err)
		return err
	}

	s.log.Debugf("after sendHistoryBlock, alreadySendHistoryBlockHeight is %d", alreadySendHistoryBlockHeight)

	return s.sendNewBlock(db, tx, server, payload.EndBlock, payload.WithRwSet, alreadySendHistoryBlockHeight)
}

// sendNewBlock - send new block to subscriber
func (s *ApiService) sendNewBlock(store protocol.BlockchainStore, tx *commonPb.Transaction,
	server apiPb.RpcNode_SubscribeServer,
	endBlockHeight int64, withRWSet bool, alreadySendHistoryBlockHeight int64) error {

	var (
		errCode         commonErr.ErrCode
		err             error
		errMsg          string
		eventSubscriber *subscriber.EventSubscriber
		blockInfo       *commonPb.BlockInfo
	)

	blockCh := make(chan model.NewBlockEvent)

	chainId := tx.Header.ChainId
	if eventSubscriber, err = s.chainMakerServer.GetEventSubscribe(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_SUBSCRIBER
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.Internal, errMsg)
	}

	sub := eventSubscriber.SubscribeBlockEvent(blockCh)
	defer sub.Unsubscribe()

	for {
		select {
		case ev := <-blockCh:
			blockInfo = ev.BlockInfo

			if alreadySendHistoryBlockHeight != -1 && blockInfo.Block.Header.BlockHeight > alreadySendHistoryBlockHeight {
				err, _ = s.sendHistoryBlock(store, server, alreadySendHistoryBlockHeight+1,
					blockInfo.Block.Header.BlockHeight, withRWSet)
				if err != nil {
					s.log.Errorf("send history block failed, %s", err)
					return err
				}

				alreadySendHistoryBlockHeight = -1
				continue
			}

			if err = s.dealBlockSubscribeResult(server, blockInfo, endBlockHeight, withRWSet); err != nil {
				s.log.Errorf(err.Error())
				return status.Error(codes.Internal, err.Error())
			}

			if endBlockHeight != -1 && blockInfo.Block.Header.BlockHeight >= endBlockHeight {
				return status.Error(codes.OK, "OK")
			}

		case <-server.Context().Done():
			return nil
		case <-s.ctx.Done():
			return nil
		}
	}
}

func (s *ApiService) dealBlockSubscribeResult(server apiPb.RpcNode_SubscribeServer, blockInfo *commonPb.BlockInfo,
	endBlockHeight int64, withRWSet bool) error {

	var (
		err    error
		result *commonPb.SubscribeResult
	)

	if !withRWSet {
		blockInfo = &commonPb.BlockInfo{
			Block:     blockInfo.Block,
			RwsetList: nil,
		}
	}
	if result, err = s.getBlockSubscribeResult(blockInfo); err != nil {
		return fmt.Errorf("get block subscribe result failed, %s", err)
	}

	if err := server.Send(result); err != nil {
		return fmt.Errorf("send block info by realtime failed, %s", err)
	}

	return nil
}

// sendNewTx - send new tx to subscriber
func (s *ApiService) sendNewTx(store protocol.BlockchainStore, tx *commonPb.Transaction,
	server apiPb.RpcNode_SubscribeServer, payload commonPb.SubscribeTxPayload,
	txIdsMap map[string]struct{}, alreadySendHistoryBlockHeight int64) error {

	var (
		errCode         commonErr.ErrCode
		err             error
		errMsg          string
		eventSubscriber *subscriber.EventSubscriber
		block           *commonPb.Block
	)

	blockCh := make(chan model.NewBlockEvent)

	chainId := tx.Header.ChainId
	if eventSubscriber, err = s.chainMakerServer.GetEventSubscribe(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_SUBSCRIBER
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.Internal, errMsg)
	}

	sub := eventSubscriber.SubscribeBlockEvent(blockCh)
	defer sub.Unsubscribe()

	for {
		select {
		case ev := <-blockCh:
			block = ev.BlockInfo.Block

			if alreadySendHistoryBlockHeight != -1 && block.Header.BlockHeight > alreadySendHistoryBlockHeight {
				err, _ = s.sendHistoryTx(store, server, alreadySendHistoryBlockHeight+1,
					block.Header.BlockHeight, payload.TxType, payload.TxIds, txIdsMap)
				if err != nil {
					s.log.Errorf("send history block failed, %s", err)
					return err
				}

				alreadySendHistoryBlockHeight = -1
				continue
			}

			if err := s.sendSubscribeTx(server, block.Txs, payload.TxType, payload.TxIds, txIdsMap); err != nil {
				errMsg = fmt.Sprintf("send subscribe tx failed, %s", err)
				s.log.Error(errMsg)
				return status.Error(codes.Internal, errMsg)
			}

			if s.checkIsFinish(payload, txIdsMap, ev.BlockInfo) {
				return status.Error(codes.OK, "OK")
			}

		case <-server.Context().Done():
			return nil
		case <-s.ctx.Done():
			return nil
		}
	}
}

func (s *ApiService) checkIsFinish(payload commonPb.SubscribeTxPayload,
	txIdsMap map[string]struct{}, blockInfo *commonPb.BlockInfo) bool {

	if len(payload.TxIds) > 0 && len(txIdsMap) == 0 {
		return true
	}

	if payload.EndBlock != -1 && blockInfo.Block.Header.BlockHeight >= payload.EndBlock {
		return true
	}

	return false
}

func (s *ApiService) getRateLimitToken() error {
	if s.subscriberRateLimiter != nil {
		if err := s.subscriberRateLimiter.Wait(s.ctx); err != nil {
			errMsg := fmt.Sprintf("subscriber rateLimiter wait token failed, %s", err.Error())
			s.log.Error(errMsg)
			return errors.New(errMsg)
		}
	}

	return nil
}

// sendHistoryBlock - send history block to subscriber
func (s *ApiService) sendHistoryBlock(store protocol.BlockchainStore, server apiPb.RpcNode_SubscribeServer,
	startBlockHeight, endBlockHeight int64, withRWSet bool) (error, int64) {

	var (
		err    error
		errMsg string
		result *commonPb.SubscribeResult
	)

	i := startBlockHeight
	for {
		select {
		case <-s.ctx.Done():
			return status.Error(codes.Internal, "chainmaker is restarting, please retry later"), -1
		default:
			if err = s.getRateLimitToken(); err != nil {
				return status.Error(codes.Internal, err.Error()), -1
			}

			if endBlockHeight != -1 && i > endBlockHeight {
				return nil, i - 1
			}

			blockInfo, alreadySendHistoryBlockHeight, err := s.getBlockInfoFromStore(store, i, withRWSet)
			if err != nil {
				return status.Error(codes.Internal, errMsg), -1
			}

			if blockInfo == nil || alreadySendHistoryBlockHeight > 0 {
				return nil, alreadySendHistoryBlockHeight
			}

			if result, err = s.getBlockSubscribeResult(blockInfo); err != nil {
				errMsg = fmt.Sprintf("get block subscribe result failed, %s", err)
				s.log.Error(errMsg)
				return errors.New(errMsg), -1
			}

			if err := server.Send(result); err != nil {
				errMsg = fmt.Sprintf("send block info by history failed, %s", err)
				s.log.Error(errMsg)
				return status.Error(codes.Internal, errMsg), -1
			}

			i++
		}
	}
}

func (s *ApiService) getBlockInfoFromStore(store protocol.BlockchainStore, curblockHeight int64, withRWSet bool) (
	blockInfo *commonPb.BlockInfo, alreadySendHistoryBlockHeight int64, err error) {
	var (
		errMsg         string
		block          *commonPb.Block
		blockWithRWSet *storePb.BlockWithRWSet
	)

	if withRWSet {
		blockWithRWSet, err = store.GetBlockWithRWSets(curblockHeight)
	} else {
		block, err = store.GetBlock(curblockHeight)
	}

	if err != nil {
		if withRWSet {
			errMsg = fmt.Sprintf("get block with rwset failed, at [height:%d], %s", curblockHeight, err)
		} else {
			errMsg = fmt.Sprintf("get block failed, at [height:%d], %s", curblockHeight, err)
		}
		s.log.Error(errMsg)
		return nil, -1, errors.New(errMsg)
	}

	if withRWSet {
		if blockWithRWSet == nil {
			return nil, curblockHeight - 1, nil
		}

		blockInfo = &commonPb.BlockInfo{
			Block:     blockWithRWSet.Block,
			RwsetList: blockWithRWSet.TxRWSets,
		}
	} else {
		if block == nil {
			return nil, curblockHeight - 1, nil
		}

		blockInfo = &commonPb.BlockInfo{
			Block:     block,
			RwsetList: nil,
		}
	}

	return blockInfo, -1, nil
}

// sendHistoryTx - send history tx to subscriber
func (s *ApiService) sendHistoryTx(store protocol.BlockchainStore,
	server apiPb.RpcNode_SubscribeServer,
	startBlockHeight, endBlockHeight int64,
	txType commonPb.TxType, txIds []string, txIdsMap map[string]struct{}) (error, int64) {

	var (
		err    error
		errMsg string
		block  *commonPb.Block
	)

	i := startBlockHeight
	for {
		select {
		case <-s.ctx.Done():
			return status.Error(codes.Internal, "chainmaker is restarting, please retry later"), -1
		default:
			if err = s.getRateLimitToken(); err != nil {
				return status.Error(codes.Internal, err.Error()), -1
			}

			if endBlockHeight != -1 && i > endBlockHeight {
				return nil, i - 1
			}

			if len(txIds) > 0 && len(txIdsMap) == 0 {
				return nil, i - 1
			}

			block, err = store.GetBlock(i)

			if err != nil {
				errMsg = fmt.Sprintf("get block failed, at [height:%d], %s", i, err)
				s.log.Error(errMsg)
				return status.Error(codes.Internal, errMsg), -1
			}

			if block == nil {
				return nil, i - 1
			}

			if err := s.sendSubscribeTx(server, block.Txs, txType, txIds, txIdsMap); err != nil {
				errMsg = fmt.Sprintf("send subscribe tx failed, %s", err)
				s.log.Error(errMsg)
				return status.Error(codes.Internal, errMsg), -1
			}

			i++
		}
	}
}

// checkSubscribePayload - check subscriber payload info
func (s *ApiService) checkSubscribePayload(startBlockHeight, endBlockHeight int64) error {
	if startBlockHeight < -1 || endBlockHeight < -1 ||
		(endBlockHeight != -1 && startBlockHeight > endBlockHeight) {

		return errors.New("invalid start block height or end block height")
	}

	return nil
}

func (s *ApiService) getTxSubscribeResult(tx *commonPb.Transaction) (*commonPb.SubscribeResult, error) {
	txBytes, err := proto.Marshal(tx)
	if err != nil {
		errMsg := fmt.Sprintf("marshal tx info failed, %s", err)
		s.log.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	result := &commonPb.SubscribeResult{
		Data: txBytes,
	}

	return result, nil
}

func (s *ApiService) getBlockSubscribeResult(blockInfo *commonPb.BlockInfo) (*commonPb.SubscribeResult, error) {

	blockBytes, err := proto.Marshal(blockInfo)
	if err != nil {
		errMsg := fmt.Sprintf("marshal block info failed, %s", err)
		s.log.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	result := &commonPb.SubscribeResult{
		Data: blockBytes,
	}

	return result, nil
}

func (s *ApiService) sendSubscribeTx(server apiPb.RpcNode_SubscribeServer,
	txs []*commonPb.Transaction, txType commonPb.TxType, txIds []string, txIdsMap map[string]struct{}) error {

	var (
		err error
	)

	for _, tx := range txs {
		if txType == -1 && len(txIds) == 0 {
			if err = s.doSendSubscribeTx(server, tx); err != nil {
				return err
			}
			continue
		}

		if s.checkIsContinue(tx, txType, txIds, txIdsMap) {
			continue
		}

		if err = s.doSendSubscribeTx(server, tx); err != nil {
			return err
		}
	}

	return nil
}

func (s *ApiService) checkIsContinue(tx *commonPb.Transaction, txType commonPb.TxType, txIds []string, txIdsMap map[string]struct{}) bool {
	if txType != -1 && tx.Header.TxType != txType {
		return true
	}

	if len(txIds) > 0 {
		_, ok := txIdsMap[tx.Header.TxId]
		if !ok {
			return true
		}

		delete(txIdsMap, tx.Header.TxId)
	}

	return false
}

func (s *ApiService) doSendSubscribeTx(server apiPb.RpcNode_SubscribeServer, tx *commonPb.Transaction) error {
	var (
		err    error
		errMsg string
		result *commonPb.SubscribeResult
	)

	if result, err = s.getTxSubscribeResult(tx); err != nil {
		errMsg = fmt.Sprintf("get tx subscribe result failed, %s", err)
		s.log.Error(errMsg)
		return errors.New(errMsg)
	}

	if err := server.Send(result); err != nil {
		errMsg = fmt.Sprintf("send subscribe tx result failed, %s", err)
		s.log.Error(errMsg)
		return errors.New(errMsg)
	}

	return nil
}

func (s *ApiService) checkAndGetLastBlockHeight(store protocol.BlockchainStore,
	payloadStartBlockHeight int64) (int64, error) {

	var (
		err       error
		errMsg    string
		errCode   commonErr.ErrCode
		lastBlock *commonPb.Block
	)

	if lastBlock, err = store.GetLastBlock(); err != nil {
		errCode = commonErr.ERR_CODE_GET_LAST_BLOCK
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return -1, status.Error(codes.Internal, errMsg)
	}

	if lastBlock.Header.BlockHeight < payloadStartBlockHeight {
		errMsg = fmt.Sprintf("payload start block height > last block height")
		s.log.Error(errMsg)
		return -1, status.Error(codes.InvalidArgument, errMsg)
	}

	return lastBlock.Header.BlockHeight, nil
}

//dealContractEventSubscription - deal contract event subscribe request
func (s *ApiService) dealContractEventSubscription(tx *commonPb.Transaction,
	server apiPb.RpcNode_SubscribeServer) error {

	var (
		err          error
		errMsg       string
		errCode      commonErr.ErrCode
		db           protocol.BlockchainStore
		startBlock   int64
		endBlock     int64
		contractName string
		topic       string
		payload commonPb.SubscribeContractEventPayload
	)

	if err = proto.Unmarshal(tx.RequestPayload, &payload); err != nil {
		errCode = commonErr.ERR_CODE_SYSTEM_CONTRACT_PB_UNMARSHAL
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.InvalidArgument, errMsg)
	}

	startBlock = payload.StartBlock
	endBlock = payload.EndBlock
	contractName = payload.ContractName
	topic = payload.Topic

	if err = s.checkSubscribeContractEventPayload(startBlock, endBlock, contractName); err != nil {
		errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_SUBSCRIBE_CONTRACT_EVENT
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.InvalidArgument, errMsg)
	}

	s.log.Infof("Recv contract event subscribe request: [start:%d]/[end:%d]/[contractName:%s]/[topic:%s]",
		startBlock, endBlock, contractName, topic)

	chainId := tx.Header.ChainId
	if db, err = s.chainMakerServer.GetStore(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.Internal, errMsg)
	}

	return s.doSendContractEvent(tx, db, server, startBlock, endBlock, contractName, topic)
}

func (s *ApiService) checkSubscribeContractEventPayload(startBlockHeight, endBlockHeight int64, contractName string) error {
	if startBlockHeight < -1 || endBlockHeight < -1 ||
		(endBlockHeight != -1 && startBlockHeight > endBlockHeight) {

		return errors.New("invalid start block height or end block height")
	}

	if contractName == "" {
		return errors.New("contractName can't be empty")
	}

	return nil
}

func (s *ApiService) doSendContractEvent(tx *commonPb.Transaction, db protocol.BlockchainStore,
	server apiPb.RpcNode_SubscribeServer, startBlock, endBlock int64,
	contractName string, topic string) error {

	var (
		alreadySendHistoryBlockHeight int64
		err                           error
	)

	if startBlock == -1 && endBlock == 0 {
		return status.Error(codes.OK, "OK")
	}

	// just send realtime contract event
	// == 0 for compatibility
	if (startBlock == -1 && endBlock == -1) || (startBlock == 0 && endBlock == 0) {
		return s.sendNewContractEvent(db, tx, server, startBlock, endBlock, contractName, topic, -1)
	}

	if startBlock != -1 {
		if alreadySendHistoryBlockHeight, err = s.doSendHistoryContractEvent(db, server, startBlock, endBlock,
			contractName, topic); err != nil {
			return err
		}
	}

	if startBlock == -1 {
		alreadySendHistoryBlockHeight = -1
	}

	if alreadySendHistoryBlockHeight == 0 {
		return status.Error(codes.OK, "OK")
	}

	return s.sendNewContractEvent(db, tx, server, startBlock, endBlock, contractName, topic,
		alreadySendHistoryBlockHeight)
}

func (s *ApiService) doSendHistoryContractEvent(db protocol.BlockchainStore, server apiPb.RpcNode_SubscribeServer,
	startBlock, endBlock int64, contractName, topic string) (int64, error) {

	var (
		err             error
		errMsg          string
		errCode         commonErr.ErrCode
		lastBlockHeight int64
	)

	var startBlockHeight int64
	if startBlock > startBlockHeight {
		startBlockHeight = startBlock
	}

	if lastBlockHeight, err = s.checkAndGetLastBlockHeight(db, startBlock); err != nil {
		errCode = commonErr.ERR_CODE_GET_LAST_BLOCK
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return -1, status.Error(codes.Internal, errMsg)
	}

	// only send history contract event
	if endBlock > 0 && endBlock <= lastBlockHeight {
		_, err = s.sendHistoryContractEvent(db, server, startBlockHeight, endBlock, contractName, topic)

		if err != nil {
			s.log.Errorf("sendHistoryContractEvent failed, %s", err)
			return -1, err
		}

		return 0, status.Error(codes.OK, "OK")
	}

	alreadySendHistoryBlockHeight, err := s.sendHistoryContractEvent(db, server, startBlockHeight, endBlock,
		contractName, topic)

	if err != nil {
		s.log.Errorf("sendHistoryContractEvent failed, %s", err)
		return -1, err
	}

	s.log.Debugf("after sendHistoryContractEvent, alreadySendHistoryBlockHeight is %d",
		alreadySendHistoryBlockHeight)

	return alreadySendHistoryBlockHeight, nil
}

// sendHistoryContractEvent - send history contract event to subscriber
func (s *ApiService) sendHistoryContractEvent(store protocol.BlockchainStore,
	server apiPb.RpcNode_SubscribeServer,
	startBlockHeight, endBlockHeight int64,
	contractName, topic string) (int64, error) {

	var (
		err    error
		errMsg string
		block  *commonPb.Block
	)

	i := startBlockHeight
	for {
		select {
		case <-s.ctx.Done():
			return -1, status.Error(codes.Internal, "chainmaker is restarting, please retry later")
		default:
			if err = s.getRateLimitToken(); err != nil {
				return -1, status.Error(codes.Internal, err.Error())
			}

			if endBlockHeight > 0 && i > endBlockHeight {
				return i - 1, nil
			}

			block, err = store.GetBlock(i)

			if err != nil {
				errMsg = fmt.Sprintf("get block failed, at [height:%d], %s", i, err)
				s.log.Error(errMsg)
				return -1, status.Error(codes.Internal, errMsg)
			}

			if block == nil {
				return i - 1, nil
			}

			if err := s.sendSubscribeContractEvent(server, block, contractName, topic); err != nil {
				errMsg = fmt.Sprintf("send subscribe tx failed, %s", err)
				s.log.Error(errMsg)
				return -1, status.Error(codes.Internal, errMsg)
			}

			i++
		}
	}
}

func (s *ApiService) sendSubscribeContractEvent(server apiPb.RpcNode_SubscribeServer,
	block *commonPb.Block, contractName, topic string) error {

	var (
		err error
	)

	for _, tx := range block.Txs {
		var eventInfos commonPb.ContractEventInfoList

		for _, event := range tx.Result.ContractResult.ContractEvent {
			if topic == "" || topic == event.Topic {
				if contractName != event.ContractName {
					continue
				}

				eventInfo := commonPb.ContractEventInfo{
					BlockHeight: block.Header.BlockHeight,
					ChainId: block.Header.ChainId,
					Topic: event.Topic,
					TxId: tx.Header.TxId,
					ContractName: event.ContractName,
					ContractVersion: event.ContractVersion,
					EventData: event.EventData,
				}

				eventInfos.ContractEvents = append(eventInfos.ContractEvents, &eventInfo)
			}
		}

		if err = s.doSendSubscribeContractEvent(server, &eventInfos); err != nil {
			return err
		}
	}

	return nil
}

func (s *ApiService) doSendSubscribeContractEvent(server apiPb.RpcNode_SubscribeServer,
	eventInfos *commonPb.ContractEventInfoList) error {

	var (
		err    error
		errMsg string
		result *commonPb.SubscribeResult
	)

	if result, err = s.getContractEventSubscribeResult(eventInfos); err != nil {
		errMsg = fmt.Sprintf("get tx subscribe result failed, %s", err)
		s.log.Error(errMsg)
		return errors.New(errMsg)
	}

	if err := server.Send(result); err != nil {
		errMsg = fmt.Sprintf("send subscribe tx result failed, %s", err)
		s.log.Error(errMsg)
		return errors.New(errMsg)
	}

	return nil
}

func (s *ApiService) sendNewContractEvent(store protocol.BlockchainStore, tx *commonPb.Transaction,
	server apiPb.RpcNode_SubscribeServer, startBlock, endBlock int64,
	contractName string, topic string, alreadySendHistoryBlockHeight int64) error {

	var (
		errCode         commonErr.ErrCode
		err             error
		errMsg          string
		eventSubscriber *subscriber.EventSubscriber
		result          *commonPb.SubscribeResult
	)

	eventCh := make(chan model.NewContractEvent)

	chainId := tx.Header.ChainId
	if eventSubscriber, err = s.chainMakerServer.GetEventSubscribe(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_SUBSCRIBER
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.Internal, errMsg)
	}

	sub := eventSubscriber.SubscribeContractEvent(eventCh)
	defer sub.Unsubscribe()
	for {
		select {
		case ev := <-eventCh:
			contractEventInfoList := ev.ContractEventInfoList.ContractEvents

			blockHeight := contractEventInfoList[0].BlockHeight
			if endBlock > 0 && blockHeight > endBlock {
				return status.Error(codes.OK, "OK")
			}

			if alreadySendHistoryBlockHeight != -1 &&
				blockHeight > alreadySendHistoryBlockHeight {
				historyBlockHeight, err := s.sendHistoryContractEvent(store, server, alreadySendHistoryBlockHeight+1,
					blockHeight, contractName, topic)
				if err != nil {
					s.log.Errorf("send history contract event failed, %s", err)
					return err
				}

				if endBlock > 0 && historyBlockHeight >= endBlock {
					return status.Error(codes.OK, "OK")
				}

				alreadySendHistoryBlockHeight = -1
				continue
			}

			sendEventInfoList := &commonPb.ContractEventInfoList{}
			for _, EventInfo := range contractEventInfoList {
				if EventInfo.ContractName != contractName || (topic != "" && EventInfo.Topic != topic) {
					continue
				}
				sendEventInfoList.ContractEvents = append(sendEventInfoList.ContractEvents, EventInfo)
			}

			if len(sendEventInfoList.ContractEvents) > 0 {
				if result, err = s.getContractEventSubscribeResult(sendEventInfoList); err != nil {
					s.log.Error(err.Error())
					return status.Error(codes.Internal, err.Error())
				}

				if err := server.Send(result); err != nil {
					err = fmt.Errorf("send block info by realtime failed, %s", err)
					s.log.Error(err.Error())
					return status.Error(codes.Internal, err.Error())
				}
			}

			if endBlock > 0 && blockHeight >= endBlock {
				return status.Error(codes.OK, "OK")
			}

		case <-server.Context().Done():
			return nil
		case <-s.ctx.Done():
			return nil
		}
	}
}

func (s *ApiService) getContractEventSubscribeResult(contractEventsInfoList *commonPb.ContractEventInfoList) (
	*commonPb.SubscribeResult, error) {

	eventBytes, err := proto.Marshal(contractEventsInfoList)
	if err != nil {
		errMsg := fmt.Sprintf("marshal contract event info failed, %s", err)
		s.log.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	result := &commonPb.SubscribeResult{
		Data: eventBytes,
	}

	return result, nil
}