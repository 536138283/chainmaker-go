"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""

import json

from utils.connect_linux import TheServerHelper
import config.public_import as gl


class Command(object):
    BASE_CMD = './cmc '

    def __init__(self, sync_result=True, sdk_config=None):
        self.sync_result = sync_result
        self.new_sdk_config = sdk_config if sdk_config else "sdk_config.yml"
        self.sdk_config_path = f'{gl.SDK_PATH}{self.new_sdk_config}'

    def recharge_gas(self, address, amount=1000000000):
        cmd = gl.CMC_TOOL_PATH + f'{self.BASE_CMD} gas recharge --address={address} --amount={amount} --sdk-conf-path={self.sdk_config_path}'
        if self.sync_result:
            cmd = cmd + " --sync-result=true"
        print(cmd)
        result = TheServerHelper(cmd).ssh_connectionServer()
        print(result)
        return result

