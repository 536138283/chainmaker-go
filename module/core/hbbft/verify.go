package hbbft

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"errors"
	"sync"
)

type Verifier struct {
	wg          sync.WaitGroup
	log         *logger.CMLogger
	hbbftCache  protocol.HbbftCache
	ledgerCache protocol.LedgerCache
}

func (v *Verifier) checkHeight(block *commonPb.Block) (bool, error) {
	currentHeight, err := v.ledgerCache.CurrentHeight()
	if err != nil {
		return false, err
	}
	if currentHeight+1 != block.Header.BlockHeight {
		return false, errors.New("the packaging signal height is inconsistent with the cache")
	}
	return true, nil
}

func (v *Verifier) verifier(block *commonPb.Block) {
	defer v.wg.Done()
	startTick := utils.CurrentTimeMillisSeconds()
	if err := utils.IsEmptyBlock(block); err != nil {
		v.log.Errorf("verify txBatch failed: %s, height: %d", err, block.Header.BlockHeight)
	}
	ok, err := v.checkHeight(block)
	if !ok {
		v.log.Errorf("verify txBatch failed: %s, height: %d", err, block.Header.BlockHeight)
	}
	v.log.Debugf("verify receive [%d](%x,%d,%d)",
		block.Header.BlockHeight, block.Header.BlockHash, block.Header.TxCount, len(block.Txs))

	txRWSetMap, timeLasts, err := v.validateBlock(block)
	if err != nil {
		v.log.Warnf("verify failed [%d](%x),preBlockHash:%x, %s",
			block.Header.BlockHeight, block.Header.BlockHash, block.Header.PreBlockHash, err.Error())
		if protocol.CONSENSUS_VERIFY == mode {
			v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, isValid))
		}
		return err
	}

	// sync mode, need to verify consensus vote signature
	if protocol.SYNC_VERIFY == mode {
		if err = v.verifyVoteSig(block); err != nil {
			v.log.Warnf("verify failed [%d](%x), votesig %s",
				block.Header.BlockHeight, block.Header.BlockHash, err.Error())
			return err
		}
	}

	// verify success, cache block and read write set
	v.log.Debugf("set proposed block(%d,%x)", block.Header.BlockHeight, block.Header.BlockHash)
	if err = v.proposalCache.SetProposedBlock(block, txRWSetMap, false); err != nil {
		return err
	}

	// mark transactions in block as pending status in txpool
	v.txPool.AddTxsToPendingCache(block.Txs, block.Header.BlockHeight)

	isValid = true
	if protocol.CONSENSUS_VERIFY == mode {
		v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, isValid))
	}
	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	v.log.Infof("verify success [%d,%x](%v,%d)", block.Header.BlockHeight, block.Header.BlockHash,
		timeLasts, elapsed)
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		v.metricBlockVerifyTime.WithLabelValues(v.chainId).Observe(float64(elapsed) / 1000)
	}
	return nil
}
