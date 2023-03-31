/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package net

import (
	"chainmaker.org/chainmaker/common/v3/msgbus"
	"chainmaker.org/chainmaker/pb-go/v3/consensus"
	"chainmaker.org/chainmaker/pb-go/v3/syscontract"
	"chainmaker.org/chainmaker/protocol/v3"
	"chainmaker.org/chainmaker/vm-native/v3/dposmgr"
	"github.com/gogo/protobuf/proto"
)

//KeyCurrentEpoch current epoch key
const KeyCurrentEpoch = "CE"

// NetServiceFactory is a net service instance factory.
type NetServiceFactory struct {
}

// NewNetService create a new net service instance.
// @param net
// @param chainId
// @param ac
// @param chainConf
// @param opts
// @return protocol.NetService
// @return error
func (nsf *NetServiceFactory) NewNetService(
	net protocol.Net,
	chainId string,
	ac protocol.AccessControlProvider,
	chainConf protocol.ChainConf,
	store protocol.BlockchainStore,
	opts ...NetServiceOption) (protocol.NetService, error) {
	//初始化工厂实例
	ns := NewNetService(chainId, net, ac)
	if err := ns.Apply(opts...); err != nil {
		return nil, err
	}
	if chainConf != nil {
		if chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_DPOS {
			if err := nsf.setAllDPoSConsensusNodeIds(ns, store); err != nil {
				return nil, err
			}
		} else {
			if err := nsf.setAllConsensusNodeIds(ns, chainConf); err != nil {
				return nil, err
			}
		}

		// set contract event subscribe
		if chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_DPOS {
			ns.msgBus.Register(msgbus.ContractEventInfo, ns.NetConfigSubscribe())
		} else {
			ns.msgBus.Register(msgbus.ChainConfig, ns.NetConfigSubscribe())
		}
		ns.msgBus.Register(msgbus.CertManageCertsRevoke, ns.NetConfigSubscribe())
		ns.msgBus.Register(msgbus.CertManageCertsFreeze, ns.NetConfigSubscribe())
		ns.msgBus.Register(msgbus.CertManageCertsUnfreeze, ns.NetConfigSubscribe())
		ns.msgBus.Register(msgbus.CertManageCertsAliasUpdate, ns.NetConfigSubscribe())
		ns.msgBus.Register(msgbus.CertManageCertsAliasDelete, ns.NetConfigSubscribe())
		ns.msgBus.Register(msgbus.PubkeyManageAdd, ns.NetConfigSubscribe())
		ns.msgBus.Register(msgbus.PubkeyManageDelete, ns.NetConfigSubscribe())
		ns.msgBus.Register(msgbus.MaxbftEpochConf, ns.NetConfigSubscribe())

		// v220_compat Deprecated
		{
			// set config watcher
			chainConf.AddWatch(ns.ConfigWatcher()) //nolint: staticcheck
			// set vm watcher
			chainConf.AddVmWatch(ns.VmWatcher()) //nolint: staticcheck
		}
	}
	return ns, nil
}

func (nsf *NetServiceFactory) setAllConsensusNodeIds(ns *NetService, chainConf protocol.ChainConf) error {
	consensusNodeUidList := make([]string, 0)
	// add all the seeds
	for _, node := range chainConf.ChainConfig().Consensus.Nodes {
		consensusNodeUidList = append(consensusNodeUidList, node.NodeId...)
	}
	// set all consensus node id for net service
	err := ns.Apply(WithConsensusNodeUid(consensusNodeUidList...))
	if err != nil {
		return err
	}
	ns.logger.Infof("[NetServiceFactory] set consensus node uid list ok(chain-id:%s)", ns.chainId)
	return nil
}

func (nsf *NetServiceFactory) setAllDPoSConsensusNodeIds(ns *NetService, store protocol.BlockchainStore) error {
	consensusNodeUidList := make([]string, 0)
	bz, err := store.ReadObject(syscontract.SystemContract_DPOS_STAKE.String(), []byte(KeyCurrentEpoch))
	if err != nil || len(bz) == 0 {
		ns.logger.Errorf("read current epoch err: %s", err)
		return nil
	}
	epoch := &syscontract.Epoch{}
	err = proto.Unmarshal(bz, epoch)
	if err != nil {
		ns.logger.Errorf("unmarshal epoch err: %s", err)
		return nil
	}

	for _, validator := range epoch.ProposerVector {
		nodeID, e := store.ReadObject(syscontract.SystemContract_DPOS_STAKE.String(), dposmgr.ToNodeIDKey(validator))
		if e != nil || len(nodeID) == 0 {
			ns.logger.Errorf("read nodeID of the validator[%s] failed, reason: %s", validator, err)
			return nil
		}
		consensusNodeUidList = append(consensusNodeUidList, string(nodeID))
	}
	// set all consensus node id for net service
	err = ns.Apply(WithConsensusNodeUid(consensusNodeUidList...))
	if err != nil {
		return err
	}
	ns.logger.Infof("[NetServiceFactory] set consensus node uid list ok(chain-id:%s)", ns.chainId)
	return nil
}
