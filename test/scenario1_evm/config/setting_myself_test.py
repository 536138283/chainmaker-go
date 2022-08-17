"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""

00

"""
chainmaker-go工程代码
"""
TESTPROJECTPATH = r"/Users/devinzeng/go/src/chainmaker.org/chainmaker-go/test/chain1/bin"

"""
项目版本
"""
VERSION = "v2.3.0_alpha"
"""
CMC工具配置
"""
# cmc工具位置
CMC_TOOL_PATH = "cd /Users/devinzeng/go/src/chainmaker.org/chainmaker-go/test/chain1/bin && "

# cmc工具背书等配置
SDK_CONFIG_PATH = r'../config/sdk_config.yml'
CRYPTO_CONFIG_PATH = r'../config'
ADMIN_KEY_FILE_PATHS = ','.join([f'{CRYPTO_CONFIG_PATH}/wx-org{i}.chainmaker.org/certs/user/admin1/admin1.sign.key'
                                 for i in range(1, 4)])
ADMIN_CRT_FILE_PATHS = ','.join([f'{CRYPTO_CONFIG_PATH}/wx-org{i}.chainmaker.org/certs/user/admin1/admin1.sign.crt'
                                 for i in range(1, 4)])
ADMIN_ORG_IDS_PWK = ','.join([f'wx-org{i}.chainmaker.org' for i in range(1, 4)])

ADMIN_KEY_FILE_PATHS_BY_PWK = ','.join([f'{CRYPTO_CONFIG_PATH}/wx-org{i}.chainmaker.org/admin/admin.key'
                                        for i in range(1, 4)])

ADMIN_KEY_FILE_PATHS_BY_GAS_PK = ','.join([f'{CRYPTO_CONFIG_PATH}/node1/admin/admin{i}/admin{i}.key'
                                           for i in range(1, 4)])

ADMIN_KEY_FILE_PATHS_BY_PK = ','.join([f'{CRYPTO_CONFIG_PATH}/node1/admin/admin{i}/admin{i}.key'
                                       for i in range(1, 4)])

# WASM_APTH = r'./testdata/claim-wasm-demo/'
WASM_APTH = r'../../testdata/'
SDK_PATH = r'--sdk-conf-path=../config/'
# 公钥身份
# SDK_PATH = r'./testdata/sdk_config_pwk.yml'
