"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""

import config.public_import as gl

def UpdateSetting():
    gl.TESTPROJECTPATH= r"/Users/devinzeng/go/src/chainmaker.org/chainmaker-go/test/chain3/bin"
    gl.CMC_TOOL_PATH = "cd /Users/devinzeng/go/src/chainmaker.org/chainmaker-go/test/chain3/bin && "
    gl.SDK_CONFIG_PATH = r'../config/sdk_config.yml'
    gl.CRYPTO_CONFIG_PATH = r'../config'
    gl.ADMIN_KEY_FILE_PATHS = ','.join([f'{gl.CRYPTO_CONFIG_PATH}/node{i}/user/admin1/admin1.sign.key'
                                 for i in range(1, 4)])
    gl.ADMIN_CRT_FILE_PATHS = ""
    gl.ADMIN_ORG_IDS_PWK = ""
    gl.ADMIN_KEY_FILE_PATHS_BY_PWK = ','.join([f'{gl.CRYPTO_CONFIG_PATH}/wx-org{i}.chainmaker.org/keys/user/admin/admin.key'
                                        for i in range(1, 4)])
    gl.ADMIN_KEY_FILE_PATHS_BY_GAS_PK = ','.join([f'{gl.CRYPTO_CONFIG_PATH}/node1/admin/admin{i}/admin{i}.key'
                                           for i in range(1, 4)])
    gl.ADMIN_KEY_FILE_PATHS_BY_PK = ','.join([f'{gl.CRYPTO_CONFIG_PATH}/node1/admin/admin{i}/admin{i}.key'
                                       for i in range(1, 4)])

    gl.WASM_APTH = r'../../testdata/'
    gl.SDK_PATH = r'../config/'
    gl.ACCOUNT_TYPE = "pk"
    gl.USER_B_KEY = r"node2/admin/admin2/admin2.key"
    gl.ENABLE_GAS = True
