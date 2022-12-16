/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockchain

import (
	"testing"

	"chainmaker.org/chainmaker-go/module/consensus"
	"chainmaker.org/chainmaker-go/module/consensus/cutover"
	"chainmaker.org/chainmaker-go/module/subscriber"
	"chainmaker.org/chainmaker/common/v3/msgbus"
	utils "chainmaker.org/chainmaker/consensus-utils/v3"
	"chainmaker.org/chainmaker/localconf/v3"
	"chainmaker.org/chainmaker/logger/v3"
	"chainmaker.org/chainmaker/pb-go/v3/common"
	commonPb "chainmaker.org/chainmaker/pb-go/v3/common"
	"chainmaker.org/chainmaker/pb-go/v3/config"
	configpb "chainmaker.org/chainmaker/pb-go/v3/config"
	consensuspb "chainmaker.org/chainmaker/pb-go/v3/consensus"
	"chainmaker.org/chainmaker/protocol/v3"
	"github.com/golang/mock/gomock"
)

func TestBlockchain_SwitchConsensus(t *testing.T) {
	var (
		localNodeID = "QmQZn3pZCcuEf34FSvucqkvVJEvfzpNjQTk17HS6CYMR35"
	)
	localconf.ChainMakerConfig.NodeConfig.NodeId = localNodeID
	consensus.RegisterConsensusProvider(consensuspb.ConsensusType_TBFT, func(config *utils.ConsensusImplConfig) (protocol.ConsensusEngine, error) {
		consensusMock := newMockConsensusEngine(t)
		consensusMock.EXPECT().Start().AnyTimes()
		consensusMock.EXPECT().Stop().AnyTimes()
		return consensusMock, nil
	})
	type fields struct {
		log                       *logger.CMLogger
		genesis                   string
		chainId                   string
		msgBus                    msgbus.MessageBus
		net                       protocol.Net
		netService                protocol.NetService
		store                     protocol.BlockchainStore
		oldStore                  protocol.BlockchainStore
		consensus                 protocol.ConsensusEngine
		txPool                    protocol.TxPool
		coreEngine                protocol.CoreEngine
		vmMgr                     protocol.VmManager
		identity                  protocol.SigningMember
		ac                        protocol.AccessControlProvider
		syncServer                protocol.SyncService
		ledgerCache               protocol.LedgerCache
		proposalCache             protocol.ProposalCache
		snapshotManager           protocol.SnapshotManager
		lastBlock                 *common.Block
		chainConf                 protocol.ChainConf
		chainNodeList             []string
		eventSubscriber           *subscriber.EventSubscriber
		txFilter                  protocol.TxFilter
		initModules               map[string]struct{}
		startModules              map[string]struct{}
		consensusSwitchSubscriber *cutover.ConsensusSwitchSubscriber
	}
	type args struct {
		consensusConfig *config.ConsensusConfig
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test0",
			fields: fields{
				log:     log,
				genesis: "",
				chainId: "chain1",
				msgBus: func() msgbus.MessageBus {
					msgbus := newMockMessageBus(t)
					msgbus.EXPECT().Register(gomock.Any(), gomock.Any()).AnyTimes()
					return msgbus
				}(),
				net:        nil,
				netService: nil,
				store:      nil,
				oldStore:   nil,
				consensus: func() protocol.ConsensusEngine {
					consensusMock := newMockConsensusEngine(t)
					consensusMock.EXPECT().Stop().AnyTimes()
					return consensusMock
				}(),
				txPool:     nil,
				coreEngine: nil,
				vmMgr: func() protocol.VmManager {
					vmMock := newMockVmManager(t)
					vmMock.EXPECT().Start().AnyTimes()
					vmMock.EXPECT().Stop().AnyTimes()
					vmMock.EXPECT().GetConsensusStateWrapper().Return(consensus.NewConsensusStateWrapper()).AnyTimes()
					return vmMock
				}(),
				identity:        nil,
				ac:              nil,
				syncServer:      nil,
				ledgerCache:     nil,
				proposalCache:   nil,
				snapshotManager: nil,
				lastBlock:       nil,
				chainConf: func() protocol.ChainConf {
					chainConf := newMockChainConf(t)
					chainConf.EXPECT().AddWatch(gomock.Any()).AnyTimes()
					chainConf.EXPECT().AddVmWatch(gomock.Any()).AnyTimes()
					chainConf.EXPECT().ChainConfig().Return(&configpb.ChainConfig{
						Consensus: &configpb.ConsensusConfig{
							Type: consensuspb.ConsensusType_TBFT,
							Nodes: []*configpb.OrgConfig{
								{
									OrgId:  "wx-org1",
									NodeId: []string{localNodeID},
								},
							},
						},
						Crypto: &configpb.CryptoConfig{Hash: "SHA256"},
						Contract: &configpb.ContractConfig{
							EnableSqlSupport: true,
						},
						Block: &configpb.BlockConfig{
							BlockInterval: 5,
						},
					}).AnyTimes()
					return chainConf
				}(),
				chainNodeList:   nil,
				eventSubscriber: nil,
				initModules: map[string]struct{}{
					moduleNameSubscriber:    {},
					moduleNameStore:         {},
					moduleNameLedger:        {},
					moduleNameChainConf:     {},
					moduleNameAccessControl: {},
					moduleNameVM:            {},
					moduleNameTxPool:        {},
					moduleNameCore:          {},
					moduleNameConsensus:     {},
					moduleNameSync:          {},
					moduleNameNetService:    {},
				},
				startModules: map[string]struct{}{
					moduleNameSubscriber:    {},
					moduleNameStore:         {},
					moduleNameLedger:        {},
					moduleNameChainConf:     {},
					moduleNameAccessControl: {},
					moduleNameVM:            {},
					moduleNameTxPool:        {},
					moduleNameCore:          {},
					moduleNameConsensus:     {},
					moduleNameSync:          {},
					moduleNameNetService:    {},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := &Blockchain{
				log:                       tt.fields.log,
				genesis:                   tt.fields.genesis,
				chainId:                   tt.fields.chainId,
				msgBus:                    tt.fields.msgBus,
				net:                       tt.fields.net,
				netService:                tt.fields.netService,
				store:                     tt.fields.store,
				oldStore:                  tt.fields.oldStore,
				consensus:                 tt.fields.consensus,
				txPool:                    tt.fields.txPool,
				coreEngine:                tt.fields.coreEngine,
				vmMgr:                     tt.fields.vmMgr,
				identity:                  tt.fields.identity,
				ac:                        tt.fields.ac,
				syncServer:                tt.fields.syncServer,
				ledgerCache:               tt.fields.ledgerCache,
				proposalCache:             tt.fields.proposalCache,
				snapshotManager:           tt.fields.snapshotManager,
				lastBlock:                 tt.fields.lastBlock,
				chainConf:                 tt.fields.chainConf,
				chainNodeList:             tt.fields.chainNodeList,
				eventSubscriber:           tt.fields.eventSubscriber,
				txFilter:                  tt.fields.txFilter,
				initModules:               tt.fields.initModules,
				startModules:              tt.fields.startModules,
				consensusSwitchSubscriber: tt.fields.consensusSwitchSubscriber,
			}
			if err := bc.SwitchConsensus(tt.args.consensusConfig); (err != nil) != tt.wantErr {
				t.Errorf("Blockchain.SwitchConsensus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlockchain_createOldStore(t *testing.T) {
	type fields struct {
		log             *logger.CMLogger
		genesis         string
		chainId         string
		msgBus          msgbus.MessageBus
		net             protocol.Net
		netService      protocol.NetService
		store           protocol.BlockchainStore
		oldStore        protocol.BlockchainStore
		consensus       protocol.ConsensusEngine
		txPool          protocol.TxPool
		coreEngine      protocol.CoreEngine
		vmMgr           protocol.VmManager
		identity        protocol.SigningMember
		ac              protocol.AccessControlProvider
		syncServer      protocol.SyncService
		ledgerCache     protocol.LedgerCache
		proposalCache   protocol.ProposalCache
		snapshotManager protocol.SnapshotManager
		lastBlock       *commonPb.Block
		chainConf       protocol.ChainConf
		chainNodeList   []string
		eventSubscriber *subscriber.EventSubscriber
		initModules     map[string]struct{}
		startModules    map[string]struct{}
	}
	type args struct {
		ok          bool
		storeEngine string
	}
	localconf.ChainMakerConfig = &localconf.CMConfig{
		StorageConfig: map[string]interface{}{
			"store_path": "./createOldStore",
			"blockdb_config": map[string]interface{}{
				"provider": "leveldb",
				"leveldb_config": map[string]interface{}{
					"store_path": "./createOldStore",
				},
			},
			"statedb_config": map[string]interface{}{
				"provider": "leveldb",
				"leveldb_config": map[string]interface{}{
					"store_path": "./createOldStore",
				},
			},
			"historydb_config": map[string]interface{}{
				"provider": "leveldb",
				"leveldb_config": map[string]interface{}{
					"store_path": "./createOldStore",
				},
			},
			"resultdb_config": map[string]interface{}{
				"provider": "leveldb",
				"leveldb_config": map[string]interface{}{
					"store_path": "./createOldStore",
				},
			},
			"txexistdb_config": map[string]interface{}{
				"provider": "leveldb",
				"leveldb_config": map[string]interface{}{
					"store_path": "./createOldStore",
				},
			},
			"disable_contract_eventdb": true,
			"contract_eventdb_config": map[string]interface{}{
				"provider": "leveldb",
				"leveldb_config": map[string]interface{}{
					"store_path": "./createOldStore",
				},
			},
		},
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test0",
			fields: fields{
				log:     log,
				genesis: "",
				chainId: "chain1",
				store: func() protocol.BlockchainStore {
					store := newMockBlockchainStore(t)
					store.EXPECT().GetDBHandle(gomock.Any()).AnyTimes()
					store.EXPECT().GetContractByName(gomock.Any()).AnyTimes()
					return store
				}(),
				chainConf: func() protocol.ChainConf {
					chainConf := newMockChainConf(t)
					chainConf.EXPECT().ChainConfig().Return(chainConfig).AnyTimes()
					return chainConf
				}(),
				initModules: map[string]struct{}{
					moduleNameStore: {},
				},
				startModules: nil,
			},
			args: args{
				ok:          true,
				storeEngine: "store-xx",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := &Blockchain{
				log:             tt.fields.log,
				genesis:         tt.fields.genesis,
				chainId:         tt.fields.chainId,
				msgBus:          tt.fields.msgBus,
				net:             tt.fields.net,
				netService:      tt.fields.netService,
				store:           tt.fields.store,
				oldStore:        tt.fields.oldStore,
				consensus:       tt.fields.consensus,
				txPool:          tt.fields.txPool,
				coreEngine:      tt.fields.coreEngine,
				vmMgr:           tt.fields.vmMgr,
				identity:        tt.fields.identity,
				ac:              tt.fields.ac,
				syncServer:      tt.fields.syncServer,
				ledgerCache:     tt.fields.ledgerCache,
				proposalCache:   tt.fields.proposalCache,
				snapshotManager: tt.fields.snapshotManager,
				lastBlock:       tt.fields.lastBlock,
				chainConf:       tt.fields.chainConf,
				chainNodeList:   tt.fields.chainNodeList,
				eventSubscriber: tt.fields.eventSubscriber,
				initModules:     tt.fields.initModules,
				startModules:    tt.fields.startModules,
			}
			if err := bc.createOldStore(tt.args.ok, tt.args.storeEngine); (err != nil) != tt.wantErr {
				t.Errorf("createOldStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlockchain_createOldStore1(t *testing.T) {
	type fields struct {
		log             *logger.CMLogger
		genesis         string
		chainId         string
		msgBus          msgbus.MessageBus
		net             protocol.Net
		netService      protocol.NetService
		store           protocol.BlockchainStore
		oldStore        protocol.BlockchainStore
		consensus       protocol.ConsensusEngine
		txPool          protocol.TxPool
		coreEngine      protocol.CoreEngine
		vmMgr           protocol.VmManager
		identity        protocol.SigningMember
		ac              protocol.AccessControlProvider
		syncServer      protocol.SyncService
		ledgerCache     protocol.LedgerCache
		proposalCache   protocol.ProposalCache
		snapshotManager protocol.SnapshotManager
		lastBlock       *commonPb.Block
		chainConf       protocol.ChainConf
		chainNodeList   []string
		eventSubscriber *subscriber.EventSubscriber
		initModules     map[string]struct{}
		startModules    map[string]struct{}
	}
	type args struct {
		ok          bool
		storeEngine string
	}

	localconf.ChainMakerConfig = &localconf.CMConfig{
		StorageConfig: map[string]interface{}{
			"store_path":      "./createOldStore1",
			"engine_provider": "store-huge",
			"block_file_config": map[string]interface{}{
				"online_file_system":  "./createOldStore1/metadb_tmp/online1",
				"archive_file_system": "./createOldStore1/metadb_tmp/archive1",
			},
			"storage_config_version": map[string]interface{}{
				"major_version": 1,
				"minor_version": 2,
			},
			"blockdb_config": map[string]interface{}{
				"provider": "leveldb",
				"leveldb_config": map[string]interface{}{
					"store_path": "./createOldStore1",
				},
			},
			"statedb_config": map[string]interface{}{
				"provider": "leveldb",
				"leveldb_config": map[string]interface{}{
					"store_path": "./createOldStore1",
				},
			},
			"historydb_config": map[string]interface{}{
				"provider": "leveldb",
				"leveldb_config": map[string]interface{}{
					"store_path": "./createOldStore1",
				},
			},
			"resultdb_config": map[string]interface{}{
				"provider": "leveldb",
				"leveldb_config": map[string]interface{}{
					"store_path": "./createOldStore1",
				},
			},
			"txexistdb_config": map[string]interface{}{
				"provider": "leveldb",
				"leveldb_config": map[string]interface{}{
					"store_path": "./createOldStore1",
				},
			},
			"disable_contract_eventdb": true,
			"contract_eventdb_config": map[string]interface{}{
				"provider": "leveldb",
				"leveldb_config": map[string]interface{}{
					"store_path": "./createOldStore1",
				},
			},
		},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test0",
			fields: fields{
				log:     log,
				genesis: "",
				chainId: "chain1",
				store: func() protocol.BlockchainStore {
					store := newMockBlockchainStore(t)
					store.EXPECT().GetDBHandle(gomock.Any()).AnyTimes()
					store.EXPECT().GetContractByName(gomock.Any()).AnyTimes()
					return store
				}(),
				chainConf: func() protocol.ChainConf {
					chainConf := newMockChainConf(t)
					chainConf.EXPECT().ChainConfig().Return(chainConfig).AnyTimes()
					return chainConf
				}(),
				initModules: map[string]struct{}{
					moduleNameStore: {},
				},
				startModules: nil,
			},
			args: args{
				ok:          true,
				storeEngine: "store-huge",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := &Blockchain{
				log:             tt.fields.log,
				genesis:         tt.fields.genesis,
				chainId:         tt.fields.chainId,
				msgBus:          tt.fields.msgBus,
				net:             tt.fields.net,
				netService:      tt.fields.netService,
				store:           tt.fields.store,
				oldStore:        tt.fields.oldStore,
				consensus:       tt.fields.consensus,
				txPool:          tt.fields.txPool,
				coreEngine:      tt.fields.coreEngine,
				vmMgr:           tt.fields.vmMgr,
				identity:        tt.fields.identity,
				ac:              tt.fields.ac,
				syncServer:      tt.fields.syncServer,
				ledgerCache:     tt.fields.ledgerCache,
				proposalCache:   tt.fields.proposalCache,
				snapshotManager: tt.fields.snapshotManager,
				lastBlock:       tt.fields.lastBlock,
				chainConf:       tt.fields.chainConf,
				chainNodeList:   tt.fields.chainNodeList,
				eventSubscriber: tt.fields.eventSubscriber,
				initModules:     tt.fields.initModules,
				startModules:    tt.fields.startModules,
			}
			delete(bc.initModules, moduleNameStore)
			if err := bc.createOldStore(tt.args.ok, tt.args.storeEngine); (err != nil) != tt.wantErr {
				t.Errorf("createOldStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
