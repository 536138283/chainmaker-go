/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockchain

import (
	"fmt"
	"os"

	"chainmaker.org/chainmaker/common/v3/container"
	"chainmaker.org/chainmaker/logger/v3"
	"chainmaker.org/chainmaker/protocol/v3"
	storeHuge "chainmaker.org/chainmaker/store-huge/v3"
	storeHugeConf "chainmaker.org/chainmaker/store-huge/v3/conf"
	"chainmaker.org/chainmaker/store/v3"
	storeConf "chainmaker.org/chainmaker/store/v3/conf"

	"chainmaker.org/chainmaker/localconf/v3"
	"chainmaker.org/chainmaker/pb-go/v3/config"

	commonErrors "chainmaker.org/chainmaker/common/v3/errors"
)

// RebuildDbs Start all the modules.
func (bc *Blockchain) RebuildDbs(needVerify bool) {
	fmt.Printf("###########################")
	fmt.Printf("###start rebuild-dbs....###")
	fmt.Printf("###########################")
	bc.log.Infof("###########################")
	bc.log.Infof("###start rebuild-dbs....###")
	bc.log.Infof("###########################")
	lastBlock, err1 := bc.oldStore.GetLastBlock()
	if err1 != nil {
		bc.log.Errorf("get lastblockerr(%s)", err1.Error())
	} else {
		bc.log.Infof("lastBlock=%d", lastBlock.Header.BlockHeight)
	}
	var i, height uint64
	var preHash []byte
	bHeight, _ := localconf.ChainMakerConfig.StorageConfig["rebuild_block_height"].(int)
	if bHeight <= 0 {
		bc.log.Warnf("error block_height!")
		height = lastBlock.GetHeader().BlockHeight
	} else {
		if uint64(bHeight) <= lastBlock.GetHeader().BlockHeight {
			height = uint64(bHeight)
		} else {
			height = lastBlock.GetHeader().BlockHeight
		}
	}
	for i = 1; i <= height; i++ {
		block, err2 := bc.oldStore.GetBlock(i)
		if err2 != nil {
			bc.log.Errorf("get block %d err(%s)", i, err2.Error())
		}
		bc.log.Debugf("block %d hash is %x", i, block.GetHeader().BlockHash)
		bc.log.Debugf("block %d prehash is %x", i, block.GetHeader().PreBlockHash)
		if preHash != nil && string(preHash) != string(block.GetHeader().PreBlockHash) {
			bc.log.Errorf("\npreHash=%x\nprehash=%x", []byte(preHash), block.GetHeader().PreBlockHash)
			bc.log.Errorf("\nError!!!!\n")
		} else {
			bc.log.Debugf("\npreHash=%x\nprehash=%x", []byte(preHash), block.GetHeader().PreBlockHash)
		}
		preHash = block.GetHeader().BlockHash

		var err3 error
		if needVerify {
			err3 = bc.coreEngine.GetBlockVerifier().VerifyBlock(block, -1)
		} else {
			blockRwSets, err31 := bc.oldStore.GetBlockWithRWSets(block.Header.BlockHeight)
			if err31 == nil {
				err3 = bc.coreEngine.GetBlockVerifier().VerifyBlockWithRwSets(
					blockRwSets.GetBlock(), blockRwSets.GetTxRWSets(), -1)
			}
		}

		if err3 != nil {
			if err3 == commonErrors.ErrBlockHadBeenCommitted {
				bc.log.Errorf("the block: %d has been committed in the blockChainStore ", block.Header.BlockHeight)
			} else {
				fmt.Printf("block[%d] verify success.", block.Header.BlockHeight)
				bc.log.Infof("block[%d] verify success.", block.Header.BlockHeight)
			}
		} else {
			fmt.Printf("block[%d] verify success.", block.Header.BlockHeight)
			bc.log.Infof("block[%d] verify success.", block.Header.BlockHeight)
		}

		//time.Sleep(500*time.Millisecond)
		if err4 := bc.coreEngine.GetBlockCommitter().AddBlock(block); err4 != nil {
			if err4 == commonErrors.ErrBlockHadBeenCommitted {
				bc.log.Errorf("the block: %d has been committed in the blockChainStore ", block.Header.BlockHeight)
			} else {
				fmt.Printf("block[%d] rebuild success.", block.Header.BlockHeight)
				bc.log.Infof("block[%d] rebuild success.", block.Header.BlockHeight)
			}

		} else {
			bc.log.Infof("block[%d] rebuild success.", block.Header.BlockHeight)
			fmt.Printf("block[%d] rebuild success.", block.Header.BlockHeight)
		}
		//time.Sleep(500 * time.Millisecond)

	}
	fmt.Printf("###########################")
	fmt.Printf("###rebuild-dbs finished!###")
	fmt.Printf("###########################")
	bc.log.Infof("###########################")
	bc.log.Infof("###rebuild-dbs finished!###")
	bc.log.Infof("###########################")
	bc.Stop()
	os.Exit(0)
}

//SwitchConsensus switch consensus algorithm， stop the old consensus and start the new consensus
func (bc *Blockchain) SwitchConsensus(consensusConfig *config.ConsensusConfig) error {
	// chainConf := bc.chainConf.ChainConfig()
	// chainConf.Consensus = consensusConfig
	delete(bc.initModules, moduleNameConsensus)
	bc.StopOnRequirements()
	if err := bc.Init(); err != nil {
		bc.log.Errorf("blockchain init failed when switching consensus, %s", err)
		return err
	}
	bc.StopOnRequirements()
	if err := bc.Start(); err != nil {
		bc.log.Errorf("blockchain start failed when witching consensus, %s", err)
		return err
	}
	return nil
}

// createOldStore create a old store for rebuild
func (bc *Blockchain) createOldStore(ok bool, storeEngine string) error {
	var err error

	// store engine is not store-huge,provider-engine is nil
	if !ok {
		storeFactory := store.NewFactory()
		//var storeFactory store.Factory // nolint: typecheck
		storeLogger := logger.GetLoggerByChain(logger.MODULE_STORAGE, bc.chainId)
		err = container.Register(func() protocol.Logger { return storeLogger }, container.Name("store"))
		if err != nil {
			return err
		}
		var config *storeConf.StorageConfig
		config, err = storeConf.NewStorageConfig(localconf.ChainMakerConfig.StorageConfig)
		//err = mapstructure.Decode(localconf.ChainMakerConfig.StorageConfig, config)
		if err != nil {
			return err
		}

		//p11Handle, err := localconf.ChainMakerConfig.GetP11Handle()
		err = container.Register(localconf.ChainMakerConfig.GetP11Handle)
		if err != nil {
			return err
		}

		err = container.Register(storeFactory.NewStore,
			container.Parameters(map[int]interface{}{0: bc.chainId, 1: config}),
			container.DependsOn(map[int]string{2: "store"}),
			container.Name(bc.chainId))
		if err != nil {
			return err
		}
		err = container.Resolve(&bc.store, container.ResolveName(bc.chainId))
		if err != nil {
			bc.log.Errorf("new store failed, %s", err.Error())
			return err
		}
		bc.initModules[moduleNameStore] = struct{}{}
		return nil
	}
	// store engine is not store-huge
	if storeEngine != STORE_HUGE {
		storeFactory := store.NewFactory()
		//var storeFactory store.Factory // nolint: typecheck
		storeLogger := logger.GetLoggerByChain(logger.MODULE_STORAGE, bc.chainId)
		err = container.Register(func() protocol.Logger { return storeLogger }, container.Name("store"))
		if err != nil {
			return err
		}
		var config *storeConf.StorageConfig
		config, err = storeConf.NewStorageConfig(localconf.ChainMakerConfig.StorageConfig)
		//err = mapstructure.Decode(localconf.ChainMakerConfig.StorageConfig, config)
		if err != nil {
			return err
		}

		//p11Handle, err := localconf.ChainMakerConfig.GetP11Handle()
		err = container.Register(localconf.ChainMakerConfig.GetP11Handle)
		if err != nil {
			return err
		}

		err = container.Register(storeFactory.NewStore,
			container.Parameters(map[int]interface{}{0: bc.chainId, 1: config}),
			container.DependsOn(map[int]string{2: "store"}),
			container.Name(bc.chainId))
		if err != nil {
			return err
		}
		err = container.Resolve(&bc.store, container.ResolveName(bc.chainId))
		if err != nil {
			bc.log.Errorf("new store failed, %s", err.Error())
			return err
		}
		bc.initModules[moduleNameStore] = struct{}{}
		return nil
	}
	// store engine is store-huge
	storeFactory := storeHuge.NewFactory()
	storeLogger := logger.GetLoggerByChain(logger.MODULE_STORAGE, bc.chainId)
	err = container.Register(func() protocol.Logger { return storeLogger }, container.Name("store"))
	if err != nil {
		return err
	}
	config, err := storeHugeConf.NewStorageConfig(localconf.ChainMakerConfig.StorageConfig)
	//err = mapstructure.Decode(localconf.ChainMakerConfig.StorageConfig, config)
	if err != nil {
		return err
	}

	//p11Handle, err := localconf.ChainMakerConfig.GetP11Handle()
	err = container.Register(localconf.ChainMakerConfig.GetP11Handle)
	if err != nil {
		return err
	}

	err = container.Register(storeFactory.NewStore,
		container.Parameters(map[int]interface{}{0: bc.chainId, 1: config}),
		container.DependsOn(map[int]string{2: "store"}),
		container.Name(bc.chainId))
	if err != nil {
		return err
	}
	err = container.Resolve(&bc.store, container.ResolveName(bc.chainId))
	if err != nil {
		bc.log.Errorf("new store failed, %s", err.Error())
		return err
	}
	bc.initModules[moduleNameStore] = struct{}{}
	return nil
}
