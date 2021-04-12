/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"bytes"
	"fmt"
	"time"

	"chainmaker.org/chainmaker-go/common/crypto/hash"
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
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

	if err := IsHeightValid(block, lastBlock.Header.BlockHeight); err != nil {
		return err
	}
	// check if this block pre hash is equal with last block hash
	if err := IsPreHashValid(block, lastBlock.Header.BlockHash); err != nil {
		return err
	}
	return nil
}

func checkVacuumBlock(block *commonpb.Block, chainConf protocol.ChainConf) error {
	if 0 == block.Header.TxCount {
		if utils.CanProposeEmptyBlock(chainConf.ChainConfig().Consensus.Type) {
			// for consensus that allows empty block, skip txs verify
			return nil
		} else {
			// for consensus that NOT allows empty block, return error
			return fmt.Errorf("tx must not empty")
		}
	}
	return nil
}

type ValidateBlockConf struct {
	ChainConf       protocol.ChainConf
	Log             *logger.CMLogger
	LedgerCache     protocol.LedgerCache
	Ac              protocol.AccessControlProvider
	SnapshotManager protocol.SnapshotManager
	VmMgr           protocol.VmManager
	TxPool          protocol.TxPool
	BlockchainStore protocol.BlockchainStore
}

type VerifyBlock struct {
	chainConf       protocol.ChainConf
	log             *logger.CMLogger
	ledgerCache     protocol.LedgerCache
	ac              protocol.AccessControlProvider
	snapshotManager protocol.SnapshotManager
	vmMgr           protocol.VmManager
	txScheduler     *TxScheduler
	txPool          protocol.TxPool
	blockchainStore protocol.BlockchainStore
}

func NewVerifyBlock(conf *ValidateBlockConf) *VerifyBlock {
	verifyBlock := &VerifyBlock{
		chainConf:       conf.ChainConf,
		log:             conf.Log,
		ledgerCache:     conf.LedgerCache,
		ac:              conf.Ac,
		snapshotManager: conf.SnapshotManager,
		vmMgr:           conf.VmMgr,
		txPool:          conf.TxPool,
		blockchainStore: conf.BlockchainStore,
	}
	verifyBlock.txScheduler = NewTxScheduler(verifyBlock.vmMgr, verifyBlock.chainConf.ChainConfig().ChainId)
	return verifyBlock
}

// validateBlock, validate block and transactions
func (vb *VerifyBlock) ValidateBlock(block *commonpb.Block) (map[string]*commonpb.TxRWSet, []int64, error) {
	hashType := vb.chainConf.ChainConfig().Crypto.Hash
	timeLasts := make([]int64, 0)
	var err error
	var lastBlock *commonpb.Block
	txCapacity := int64(vb.chainConf.ChainConfig().Block.BlockTxCapacity)
	if block.Header.TxCount > txCapacity {
		return nil, timeLasts, fmt.Errorf("txcapacity expect <= %d, got %d)", txCapacity, block.Header.TxCount)
	}

	if err = IsTxCountValid(block); err != nil {
		return nil, timeLasts, err
	}

	lastBlock = vb.ledgerCache.GetLastCommittedBlock()

	err = checkPreBlock(block, lastBlock)
	if err != nil {
		return nil, timeLasts, err
	}

	if err = IsBlockHashValid(block, vb.chainConf.ChainConfig().Crypto.Hash); err != nil {
		return nil, timeLasts, err
	}

	// verify block sig and also verify identity and auth of block proposer
	startSigTick := utils.CurrentTimeMillisSeconds()

	vb.log.Debugf("verify block \n %s", utils.FormatBlock(block))
	if ok, err := utils.VerifyBlockSig(hashType, block, vb.ac); !ok || err != nil {
		return nil, timeLasts, fmt.Errorf("(%d,%x - %x,%x) [signature]",
			block.Header.BlockHeight, block.Header.BlockHash, block.Header.Proposer, block.Header.Signature)
	}
	sigLasts := utils.CurrentTimeMillisSeconds() - startSigTick
	timeLasts = append(timeLasts, sigLasts)

	err = checkVacuumBlock(block, vb.chainConf)
	if err != nil {
		return nil, timeLasts, err
	}
	if len(block.Txs) == 0 {
		return nil, timeLasts, nil
	}

	// verify if txs are duplicate in this block
	if IsTxDuplicate(block.Txs) {
		err := fmt.Errorf("tx duplicate")
		return nil, timeLasts, err
	}

	// simulate with DAG, and verify read write set
	startVMTick := utils.CurrentTimeMillisSeconds()
	snapshot := vb.snapshotManager.NewSnapshot(lastBlock, block)

	txRWSetMap, txResultMap, err := vb.txScheduler.SimulateWithDag(block, snapshot)

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
	verifyTxConf := &VerifyTxConfig{
		Block:       block,
		TxResultMap: txResultMap,
		TxRWSetMap:  txRWSetMap,
		ChainConf:   vb.chainConf,
		Log:         vb.log,
		Ac:          vb.ac,
		TxPool:      vb.txPool,
		Store:       vb.blockchainStore,
	}
	verifytx := NewVerifyTx(verifyTxConf)
	txHashes, _, errTxs, err := verifytx.VerifyTxs()
	txLasts := utils.CurrentTimeMillisSeconds() - startTxTick
	timeLasts = append(timeLasts, txLasts)
	if err != nil {
		// verify failed, need to put transactions back to txpool
		if len(errTxs) > 0 {
			vb.log.Warn("[Duplicate txs] delete the err txs")
			vb.txPool.RetryAndRemoveTxs(nil, errTxs)
		}
		return nil, timeLasts, fmt.Errorf("verify failed [%d](%x), %s ",
			block.Header.BlockHeight, block.Header.PreBlockHash, err)
	}
	// verify TxRoot
	startRootsTick := utils.CurrentTimeMillisSeconds()
	err = checkBlockDigests(block, txHashes, hashType, vb.log)
	if err != nil {
		return txRWSetMap, timeLasts, err
	}
	rootsLast := utils.CurrentTimeMillisSeconds() - startRootsTick
	timeLasts = append(timeLasts, rootsLast)
	return txRWSetMap, timeLasts, nil
}

func checkBlockDigests(block *commonpb.Block, txHashes [][]byte, hashType string, log *logger.CMLogger) error {
	if err := IsMerkleRootValid(block, txHashes, hashType); err != nil {
		log.Error(err)
		return err
	}
	// verify DAG hash
	if err := IsDagHashValid(block, hashType); err != nil {
		log.Error(err)
		return err
	}
	// verify read write set, check if simulate result is equal with rwset in block header
	if err := IsRWSetHashValid(block, hashType); err != nil {
		log.Error(err)
		return err
	}
	return nil
}
