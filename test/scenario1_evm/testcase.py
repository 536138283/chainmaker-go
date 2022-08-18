"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""
import json
import sys
import unittest

sys.path.append("..")

import config.public_import as gl
from utils.cmc_tools_query import get_user_addr, get_user_balance
from utils.cmc_tools_contract_deal_by_all import ContractDeal
from utils.cmc_command import Command


class Test(unittest.TestCase):
    def test_balance_a_compare_pwk(self):
        print("query UserA address: org1 admin".center(50, "="))
        user_a_address = get_user_addr("1", "1")
        print("query UserB address: org2 admin".center(50, "="))
        user_b_address = get_user_addr("2", "2")
        print("query UserC address: org3 admin".center(50, "="))
        user_c_address = get_user_addr("3", "3")
        print("query UserD address: org4 admin".center(50, "="))
        user_d_address = get_user_addr("4", "4")
        if gl.ACCOUNT_TYPE == "pk":
            cmd = Command(sync_result=True)
            cmd.recharge_gas(user_a_address)
            cmd.recharge_gas(user_b_address)
        print("ERC20合约安装".center(50, "="))
        cd_erc = ContractDeal("ERC20", sync_result=True)
        result_erc = cd_erc.create("EVM", "erc20.bin", public_identity=f'{gl.ACCOUNT_TYPE}', sdk_config='sdk_config.yml')
        erc_address = json.loads(result_erc).get("contract_result").get("result").get("address")
        print("ERC20 contract address: ", erc_address)

        print("withdraw合约安装".center(50, "="))
        cd_withdraw = ContractDeal("withdraw", sync_result=True)
        result_withdraw = cd_withdraw.create("EVM", "withdraw.bin", public_identity=f'{gl.ACCOUNT_TYPE}',
                                             sdk_config='sdk_config.yml')
        withdraw_address = json.loads(result_withdraw).get("contract_result").get("result").get("address")
        print("withdraw contract address: ", withdraw_address)



        print(user_a_address, user_b_address, user_c_address, user_d_address)
        print("A转账给B".center(50, "="))
        cd_erc.invoke("transfer", r"[{\"address\": \"%s\"},{\"uint256\": \"100\"}]" % user_b_address,
                      sdk_config="sdk_config.yml",
                      abi="erc20.abi")
        print("A转账给withdraw合约".center(50, "="))
        cd_erc.invoke("transfer", r"[{\"address\": \"%s\"},{\"uint256\": \"200\"}]" % withdraw_address,
                      sdk_config="sdk_config.yml",
                      abi="erc20.abi")

        print("B调用withdraw合约，提款10".center(50, "="))
        cd_withdraw.invoke("withdraw", r"[{\"address\": \"%s\"},{\"uint256\": \"10\"}]" % erc_address,
                           sdk_config="sdk_config.yml",
                           abi="withdraw.abi", signkey=gl.USER_B_KEY,
                           signcrt="wx-org2.chainmaker.org/certs/user/admin1/admin1.sign.crt",
                           org="wx-org2.chainmaker.org")

        print("UserA balance:".center(50, "="))
        balance_a_result = cd_erc.get("balanceOf", r"[{\"address\":\"%s\"}]" % user_a_address,
                                      sdk_config="sdk_config.yml", abi="erc20.abi")
        expect_a = "[999999999999999999999999700]"
        balance_a = get_user_balance(balance_a_result)
        self.assertEqual(expect_a, balance_a, "success")

        print("UserB balance:".center(50, "="))
        balance_b_result = cd_erc.get("balanceOf", r"[{\"address\":\"%s\"}]" % user_b_address,
                                      sdk_config="sdk_config.yml", abi="erc20.abi")
        expect_b = "[110]"
        balance_b = get_user_balance(balance_b_result)
        self.assertEqual(expect_b, balance_b, "success")

        print("withdraw contract balance:".center(50, "="))
        balance_c_result = cd_erc.get("balanceOf", r"[{\"address\":\"%s\"}]" % withdraw_address,
                                      sdk_config="sdk_config.yml", abi="erc20.abi")
        expect_c = "[190]"
        balance_c = get_user_balance(balance_c_result)
        self.assertEqual(expect_c, balance_c, "success")


if __name__ == '__main__':
    unittest.main()
