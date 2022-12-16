package cmd

// ImportLedgerSnapshotCMD import-ledger-snapshot命令的实现
// @return *cobra.Command
//func ImportLedgerSnapshotCMD() *cobra.Command {
//	importSnapshotCmd := &cobra.Command{
//		Use:   "import-ledger-snapshot",
//		Short: "ImportLedgerSnapshot ChainMaker",
//		Long:  "ImportLedgerSnapshot ChainMaker",
//		RunE: func(cmd *cobra.Command, _ []string) error {
//			initLocalConfig(cmd)
//			//backupDbs(rebuildChainId)
//			importSnapshotStart(rebuildChainId, snapshotHeight)
//			fmt.Println("ChainMaker exit")
//			return nil
//		},
//	}
//	attachFlags(importSnapshotCmd, []string{flagNameOfConfigFilepath, flagNameSnapshotHeight})
//	return importSnapshotCmd
//}

//func importSnapshotStart(chainID string, snapshotHeight uint64) {
//	if localconf.ChainMakerConfig.DebugConfig.IsTraceMemoryUsage {
//		traceMemoryUsage()
//	}
//	// set snapshot_height to local conf
//	localconf.ChainMakerConfig.StorageConfig["snapshot_height"] = snapshotHeight
//
//	// init chainmaker server
//	chainMakerServer := blockchain.NewChainMakerServer()
//
//	// init block chain
//	if err := chainMakerServer.InitForImportSnapshot(chainID); err != nil {
//		log.Errorf("init chain make server error,errInfo:[%s]", err)
//	}
//	getBlockchain, err := chainMakerServer.GetBlockchain(chainID)
//	if err != nil {
//		log.Errorf("get block chain error,errInfo:[%s]", err)
//		return
//	}
//	// init storage-ImportLedgerSnapshot and import data
//	if err := getBlockchain.InitForImportSnapshot(); err != nil {
//		log.Errorf("init import snapshot error,errInfo:[%s]", err)
//		return
//	}
//
//	log.Infof("Congratulation, import data success \n")
//}
