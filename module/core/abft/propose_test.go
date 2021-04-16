package abft

import (
	"testing"

	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/mock"
	"github.com/golang/mock/gomock"
)

var (
	chainId      = "chain1"
	contractName = "testContract"
)

func TestVerifyHeight(t *testing.T) {

}

func NewProposerTest(t *testing.T) *Proposer {
	ctl := gomock.NewController(t)
	blockchainStoreImpl := mock.NewMockBlockchainStore(ctl)
	txPool := mock.NewMockTxPool(ctl)
	snapshotManager := mock.NewMockSnapshotManager(ctl)
	ledgerCache := cache.NewLedgerCache("chain1")
	chainConf := mock.NewMockChainConf(ctl)
	identity := mock.NewMockSigningMember(ctl)
	msgBus := mock.NewMockMessageBus(ctl)
	ac := mock.NewMockAccessControlProvider(ctl)
	log := logger.GetLoggerByChain(logger.MODULE_CORE, chainId)
	vmMgr := mock.NewMockVmManager(ctl)
	ce := &CoreExecute{
		chainId:         chainId,
		ledgerCache:     ledgerCache,
		txPool:          txPool,
		snapshotManager: snapshotManager,
		identity:        identity,
		msgBus:          msgBus,
		ac:              ac,
		blockchainStore: blockchainStoreImpl,
		chainConf:       chainConf,
		log:             log,
		vmMgr:           vmMgr,
	}
	proposer := NewProposer(ce)
	return proposer
}
