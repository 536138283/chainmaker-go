"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""
import base64
import json
import sys
import unittest

sys.path.append("..")

import config.public_import as gl
from utils.cmc_tools_contract_deal_by_all import ContractDeal


class Test(unittest.TestCase):
    def test_balance_a_compare_cert(self):

        print("\n","rust asset 合约安装".center(50, "="))
        cd_asset = ContractDeal("asset", sync_result=True)
        result_erc = cd_asset.create("WASMER", "rust-asset-2.0.0.wasm",params=r"{\"issue_limit\":\"10000000\",\"total_supply\":\"1000000000\"}", public_identity=f'{gl.ACCOUNT_TYPE}', sdk_config='sdk_config.yml',endorserKeys=f'{gl.ADMIN_KEY_FILE_PATHS}',endorserCerts=f'{gl.ADMIN_CRT_FILE_PATHS}',endorserOrgs=f'{gl.ADMIN_ORG_IDS}')
        asset_address = json.loads(result_erc).get("contract_result").get("result").get("address")
        print("rust asset 合约地址:",asset_address,"\n")


        print("注册B账户".center(50, "="))
        user_b_address_result = cd_asset.invoke("register", "",
                                                sdk_config="sdk_config.yml",
                                                signkey="wx-org2.chainmaker.org/certs/user/admin1/admin1.sign.key",
                                                signcrt="wx-org2.chainmaker.org/certs/user/admin1/admin1.sign.crt",
                                                org="wx-org2.chainmaker.org")
        user_b_address = base64.b64decode(json.loads(user_b_address_result).get("contract_result").get("result"))


        print("注册C账户".center(50, "="))
        user_c_address_result = cd_asset.invoke("register", "",
                                                sdk_config="sdk_config.yml",
                                                signkey="wx-org3.chainmaker.org/certs/user/admin1/admin1.sign.key",
                                                signcrt="wx-org3.chainmaker.org/certs/user/admin1/admin1.sign.crt",
                                                org="wx-org3.chainmaker.org")
        user_c_address = base64.b64decode(json.loads(user_c_address_result).get("contract_result").get("result"))



        print("query UserA address: org1 admin".center(50, "="))
        user_a_address_result = cd_asset.get("query_address", "", sdk_config="sdk_config2.yml",
                                             signkey="wx-org1.chainmaker.org/certs/user/admin1/admin1.sign.key",
                                             signcrt="wx-org1.chainmaker.org/certs/user/admin1/admin1.sign.crt",
                                             org="wx-org1.chainmaker.org")
        user_a_address = base64.b64decode(json.loads(user_a_address_result).get("contract_result").get("result"))


        print("query UserB address: org2 admin".center(50, "="))
        user_b_address_result2 = cd_asset.get("query_address", "", sdk_config="sdk_config2.yml",
                                             signkey="wx-org2.chainmaker.org/certs/user/admin1/admin1.sign.key",
                                             signcrt="wx-org2.chainmaker.org/certs/user/admin1/admin1.sign.crt",
                                             org="wx-org2.chainmaker.org")
        user_b_address2 = base64.b64decode(json.loads(user_b_address_result2).get("contract_result").get("result"))
        self.assertEqual(user_b_address2, user_b_address, "success")

        print("query UserC address: org3 admin".center(50, "="))
        user_c_address_result2 = cd_asset.get("query_address", "", sdk_config="sdk_config2.yml",
                                              signkey="wx-org3.chainmaker.org/certs/user/admin1/admin1.sign.key",
                                              signcrt="wx-org3.chainmaker.org/certs/user/admin1/admin1.sign.crt",
                                              org="wx-org3.chainmaker.org")
        user_c_address2 = base64.b64decode(json.loads(user_c_address_result2).get("contract_result").get("result"))
        self.assertEqual(user_c_address2, user_c_address, "success")


        print("\n","User A address:",user_a_address,"\n","User B address:",user_b_address2,"\n","User C address:",user_c_address2,"\n")



        print("给A账户增发代币100".center(50, "="))
        cd_asset.invoke("issue_amount", r"{\"amount\":\"100\",\"to\":\"fe2bb3b5b09cb9e506912d605f0d62947ed7154400ce9775113d720239b51f72\"}",
                                                sdk_config="sdk_config.yml",
                                                signkey="",
                                                signcrt="",
                                                org="")

        print("给B账户增发代币100".center(50, "="))
        cd_asset.invoke("issue_amount", r"{\"amount\":\"100\",\"to\":\"9afc94e4343b5d6c1d3017e1cdc4ab3dd953ab5250bd1f2f6f8903037075cd77\"}",
                    sdk_config="sdk_config.yml",
                    signkey="",
                    signcrt="",
                    org="")



        print("A账户给B账户转账10".center(50, "="))
        cd_asset.invoke("transfer", r"{\"amount\":\"10\",\"to\":\"9afc94e4343b5d6c1d3017e1cdc4ab3dd953ab5250bd1f2f6f8903037075cd77\"}",
                        sdk_config="sdk_config.yml",
                        signkey="",
                        signcrt="",
                        org="")



        print("查询A账户余额，应该为90".center(50, "="))
        balance_a_result = cd_asset.get("balance_of",
                                        r"{\"owner\":\"fe2bb3b5b09cb9e506912d605f0d62947ed7154400ce9775113d720239b51f72\"}",
                                        sdk_config="sdk_config2.yml", signkey="", signcrt="", org="")

        balance_user_a = base64.b64decode(json.loads(balance_a_result).get("contract_result").get("result"))
        print("查询结果:A账户余额:",balance_user_a,"\n")




        print("查询B账户余额，应该为110".center(50, "="))
        balance_b_result = cd_asset.get("balance_of",
                                        r"{\"owner\":\"9afc94e4343b5d6c1d3017e1cdc4ab3dd953ab5250bd1f2f6f8903037075cd77\"}",
                                        sdk_config="sdk_config2.yml", signkey="", signcrt="", org="")

        balance_user_b = base64.b64decode(json.loads(balance_b_result).get("contract_result").get("result"))
        print("查询结果：B账户余额",balance_user_b,"\n")




        print("B账户给A账户授权代转账金额为50".center(50, "="))
        cd_asset.invoke("approve", r"{\"amount\":\"50\",\"spender\":\"fe2bb3b5b09cb9e506912d605f0d62947ed7154400ce9775113d720239b51f72\"}",
                        sdk_config="sdk_config.yml",
                        signkey="../config/wx-org2.chainmaker.org/certs/user/admin1/admin1.sign.key",
                        signcrt="../config/wx-org2.chainmaker.org/certs/user/admin1/admin1.sign.crt",
                        org="wx-org2.chainmaker.org")



        print("A账户用B账户授权的代币给C账户转账10".center(50, "="))
        cd_asset.invoke("transfer_from", r"{\"amount\":\"10\",\"from\":\"9afc94e4343b5d6c1d3017e1cdc4ab3dd953ab5250bd1f2f6f8903037075cd77\",\"to\":\"b1887445b97e0bbad8f366c357a8e73cc93812f10fcec372d20313984f62a1fc\"}",
                        sdk_config="sdk_config.yml",
                        signkey="",
                        signcrt="",
                        org="")




        print("查询B账户给A账户授权代转账的余额,应该为40".center(50, "="))
        balance_b_allowance_a_result = cd_asset.get("allowance",
                                        r"{\"spender\":\"fe2bb3b5b09cb9e506912d605f0d62947ed7154400ce9775113d720239b51f72\",\"owner\":\"9afc94e4343b5d6c1d3017e1cdc4ab3dd953ab5250bd1f2f6f8903037075cd77\"}",
                                        sdk_config="sdk_config2.yml", signkey="", signcrt="", org="")

        balance_b_allowance_a = base64.b64decode(json.loads(balance_b_allowance_a_result).get("contract_result").get("result"))
        print("查询结果：B账户给A账户授权的代转账余额:",balance_b_allowance_a)


if __name__ == '__main__':
    unittest.main()
