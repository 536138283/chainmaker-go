"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""

import json

from config.public_import import *
from utils.connect_linux import TheServerHelper


class ContractQuery(object):
    BASE_CMD = './cmc query'

    def __init__(self, query_param, sdk_config=None):
        self.query_param = query_param
        self.new_sdk_config = sdk_config if sdk_config else "sdk_config.yml"
        self.sdk_config_path = f'{SDK_PATH}{self.new_sdk_config}'

    def query_block_height(self):
        if self.query_param:
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} block-by-height {self.query_param} --chain-id=chain1 {self.sdk_config_path}'
        else:
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} block-by-height --chain-id=chain1 {self.sdk_config_path}'

        print(cmd)
        result = TheServerHelper(cmd).ssh_connectionServer()
        # print(result)
        return result

    def query_block_hash(self):
        cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} block-by-hash {self.query_param} --chain-id=chain1 {self.sdk_config_path}'
        print(cmd)
        result = TheServerHelper(cmd).ssh_connectionServer()
        print(result)
        return result

    def query_block_tx_id(self):
        cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} block-by-txid {self.query_param} --chain-id=chain1 {self.sdk_config_path}'
        print(cmd)
        result = TheServerHelper(cmd).ssh_connectionServer()
        print(result)
        return result

    def query_by_tx(self):
        cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} tx {self.query_param} --chain-id=chain1 {self.sdk_config_path}'
        print(cmd)
        result = TheServerHelper(cmd).ssh_connectionServer()
        print(result)
        return result

    # 指定在某个节点的txid查块儿高
    def query_tx_id_get_height(self):
        result = json.loads(self.query_block_tx_id())
        # print(result)
        block_height = result.get("block").get("header").get("block_height")
        return block_height

    def query_last_height(self):
        result = json.loads(self.query_block_height())
        block_height = result.get("block").get("header").get("block_height")
        print(block_height)
        return block_height


def query_address(org, user):
    cmd = CMC_TOOL_PATH + f"./cmc address cert-to-addr {CRYPTO_CONFIG_PATH}/wx-org{org}.chainmaker.org/certs/user/{user}/{user}.sign.crt"
    print(cmd)
    result = json.loads(TheServerHelper(cmd).ssh_connectionServer())
    print(result)
    return result


if __name__ == "__main__":
    # print(ContractQuery("1").query_block_height())
    # a = ContractQuery("16fa37171eced262ca52fdfc07218265bfe0bd9caa554b5db3c1683ebacae377",
    #                   sdk_config="sdk_config.yml").query_by_tx()
    # print(a)
    # a = ContractQuery("83", sdk_config="sdk_config_pk.yml").query_block_height()
    # print(a)
    # a = ContractQuery("", sdk_config="sdk_config4.yml").query_last_height()
    # b = json.loads(a)
    # print(type(b))
    # base64_decode(b)
    # ContractQuery("47032febf20a40eaa1328de8e25e8f7b00b6a84be3744a968271ceacc9222818").query_block_tx_id()
    # ContractQuery("cab9933710024c99a2609c7c540a84840f02f71607424d239298b579f63da642").query_by_tx()
    query_address("1", "client1")
    query_address("1", "admin1")
