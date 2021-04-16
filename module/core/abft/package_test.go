package abft

import (
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/mock"
	"github.com/golang/mock/gomock"
	"testing"
)

var (
	chainId      = "chain1"
	contractName = "testContract"
)

func TestVerifyHeight(t *testing.T) {

}

func NewPackagerTest(t *testing.T) *Packager {
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
	packager := NewPackager(ce)
	return packager
}
