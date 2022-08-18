"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""

import config.public_import as gl
from utils.connect_linux import TheServerHelper


class ContractDeal(object):
    BASE_CMD = './cmc client contract user'

    def __init__(self, contract_name, sync_result=True):
        """
        :param contract_name: 合约名称
        :param sync_result: 同步参数，默认true，传入false就异步
        """
        self.contract_name = contract_name
        self.sync_result = "true" if sync_result else "false"

    def create(self, runtime, wasm, abi=None, params=None, public_identity=None, sdk_config=None):
        """
        支持创建普通sql合约以及kv合约
        :param runtime: GASM,WASMER,DOCKER_GO,EVM
        :param wasm: 使用的是哪个wasm文件
        :param params: 创建合约的时候的参数
        :param public_identity: 是否选择公钥方式，共有三种模式，pk，pwk，cert模式
        :param sdk_config: 默认是节点1，传入的这个sdk的配置文件表示在哪个节点上跑
        :return:
        """
        wasm_path = gl.WASM_APTH + wasm
        params_new = params if params else "{}"
        new_sdk_config = sdk_config if sdk_config else "sdk_config.yml"
        new_abi = abi if abi else wasm.split(".")[0] + ".abi"
        sdk_config_path = f'{gl.SDK_PATH}{new_sdk_config}'
        if public_identity == "pwk":
            print("合约创建-pwk模式".center(50, "="))
            ADMIN_ORG_KEY = f'--admin-org-ids={gl.ADMIN_ORG_IDS_PWK} --admin-key-file-paths={gl.ADMIN_KEY_FILE_PATHS_BY_PWK}'
            cmd = gl.CMC_TOOL_PATH + f'{self.BASE_CMD} create --abi-file-path={gl.WASM_APTH}{new_abi} --contract-name={self.contract_name} --runtime-type={runtime} --byte-code-path={wasm_path} --version=1.0 --sdk-conf-path={sdk_config_path} {ADMIN_ORG_KEY} --sync-result={self.sync_result}'
        elif public_identity == "pk":
            print("合约创建-pk模式".center(50, "="))
            ADMIN_KEY_FILE_PATHS_BY_PK_ONE = f'--admin-key-file-paths={gl.ADMIN_KEY_FILE_PATHS_BY_PK}'
            cmd = gl.CMC_TOOL_PATH + f'{self.BASE_CMD} create --abi-file-path={gl.WASM_APTH}{new_abi} --contract-name={self.contract_name} --runtime-type={runtime} --byte-code-path={wasm_path} --version=1.0 --sdk-conf-path={sdk_config_path} {ADMIN_KEY_FILE_PATHS_BY_PK_ONE} --sync-result={self.sync_result}'
        else:
            print("合约创建-cert模式".center(50, "="))
            ADMIN_KEY_AND_CRT = f'--admin-key-file-paths={gl.ADMIN_KEY_FILE_PATHS} --admin-crt-file-paths={gl.ADMIN_CRT_FILE_PATHS}'

            if runtime == "EVM":
                cmd = gl.CMC_TOOL_PATH + f'{self.BASE_CMD} create --abi-file-path={gl.WASM_APTH}{new_abi} --contract-name={self.contract_name} --runtime-type={runtime} --byte-code-path={wasm_path} --version=1.0 --sdk-conf-path={sdk_config_path} {ADMIN_KEY_AND_CRT}  --sync-result={self.sync_result} '
            else:
                cmd = gl.CMC_TOOL_PATH + f'{self.BASE_CMD} create --contract-name={self.contract_name} --runtime-type={runtime} --byte-code-path={wasm_path} --version=1.0 --sdk-conf-path={sdk_config_path} {ADMIN_KEY_AND_CRT}  --sync-result={self.sync_result} '
        if gl.ENABLE_GAS:
            cmd = cmd + " --gas-limit=99999999"
        print(cmd)
        result = TheServerHelper(cmd).ssh_connectionServer()
        print(result)
        return result

    def invoke(self, method, params, sdk_config=None, txid=None, abi=None, signkey=None, signcrt=None, org=None):
        """
        调用合约
        :param method: 调用合约方法
        :param params: 调用合约的参数
        :param org: 指定在某个节点上执行
        :return: 返回合约调用结果
        """
        new_sdk_config = sdk_config if sdk_config else "sdk_config.yml"
        sdk_config_path = f'{gl.SDK_PATH}{new_sdk_config}'
        txid = f" --tx-id={txid}" if txid else ""
        new_sign_key = f" --user-signkey-file-path={gl.CRYPTO_CONFIG_PATH}/{signkey} " if signkey else ""
        new_sign_crt = f" --user-signcrt-file-path={gl.CRYPTO_CONFIG_PATH}/{signcrt} " if signkey else " "
        if abi:
            cmd = gl.CMC_TOOL_PATH + f'{self.BASE_CMD} invoke --contract-name={self.contract_name} --abi-file-path={gl.WASM_APTH}{abi} --method={method} --sdk-conf-path={sdk_config_path}{new_sign_key}{new_sign_crt}--params="{params}" --sync-result={self.sync_result}{txid}'
        else:
            cmd = gl.CMC_TOOL_PATH + f'{self.BASE_CMD} invoke --contract-name={self.contract_name} --method={method} --sdk-conf-path={sdk_config_path} --params="{params}" --sync-result={self.sync_result}{txid}'
        orgid =  f" --org-id={org}" if org else ""
        cmd = cmd + orgid
        if gl.ENABLE_GAS:
            cmd = cmd + " --gas-limit=99999999"
        print(cmd)
        result = TheServerHelper(cmd).ssh_connectionServer()
        print(result)
        return result

    def get(self, method, params, sdk_config=None, abi=None):
        """
        查询合约
        :param method: 查询合约的方法
        :param params: 查询合约的参数
        :return:
        """
        new_sdk_config = sdk_config if sdk_config else "sdk_config.yml"
        sdk_config_path = f'{gl.SDK_PATH}{new_sdk_config}'
        if abi:
            cmd = gl.CMC_TOOL_PATH + f'{self.BASE_CMD} get --contract-name={self.contract_name} --abi-file-path={gl.WASM_APTH}{abi} --method={method} --sdk-conf-path={sdk_config_path} --params="{params}"'
        else:
            cmd = gl.CMC_TOOL_PATH + f'{self.BASE_CMD} get --contract-name={self.contract_name} --method={method} --sdk-conf-path={sdk_config_path} --params="{params}"'
        print(cmd)
        result = TheServerHelper(cmd).ssh_connectionServer()
        print(result)
        return result


if __name__ == "__main__":
    # cd = ContractDeal("feifei_test_001", sync_result=True)
    # cd.create("WASMER", "rust-fact-2.0.0.wasm", public_identity='pk', sdk_config='sdk_config_pk.yml')
    # cd.create("WASMER", "rust-fact-2.0.0.wasm")
    # cd.create("WASMER", "chainmaker_contract_no_upgrade.wasm")
    # cd.create("WASMER", "rust-asset-2.1.0.wasm",
    #           params=r'{\"issue_limit\":\"100\",\"total_supply\":\"1000\",\"manager_pk\":\"\"}')
    # print(cd.create("WASMER",r'{\"status\":\"success\",\"tblinfo\":\"CREAE TABLE Persons(Id_P int,LastName varchar(255),FirstName varchar(255),Address varchar(255),City varchar(255))\"}'))
    # cd.invoke("register", '', org=2)

    # cd.invoke("save",
    #                 r'{\"file_name\":\"name007\",\"file_hash\":\"ab3456df5799b87c77e7f88\",\"time\":\"6543234\"}')
    # cd.get("query_address", '')
    # cd.get("balance_of", r'{\"owner\":\"2d0e03297ff63ce802d2b8a71ee8efe17001f6c9da1816cf15540c982849520b\"}')
    # print(cd.upgrade("WASMER", "2.0", "chainmaker_contract.wasm"))
    # print(cd.freeze())
    # print(cd.unfreeze())
    # print(cd.revoke())
    cd = ContractDeal("ERC20", sync_result=True)
    a = cd.get("balanceOf", r"[{\"address\":\"d8551a6f75e0b76cb22d1e0a2770355396f66963\"}]", abi="erc20.abi")
    print(type(a))
