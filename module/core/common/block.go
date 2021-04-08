/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"bytes"
	"chainmaker.org/chainmaker-go/common/crypto/hash"
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	consensuspb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"fmt"
	"time"
)

const (
	DEFAULTDURATION = 1000     // default proposal duration, millis seconds
	DEFAULTVERSION  = "v1.0.0" // default version of chain
)

func InitNewBlock(lastBlock *commonpb.Block, identity protocol.SigningMember, chainId string, chainConf protocol.ChainConf) (*commonpb.Block, error) {
	// get node pk from identity
	proposer, err := identity.Serialize(true)
	if err != nil {
		return nil, fmt.Errorf("identity serialize failed, %s", err)
	}
	preConfHeight := lastBlock.Header.PreConfHeight
	// if last block is config block, then this block.preConfHeight is last block height
	if utils.IsConfBlock(lastBlock) {
		preConfHeight = lastBlock.Header.BlockHeight
	}

	block := &commonpb.Block{
		Header: &commonpb.BlockHeader{
			ChainId:        chainId,
			BlockHeight:    lastBlock.Header.BlockHeight + 1,
			PreBlockHash:   lastBlock.Header.BlockHash,
			BlockHash:      nil,
			PreConfHeight:  preConfHeight,
			BlockVersion:   getChainVersion(chainConf),
			DagHash:        nil,
			RwSetRoot:      nil,
			TxRoot:         nil,
			BlockTimestamp: utils.CurrentTimeSeconds(),
			Proposer:       proposer,
			ConsensusArgs:  nil,
			TxCount:        0,
			Signature:      nil,
		},
		Dag:            &commonpb.DAG{},
		Txs:            nil,
		AdditionalData: nil,
	}
	return block, nil
}

func FinalizeBlock(block *commonpb.Block, txRWSetMap map[string]*commonpb.TxRWSet, aclFailTxs []*commonpb.Transaction, hashType string) error {

	if aclFailTxs != nil && len(aclFailTxs) > 0 {
		// append acl check failed txs to the end of block.Txs
		block.Txs = append(block.Txs, aclFailTxs...)
	}

	// TxCount contains acl verify failed txs and invoked contract txs
	txCount := len(block.Txs)
	block.Header.TxCount = int64(txCount)

	// TxRoot/RwSetRoot
	var err error
	txHashes := make([][]byte, txCount)
	for i, tx := range block.Txs {
		// finalize tx, put rwsethash into tx.Result
		rwSet := txRWSetMap[tx.Header.TxId]
		if rwSet == nil {
			rwSet = &commonpb.TxRWSet{
				TxId:     tx.Header.TxId,
				TxReads:  nil,
				TxWrites: nil,
			}
		}
		rwSetHash, err := utils.CalcRWSetHash(hashType, rwSet)
		if err != nil {
			return fmt.Errorf("failed to calc rwset hash: %s", err.Error())
		}
		if tx.Result == nil {
			// in case tx.Result is nil, avoid panic
			e := fmt.Errorf("tx(%s) result == nil", tx.Header.TxId)
			return e
		}
		tx.Result.RwSetHash = rwSetHash
		// calculate complete tx hash, include tx.Header, tx.Payload, tx.Result
		txHash, err := utils.CalcTxHash(hashType, tx)
		if err != nil {
			return fmt.Errorf("failed to calc tx hash: %s", err.Error())
		}
		txHashes[i] = txHash
	}

	block.Header.TxRoot, err = hash.GetMerkleRoot(hashType, txHashes)
	if err != nil {
		return fmt.Errorf("failed to get merkle root hash: %s", err.Error())
	}
	block.Header.RwSetRoot, err = utils.CalcRWSetRoot(hashType, block.Txs)
	if err != nil {
		return fmt.Errorf("failed to calc rwset root hash: %s", err.Error())
	}

	// DagDigest
	dagHash, err := utils.CalcDagHash(hashType, block.Dag)
	if err != nil {
		return fmt.Errorf("failed to calc dag hash: %s", err.Error())
	}
	block.Header.DagHash = dagHash

	return nil
}

// IsTxCountValid, to check if txcount in block is valid
func IsTxCountValid(block *commonpb.Block) error {
	if block.Header.TxCount != int64(len(block.Txs)) {
		return fmt.Errorf("txcount expect %d, got %d", block.Header.TxCount, len(block.Txs))
	}
	return nil
}

// IsHeightValid, to check if block height is valid
func IsHeightValid(block *commonpb.Block, currentHeight int64) error {
	if currentHeight+1 != block.Header.BlockHeight {
		return fmt.Errorf("height expect %d, got %d", currentHeight+1, block.Header.BlockHeight)
	}
	return nil
}

// IsPreHashValid, to check if block.preHash equals with last block hash
func IsPreHashValid(block *commonpb.Block, preHash []byte) error {
	if !bytes.Equal(preHash, block.Header.PreBlockHash) {
		return fmt.Errorf("prehash expect %x, got %x", preHash, block.Header.BlockHash)
	}
	return nil
}

// IsBlockHashValid, to check if block hash equals with result calculated from block
func IsBlockHashValid(block *commonpb.Block, hashType string) error {
	hash, err := utils.CalcBlockHash(hashType, block)
	if err != nil {
		return fmt.Errorf("calc block hash error")
	}
	if !bytes.Equal(hash, block.Header.BlockHash) {
		return fmt.Errorf("block hash expect %x, got %x", block.Header.BlockHash, hash)
	}
	return nil
}

// IsTxDuplicate, to check if there is duplicated transactions in one block
func IsTxDuplicate(txs []*commonpb.Transaction) bool {
	txSet := make(map[string]struct{})
	exist := struct{}{}
	for _, tx := range txs {
		if tx == nil || tx.Header == nil {
			return true
		}
		txSet[tx.Header.TxId] = exist
	}
	// length of set < length of txs, means txs have duplicate tx
	return len(txSet) < len(txs)
}

// IsMerkleRootValid, to check if block merkle root equals with simulated merkle root
func IsMerkleRootValid(block *commonpb.Block, txHashes [][]byte, hashType string) error {
	txRoot, err := hash.GetMerkleRoot(hashType, txHashes)
	if err != nil || !bytes.Equal(txRoot, block.Header.TxRoot) {
		return fmt.Errorf("txroot expect %x, got %x", block.Header.TxRoot, txRoot)
	}
	return nil
}

// IsDagHashValid, to check if block dag equals with simulated block dag
func IsDagHashValid(block *commonpb.Block, hashType string) error {
	dagHash, err := utils.CalcDagHash(hashType, block.Dag)
	if err != nil || !bytes.Equal(dagHash, block.Header.DagHash) {
		return fmt.Errorf("dag expect %x, got %x", block.Header.DagHash, dagHash)
	}
	return nil
}

// IsRWSetHashValid, to check if read write set is valid
func IsRWSetHashValid(block *commonpb.Block, hashType string) error {
	rwSetRoot, err := utils.CalcRWSetRoot(hashType, block.Txs)
	if err != nil {
		return fmt.Errorf("calc rwset error, %s", err)
	}
	if !bytes.Equal(rwSetRoot, block.Header.RwSetRoot) {
		return fmt.Errorf("rwset expect %x, got %x", block.Header.RwSetRoot, rwSetRoot)
	}
	return nil
}

// getDuration, get propose duration from config.
// If not access from config, use default value.
func getDuration(chainConf protocol.ChainConf) time.Duration {
	if chainConf == nil || chainConf.ChainConfig() == nil {
		return DEFAULTDURATION * time.Millisecond
	}
	chainConfig := chainConf.ChainConfig()
	duration := chainConfig.Block.BlockInterval
	if duration <= 0 {
		return DEFAULTDURATION * time.Millisecond
	} else {
		return time.Duration(duration) * time.Millisecond
	}
}

// getChainVersion, get chain version from config.
// If not access from config, use default value.
func getChainVersion(chainConf protocol.ChainConf) []byte {
	if chainConf == nil || chainConf.ChainConfig() == nil {
		return []byte(DEFAULTVERSION)
	}
	return []byte(chainConf.ChainConfig().Version)
}

func checkPreBlock(block *commonpb.Block, lastBlock *commonpb.Block) error {
	if consensuspb.ConsensusType_HOTSTUFF != v.chainConf.ChainConfig().Consensus.Type {
		if err = v.blockValidator.IsHeightValid(block, proposedHeight); err != nil {
			return err
		}
		// check if this block pre hash is equal with last block hash
		return v.blockValidator.IsPreHashValid(block, lastBlockHash)
	}

	if block.Header.BlockHeight == lastBlock.Header.BlockHeight+1 {
		if err := v.blockValidator.IsPreHashValid(block, lastBlock.Header.BlockHash); err != nil {
			return err
		}
	} else {
		// for chained bft consensus type
		proposedBlock, _ := v.proposalCache.GetProposedBlockByHashAndHeight(block.Header.PreBlockHash, block.Header.BlockHeight-1)
		if proposedBlock == nil {
			return fmt.Errorf("no last block found [%d](%x) %s", block.Header.BlockHeight-1, block.Header.PreBlockHash, err)
		}
	}

	// remove unconfirmed block from proposal cache and txpool
	cutBlocks := v.proposalCache.KeepProposedBlock(lastBlockHash, lastBlock.Header.BlockHeight)
	if len(cutBlocks) > 0 {
		cutTxs := make([]*commonpb.Transaction, 0)
		for _, cutBlock := range cutBlocks {
			cutTxs = append(cutTxs, cutBlock.Txs...)
		}
		v.txPool.RetryAndRemoveTxs(cutTxs, nil)
	}
	return nil
	cache.HbbftTxBatch{

	}
}

type ValidateBlockConf struct {
	chainConf   protocol.ChainConf
	log         *logger.CMLogger
	ledgerCache *cache.LedgerCache
	ac          protocol.AccessControlProvider
}

// validateBlock, validate block and transactions
func ValidateBlock(block *commonpb.Block, v *ValidateBlockConf) (map[string]*commonpb.TxRWSet, []int64, error) {
	hashType := v.chainConf.ChainConfig().Crypto.Hash
	timeLasts := make([]int64, 0)
	var err error
	var lastBlock *commonpb.Block
	txCapacity := int64(v.chainConf.ChainConfig().Block.BlockTxCapacity)
	if block.Header.TxCount > txCapacity {
		return nil, timeLasts, fmt.Errorf("txcapacity expect <= %d, got %d)", txCapacity, block.Header.TxCount)
	}

	if err = IsTxCountValid(block); err != nil {
		return nil, timeLasts, err
	}

	lastBlock = v.ledgerCache.GetLastCommittedBlock()

	// proposed height == proposing height - 1
	proposedHeight := lastBlock.Header.BlockHeight
	// check if this block height is 1 bigger than last block height
	lastBlockHash := lastBlock.Header.BlockHash
	err = v.checkPreBlock(block, lastBlock, err, lastBlockHash, proposedHeight)
	if err != nil {
		return nil, timeLasts, err
	}

	if err = IsBlockHashValid(block, chainConf.ChainConfig().Crypto.Hash); err != nil {
		return nil, timeLasts, err
	}

	// verify block sig and also verify identity and auth of block proposer
	startSigTick := utils.CurrentTimeMillisSeconds()

	log.Debugf("verify block \n %s", utils.FormatBlock(block))
	if ok, err := utils.VerifyBlockSig(hashType, block, v.ac); !ok || err != nil {
		return nil, timeLasts, fmt.Errorf("(%d,%x - %x,%x) [signature]",
			block.Header.BlockHeight, block.Header.BlockHash, block.Header.Proposer, block.Header.Signature)
	}
	sigLasts := utils.CurrentTimeMillisSeconds() - startSigTick
	timeLasts = append(timeLasts, sigLasts)

	err = v.checkVacuumBlock(block)
	if err != nil {
		return nil, timeLasts, err
	}
	if len(block.Txs) == 0 {
		return nil, timeLasts, nil
	}

	// verify if txs are duplicate in this block
	if v.blockValidator.IsTxDuplicate(block.Txs) {
		err := fmt.Errorf("tx duplicate")
		return nil, timeLasts, err
	}

	// simulate with DAG, and verify read write set
	startVMTick := utils.CurrentTimeMillisSeconds()
	snapshot := v.snapshotManager.NewSnapshot(lastBlock, block)
	txRWSetMap, txResultMap, err := v.txScheduler.SimulateWithDag(block, snapshot)
	vmLasts := utils.CurrentTimeMillisSeconds() - startVMTick
	timeLasts = append(timeLasts, vmLasts)
	if err != nil {
		return nil, timeLasts, fmt.Errorf("simulate %s", err)
	}
	if block.Header.TxCount != int64(len(txRWSetMap)) {
		err = fmt.Errorf("simulate txcount expect %d, got %d", block.Header.TxCount, len(txRWSetMap))
		return nil, timeLasts, err
	}

	// 2.transaction verify
	startTxTick := utils.CurrentTimeMillisSeconds()
	txHashes, _, errTxs, err := v.txValidator.VerifyTxs(block, txRWSetMap, txResultMap)
	txLasts := utils.CurrentTimeMillisSeconds() - startTxTick
	timeLasts = append(timeLasts, txLasts)
	if err != nil {
		// verify failed, need to put transactions back to txpool
		if len(errTxs) > 0 {
			v.log.Warn("[Duplicate txs] delete the err txs")
			v.txPool.RetryAndRemoveTxs(nil, errTxs)
		}
		return nil, timeLasts, fmt.Errorf("verify failed [%d](%x), %s ",
			block.Header.BlockHeight, block.Header.PreBlockHash, err)
	}
	//if protocol.CONSENSUS_VERIFY == mode && len(newAddTx) > 0 {
	//	v.txPool.AddTrustedTx(newAddTx)
	//}

	// verify TxRoot
	startRootsTick := utils.CurrentTimeMillisSeconds()
	err = v.checkBlockDigests(block, txHashes, hashType)
	if err != nil {
		return txRWSetMap, timeLasts, err
	}
	rootsLast := utils.CurrentTimeMillisSeconds() - startRootsTick
	timeLasts = append(timeLasts, rootsLast)
	return txRWSetMap, timeLasts, nil
}
