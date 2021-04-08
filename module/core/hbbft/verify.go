package hbbft

import (
	commonErrors "chainmaker.org/chainmaker-go/common/errors"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	consensuspb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"sync"
)

type Verifier struct {
	wg          sync.WaitGroup
	log         *logger.CMLogger
	hbbftCache  protocol.HbbftCache
	ledgerCache protocol.LedgerCache
}

func (v *Verifier) checkHeight(block *commonPb.Block) (bool, error) {
	get
}

func (v *Verifier) verifier(block *commonPb.Block) {
	defer v.wg.Done()
	startTick := utils.CurrentTimeMillisSeconds()
	var err error
	if err = utils.IsEmptyBlock(block); err != nil {
		v.log.Error(err)
	}

	v.log.Debugf("verify receive [%d](%x,%d,%d), from sync %d",
		block.Header.BlockHeight, block.Header.BlockHash, block.Header.TxCount, len(block.Txs), mode)

	var isValid bool
	// to check if the block has verified before
	if b, _ := v.proposalCache.GetProposedBlock(block); b != nil &&
		consensuspb.ConsensusType_SOLO != v.chainConf.ChainConfig().Consensus.Type {
		// the block has verified before
		v.log.Infof("verify success repeat [%d](%x)", block.Header.BlockHeight, block.Header.BlockHash)
		isValid = true
		if protocol.CONSENSUS_VERIFY == mode {
			// consensus mode, publish verify result to message bus
			v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, isValid))
		}
		return nil
	}

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
