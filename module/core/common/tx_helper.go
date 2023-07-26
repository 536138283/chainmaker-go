/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"bytes"
	"fmt"
	"strconv"
	"sync"

	commonErr "chainmaker.org/chainmaker/common/v3/errors"
	"chainmaker.org/chainmaker/common/v3/json"
	commonpb "chainmaker.org/chainmaker/pb-go/v3/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v3/consensus"
	"chainmaker.org/chainmaker/pb-go/v3/txfilter"
	"chainmaker.org/chainmaker/protocol/v3"
	batch "chainmaker.org/chainmaker/txpool-batch/v3"
	"chainmaker.org/chainmaker/utils/v3"
)

// TxPoolType tx pool type
var TxPoolType string

// VerifyBlockBatch verify block batch struct
type VerifyBlockBatch struct {
	// transaction list
	txs []*commonpb.Transaction
	// new add transaction list
	newAddTxs []*commonpb.Transaction
	// tx hash
	txHash [][]byte
}

// NewVerifyBlockBatch new verify block batch
func NewVerifyBlockBatch(txs, newAddTxs []*commonpb.Transaction, txHash [][]byte) VerifyBlockBatch {
	return VerifyBlockBatch{
		txs:       txs,
		newAddTxs: newAddTxs,
		txHash:    txHash,
	}
}

// VerifyStat statistic for verify steps
type VerifyStat struct {
	TotalCount  int
	DBLasts     int64
	SigLasts    int64
	OthersLasts int64
	SigCount    int
	txfilter.Stat
}

// Sum sum stat data
// @param filter
func (stat *VerifyStat) Sum(filter *txfilter.Stat) {
	if filter != nil {
		stat.FpCount += filter.FpCount
		stat.FilterCosts += filter.FilterCosts
		stat.DbCosts += filter.DbCosts
	}
}

// RwSetVerifyFailTx rw set verify fail tx struct
type RwSetVerifyFailTx struct {
	TxIds       []string
	BlockHeight uint64
}

// IfExitInSameBranch 判断相同分支上是否存在交易重复（防止双花）
func IfExitInSameBranch(height uint64, txId string, proposalCache protocol.ProposalCache, preBlockHash []byte) (
	bool, error) {
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

// GetTxsMapFromSameBranch 将该分支下的所有交易的txId，放入map中
func GetTxsMapFromSameBranch(
	height uint64, proposalCache protocol.ProposalCache, preBlockHash []byte) map[string]struct{} {
	hash := preBlockHash
	txMap := make(map[string]struct{})
	for i := uint64(1); i <= 3; i++ {
		b, _ := proposalCache.GetProposedBlockByHashAndHeight(hash, height-i)
		if b == nil || b.Header == nil {
			return nil
		}

		for _, tx := range b.Txs {
			txMap[tx.Payload.TxId] = struct{}{}
		}
		hash = b.Header.PreBlockHash
	}

	return txMap
}

// ValidateTx validate tx, return error
func ValidateTx(txsRet map[string]*commonpb.Transaction, tx *commonpb.Transaction,
	stat *VerifyStat, newAddTxs []*commonpb.Transaction, block *commonpb.Block,
	consensusType consensuspb.ConsensusType, filter protocol.TxFilter,
	chainId string, ac protocol.AccessControlProvider, proposalCache protocol.ProposalCache,
	mode protocol.VerifyMode, verifyMode uint8, options ...string) error {

	if TxPoolType == batch.TxPoolType {

		// tx pool batch not need to verify TxHash
		return nil
	}

	txInPool, existTx := txsRet[tx.Payload.TxId]
	if existTx && mode == protocol.CONSENSUS_VERIFY {
		// not necessary to verify tx hash when in SYNC_VERIFY
		return IsTxRequestValid(tx, txInPool)
	}

	startDBTicker := utils.CurrentTimeMillisSeconds()
	var (
		isExist    bool
		err        error
		filterStat *txfilter.Stat
	)

	if verifyMode != QuickSyncVerifyMode {
		if mode == protocol.CONSENSUS_VERIFY {
			isExist, filterStat, err = filter.IsExists(tx.Payload.TxId, commonpb.RuleType_AbsoluteExpireTime)
		} else {
			isExist, filterStat, err = filter.IsExists(tx.Payload.TxId)
		}
	}

	// calc db use time
	stat.DBLasts += utils.CurrentTimeMillisSeconds() - startDBTicker
	stat.Sum(filterStat)

	if err != nil || isExist {
		err = fmt.Errorf("tx duplicate in DB (tx:%s) error: %v", tx.Payload.TxId, err)
		return err
	}
	stat.SigCount++
	startSigTicker := utils.CurrentTimeMillisSeconds()
	// if tx in txpool, means tx has already validated. tx noIt in txpool, need to validate.
	if err = utils.VerifyTxWithoutPayload(tx, chainId, ac, options...); err != nil {
		err = fmt.Errorf("acl error (tx:%s), %s", tx.Payload.TxId, err.Error())
		return err
	}
	// calc sig use time
	stat.SigLasts += utils.CurrentTimeMillisSeconds() - startSigTicker
	// tx valid and put into txpool
	newAddTxs = append(newAddTxs, tx) //nolint

	return nil
}

// TxVerifyResultsMerge tx verify results merge
func TxVerifyResultsMerge(resultTasks map[int]VerifyBlockBatch,
	verifyBatchs map[int][]*commonpb.Transaction) ([][]byte, []*commonpb.Transaction, error) {

	txHashes := make([][]byte, 0)
	txNewAdd := make([]*commonpb.Transaction, 0)
	if len(resultTasks) < len(verifyBatchs) {
		return nil, nil, fmt.Errorf("tx verify error, batch num mismatch, received: %d,expected:%d",
			len(resultTasks), len(verifyBatchs))
	}
	for i := 0; i < len(resultTasks); i++ {
		batch := resultTasks[i]
		if len(batch.txs) != len(batch.txHash) {
			return nil, nil,
				fmt.Errorf("tx verify error, txs in batch mismatch, received: %d, expected:%d",
					len(batch.txHash), len(batch.txs))
		}
		txHashes = append(txHashes, batch.txHash...)
		txNewAdd = append(txNewAdd, batch.newAddTxs...)

	}
	return txHashes, txNewAdd, nil
}

// RearrangeRWSet rearrange rw set
func RearrangeRWSet(block *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet) []*commonpb.TxRWSet {
	rwSet := make([]*commonpb.TxRWSet, 0)
	if rwSetMap == nil {
		return rwSet
	}
	// range block txs to collect rw set
	for _, tx := range block.Txs {
		if set, ok := rwSetMap[tx.Payload.TxId]; ok {
			rwSet = append(rwSet, set)
		}
	}
	return rwSet

}

// IsTxRequestValid to check if transaction request payload is valid
func IsTxRequestValid(tx *commonpb.Transaction, txInPool *commonpb.Transaction) error {
	// calc unsigned tx bytes by tx in pool
	poolTxRawBytes, err := utils.CalcUnsignedTxBytes(txInPool)
	if err != nil {
		return fmt.Errorf("calc pool tx bytes error (tx:%s), %s", tx.Payload.TxId, err.Error())
	}
	// calc unsigned tx bytes by tx
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

// VerifyTxResult to check if transaction result is valid,
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

// IsTxRWSetValid to check if transaction read write set is valid
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

// VerifierTx verifier tx
type VerifierTx struct {
	// block
	block *commonpb.Block
	// tx rw set map
	txRWSetMap map[string]*commonpb.TxRWSet
	// tx result map
	txResultMap map[string]*commonpb.Result
	// log
	log protocol.Logger
	// tx filter
	txFilter protocol.TxFilter
	// tx pool
	txPool protocol.TxPool
	// access control provider
	ac protocol.AccessControlProvider
	// chain config
	chainConf protocol.ChainConf
	// proposal cache
	proposalCache protocol.ProposalCache
}

// VerifierTxConfig verifier tx config
type VerifierTxConfig struct {
	// block
	Block *commonpb.Block
	// tx rw set map
	TxRWSetMap map[string]*commonpb.TxRWSet
	// tx result map
	TxResultMap map[string]*commonpb.Result
	// log
	Log protocol.Logger
	// tx filter
	TxFilter protocol.TxFilter
	// tx pool
	TxPool protocol.TxPool
	// access control provider
	Ac protocol.AccessControlProvider
	// chain config
	ChainConf protocol.ChainConf
	// proposal cache
	ProposalCache protocol.ProposalCache
}

// NewVerifierTx new verifier tx
func NewVerifierTx(conf *VerifierTxConfig) *VerifierTx {
	// construct verifier tx
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

// verifierTxs verify transactions in block
// include if transaction is double spent, transaction signature
func (vt *VerifierTx) verifierTxs(block *commonpb.Block, mode protocol.VerifyMode, verifyMode uint8) (
	[][]byte, []*commonpb.Transaction, *RwSetVerifyFailTx, error) {

	verifyBatch := utils.DispatchTxVerifyTask(block.Txs)
	resultTasks := make(map[int]VerifyBlockBatch, len(verifyBatch))
	stats := make(map[int]*VerifyStat, len(verifyBatch))
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
	// calc pool use time
	poolLasts := utils.CurrentTimeMillisSeconds() - poolStart

	var err error
	startTicker := utils.CurrentTimeMillisSeconds()

	// maxbft 共识下，从同分支下的其他区块中获取txsMap，用以判断交易是否重复
	txsMap := make(map[string]struct{})
	if vt.chainConf.ChainConfig().Consensus.Type == consensuspb.ConsensusType_MAXBFT {
		txsMap = GetTxsMapFromSameBranch(
			block.Header.BlockHeight,
			vt.proposalCache,
			block.Header.PreBlockHash)
	}

	// collect rw set verify failed txs
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

			txHashes1, newAddTxs, rwSetVerifyFailTxIdsIncr, err1 := vt.verifyTx(
				txs, txsRet, stat, block, txsMap, mode, verifyMode)
			if err1 != nil {
				vt.log.Errorf("verify tx failed, block height:%d, err:%v", block.Header.BlockHeight, err1)

				err = err1
				if rwSetVerifyFailTxIdsIncr != nil {
					failTxLock.Lock()
					rwSetVerifyFailTxIds = append(rwSetVerifyFailTxIds, rwSetVerifyFailTxIdsIncr...)
					failTxLock.Unlock()
					vt.log.Errorf("verify tx failed, block height:%d, rw set verify failed tx ids:%v, err:%v",
						block.Header.BlockHeight, rwSetVerifyFailTxIds, err1)
				}
				vt.log.Warn(err1.Error())
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
	// calc verify txs time
	concurrentLasts := utils.CurrentTimeMillisSeconds() - startTicker
	// if rw set verify fail tx ids exists, return rwSetVerifyFailTx and error
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
		return nil, nil, rwSetVerifyFailTx, err
	}

	resultStart := utils.CurrentTimeMillisSeconds()
	txHashes, txNewAdd, err := TxVerifyResultsMerge(resultTasks, verifyBatch)
	if err != nil {
		return txHashes, txNewAdd, nil, err
	}
	// calc result use time
	resultLasts := utils.CurrentTimeMillisSeconds() - resultStart

	for i, stat := range stats {
		if stat != nil {
			vt.log.Debugf(
				"verify stat (index:%d,sigcount:%d/%d,db:%d,sig:%d,other:%d,total:%d) "+
					"txfilter (fp:%d,exists:%d,fpdb:%d)",
				i, stat.SigLasts, stat.TotalCount, stat.DBLasts, stat.SigLasts, stat.OthersLasts, concurrentLasts,
				stat.FpCount, stat.FilterCosts, stat.DbCosts,
			)
		}
	}

	total, sig, db, other, fp, filterCosts, dbCosts, totalFilterCosts, totalDbCosts := calStatsAvg(stats)

	vt.log.Infof("verify txs,height: [%d] (count:%v,pool:%d,txVerify:%d,results:%d) "+
		"avg(sigcount:%d/%d,db:%d,sig:%d,other:%d) "+
		"filter total(fp:%d,exists:%d,fpdb:%d) filter avg(fp:%d,exists:%d,fpdb:%d)",
		block.Header.BlockHeight, block.Header.TxCount, poolLasts, concurrentLasts, resultLasts,
		sig, total, db, sig, other,
		fp, totalFilterCosts, totalDbCosts,
		fp, filterCosts, dbCosts,
	)
	return txHashes, txNewAdd, nil, nil
}

// verifyTx verify tx, return tx hashes, tx list, rw set verify failed tx tds, error
// VerifyTxs verify transactions in block
// include if transaction is double spent, transaction signature
func (vt *VerifierTx) verifierTxsWithRWSet(block *commonpb.Block, mode protocol.VerifyMode, verifyMode uint8) (
	[][]byte, error) {

	verifyBatch := utils.DispatchTxVerifyTask(block.Txs)
	resultTasks := make(map[int]VerifyBlockBatch)
	stats := make(map[int]*VerifyStat)
	var resultMu sync.Mutex
	var wg sync.WaitGroup
	waitCount := len(verifyBatch)
	wg.Add(waitCount)
	//txIds := utils.GetTxIds(block.Txs)

	poolStart := utils.CurrentTimeMillisSeconds()
	//txsRet := make(map[string]*commonpb.Transaction)
	//if !IfOpenConsensusMessageTurbo(vt.chainConf) {
	//	if TxPoolType != batch.TxPoolType {
	//		txsRet, _ = vt.txPool.GetTxsByTxIds(txIds)
	//	}
	//}
	poolLasts := utils.CurrentTimeMillisSeconds() - poolStart

	var err error
	startTicker := utils.CurrentTimeMillisSeconds()

	for i := 0; i < waitCount; i++ {
		index := i
		go func() {
			defer wg.Done()
			txs := verifyBatch[index]
			stat := &VerifyStat{
				TotalCount: len(txs),
			}
			txHashes1, err1 := vt.verifyTxWithRWSet(txs, stat, block)
			if err1 != nil {
				vt.log.Errorf("verify tx failed, block height:%d, err:%v", block.Header.BlockHeight, err1)
				err = err1

				return
			}
			resultMu.Lock()
			defer resultMu.Unlock()
			resultTasks[index] = VerifyBlockBatch{
				txs:    txs,
				txHash: txHashes1,
			}
			stats[index] = stat
		}()
	}
	wg.Wait()
	concurrentLasts := utils.CurrentTimeMillisSeconds() - startTicker

	resultStart := utils.CurrentTimeMillisSeconds()
	txHashes, _, err := TxVerifyResultsMerge(resultTasks, verifyBatch)
	if err != nil {
		return txHashes, err
	}
	resultLasts := utils.CurrentTimeMillisSeconds() - resultStart

	for i, stat := range stats {
		if stat != nil {
			vt.log.Debugf(
				"verify stat (index:%d,sigcount:%d/%d,db:%d,sig:%d,other:%d,total:%d) "+
					"txfilter (fp:%d,exists:%d,fpdb:%d)",
				i, stat.SigLasts, stat.TotalCount, stat.DBLasts, stat.SigLasts, stat.OthersLasts, concurrentLasts,
				stat.FpCount, stat.FilterCosts, stat.DbCosts,
			)
		}
	}

	total, sig, db, other, fp, filterCosts, dbCosts, totalFilterCosts, totalDbCosts := calStatsAvg(stats)

	vt.log.Infof("verify txs,height: [%d] (count:%v,pool:%d,txVerify:%d,results:%d) "+
		"avg(sigcount:%d/%d,db:%d,sig:%d,other:%d) "+
		"filter total(fp:%d,exists:%d,fpdb:%d) filter avg(fp:%d,exists:%d,fpdb:%d)",
		block.Header.BlockHeight, block.Header.TxCount, poolLasts, concurrentLasts, resultLasts,
		sig, total, db, sig, other,
		fp, totalFilterCosts, totalDbCosts,
		fp, filterCosts, dbCosts,
	)
	return txHashes, nil
}

// verifyTx verify txs
//
//	@receiver vt
//	@param txs
//	@param txsRet
//	@param stat
//	@param block
//	@param txsMap
//	@param mode
//	@param verifyMode
func (vt *VerifierTx) verifyTx(txs []*commonpb.Transaction, txsRet map[string]*commonpb.Transaction,
	stat *VerifyStat, block *commonpb.Block, txsMap map[string]struct{}, mode protocol.VerifyMode, verifyMode uint8) (
	[][]byte, []*commonpb.Transaction, []string, error) {
	txHashes := make([][]byte, 0, len(txs))
	// tx that verified and not in txpool, need to be added to txpool
	newAddTxs := make([]*commonpb.Transaction, 0, len(txs))

	// maxbft 共识下判断相同分支上是否存在交易重复（防止双花）
	if vt.chainConf.ChainConfig().Consensus.Type == consensuspb.ConsensusType_MAXBFT {
		if txsMap != nil {
			for _, tx := range txs {
				if _, exit := txsMap[tx.Payload.TxId]; exit {
					return nil, nil, nil, fmt.Errorf("found the same tx[%s], height: %d",
						tx.Payload.TxId, block.Header.BlockHeight)
				}
			}
		}
	}

	rwSetVerifyFailTxIds := make([]string, 0)
	for _, tx := range txs {
		// tx must in txpool when open consensus message turbo
		if !IfOpenConsensusMessageTurbo(vt.chainConf) {
			blockVersion := strconv.Itoa(int(vt.block.Header.BlockVersion))
			if err := ValidateTx(txsRet, tx, stat, newAddTxs, block,
				vt.chainConf.ChainConfig().Consensus.Type,
				vt.txFilter, vt.chainConf.ChainConfig().ChainId, vt.ac,
				vt.proposalCache, mode, verifyMode, blockVersion); err != nil {
				return nil, nil, nil, err
			}
		}

		startOthersTicker := utils.CurrentTimeMillisSeconds()
		rwSet := vt.txRWSetMap[tx.Payload.TxId]
		result := vt.txResultMap[tx.Payload.TxId]

		if TxPoolType == batch.TxPoolType {
			// calc rw set hash
			rwsetHash, err := utils.CalcRWSetHash(vt.chainConf.ChainConfig().Crypto.Hash, rwSet)
			if err != nil {
				vt.log.Warnf("calc rwset hash error (tx:%s), rwSet: %v, %s",
					tx.Payload.TxId, rwSet, err)
				return nil, nil, nil, err
			}
			tx.Result.RwSetHash = rwsetHash
			// calc tx hash with version
			hash, err := utils.CalcTxHashWithVersion(
				vt.chainConf.ChainConfig().Crypto.Hash, tx, int(block.Header.BlockVersion))
			if err != nil {
				vt.log.Warnf("calc txhash error (tx:%s), %s", tx.Payload.TxId, err)
				return nil, nil, nil, err
			}

			txHashes = append(txHashes, hash)

		} else {
			// calc rw set hash
			rwsetHash, err := utils.CalcRWSetHash(vt.chainConf.ChainConfig().Crypto.Hash, rwSet)
			if err != nil {
				vt.log.Warnf("calc rwset hash error (tx:%s), rwSet: %v, %s",
					tx.Payload.TxId, rwSet, err)
				return nil, nil, nil, err
			}
			// check rw set
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
			// calc tx hash with version
			hash, err := utils.CalcTxHashWithVersion(
				vt.chainConf.ChainConfig().Crypto.Hash, tx, int(block.Header.BlockVersion))
			if err != nil {
				vt.log.Warnf("calc txhash error (tx:%s), %s", tx.Payload.TxId, err)
				return nil, nil, nil, err
			}

			txHashes = append(txHashes, hash)
		}
		// calc other use time
		stat.OthersLasts += utils.CurrentTimeMillisSeconds() - startOthersTicker
	}
	// if rw set verify fail tx ids has existed, return rwSetVerifyFailTxIds, error
	if len(rwSetVerifyFailTxIds) > 0 {
		vt.log.Warn(commonErr.WarnRwSetVerifyFailTxs.Message)
		return nil, nil, rwSetVerifyFailTxIds, commonErr.WarnRwSetVerifyFailTxs
	}

	return txHashes, newAddTxs, nil, nil
}

// verifyTxWithRWSet verify tx with rw set
//
//	@receiver vt
//	@param txs
//	@param stat
//	@param block
//	@return [][]byte
//	@return error
func (vt *VerifierTx) verifyTxWithRWSet(txs []*commonpb.Transaction,
	stat *VerifyStat, block *commonpb.Block) ([][]byte, error) {
	txHashes := make([][]byte, 0)

	for _, tx := range txs {

		startOthersTicker := utils.CurrentTimeMillisSeconds()
		rwSet := vt.txRWSetMap[tx.Payload.TxId]

		// 将得到的读写集hash带入到tx的result中
		rwsetHash, err := utils.CalcRWSetHash(vt.chainConf.ChainConfig().Crypto.Hash, rwSet)
		if err != nil {
			vt.log.Warnf("calc rwset hash error (tx:%s), rwSet: %v, %s",
				tx.Payload.TxId, rwSet, err)
			return nil, err
		}

		tx.Result.RwSetHash = rwsetHash

		hash, err := utils.CalcTxHashWithVersion(
			vt.chainConf.ChainConfig().Crypto.Hash, tx, int(block.Header.BlockVersion))
		if err != nil {
			vt.log.Warnf("calc txhash error (tx:%s), %s", tx.Payload.TxId, err)
			return nil, err
		}

		txHashes = append(txHashes, hash)

		stat.OthersLasts += utils.CurrentTimeMillisSeconds() - startOthersTicker
	}

	return txHashes, nil
}

// ValidateTxRules validate Transactions and return remain txs and txs that need to be removed
//
//	@param filter
//	@param txs
//	@return removeTxs
//	@return remainTxs
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

// validateTxIds validate tx ids
//
//	@param filter
//	@param ids
//	@return errorIdIndexes
func validateTxIds(filter protocol.TxFilter, ids []string) (errorIdIndexes []int) {
	for i, id := range ids {
		err := filter.ValidateRule(id, commonpb.RuleType_AbsoluteExpireTime)
		if err != nil {
			errorIdIndexes = append(errorIdIndexes, i)
		}
	}
	return
}

// IntegersContains integers contains
//
//	@Description:
//	@param array
//	@param val
//	@return bool
func IntegersContains(array []int, val int) bool {
	for i := 0; i < len(array); i++ {
		if array[i] == val {
			return true
		}
	}
	return false
}

// GetBatchIds get batch ids
//
//	@param block
//	@return []string
//	@return []uint32
//	@return error
func GetBatchIds(block *commonpb.Block) ([]string, []uint32, error) {
	if batchIdsByte, ok := block.AdditionalData.ExtraData[batch.BatchPoolAddtionalDataKey]; ok {
		txBatchInfo, err := DeserializeTxBatchInfo(batchIdsByte)
		if err != nil {
			return nil, nil, err
		}

		return txBatchInfo.BatchIds, txBatchInfo.Index, nil
	}
	return []string{}, []uint32{}, nil
}

// calStatsAvg Calculate STATS averages
//
//	@param stats
//	@return total
//	@return sig
//	@return db
//	@return other
//	@return fp
//	@return filterCosts
//	@return dbCosts
//	@return totalFilterCosts
//	@return totalDbCosts
func calStatsAvg(stats map[int]*VerifyStat) (total, sig, db, other int, fp uint32,
	filterCosts, dbCosts, totalFilterCosts, totalDbCosts int64) {
	var count int
	if len(stats) == 0 {
		return
	}

	for _, stat := range stats {
		if stat != nil {
			total += stat.TotalCount
			sig += int(stat.SigLasts)
			db += int(stat.DBLasts)
			other += int(stat.OthersLasts)
			fp += stat.FpCount
			filterCosts += stat.FilterCosts
			dbCosts += stat.DbCosts
			count++
		}
	}
	totalFilterCosts = filterCosts
	totalDbCosts = dbCosts
	if count == 0 {
		return
	}
	total /= count
	sig /= count
	db /= count
	other /= count
	filterCosts /= int64(count)
	if fp != 0 {
		dbCosts /= int64(fp)
	}
	return
}

// GetInvalidTxSets get invalid tx sets(txId=>struct{}),use for OnReceiveRwSetVerifyFailTxs
//
//	@param invalidTxIds
//	@return map[string]struct{}
func GetInvalidTxSets(invalidTxIds []string) map[string]struct{} {
	invalidTxSets := make(map[string]struct{}, len(invalidTxIds))
	for _, txIds := range invalidTxIds {
		invalidTxSets[txIds] = struct{}{}
	}

	return invalidTxSets
}

// RemoveInvalidTxsForFollower remove invalid txs for follower,use for OnReceiveRwSetVerifyFailTxs
//
//	@param txPool
//	@param invalidTxIds
func RemoveInvalidTxsForFollower(txPool protocol.TxPool, invalidTxIds []string) {
	txsRet, _ := txPool.GetTxsByTxIds(invalidTxIds)
	txs := make([]*commonpb.Transaction, 0)
	for _, v := range txsRet {
		txs = append(txs, v)
	}
	txPool.RemoveTxs(txs, protocol.EVIL)
}

// RemoveInvalidTxsForProposer remove invalid txs for proposer, use for OnReceiveRwSetVerifyFailTxs
//
//	@param txPool
//	@param invalidTxSets
//	@param block
func RemoveInvalidTxsForProposer(txPool protocol.TxPool,
	invalidTxSets map[string]struct{}, block *commonpb.Block) {

	retryTxs := make([]*commonpb.Transaction, 0, len(block.Txs))
	removeTxs := make([]*commonpb.Transaction, 0, len(block.Txs))
	for _, tx := range block.Txs {
		if _, ok := invalidTxSets[tx.Payload.TxId]; ok {
			removeTxs = append(removeTxs, tx)
			continue
		}

		retryTxs = append(retryTxs, tx)
	}

	// retry txs and remove txs in tx pool
	txPool.RetryTxs(retryTxs)
	txPool.RemoveTxs(removeTxs, protocol.EVIL)
}
