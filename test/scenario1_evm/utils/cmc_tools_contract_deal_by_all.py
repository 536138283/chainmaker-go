"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""



from config.public_import import *
from utils.connect_linux import TheServerHelper


class ContractDeal(object):
    BASE_CMD = './cmc client contract user'

    # 普通的cert模式
    ADMIN_KEY_AND_CRT = f'--admin-key-file-paths={ADMIN_KEY_FILE_PATHS} --admin-crt-file-paths={ADMIN_CRT_FILE_PATHS}'
    # pwk模式
    ADMIN_ORG_KEY = f'--admin-org-ids={ADMIN_ORG_IDS_PWK} --admin-key-file-paths={ADMIN_KEY_FILE_PATHS_BY_PWK}'
    # pk模式
    ADMIN_KEY_FILE_PATHS_BY_PK_ONE = f'--admin-key-file-paths={ADMIN_KEY_FILE_PATHS_BY_PK}'

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
        wasm_path = WASM_APTH + wasm
        params_new = params if params else "{}"
        new_sdk_config = sdk_config if sdk_config else "sdk_config.yml"
        new_abi = abi if abi else wasm.split(".")[0] + ".abi"
        sdk_config_path = f'{SDK_PATH}{new_sdk_config}'
        if public_identity == "pwk":
            print("合约创建-pwk模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} create --contract-name={self.contract_name} --runtime-type={runtime} --byte-code-path={wasm_path} --version=1.0 {sdk_config_path} {self.ADMIN_ORG_KEY} --sync-result={self.sync_result} ' + '--params="%s"' % params_new
        elif public_identity == "pk":
            print("合约创建-pk模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} create --contract-name={self.contract_name} --runtime-type={runtime} --byte-code-path={wasm_path} --version=1.0 {sdk_config_path} {self.ADMIN_KEY_FILE_PATHS_BY_PK_ONE} --sync-result={self.sync_result} ' + '--params="%s"' % params_new
        else:
            print("合约创建-cert模式".center(50, "="))
            if runtime == "EVM":
                cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} create --abi-file-path={WASM_APTH}{new_abi} --contract-name={self.contract_name} --runtime-type={runtime} --byte-code-path={wasm_path} --version=1.0 {sdk_config_path} {self.ADMIN_KEY_AND_CRT}  --sync-result={self.sync_result} '
            else:
                cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} create --contract-name={self.contract_name} --runtime-type={runtime} --byte-code-path={wasm_path} --version=1.0 {sdk_config_path} {self.ADMIN_KEY_AND_CRT}  --sync-result={self.sync_result} ' + '--params="%s"' % params_new
        print(cmd)
        result = TheServerHelper(cmd).ssh_connectionServer()
        print(result)
        return result

    def invoke(self, method, params, sdk_config=None, txid=None, abi=None, signkey=None, signcrt=None):
        """
        调用合约
        :param method: 调用合约方法
        :param params: 调用合约的参数
        :param org: 指定在某个节点上执行
        :return: 返回合约调用结果
        """
        new_sdk_config = sdk_config if sdk_config else "sdk_config.yml"
        sdk_config_path = f'{SDK_PATH}{new_sdk_config}'
        txid = f" --tx-id={txid}" if txid else ""
        new_sign_key = f" --user-signkey-file-path={CRYPTO_CONFIG_PATH}/{signkey} " if signkey else ""
        new_sign_crt = f" --user-signcrt-file-path={CRYPTO_CONFIG_PATH}/{signcrt} " if signkey else " "
        if abi:
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} invoke --contract-name={self.contract_name} --abi-file-path={WASM_APTH}{abi} --method={method} {sdk_config_path}{new_sign_key}{new_sign_crt}--params="{params}" --sync-result={self.sync_result}{txid}'
        else:
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} invoke --contract-name={self.contract_name} --method={method} {sdk_config_path} --params="{params}" --sync-result={self.sync_result}{txid}'
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
        sdk_config_path = f'{SDK_PATH}{new_sdk_config}'
        if abi:
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} get --contract-name={self.contract_name} --abi-file-path={WASM_APTH}{abi} --method={method} {sdk_config_path} --params="{params}"'
        else:
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} get --contract-name={self.contract_name} --method={method} {sdk_config_path} --params="{params}"'
        print(cmd)
        result = TheServerHelper(cmd).ssh_connectionServer()
        print(result)
        return result

    def upgrade(self, runtime, version, wasm, public_identity=None, org=None, params=None, sdk_config=None):
        """
        合约升级
        :param runtime: GASM,WASMER
        :param version: 合约升级的版本
        :param org: 在哪个节点上执行合约
        :return: 返回合约升级的结果
        """
        wasm_path = WASM_APTH + wasm
        params_new = params if params else "{}"
        new_sdk_config = sdk_config if sdk_config else "sdk_config.yml"
        sdk_config_path = f'{SDK_PATH}{new_sdk_config}'
        new_org = org if org else "1"
        if public_identity == "pk":
            print("合约升级-pk模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} upgrade --contract-name={self.contract_name} --runtime-type={runtime} --byte-code-path={wasm_path} --version={version} {sdk_config_path} {self.ADMIN_KEY_FILE_PATHS_BY_PK_ONE} --sync-result={self.sync_result} ' + '--params="%s"' % params_new

        elif public_identity == "pwk":
            print("合约升级-pwk模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} upgrade --contract-name={self.contract_name} --runtime-type={runtime} --byte-code-path={wasm_path} --version={version} {sdk_config_path} {self.ADMIN_ORG_KEY} --org-id=wx-org{new_org}.chainmaker.org --sync-result={self.sync_result} ' + '--params="%s"' % params_new

        else:
            print("合约升级-cert模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} upgrade --contract-name={self.contract_name} --runtime-type={runtime} --byte-code-path={wasm_path} --version={version} {sdk_config_path} {self.ADMIN_KEY_AND_CRT} --org-id=wx-org{new_org}.chainmaker.org --sync-result={self.sync_result} ' + '--params="%s"' % params_new
        print(cmd)
        result = TheServerHelper(cmd).ssh_connectionServer()
        print(result)
        return result

    def freeze(self, org=None, public_identity=None, sdk_config=None):
        """
        合约冻结
        :param org: 在哪个节点上执行合约
        :return: 返回合约冻结结果
        """
        org_new = org if org else "1"
        new_sdk_config = sdk_config if sdk_config else "sdk_config.yml"
        sdk_config_path = f'{SDK_PATH}{new_sdk_config}'
        if public_identity == "pwk":
            print("合约冻结-pwk模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} freeze --contract-name={self.contract_name} {sdk_config_path} {self.ADMIN_ORG_KEY} --org-id=wx-org{org_new}.chainmaker.org --sync-result={self.sync_result}'

        elif public_identity == "pk":
            print("合约冻结-pk模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} freeze --contract-name={self.contract_name} {sdk_config_path} {self.ADMIN_KEY_FILE_PATHS_BY_PK_ONE} --sync-result={self.sync_result}'
        else:
            print("合约冻结-cert模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} freeze --contract-name={self.contract_name} {sdk_config_path} {self.ADMIN_KEY_AND_CRT} --org-id=wx-org{org_new}.chainmaker.org --sync-result={self.sync_result}'
        print(cmd)
        result = TheServerHelper(cmd).ssh_connectionServer()
        print(result)
        return result

    def unfreeze(self, org=None, public_identity=None, sdk_config=None):
        """
        合约解冻
        :param org: 在哪个节点上执行
        :return: 返回合约解冻结果
        """
        org_new = org if org else "1"
        new_sdk_config = sdk_config if sdk_config else "sdk_config.yml"
        sdk_config_path = f'{SDK_PATH}{new_sdk_config}'
        if public_identity == "pwk":
            print("合约解冻-pwk模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} unfreeze --contract-name={self.contract_name} {sdk_config_path} {self.ADMIN_ORG_KEY} --org-id=wx-org{org_new}.chainmaker.org --sync-result={self.sync_result}'
        elif public_identity == "pk":
            print("合约解冻-pk模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} unfreeze --contract-name={self.contract_name} {sdk_config_path} {self.ADMIN_KEY_FILE_PATHS_BY_PK_ONE} --sync-result={self.sync_result}'
        else:
            print("合约解冻-cert模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} unfreeze --contract-name={self.contract_name} {sdk_config_path} {self.ADMIN_KEY_AND_CRT} --org-id=wx-org{org_new}.chainmaker.org --sync-result={self.sync_result}'
        print(cmd)
        result = TheServerHelper(cmd).ssh_connectionServer()
        print(result)
        return result

    def revoke(self, org=None, public_identity=None, sdk_config=None):
        """
        合约吊销
        :param org: 在哪个节点上执行
        :return: 返回吊销合约结果
        """
        org_new = org if org else "1"
        new_sdk_config = sdk_config if sdk_config else "sdk_config.yml"
        sdk_config_path = f'{SDK_PATH}{new_sdk_config}'
        if public_identity == "pwk":
            print("合约吊销-pwk模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} revoke --contract-name={self.contract_name} {sdk_config_path} {self.ADMIN_ORG_KEY} --org-id=wx-org{org_new}.chainmaker.org --sync-result={self.sync_result}'
        elif public_identity == "pk":
            print("合约吊销-pk模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} revoke --contract-name={self.contract_name} {sdk_config_path} {self.ADMIN_KEY_FILE_PATHS_BY_PK_ONE} --sync-result={self.sync_result}'
        else:
            print("合约吊销-cert模式".center(50, "="))
            cmd = CMC_TOOL_PATH + f'{self.BASE_CMD} revoke --contract-name={self.contract_name} {sdk_config_path} {self.ADMIN_KEY_AND_CRT} --org-id=wx-org{org_new}.chainmaker.org --sync-result={self.sync_result}'
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
