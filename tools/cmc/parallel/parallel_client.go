package parallel

import sdk "chainmaker.org/chainmaker/sdk-go/v2"

// initSubClient 初始化订阅节点使用的sdkClient
func initSubClient() error {
	for i := range hosts {
		sdkClient, err := getSdkClient(i)
		if err != nil {
			return err
		}
		subSdkClients = append(subSdkClients, sdkClient)
	}
	return nil
}

// getSdkClient 用来获取一个SdkClient对象， i为对应node的数组下标
func getSdkClient(i int) (*sdk.ChainClient, error) {
	nodeConf := sdk.NewNodeConfig(
		// 节点地址，格式：127.0.0.1:12301
		sdk.WithNodeAddr(hosts[i]),
		// 节点连接数
		sdk.WithNodeConnCnt(threadNum),
		// 节点是否启用TLS认证
		sdk.WithNodeUseTLS(useTLS),
		// 根证书路径，支持多个
		sdk.WithNodeCAPaths(caPaths),
		// TLS Hostname
		sdk.WithNodeTLSHostName(hostnamesString),
	)
	opts := make([]sdk.ChainClientOption, 0)
	switch sdk.AuthType(authTypeUint32) {
	case sdk.Public:
		opts = append(opts, sdk.WithAuthType(sdk.AuthTypeToStringMap[sdk.AuthType(authTypeUint32)]))
		opts = append(opts, sdk.WithChainClientChainId(chainId))
		opts = append(opts, sdk.WithUserSignKeyFilePath(signKeyPaths[i]))
		opts = append(opts, sdk.WithCryptoConfig(sdk.NewCryptoConfig(sdk.WithHashAlgo(hashAlgo))))
		opts = append(opts, sdk.WithUserKeyFilePath(userKeyPaths[i]))
		opts = append(opts, sdk.WithUserSignCrtFilePath(caPaths[i]))
		if len(encCrtPaths) > 0 && len(encKeyPaths) > 0 {
			opts = append(opts, sdk.WithUserEncKeyBytes(encKeyBytes[i]))
			opts = append(opts, sdk.WithUserEncCrtBytes(encCrtBytes[i]))
		}
		opts = append(opts, sdk.AddChainClientNodeConfig(nodeConf))
	case sdk.PermissionedWithCert:
		opts = append(opts, sdk.WithAuthType(sdk.AuthTypeToStringMap[sdk.AuthType(authTypeUint32)]))
		opts = append(opts, sdk.WithChainClientOrgId(orgIDs[i]))
		opts = append(opts, sdk.WithChainClientChainId(chainId))
		opts = append(opts, sdk.WithUserKeyFilePath(userKeyPaths[i]))
		opts = append(opts, sdk.WithUserCrtFilePath(userCrtPaths[i]))
		opts = append(opts, sdk.WithUserSignCrtFilePath(signCrtPaths[i]))
		opts = append(opts, sdk.WithUserSignKeyFilePath(signKeyPaths[i]))
		opts = append(opts, sdk.AddChainClientNodeConfig(nodeConf))
		if len(encCrtPaths) > 0 && len(encKeyPaths) > 0 {
			opts = append(opts, sdk.WithUserEncKeyBytes(encKeyBytes[i]))
			opts = append(opts, sdk.WithUserEncCrtBytes(encCrtBytes[i]))
		}
	case sdk.PermissionedWithKey:
		opts = append(opts, sdk.WithAuthType(sdk.AuthTypeToStringMap[sdk.AuthType(authTypeUint32)]))
		opts = append(opts, sdk.WithChainClientChainId(chainId))
		opts = append(opts, sdk.WithUserSignKeyFilePath(signKeyPaths[i]))
		opts = append(opts, sdk.WithCryptoConfig(sdk.NewCryptoConfig(sdk.WithHashAlgo(hashAlgo))))
		opts = append(opts, sdk.WithChainClientOrgId(orgIDs[i]))
		opts = append(opts, sdk.AddChainClientNodeConfig(nodeConf))
	}
	return sdk.NewChainClient(opts...)
}
