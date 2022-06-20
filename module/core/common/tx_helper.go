/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	bn "chainmaker.org/chainmaker/common/v2/birdsnest"
	commonErr "chainmaker.org/chainmaker/common/v2/errors"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
	batch "chainmaker.org/chainmaker/txpool-batch/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

var TxPoolType string

type VerifyBlockBatch struct {
	txs       []*commonpb.Transaction
	newAddTxs []*commonpb.Transaction
	txHash    [][]byte
}

func NewVerifyBlockBatch(txs, newAddTxs []*commonpb.Transaction, txHash [][]byte) VerifyBlockBatch {
	return VerifyBlockBatch{
		txs:       txs,
		newAddTxs: newAddTxs,
		txHash:    txHash,
	}
}

// verifyStat, statistic for verify steps
type VerifyStat struct {
	TotalCount  int
	DBLasts     int64
	SigLasts    int64
	OthersLasts int64
	SigCount    int
}

type RwSetVerifyFailTx struct {
	TxIds       []string
	BlockHeight uint64
}

// 判断相同分支上是否存在交易重复（防止双花）
func IfExitInSameBranch(height uint64, txId string, proposalCache protocol.ProposalCache, preBlockHash []byte) (bool, error) {
	hash := preBlockHash

	for i := uint64(1); i <= 3; i++ {
		b, _ := proposalCache.GetProposedBlockByHashAndHeight(hash, height-i)
		if b == nil || b.Header == nil {
			return false, nil
		}

		for _, tx := range b.Txs {
			if tx.Payload.TxId == txId {
				return true, fmt.Errorf("found the same tx[%s], height: %d", txId, b.Header.BlockHeight)
			}
		}
		hash = b.Header.PreBlockHash
	}

	return false, nil
}

func ValidateTx(txsRet map[string]*commonpb.Transaction, tx *commonpb.Transaction,
	stat *VerifyStat, newAddTxs []*commonpb.Transaction, block *commonpb.Block,
	consensusType consensuspb.ConsensusType, filter protocol.TxFilter,
	chainId string, ac protocol.AccessControlProvider, proposalCache protocol.ProposalCache,
	mode protocol.VerifyMode, verifyMode uint8) error {

	if TxPoolType == batch.TxPoolType {
		isExit, err := IfExitInSameBranch(block.Header.BlockHeight, tx.Payload.TxId, proposalCache, block.Header.PreBlockHash)
		if consensuspb.ConsensusType_MAXBFT == consensusType && isExit {

			err = fmt.Errorf("tx duplicate in pending (tx:%s), txInBlockHeight:%d, exit:%s",
				tx.Payload.TxId, block.Header.BlockHeight, err.Error())
			return err
		}

		// tx pool batch not need to verify TxHash
		return nil
	}

	txInPool, existTx := txsRet[tx.Payload.TxId]
	if existTx {
		isExit, err := IfExitInSameBranch(block.Header.BlockHeight, tx.Payload.TxId, proposalCache, block.Header.PreBlockHash)
		if consensuspb.ConsensusType_MAXBFT == consensusType && isExit {

			err = fmt.Errorf("tx duplicate in pending (tx:%s), txInBlockHeight:%d, exit:%s",
				tx.Payload.TxId, block.Header.BlockHeight, err.Error())
			return err
		}

		return IsTxRequestValid(tx, txInPool)
	}
	startDBTicker := utils.CurrentTimeMillisSeconds()
	var (
		isExist bool
		err     error
	)

	if verifyMode != QuickSyncVerifyMode {
		if mode == protocol.CONSENSUS_VERIFY {
			isExist, err = filter.IsExists(tx.Payload.TxId, bn.RuleType_AbsoluteExpireTime)
		} else {
			isExist, err = filter.IsExists(tx.Payload.TxId)
		}
	}

	stat.DBLasts += utils.CurrentTimeMillisSeconds() - startDBTicker
	if err != nil || isExist {
		err = fmt.Errorf("tx duplicate in DB (tx:%s) error: %v", tx.Payload.TxId, err)
		return err
	}
	stat.SigCount++
	startSigTicker := utils.CurrentTimeMillisSeconds()
	// if tx in txpool, means tx has already validated. tx noIt in txpool, need to validate.
	if err = utils.VerifyTxWithoutPayload(tx, chainId, ac); err != nil {
		err = fmt.Errorf("acl error (tx:%s), %s", tx.Payload.TxId, err.Error())
		return err
	}
	stat.SigLasts += utils.CurrentTimeMillisSeconds() - startSigTicker
	// tx valid and put into txpool
	newAddTxs = append(newAddTxs, tx) //nolint

	return nil
}

func TxVerifyResultsMerge(resultTasks map[int]VerifyBlockBatch,
	verifyBatchs map[int][]*commonpb.Transaction) ([][]byte, []*commonpb.Transaction, []*commonpb.Transaction, error) {

	errTxs := make([]*commonpb.Transaction, 0)
	txHashes := make([][]byte, 0)
	txNewAdd := make([]*commonpb.Transaction, 0)
	if len(resultTasks) < len(verifyBatchs) {
		return nil, nil, errTxs, fmt.Errorf("tx verify error, batch num mismatch, received: %d,expected:%d",
			len(resultTasks), len(verifyBatchs))
	}
	for i := 0; i < len(resultTasks); i++ {
		batch := resultTasks[i]
		if len(batch.txs) != len(batch.txHash) {
			return nil, nil, errTxs,
				fmt.Errorf("tx verify error, txs in batch mismatch, received: %d, expected:%d",
					len(batch.txHash), len(batch.txs))
		}
		txHashes = append(txHashes, batch.txHash...)
		txNewAdd = append(txNewAdd, batch.newAddTxs...)

	}
	return txHashes, txNewAdd, nil, nil
}

func RearrangeRWSet(block *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet) []*commonpb.TxRWSet {
	rwSet := make([]*commonpb.TxRWSet, 0)
	if rwSetMap == nil {
		return rwSet
	}
	for _, tx := range block.Txs {
		if set, ok := rwSetMap[tx.Payload.TxId]; ok {
			rwSet = append(rwSet, set)
		}
	}
	return rwSet
}

// IsTxRequestValid, to check if transaction request payload is valid
func IsTxRequestValid(tx *commonpb.Transaction, txInPool *commonpb.Transaction) error {
	poolTxRawBytes, err := utils.CalcUnsignedTxBytes(txInPool)
	if err != nil {
		return fmt.Errorf("calc pool tx bytes error (tx:%s), %s", tx.Payload.TxId, err.Error())
	}
	txRawBytes, err := utils.CalcUnsignedTxBytes(tx)
	if err != nil {
		return fmt.Errorf("calc req tx bytes error (tx:%s), %s", tx.Payload.TxId, err.Error())
	}
	// check if tx equals with tx in pool
	if !bytes.Equal(txRawBytes, poolTxRawBytes) {
		return fmt.Errorf("txhash (tx:%s) expect %x, got %x", tx.Payload.TxId, poolTxRawBytes, txRawBytes)
	}
	return nil
}

// VerifyTxResult, to check if transaction result is valid,
// compare result simulate in this node with executed in other node
func VerifyTxResult(tx *commonpb.Transaction, result *commonpb.Result) error {
	// verify if result is equal
	txResultBytes, err := utils.CalcResultBytes(tx.Result)
	if err != nil {
		return fmt.Errorf("calc tx result (tx:%s), %s)", tx.Payload.TxId, err.Error())
	}
	resultBytes, err := utils.CalcResultBytes(result)
	if err != nil {
		return fmt.Errorf("calc tx result (tx:%s), %s)", tx.Payload.TxId, err.Error())
	}
	if !bytes.Equal(txResultBytes, resultBytes) {
		debugInfo := "tx.Result:"
		r1, _ := json.Marshal(tx.Result)
		r2, _ := json.Marshal(result)
		debugInfo += string(r1) + "\ncurrent result:\n" + string(r2)
		return fmt.Errorf("tx result (tx:%s) expect %x, got %x\nDebug info:%s",
			tx.Payload.TxId, txResultBytes, resultBytes, debugInfo)
	}
	return nil
}

// IsTxRWSetValid, to check if transaction read write set is valid
func IsTxRWSetValid(block *commonpb.Block, tx *commonpb.Transaction, rwSet *commonpb.TxRWSet, result *commonpb.Result,
	rwsetHash []byte) error {
	if rwSet == nil || result == nil {
		return fmt.Errorf("txresult, rwset == nil (blockHeight: %d) (blockHash: %s) (tx:%s)",
			block.Header.BlockHeight, block.Header.BlockHash, tx.Payload.TxId)
	}
	if !bytes.Equal(tx.Result.RwSetHash, rwsetHash) {
		rwSetJ, _ := json.Marshal(rwSet)
		return fmt.Errorf("tx[%s] rwset hash expect %x, got %x, rwset details:%s",
			tx.Payload.TxId, tx.Result.RwSetHash, rwsetHash, string(rwSetJ))
	}
	return nil
}

type VerifierTx struct {
	block         *commonpb.Block
	txRWSetMap    map[string]*commonpb.TxRWSet
	txResultMap   map[string]*commonpb.Result
	log           protocol.Logger
	txFilter      protocol.TxFilter
	txPool        protocol.TxPool
	ac            protocol.AccessControlProvider
	chainConf     protocol.ChainConf
	proposalCache protocol.ProposalCache
}

type VerifierTxConfig struct {
	Block         *commonpb.Block
	TxRWSetMap    map[string]*commonpb.TxRWSet
	TxResultMap   map[string]*commonpb.Result
	Log           protocol.Logger
	TxFilter      protocol.TxFilter
	TxPool        protocol.TxPool
	Ac            protocol.AccessControlProvider
	ChainConf     protocol.ChainConf
	ProposalCache protocol.ProposalCache
}

func NewVerifierTx(conf *VerifierTxConfig) *VerifierTx {
	return &VerifierTx{
		block:         conf.Block,
		txRWSetMap:    conf.TxRWSetMap,
		txResultMap:   conf.TxResultMap,
		log:           conf.Log,
		txFilter:      conf.TxFilter,
		txPool:        conf.TxPool,
		ac:            conf.Ac,
		chainConf:     conf.ChainConf,
		proposalCache: conf.ProposalCache,
	}
}

// VerifyTxs verify transactions in block
// include if transaction is double spent, transaction signature
func (vt *VerifierTx) verifierTxs(block *commonpb.Block, mode protocol.VerifyMode, verifyMode uint8) (
	[][]byte, []*commonpb.Transaction,
	[]*commonpb.Transaction, *RwSetVerifyFailTx, error) {

	verifyBatch := utils.DispatchTxVerifyTask(block.Txs)
	resultTasks := make(map[int]VerifyBlockBatch)
	stats := make(map[int]*VerifyStat)
	var resultMu sync.Mutex
	var wg sync.WaitGroup
	waitCount := len(verifyBatch)
	wg.Add(waitCount)
	txIds := utils.GetTxIds(block.Txs)

	poolStart := utils.CurrentTimeMillisSeconds()
	txsRet := make(map[string]*commonpb.Transaction)
	if !IfOpenConsensusMessageTurbo(vt.chainConf) {
		if TxPoolType != batch.TxPoolType {
			txsRet, _ = vt.txPool.GetTxsByTxIds(txIds)
		}
	}
	poolLasts := utils.CurrentTimeMillisSeconds() - poolStart

	var err error
	startTicker := utils.CurrentTimeMillisSeconds()

	var failTxLock sync.Mutex
	rwSetVerifyFailTxIds := make([]string, 0)
	for i := 0; i < waitCount; i++ {
		index := i
		go func() {
			defer wg.Done()
			txs := verifyBatch[index]
			stat := &VerifyStat{
				TotalCount: len(txs),
			}
			txHashes1, newAddTxs, rwSetVerifyFailTxIdsIncr, err1 := vt.verifyTx(txs, txsRet, stat, block, mode, verifyMode)
			if err1 != nil {
				vt.log.Errorf("VerifyTx => verifyTx(...) failed, err = %v", err1)
				err = err1

				if rwSetVerifyFailTxIdsIncr != nil {
					failTxLock.Lock()
					rwSetVerifyFailTxIds = append(rwSetVerifyFailTxIds, rwSetVerifyFailTxIdsIncr...)
					failTxLock.Unlock()
				}
				return
			}
			resultMu.Lock()
			defer resultMu.Unlock()
			resultTasks[index] = VerifyBlockBatch{
				txs:       txs,
				txHash:    txHashes1,
				newAddTxs: newAddTxs,
			}
			stats[index] = stat
		}()
	}
	wg.Wait()
	concurrentLasts := utils.CurrentTimeMillisSeconds() - startTicker

	if len(rwSetVerifyFailTxIds) > 0 {
		rwSetVerifyFailTx := &RwSetVerifyFailTx{
			TxIds:       rwSetVerifyFailTxIds,
			BlockHeight: block.Header.BlockHeight,
		}
		vt.log.DebugDynamic(func() string {
			return fmt.Sprintf("collected verfiy failed rw set txs, count %d, "+
				"block height:%d, err: %s", len(rwSetVerifyFailTxIds),
				block.Header.BlockHeight, err.Error())
		})
		return nil, nil, nil, rwSetVerifyFailTx, err
	}

	resultStart := utils.CurrentTimeMillisSeconds()
	txHashes, txNewAdd, errTxs, err := TxVerifyResultsMerge(resultTasks, verifyBatch)
	if err != nil {
		return txHashes, txNewAdd, errTxs, nil, err
	}
	resultLasts := utils.CurrentTimeMillisSeconds() - resultStart

	for i, stat := range stats {
		if stat != nil {
			vt.log.Debugf("verify stat (index:%d,sigcount:%d/%d,db:%d,sig:%d,other:%d,total:%d)",
				i, stat.SigLasts, stat.TotalCount, stat.DBLasts, stat.SigLasts, stat.OthersLasts, concurrentLasts)
		}
	}

	vt.log.Infof("verify txs,height: [%d] (pool:%d,txVerify:%d,results:%d)",
		block.Header.BlockHeight, poolLasts, concurrentLasts, resultLasts)
	return txHashes, txNewAdd, nil, nil, nil
}

func (vt *VerifierTx) verifyTx(txs []*commonpb.Transaction, txsRet map[string]*commonpb.Transaction,
	stat *VerifyStat, block *commonpb.Block, mode protocol.VerifyMode, verifyMode uint8) (
	[][]byte, []*commonpb.Transaction, []string, error) {
	txHashes := make([][]byte, 0)
	newAddTxs := make([]*commonpb.Transaction, 0) // tx that verified and not in txpool, need to be added to txpool

	rwSetVerifyFailTxIds := make([]string, 0)
	for _, tx := range txs {
		// tx must in txpool when open consensus message turbo
		if !IfOpenConsensusMessageTurbo(vt.chainConf) {
			if err := ValidateTx(txsRet, tx, stat, newAddTxs, block,
				vt.chainConf.ChainConfig().Consensus.Type,
				vt.txFilter, vt.chainConf.ChainConfig().ChainId, vt.ac, vt.proposalCache, mode, verifyMode); err != nil {
				return nil, nil, nil, err
			}
		}

		if mode == protocol.CONSENSUS_VERIFY {
			if vt.chainConf.ChainConfig().Block.TxTimestampVerify {
				currentTime := utils.CurrentTimeSeconds()
				if (tx.Payload.Timestamp + int64(vt.chainConf.ChainConfig().Block.TxTimeout)) < currentTime {
					errMsg := fmt.Sprintf("verify tx timestamp fail, tx id:%s, tx payload timestamp:%d, current timestamp:%d",
						tx.Payload.TxId, tx.Payload.Timestamp, currentTime)
					vt.log.Errorf(errMsg)
					return nil, nil, nil, errors.New(errMsg)
				}
			}
		}

		startOthersTicker := utils.CurrentTimeMillisSeconds()
		rwSet := vt.txRWSetMap[tx.Payload.TxId]
		result := vt.txResultMap[tx.Payload.TxId]

		if TxPoolType == batch.TxPoolType {
			// recover result
			tx.Result = result

			rwsetHash, err := utils.CalcRWSetHash(vt.chainConf.ChainConfig().Crypto.Hash, rwSet)
			if err != nil {
				vt.log.Warnf("calc rwset hash error (tx:%s), rwSet: %v, %s",
					tx.Payload.TxId, rwSet, err)
				return nil, nil, nil, err
			}
			result.RwSetHash = rwsetHash

			hash, err := utils.CalcTxHashWithVersion(
				vt.chainConf.ChainConfig().Crypto.Hash, tx, int(block.Header.BlockVersion))
			if err != nil {
				vt.log.Warnf("calc txhash error (tx:%s), %s", tx.Payload.TxId, err)
				return nil, nil, nil, err
			}

			txHashes = append(txHashes, hash)

		} else {
			rwsetHash, err := utils.CalcRWSetHash(vt.chainConf.ChainConfig().Crypto.Hash, rwSet)
			if err != nil {
				vt.log.Warnf("calc rwset hash error (tx:%s), rwSet: %v, %s",
					tx.Payload.TxId, rwSet, err)
				return nil, nil, nil, err
			}
			if err = IsTxRWSetValid(vt.block, tx, rwSet, result, rwsetHash); err != nil {
				vt.log.Warnf("verify tx rw set failed, block height:%d, err:%s", vt.block.Header.BlockHeight, err)
				rwSetVerifyFailTxIds = append(rwSetVerifyFailTxIds, tx.Payload.TxId)
				continue
			}
			result.RwSetHash = rwsetHash
			// verify if rwset hash is equal
			if err = VerifyTxResult(tx, result); err != nil {
				vt.log.Warnf("verify tx result failed, block height:%d, err:%s", vt.block.Header.BlockHeight, err)
				rwSetVerifyFailTxIds = append(rwSetVerifyFailTxIds, tx.Payload.TxId)
				continue
			}
			hash, err := utils.CalcTxHashWithVersion(
				vt.chainConf.ChainConfig().Crypto.Hash, tx, int(block.Header.BlockVersion))
			if err != nil {
				vt.log.Warnf("calc txhash error (tx:%s), %s", tx.Payload.TxId, err)
				return nil, nil, nil, err
			}

			txHashes = append(txHashes, hash)
		}

		stat.OthersLasts += utils.CurrentTimeMillisSeconds() - startOthersTicker
	}

	if len(rwSetVerifyFailTxIds) > 0 {
		vt.log.Warn(commonErr.WarnRwSetVerifyFailTxs.Message)
		return nil, nil, rwSetVerifyFailTxIds, commonErr.WarnRwSetVerifyFailTxs
	}

	return txHashes, newAddTxs, nil, nil
}

// ValidateTxRules validate Transactions and return remain Transactions and Transactions that
// need to be removed
func ValidateTxRules(filter protocol.TxFilter, txs []*commonpb.Transaction) (
	removeTxs []*commonpb.Transaction, remainTxs []*commonpb.Transaction) {
	txIds := utils.GetTxIds(txs)
	// validate txFilter rules
	errorIdIndexes := validateTxIds(filter, txIds)
	// quick response None at all
	if len(errorIdIndexes) == 0 {
		return removeTxs, txs
	}
	// quick response None of the transactions were in compliance with the rules
	if len(errorIdIndexes) == len(txs) {
		return txs, []*commonpb.Transaction{}
	}
	remainTxs = make([]*commonpb.Transaction, 0, len(errorIdIndexes))
	removeTxs = make([]*commonpb.Transaction, 0, len(txs)-len(errorIdIndexes))
	for i, tx := range txs {
		if IntegersContains(errorIdIndexes, i) {
			removeTxs = append(removeTxs, tx)
		} else {
			remainTxs = append(remainTxs, tx)
		}
	}
	return removeTxs, remainTxs
}

func validateTxIds(filter protocol.TxFilter, ids []string) (errorIdIndexes []int) {
	for i, id := range ids {
		err := filter.ValidateRule(id, bn.RuleType_AbsoluteExpireTime)
		if err != nil {
			errorIdIndexes = append(errorIdIndexes, i)
		}
	}
	return
}

func IntegersContains(array []int, val int) bool {
	for i := 0; i < len(array); i++ {
		if array[i] == val {
			return true
		}
	}
	return false
}

func GetBatchIds(block *commonpb.Block) ([]string, []uint32) {
	batchIdsByte := block.AdditionalData.ExtraData[batch.BatchPoolAddtionalDataKey]

	txBatchInfo, _ := DeserializeTxBatchInfo(batchIdsByte)

	return txBatchInfo.BatchIds, txBatchInfo.Index
}
